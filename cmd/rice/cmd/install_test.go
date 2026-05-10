package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet/rice/internal/state"
)

func resetInstallFlags() {
	flagProfile = ""
	flagYes = false
	flagRepo = "."
	flagState = state.DefaultPath()
	flagLogLevel = ""
}

func setupTestRepo(t *testing.T) (repoRoot, statePath, homeDir string) {
	t.Helper()
	root := t.TempDir()
	repoRoot = filepath.Join(root, "repo")
	homeDir = filepath.Join(root, "home")
	statePath = filepath.Join(root, "state.json")

	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	pkgDir := filepath.Join(repoRoot, "mypkg")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "common", ".config", "mypkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "macbook", ".config", "mypkg"), 0o755))

	manifest := `schema_version = 1
name = "mypkg"
description = "Test package"
supported_os = ["linux", "darwin", "windows"]
target = "$HOME"

[profiles.common]
sources = ["common"]

[profiles.macbook]
sources = ["common", "macbook"]
`
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "rice.toml"), []byte(manifest), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "common", ".config", "mypkg", "base.toml"), []byte("base=true\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "macbook", ".config", "mypkg", "machine.toml"), []byte("machine=true\n"), 0o644))

	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	return
}

func runInstallCmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	// Restore I/O so leaked writers don't affect later tests.
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	return buf.String(), err
}

func TestInstall_WithYesFlag(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)

	out, err := runInstallCmd(t, "",
		"--repo", repoRoot,
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: install mypkg")
	assert.Contains(t, out, "CREATE")

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected symlink at %s", link)
}

func TestInstall_StdinYesProceeds(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)

	out, err := runInstallCmd(t, "y\n",
		"--repo", repoRoot,
		"--state", statePath,
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: install mypkg")

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, err = os.Lstat(link)
	require.NoError(t, err, "symlink should exist")
}

func TestInstall_StdinNoAborts(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)

	out, err := runInstallCmd(t, "n\n",
		"--repo", repoRoot,
		"--state", statePath,
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Aborted.")

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, err = os.Lstat(link)
	assert.True(t, os.IsNotExist(err), "symlink should NOT exist; lstat err=%v", err)
}

func TestInstall_NoArgsErrors(t *testing.T) {
	resetInstallFlags()
	_, err := runInstallCmd(t, "", "install")
	require.Error(t, err)
}

func TestInstall_WithProfileFlag(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)

	out, err := runInstallCmd(t, "",
		"--repo", repoRoot,
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "macbook",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "profile: macbook")

	for _, rel := range []string{".config/mypkg/base.toml", ".config/mypkg/machine.toml"} {
		_, err := os.Lstat(filepath.Join(homeDir, rel))
		assert.NoError(t, err, "expected symlink %s", rel)
	}
}
