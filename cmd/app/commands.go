package main

import (
	"github.com/urfave/cli/v3"
)

func getCommands(version string) []*cli.Command {
	cmds := []*cli.Command{}
	cmds = append(cmds, getSystemCommands(version)...)
	cmds = append(cmds, getKeyCommands()...)
	cmds = append(cmds, getAuthCommands()...)
	return cmds
}
