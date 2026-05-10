package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func installForUninstall(t *testing.T, repoRoot, statePath string) {
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

func TestUninstall_WithYesFlag(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installForUninstall(t, repoRoot, statePath)

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, err := os.Lstat(link)
	require.NoError(t, err, "precondition: link should exist after install")

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"uninstall", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: uninstall mypkg")
	assert.Contains(t, out, "REMOVE")

	_, err = os.Lstat(link)
	assert.True(t, os.IsNotExist(err), "symlink should be removed; lstat err=%v", err)
}

func TestUninstall_StdinYesProceeds(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installForUninstall(t, repoRoot, statePath)

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")

	resetInstallFlags()
	out, err := runInstallCmd(t, "y\n",
		"--state", statePath,
		"uninstall", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: uninstall mypkg")

	_, err = os.Lstat(link)
	assert.True(t, os.IsNotExist(err), "symlink should be removed")
}

func TestUninstall_StdinNoAborts(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installForUninstall(t, repoRoot, statePath)

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")

	resetInstallFlags()
	out, err := runInstallCmd(t, "n\n",
		"--state", statePath,
		"uninstall", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Aborted.")

	_, err = os.Lstat(link)
	require.NoError(t, err, "symlink should still exist after abort")
}

func TestUninstall_NoArgsErrors(t *testing.T) {
	resetInstallFlags()
	_, err := runInstallCmd(t, "", "uninstall")
	require.Error(t, err)
}

func TestUninstall_NotInstalledErrors(t *testing.T) {
	resetInstallFlags()
	_, statePath, _ := setupTestRepo(t)

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"uninstall", "notinstalled",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}
