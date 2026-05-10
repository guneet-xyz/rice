# Folder Mode Symlinks - Learnings

## Implementation Summary
Added safety check in `ExecuteUninstallPlan` to detect when folder symlinks have been replaced with real directories.

## Key Patterns

### 1. Using `os.Lstat` for Symlink Detection
- `os.Lstat` (not `os.Stat`) is required to detect symlinks without following them
- Check: `fi.Mode()&os.ModeSymlink == 0` to detect if NOT a symlink
- This is the correct pattern used in `internal/symlink/symlink.go`

### 2. State Extraction Pattern
- Extract both `Source` and `IsDir` from `InstalledLink` in the same loop
- `IsDir` field exists in `state.InstalledLink` struct (line 15 of state.go)
- Use this flag to conditionally apply safety checks

### 3. Safety Check Flow for Folder Symlinks
```go
if isDir {
    fi, err := os.Lstat(op.Target)
    if err != nil {
        // Handle error (permission denied, etc.)
        logger.Warn("failed to check symlink", ...)
        skipped++
        continue
    }
    
    // Check if it's still a symlink
    if fi.Mode()&os.ModeSymlink == 0 {
        // Replaced with real directory - skip removal
        logger.Warn("folder symlink replaced with real directory, skipping removal", ...)
        skipped++
        continue
    }
}
```

### 4. Import Addition
- Added `"os"` import to access `os.Lstat` and `os.ModeSymlink`
- This is the only new import needed

## Testing
- All existing tests pass with `-race` flag
- No new test cases needed (existing test coverage sufficient)
- Build succeeds with `go build ./...`

## Safety Guarantees
- Never uses `os.RemoveAll` (only `os.Remove` via `symlink.RemoveSymlink`)
- Logs warnings with target path for debugging
- Gracefully skips removal if folder symlink has been replaced
- Maintains exact uninstall semantics (state is still removed)

## [2026-05-11] Prompt rendering: CREATE-DIR / REMOVE-DIR labels

### Implementation
- `internal/prompt/prompt.go`: Updated three functions to render folder-mode ops with distinct labels:
  1. **RenderPlan** (lines 53-72): In the operations table loop, check `op.IsDir` and use `"CREATE-DIR"` instead of `"CREATE"` for OpCreate, and `"REMOVE-DIR"` instead of `"REMOVE"` for OpRemove.
  2. **RenderSwitchPlan** (lines 95-121): Applied same logic to both uninstall phase (lines 100-103) and install phase (lines 114-117).
  3. **RenderConflicts** (lines 137-145): When `c.IsDir == true`, append `" (directory)"` to the reason string before rendering.

### Key details
- File-mode ops (IsDir == false) retain original labels: `"CREATE"`, `"REMOVE"`
- Layout and spacing unchanged — only label strings differ
- Conflicts now show `"CONFLICT  <target>: <reason> (directory)"` for folder conflicts

### Verification
- `go build ./...` ✓ (no errors)
- `go test -race ./internal/prompt/...` ✓ (1.601s)
- LSP diagnostics: clean

### Pattern
The rendering logic is straightforward: conditional label assignment based on `op.IsDir` or `c.IsDir` boolean. No structural changes to the output format.

## [2026-05-11] Test Cases for Folder-Mode Validation

### Added Test Cases to `internal/manifest/validate_test.go`
Added 5 new test cases to the existing `TestValidate` table covering folder-mode validation scenarios:

1. **"folder-mode source with target"** (valid case)
   - SourceSpec: `{Path: "nvim", Mode: "folder", Target: ".config/nvim"}`
   - Expected: No error
   - Validates that folder-mode with non-empty target passes validation

2. **"folder-mode source missing target"** (error case)
   - SourceSpec: `{Path: "nvim", Mode: "folder"}` (no Target)
   - Expected: Error containing "folder-mode requires a non-empty target field"
   - Validates that folder-mode without target is rejected

3. **"unknown mode value"** (error case)
   - SourceSpec: `{Path: "nvim", Mode: "symlink"}` (invalid mode)
   - Expected: Error containing "unknown mode"
   - Validates that only "file" and "folder" modes are accepted

4. **"file-mode source with target field set"** (error case)
   - SourceSpec: `{Path: "config.txt", Mode: "file", Target: ".config"}`
   - Expected: Error containing "target field is only valid for folder-mode"
   - Validates that file-mode rejects target field

5. **"table-form source with mode=file and no target"** (valid case)
   - SourceSpec: `{Path: "config.txt", Mode: "file"}` (explicit mode, no target)
   - Expected: No error
   - Validates that explicit mode="file" without target is valid

### Test Execution
- All 5 new test cases pass with `go test -race ./internal/manifest/...`
- All existing tests continue to pass (no regressions)
- No LSP diagnostics on the modified file

### Key Validation Rules Covered
From `internal/manifest/validate.go` (lines 48-59):
- Empty Mode defaults to "file" (handled by UnmarshalTOML)
- Mode must be "file" or "folder"
- Folder-mode requires non-empty Target
- File-mode must have empty Target
- Unknown modes are rejected with descriptive error

### Test Table Structure
- Tests added before the closing `}` of the tests slice (line 491)
- Follows existing table-driven test pattern
- Each case includes: name, manifest, wantErr, errMsg
- Error messages use `assert.Contains` for substring matching

## Folder-mode unit tests (T-tests)

- `BuildInstallPlan` for folder-mode emits exactly 1 `Op` with `IsDir: true`; Source is the absolute path to the source dir (ends `/cfg`), Target is `<HomeDir>/<spec.Target>` (e.g. `<home>/.config/myfolder`).
- `DetectConflicts` folder-mode behavior:
  - absent target → no conflict
  - real directory at target → conflict ("existing directory")
  - symlink already pointing to our source → no conflict (idempotent)
  - symlink pointing elsewhere → conflict ("symlink points to ...")
- Overlay validation: two folder-mode sources with same target → `BuildInstallPlan` returns error containing `planning error` and a nil plan.
- Fixture pattern: copy `testdata/install/` into a temp dir via `copyDir` (defined in `install_test.go`); use the copied repo as `RepoRoot`. New fixtures added: `folder-pkg/` and `folder-overlay-pkg/`.
- New fixtures must be valid for ALL `Discover` invocations across other tests in the package — keep `supported_os` broad (`linux`, `darwin`, `windows`) so existing tests don't break.

## [2026-05-11] CLI End-to-End Folder Mode Tests

### Test additions
- `cli/install_test.go`: Added `setupFolderTestRepo(t)` helper + `TestInstall_FolderMode_CreatesSingleSymlink`.
- `cli/uninstall_test.go`: Added `installFolderpkg(t,...)` helper + `TestUninstall_FolderMode_RemovesSymlinkOnly` + `TestUninstall_FolderMode_SkipsWhenReplacedWithDir`.
- `cli/switch_test.go`: Added `TestSwitch_FolderModeToFileMode`.

### Manifest pattern used in fixtures
- Folder-mode profile: `sources = [{path = "cfg", mode = "folder", target = ".config/folderpkg"}]`
- File-mode profile (same package): `sources = ["cfg"]` — bare-string form parses to `{Path:"cfg", Mode:"file"}` and walks `cfg/` placing files at `$HOME/<name>` (cfg/init.conf -> $HOME/init.conf).

### Verification of "replaced-with-dir" path
- Install creates folder symlink at `$HOME/.config/folderpkg`.
- `os.Remove(target)` then `os.MkdirAll(target, 0o755)` + write a user file inside.
- Uninstall returns no error (warning logged via `installer.ExecuteUninstallPlan`), and the real dir + user file remain intact.

### Switch folder→file
- Profile A folder symlink at `~/.config/folderpkg` is removed cleanly during switch's uninstall phase.
- Profile B's file-mode walk creates `$HOME/init.conf` as an individual file symlink.
- Verified with `os.Lstat` checking `ModeSymlink` bit AND `IsDir() == false`.
# Folder-mode Symlinks Migration - Learnings

## Completed Tasks

### 1. nvim/rice.toml Migration
- Changed from `sources = ["."]` to folder-mode syntax
- New syntax: `sources = [{path = ".config/nvim", mode = "folder", target = ".config/nvim"}]`
- This ensures nvim's entire `.config/nvim` directory is symlinked as a single unit to `~/.config/nvim`
- Verified: `nvim/.config/nvim/` directory exists with actual config files

### 2. AGENTS.md Documentation Update
- Updated the `profiles.<name>.sources` field description in the schema table to mention both string and table forms
- Added new "Folder-mode sources" subsection after the sources explanation
- Documented:
  - When to use: tools that need their config dir to be a single symlink (nvim, opencode)
  - Syntax with example
  - Constraints: cannot be overlaid, requires both `path` and `target`
  - Use cases

### 3. Verification
- `go build ./...` passes
- `go test -race ./...` passes (all 10 packages tested, no failures)
- No Go source or test files were modified
- No other dotfile packages were affected

## Key Insights

1. **Folder-mode vs String sources**: String sources (e.g., `"common"`) are overlaid; folder-mode sources are single symlinks
2. **Path vs Target**: `path` is relative to package dir; `target` is relative to `$HOME` (or the `target` root)
3. **nvim use case**: Perfect candidate for folder-mode because nvim manages its entire config directory as a unit
4. **Documentation placement**: The new subsection fits naturally after the sources explanation and before Profile Conventions

## No Issues Encountered
- All changes applied cleanly
- Tests passed without modification
- Build succeeded without errors
