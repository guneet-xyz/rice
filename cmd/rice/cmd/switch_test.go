package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func installCommonForSwitch(t *testing.T, repoRoot, statePath string) {
	t.Helper()
	out, err := runInstallCmd(t, "",
		"--repo", repoRoot,
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "setup install failed: out=%s", out)
}

func TestSwitch_WithYesFlag(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--repo", repoRoot,
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "macbook",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: uninstall mypkg")
	assert.Contains(t, out, "Plan: install mypkg")
	assert.Contains(t, out, "REMOVE")
	assert.Contains(t, out, "CREATE")

	machineLink := filepath.Join(homeDir, ".config", "mypkg", "machine.toml")
	fi, err := os.Lstat(machineLink)
	require.NoError(t, err, "expected machine.toml symlink after switch")
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected symlink")
}

func TestSwitch_StdinYesProceeds(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	resetInstallFlags()
	out, err := runInstallCmd(t, "y\n",
		"--repo", repoRoot,
		"--state", statePath,
		"switch", "mypkg", "macbook",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: install mypkg")

	machineLink := filepath.Join(homeDir, ".config", "mypkg", "machine.toml")
	_, err = os.Lstat(machineLink)
	require.NoError(t, err, "expected machine.toml symlink after switch")
}

func TestSwitch_StdinNoAborts(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	resetInstallFlags()
	out, err := runInstallCmd(t, "n\n",
		"--repo", repoRoot,
		"--state", statePath,
		"switch", "mypkg", "macbook",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Aborted.")

	machineLink := filepath.Join(homeDir, ".config", "mypkg", "machine.toml")
	_, err = os.Lstat(machineLink)
	assert.True(t, os.IsNotExist(err), "machine.toml should not exist after abort")

	baseLink := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, err = os.Lstat(baseLink)
	require.NoError(t, err, "base.toml should still exist after abort")
}

func TestSwitch_NoArgsErrors(t *testing.T) {
	resetInstallFlags()
	_, err := runInstallCmd(t, "", "switch")
	require.Error(t, err)
}

func TestSwitch_OneArgErrors(t *testing.T) {
	resetInstallFlags()
	_, err := runInstallCmd(t, "", "switch", "mypkg")
	require.Error(t, err)
}

func TestSwitch_NotInstalledErrors(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, _ := setupTestRepo(t)

	_, err := runInstallCmd(t, "",
		"--repo", repoRoot,
		"--state", statePath,
		"--yes",
		"switch", "notinstalled", "macbook",
	)
	require.Error(t, err)
}
