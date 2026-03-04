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

	expectedCmds := []string{
		"server",
		"migrate",
		"migrate-down",
		"clean-audit-logs",
		"purge-secrets",
		"purge-transit-keys",
		"purge-tokenization-keys",
		"verify-audit-logs",
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

	// Basic flag checks for specific commands
	for _, cmd := range cmds {
		switch cmd.Name {
		case "migrate-down":
			require.NotEmpty(t, cmd.Flags)
			require.Equal(t, "steps", cmd.Flags[0].Names()[0])
		case "purge-secrets", "purge-transit-keys", "purge-tokenization-keys", "clean-audit-logs":
			require.NotEmpty(t, cmd.Flags)
			// check that they have a --days flag
			hasDaysFlag := false
			for _, flag := range cmd.Flags {
				if flag.Names()[0] == "days" {
					hasDaysFlag = true
					break
				}
			}
			require.True(t, hasDaysFlag, "command %s missing --days flag", cmd.Name)
		}
	}
}
