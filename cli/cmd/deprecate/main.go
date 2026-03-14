package main

import (
	"os"

	"github.com/agud97/idp-platform/cli/internal/cliapp"
)

func main() {
	os.Exit(cliapp.Execute("deprecate"))
}

