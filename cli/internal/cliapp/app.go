package cliapp

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/agud97/idp-platform/cli/pkg/converter"
	"github.com/agud97/idp-platform/cli/pkg/exporter"
	"github.com/agud97/idp-platform/cli/pkg/validator"
	"github.com/agud97/idp-platform/cli/internal/tracker"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// Execute runs a single command entrypoint for the requested subcommand.
func Execute(name string) int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return executeCommand(newNamedCommand(name, logger))
}

// ExecuteRoot runs the multi-command idp entrypoint.
func ExecuteRoot() int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	root := &cobra.Command{
		Use:   "idp",
		Short: "IDP environment migration CLI",
	}

	root.AddCommand(
		NewMigrateCommand(logger),
		NewExportCommand(logger),
		NewValidateCommand(logger),
		NewRegisterLegacyCommand(logger),
		NewCompleteMigrationCommand(logger),
		NewDeprecateLegacyCommand(logger),
	)

	return executeCommand(root)
}

func executeCommand(cmd *cobra.Command) int {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

func newNamedCommand(name string, logger *slog.Logger) *cobra.Command {
	switch name {
	case "migrate":
		return NewMigrateCommand(logger)
	case "export":
		return NewExportCommand(logger)
	case "validate":
		return NewValidateCommand(logger)
	case "register":
		return NewRegisterLegacyCommand(logger)
	case "complete":
		return NewCompleteMigrationCommand(logger)
	case "deprecate":
		return NewDeprecateLegacyCommand(logger)
	default:
		return &cobra.Command{
			Use:   name,
			Short: fmt.Sprintf("%s command placeholder", name),
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Help()
			},
		}
	}
}

func NewMigrateCommand(logger *slog.Logger) *cobra.Command {
	var fromPath string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Convert a docker-compose file into an Environment manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFile(fromPath); err != nil {
				return err
			}

			input, err := os.Open(filepath.Clean(fromPath))
			if err != nil {
				return fileError(err)
			}
			defer input.Close()

			manifest, report, err := converter.NewComposeConverter().Convert(input)
			if err != nil {
				logger.Error("compose conversion failed", "file", fromPath, "error", err)
				return err
			}

			body, err := converter.MarshalYAML(manifest)
			if err != nil {
				logger.Error("manifest serialisation failed", "file", fromPath, "error", err)
				return err
			}

			output := append([]byte("# Source: idp-migrate\n"), body...)
			if err := os.WriteFile(filepath.Clean(outputPath), output, 0o644); err != nil {
				logger.Error("manifest write failed", "file", outputPath, "error", err)
				return err
			}

			reportYAML, err := yaml.Marshal(report)
			if err != nil {
				logger.Error("report serialisation failed", "file", fromPath, "error", err)
				return err
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), string(reportYAML))
			return err
		},
	}

	cmd.Flags().StringVar(&fromPath, "from", "", "path to docker-compose file")
	cmd.Flags().StringVar(&outputPath, "output", "", "path to write the Environment manifest")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("output")

	return cmd
}

func NewExportCommand(logger *slog.Logger) *cobra.Command {
	var envName string
	var team string
	var outputPath string
	var namespace string
	var redact bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export legacy environment configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if stringsTrimmedEmpty(envName, team, outputPath) {
				return fmt.Errorf("required flags are missing")
			}

			client := unsupportedLegacyClient{team: team}
			config, summary, err := exporter.NewLegacyExporter(client).Export(envName, exporter.ExportOptions{
				Namespace: namespace,
				Redact:    redact,
			})
			if err != nil {
				logger.Error("legacy export failed", "environment", envName, "team", team, "error", err)
				return err
			}

			configYAML, err := yaml.Marshal(config)
			if err != nil {
				logger.Error("legacy config serialisation failed", "environment", envName, "error", err)
				return err
			}

			if err := os.WriteFile(filepath.Clean(outputPath), configYAML, 0o644); err != nil {
				logger.Error("legacy config write failed", "file", outputPath, "error", err)
				return err
			}

			summaryYAML, err := yaml.Marshal(summary)
			if err != nil {
				logger.Error("export summary serialisation failed", "environment", envName, "error", err)
				return err
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), string(summaryYAML))
			return err
		},
	}

	cmd.Flags().StringVar(&envName, "name", "", "legacy environment name")
	cmd.Flags().StringVar(&team, "team", "", "team owning the legacy environment")
	cmd.Flags().StringVar(&outputPath, "output", "", "path to write exported configuration")
	cmd.Flags().StringVar(&namespace, "namespace", "", "legacy environment namespace")
	cmd.Flags().BoolVar(&redact, "redact", true, "redact secret values in exported output")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("team")
	_ = cmd.MarkFlagRequired("output")

	return cmd
}

func NewValidateCommand(logger *slog.Logger) *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate an Environment manifest against the CRD schema and business rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFile(filePath); err != nil {
				return err
			}

			raw, err := os.ReadFile(filepath.Clean(filePath))
			if err != nil {
				return fileError(err)
			}

			var manifest converter.EnvironmentManifest
			if err := yaml.Unmarshal(raw, &manifest); err != nil {
				logger.Error("manifest parse failed", "file", filePath, "error", err)
				return err
			}

			envValidator, err := validator.NewEnvironmentValidator()
			if err != nil {
				logger.Error("validator initialisation failed", "file", filePath, "error", err)
				return err
			}

			errs := envValidator.Validate(&manifest)
			if len(errs) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Valid")
				return err
			}

			for _, validationErr := range errs {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", validationErr.Path, validationErr.Message); err != nil {
					return err
				}
			}
			return fmt.Errorf("manifest validation failed")
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "path to Environment manifest")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func requireFile(path string) error {
	if stringsTrimmedEmpty(path) {
		return fmt.Errorf("required flags are missing")
	}
	return nil
}

func fileError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("File not found")
	}
	return err
}

func stringsTrimmedEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

type unsupportedLegacyClient struct {
	team string
}

func (c unsupportedLegacyClient) GetComposeConfig(string, exporter.ExportOptions) (*exporter.LegacyConfig, error) {
	return nil, fmt.Errorf("legacy export client is not configured for team %q", c.team)
}

func (c unsupportedLegacyClient) InspectRuntime(string, exporter.ExportOptions) (*exporter.LegacyConfig, error) {
	return nil, fmt.Errorf("legacy runtime inspection client is not configured for team %q", c.team)
}

func (c unsupportedLegacyClient) ListInaccessibleServices(string, exporter.ExportOptions) ([]string, error) {
	return nil, nil
}

func NewRegisterLegacyCommand(logger *slog.Logger) *cobra.Command {
	var name string
	var team string
	var composePath string

	cmd := &cobra.Command{
		Use:   "register-legacy",
		Short: "Register a legacy environment in the migration tracker",
		RunE: func(cmd *cobra.Command, args []string) error {
			if stringsTrimmedEmpty(name, team) {
				return fmt.Errorf("required flags are missing")
			}

			store, err := openTracker()
			if err != nil {
				logger.Error("tracker open failed", "error", err)
				return err
			}
			defer store.Close()

			record, err := store.RegisterLegacyEnvironment(team, name, composePath)
			if err != nil {
				logger.Error("legacy registration failed", "team", team, "name", name, "error", err)
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Registered legacy environment %q for team %q with id %d\n", record.Name, record.TeamName, record.ID)
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "legacy environment name")
	cmd.Flags().StringVar(&team, "team", "", "team owning the legacy environment")
	cmd.Flags().StringVar(&composePath, "compose-path", "", "path to docker-compose file if available")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("team")

	return cmd
}

func NewCompleteMigrationCommand(logger *slog.Logger) *cobra.Command {
	var envID int64

	cmd := &cobra.Command{
		Use:   "complete-migration",
		Short: "Mark a validated legacy environment migration as completed",
		RunE: func(cmd *cobra.Command, args []string) error {
			if envID == 0 {
				return fmt.Errorf("required flags are missing")
			}

			store, err := openTracker()
			if err != nil {
				logger.Error("tracker open failed", "error", err)
				return err
			}
			defer store.Close()

			record, err := store.CompleteMigration(envID)
			if err != nil {
				logger.Error("complete migration failed", "env_id", envID, "error", err)
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Migration completed for environment %d (%s/%s)\n", record.ID, record.TeamName, record.Name)
			return err
		},
	}

	cmd.Flags().Int64Var(&envID, "env-id", 0, "legacy environment identifier")
	_ = cmd.MarkFlagRequired("env-id")

	return cmd
}

func NewDeprecateLegacyCommand(logger *slog.Logger) *cobra.Command {
	var dryRun bool
	var confirm bool

	cmd := &cobra.Command{
		Use:   "deprecate-legacy",
		Short: "Evaluate or execute the legacy platform deprecation gate",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun && confirm {
				return fmt.Errorf("use either --dry-run or --confirm")
			}

			store, err := openTracker()
			if err != nil {
				logger.Error("tracker open failed", "error", err)
				return err
			}
			defer store.Close()

			records, err := store.ListLegacyEnvironments()
			if err != nil {
				logger.Error("load legacy environments failed", "error", err)
				return err
			}
			if len(records) == 0 {
				return fmt.Errorf("No legacy environments registered. Register environments first with `idp register-legacy` before initiating deprecation.")
			}

			summary, err := store.SummarizeLegacyEnvironments()
			if err != nil {
				logger.Error("summarise legacy environments failed", "error", err)
				return err
			}

			effectiveDryRun := dryRun || !confirm
			if err := printGateSummary(cmd.OutOrStdout(), summary); err != nil {
				return err
			}

			blockers := blockingEnvironments(records)
			if len(blockers) > 0 {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Gate FAILED — deprecation blocked."); err != nil {
					return err
				}
				for _, blocker := range blockers {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s | %s | %s\n", blocker.Name, blocker.TeamName, blocker.MigrationStatus); err != nil {
						return err
					}
				}
				return fmt.Errorf("deprecation gate blocked")
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Gate PASSED — all %d environments completed.\n", summary.Total); err != nil {
				return err
			}
			if effectiveDryRun {
				return nil
			}

			reportPath, err := writeDeprecationReport(records)
			if err != nil {
				logger.Error("write deprecation report failed", "error", err)
				return err
			}
			if err := store.SetDeprecatedFlag("true"); err != nil {
				logger.Error("set deprecated flag failed", "error", err)
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deprecation report written to %s\n", reportPath)
			return err
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview the deprecation gate")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "write the deprecation report and set the deprecated flag")

	return cmd
}

func openTracker() (*tracker.Store, error) {
	path, err := tracker.DefaultDBPath()
	if err != nil {
		return nil, err
	}
	return tracker.Open(path)
}

func blockingEnvironments(records []tracker.LegacyEnvironment) []tracker.LegacyEnvironment {
	var blockers []tracker.LegacyEnvironment
	for _, record := range records {
		if record.MigrationStatus != "completed" {
			blockers = append(blockers, record)
		}
	}
	return blockers
}

func printGateSummary(w io.Writer, summary tracker.StatusSummary) error {
	_, err := fmt.Fprintf(
		w,
		"total=%d completed=%d validated=%d in_progress=%d pending=%d blocked=%d\n",
		summary.Total,
		summary.Completed,
		summary.Validated,
		summary.InProgress,
		summary.Pending,
		summary.Blocked,
	)
	return err
}

func writeDeprecationReport(records []tracker.LegacyEnvironment) (string, error) {
	reportDir := os.Getenv("IDP_REPORT_DIR")
	if reportDir == "" {
		reportDir = "."
	}

	filename := fmt.Sprintf("deprecation-report-%s.md", time.Now().UTC().Format("2006-01-02"))
	reportPath := filepath.Join(reportDir, filename)

	operator := os.Getenv("USER")
	if operator == "" {
		if currentUser, err := user.Current(); err == nil {
			operator = currentUser.Username
		}
	}
	if operator == "" {
		operator = "unknown"
	}

	var builder strings.Builder
	builder.WriteString("# Legacy Platform Deprecation Report\n\n")
	builder.WriteString(fmt.Sprintf("- Generated at: %s\n", time.Now().UTC().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Operator: %s\n", operator))
	builder.WriteString(fmt.Sprintf("- Total environments: %d\n\n", len(records)))
	builder.WriteString("| Team | Environment | Completed At | Target Environment ID |\n")
	builder.WriteString("|------|-------------|--------------|-----------------------|\n")
	for _, record := range records {
		completedAt := ""
		if record.CompletedAt.Valid {
			completedAt = record.CompletedAt.Time.Format(time.RFC3339)
		}
		target := ""
		if record.TargetEnvironmentID.Valid {
			target = fmt.Sprintf("%d", record.TargetEnvironmentID.Int64)
		}
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", record.TeamName, record.Name, completedAt, target))
	}

	if err := os.WriteFile(filepath.Clean(reportPath), []byte(builder.String()), 0o644); err != nil {
		return "", err
	}
	return reportPath, nil
}
