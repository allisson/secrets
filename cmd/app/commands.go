// Package main provides the CLI command definitions for the application.
package main

import (
	"github.com/urfave/cli/v3"
)

// getCommands aggregates and returns all CLI commands for the application.
func getCommands(version string) []*cli.Command {
	cmds := []*cli.Command{}
	cmds = append(cmds, getSystemCommands(version)...)
	cmds = append(cmds, getKeyCommands()...)
	cmds = append(cmds, getAuthCommands()...)
	return cmds
}
