package tracker

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultDBRelativePath = ".idp/tracker.db"
	deprecatedFlagKey     = "LEGACY_PLATFORM_DEPRECATED"
)

type Team struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

type LegacyEnvironment struct {
	ID                  int64
	TeamID              int64
	TeamName            string
	Name                string
	ComposeFilePath     string
	MigrationStatus     string
	TargetEnvironmentID sql.NullInt64
	CreatedAt           time.Time
	CompletedAt         sql.NullTime
}

type StatusSummary struct {
	Total      int
	Pending    int
	InProgress int
	Validated  int
	Completed  int
	Blocked    int
}

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func DefaultDBPath() (string, error) {
	if override := os.Getenv("IDP_TRACKER_DB"); override != "" {
		return filepath.Clean(override), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	return filepath.Join(home, defaultDBRelativePath), nil
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("tracker path is required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create tracker directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open tracker database: %w", err)
	}

	store := &Store{
		db:  db,
		now: func() time.Time { return time.Now().UTC() },
	}
	if err := store.init(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) init() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS legacy_environments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			compose_file_path TEXT,
			migration_status TEXT NOT NULL,
			target_environment_id INTEGER,
			created_at TEXT NOT NULL,
			completed_at TEXT,
			FOREIGN KEY(team_id) REFERENCES teams(id),
			UNIQUE(team_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS tracker_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("initialise tracker schema: %w", err)
		}
	}
	return nil
}

func (s *Store) CreateTeam(name string) (*Team, error) {
	now := s.now()
	res, err := s.db.Exec(`INSERT INTO teams(name, created_at) VALUES (?, ?)`, name, now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read team id: %w", err)
	}

	return &Team{ID: id, Name: name, CreatedAt: now}, nil
}

func (s *Store) FindTeamByName(name string) (*Team, error) {
	var team Team
	var createdAt string
	err := s.db.QueryRow(`SELECT id, name, created_at FROM teams WHERE name = ?`, name).Scan(&team.ID, &team.Name, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find team: %w", err)
	}

	team.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse team timestamp: %w", err)
	}

	return &team, nil
}

func (s *Store) RegisterLegacyEnvironment(teamName, envName, composePath string) (*LegacyEnvironment, error) {
	team, err := s.FindTeamByName(teamName)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, fmt.Errorf("Team %q not found", teamName)
	}

	existing, err := s.FindLegacyEnvironmentByName(team.ID, envName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("Legacy environment %q is already registered for team %q", envName, teamName)
	}

	now := s.now()
	res, err := s.db.Exec(
		`INSERT INTO legacy_environments(team_id, name, compose_file_path, migration_status, created_at) VALUES (?, ?, ?, ?, ?)`,
		team.ID,
		envName,
		nullIfEmpty(composePath),
		"pending",
		now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("register legacy environment: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read legacy environment id: %w", err)
	}

	record, err := s.GetLegacyEnvironment(id)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Store) FindLegacyEnvironmentByName(teamID int64, envName string) (*LegacyEnvironment, error) {
	rows, err := s.queryLegacyEnvironments(`WHERE le.team_id = ? AND le.name = ?`, teamID, envName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	record, err := scanLegacyEnvironment(rows)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Store) GetLegacyEnvironment(id int64) (*LegacyEnvironment, error) {
	rows, err := s.queryLegacyEnvironments(`WHERE le.id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("Legacy environment %d not found", id)
	}

	return scanLegacyEnvironment(rows)
}

func (s *Store) SetMigrationStatus(id int64, status string) error {
	var completedAt any
	if status == "completed" {
		completedAt = s.now().Format(time.RFC3339)
	}

	result, err := s.db.Exec(`UPDATE legacy_environments SET migration_status = ?, completed_at = ? WHERE id = ?`, status, completedAt, id)
	if err != nil {
		return fmt.Errorf("update migration status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated rows: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("Legacy environment %d not found", id)
	}

	return nil
}

func (s *Store) CompleteMigration(id int64) (*LegacyEnvironment, error) {
	record, err := s.GetLegacyEnvironment(id)
	if err != nil {
		return nil, err
	}
	if record.MigrationStatus != "validated" {
		return nil, fmt.Errorf("environment %d has status %q; complete validation first", id, record.MigrationStatus)
	}

	now := s.now().Format(time.RFC3339)
	if _, err := s.db.Exec(`UPDATE legacy_environments SET migration_status = 'completed', completed_at = ? WHERE id = ?`, now, id); err != nil {
		return nil, fmt.Errorf("complete migration: %w", err)
	}

	return s.GetLegacyEnvironment(id)
}

func (s *Store) ListLegacyEnvironments() ([]LegacyEnvironment, error) {
	rows, err := s.queryLegacyEnvironments(`ORDER BY t.name, le.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LegacyEnvironment
	for rows.Next() {
		record, err := scanLegacyEnvironment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy environments: %w", err)
	}
	return out, nil
}

func (s *Store) SummarizeLegacyEnvironments() (StatusSummary, error) {
	records, err := s.ListLegacyEnvironments()
	if err != nil {
		return StatusSummary{}, err
	}

	var summary StatusSummary
	summary.Total = len(records)
	for _, record := range records {
		switch record.MigrationStatus {
		case "pending":
			summary.Pending++
		case "in_progress":
			summary.InProgress++
		case "validated":
			summary.Validated++
		case "completed":
			summary.Completed++
		case "blocked":
			summary.Blocked++
		}
	}
	return summary, nil
}

func (s *Store) SetDeprecatedFlag(value string) error {
	_, err := s.db.Exec(
		`INSERT INTO tracker_config(key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		deprecatedFlagKey,
		value,
		s.now().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("set deprecated flag: %w", err)
	}
	return nil
}

func (s *Store) DeprecatedFlag() (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM tracker_config WHERE key = ?`, deprecatedFlagKey).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read deprecated flag: %w", err)
	}
	return value, nil
}

func (s *Store) queryLegacyEnvironments(whereClause string, args ...any) (*sql.Rows, error) {
	query := `SELECT le.id, le.team_id, t.name, le.name, COALESCE(le.compose_file_path, ''), le.migration_status,
		le.target_environment_id, le.created_at, le.completed_at
		FROM legacy_environments le
		JOIN teams t ON t.id = le.team_id ` + whereClause

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query legacy environments: %w", err)
	}
	return rows, nil
}

func scanLegacyEnvironment(scanner interface{ Scan(dest ...any) error }) (*LegacyEnvironment, error) {
	var record LegacyEnvironment
	var createdAt string
	var completedAt sql.NullString
	if err := scanner.Scan(
		&record.ID,
		&record.TeamID,
		&record.TeamName,
		&record.Name,
		&record.ComposeFilePath,
		&record.MigrationStatus,
		&record.TargetEnvironmentID,
		&createdAt,
		&completedAt,
	); err != nil {
		return nil, fmt.Errorf("scan legacy environment: %w", err)
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	record.CreatedAt = parsedCreatedAt
	if completedAt.Valid {
		parsedCompletedAt, err := time.Parse(time.RFC3339, completedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse completed_at: %w", err)
		}
		record.CompletedAt = sql.NullTime{Time: parsedCompletedAt, Valid: true}
	}
	return &record, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
