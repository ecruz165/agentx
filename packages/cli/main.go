package main

import (
	"os"

	"github.com/agentx-labs/agentx/internal/cli"
)

// version, commit, and date are set via ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := cli.Execute(version, commit, date); err != nil {
		os.Exit(1)
	}
}
