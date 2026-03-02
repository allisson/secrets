package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCommands(t *testing.T) {
	version := "v0.1.0"
	cmds := getCommands(version)

	require.NotEmpty(t, cmds)

	// Check if key commands exist
	foundCreateMasterKey := false
	for _, cmd := range cmds {
		if cmd.Name == "create-master-key" {
			foundCreateMasterKey = true
			break
		}
	}
	require.True(t, foundCreateMasterKey)
}

func TestGetAuthCommands(t *testing.T) {
	cmds := getAuthCommands()
	require.NotEmpty(t, cmds)

	expectedCmds := []string{"clean-expired-tokens", "create-client", "update-client"}
	for _, name := range expectedCmds {
		found := false
		for _, cmd := range cmds {
			if cmd.Name == name {
				found = true
				break
			}
		}
		require.Truef(t, found, "command %s not found", name)
	}
}

func TestGetKeyCommands(t *testing.T) {
	cmds := getKeyCommands()
	require.NotEmpty(t, cmds)

	expectedCmds := []string{
		"create-master-key",
		"rotate-master-key",
		"create-kek",
		"rotate-kek",
		"rewrap-deks",
		"create-tokenization-key",
		"rotate-tokenization-key",
	}
	for _, name := range expectedCmds {
		found := false
		for _, cmd := range cmds {
			if cmd.Name == name {
				found = true
				break
			}
		}
		require.Truef(t, found, "command %s not found", name)
	}
}

func TestGetSystemCommands(t *testing.T) {
	version := "v0.1.0"
	cmds := getSystemCommands(version)
	require.NotEmpty(t, cmds)

	expectedCmds := []string{"server", "migrate", "clean-audit-logs", "verify-audit-logs"}
	for _, name := range expectedCmds {
		found := false
		for _, cmd := range cmds {
			if cmd.Name == name {
				found = true
				break
			}
		}
		require.Truef(t, found, "command %s not found", name)
	}
}
