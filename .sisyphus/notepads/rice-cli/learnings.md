# Rice CLI — Learnings

## Project Conventions
- Module path: github.com/guneet/rice (placeholder — agent should use this)
- Go version: 1.22+ (use range-over-int, slices package)
- All internal packages under internal/
- CLI entry point: cmd/rice/main.go
- Test fixtures: testdata/ subdirs per package
- Commit style: conventional commits (feat/fix/refactor/docs/chore/test)

## Key Design Decisions
- File-level symlinking only (no directory folding)
- sources = [...] list in rice.toml — NO implicit "common" base
- State file is authoritative for "what rice installed"
- Conflict policy: ABORT on any conflict (no --force)
- Switch = full uninstall + reinstall (not diff-based)
- Confirmation prompt: default NO on bare Enter; --yes bypasses prompt but still prints plan
- Logging: STDERR for console (default WARN), always-DEBUG JSON to ~/.config/rice/logs/rice.log
- Stdout reserved for command output (plan, status table, summaries)

## Task 1: Go Module Init + Scaffolding (COMPLETED)

### Execution Summary
- Module initialized: github.com/guneet/rice (go 1.22)
- Dependencies added: cobra, toml, testify, zap
- All internal packages created with placeholder doc.go files
- cmd/rice/main.go → cmd/rice/cmd/root.go structure established
- Build verified: `go build ./...` passes
- Tests verified: `go test ./...` passes (no tests yet)
- Binary builds: `go build ./cmd/rice` produces executable

### Key Artifacts
- go.mod / go.sum: Dependency management
- cmd/rice/main.go: Entry point calling cmd.Execute()
- cmd/rice/cmd/root.go: Stub with Execute() function
- internal/{manifest,symlink,state,profile,installer,doctor,logger,plan,prompt}/doc.go: Package stubs
- testdata/.gitkeep: Test fixtures directory
- .gitignore: Updated with /rice, /dist/, *.test, coverage.out, .sisyphus/evidence/

### Conventions Established
- All internal packages use doc.go pattern for package documentation
- CLI entry point follows cobra convention (cmd.Execute())
- Commit message: conventional commits format (chore: ...)

## Task 2: rice.toml Schema Definition (COMPLETED)

### Execution Summary
- Manifest struct: SchemaVersion, Name, Description, SupportedOS, Target, ProfileKey, Profiles
- ProfileDef struct: Sources (list of relative paths)
- Validate() function: 7 validation rules enforced
- Test coverage: 37 table-driven test cases covering all rules (positive + negative)
- All tests pass with race detector enabled
- Build verified: `go build ./...` passes
- Commit: feat(manifest): add rice.toml schema structs and validation

### Key Implementation Details
- SchemaVersion: Must be exactly 1 (no backward compat)
- Name: Non-empty, whitespace trimmed
- SupportedOS: Non-empty list, each element ∈ {linux, darwin, windows}
- Profiles: At least one required; each profile must have non-empty Sources
- Sources: Relative paths only (no leading /, no .. segments)
  - Implementation: Check for ".." in original string (before path.Clean) to catch "a/../b"
- Sources uniqueness: Per-profile (duplicates allowed across profiles)
- Target: If set, must start with $HOME, $XDG_CONFIG_HOME, %USERPROFILE%, or %APPDATA%

### Test Coverage
- Rule 1 (SchemaVersion): 3 cases (valid 1, invalid 0, invalid 2)
- Rule 2 (Name): 3 cases (valid, empty, whitespace-only)
- Rule 3 (SupportedOS): 6 cases (single OS, multiple, empty, invalid, mixed)
- Rule 4 (Profiles): 5 cases (single, multiple, none, empty sources)
- Rule 5 (Relative paths): 5 cases (valid, nested, absolute, .., ..in-middle)
- Rule 6 (Uniqueness): 4 cases (unique, duplicate, cross-profile, multiple duplicates)
- Rule 7 (Target): 7 cases ($HOME, $XDG_CONFIG_HOME, %USERPROFILE%, %APPDATA%, empty, /etc, relative)
- Integration: 2 comprehensive cases (valid manifest, multiple errors)

### Conventions Established
- Validation errors use fmt.Errorf with descriptive messages
- Error messages include context (profile name, index, value)
- Table-driven tests organized by rule number for clarity
- All validation rules checked in order (first error returned)

## Task 3: Cross-platform Symlink Primitives (COMPLETED)

### Execution Summary
- Package: internal/symlink
- Functions: CreateSymlink, RemoveSymlink, IsSymlinkTo, ReadLink
- Tests: 11 comprehensive test cases covering all scenarios
- All tests pass with race detector enabled
- Build verified: `go build ./...` passes
- Commit: feat(symlink): add cross-platform symlink primitives

### Implementation Details
- **CreateSymlink(source, target)**:
  - Uses os.Symlink (pure Go, cross-platform)
  - Creates parent directories via os.MkdirAll(0755)
  - Checks target existence with os.Lstat before creating
  - Returns descriptive error if target already exists
  - Windows note: Requires Developer Mode (documented in docstring, doctor package handles check)

- **RemoveSymlink(target)**:
  - Verifies target exists and is a symlink (not regular file)
  - Uses os.Remove to delete the symlink
  - Returns error if target doesn't exist or is not a symlink

- **IsSymlinkTo(target, source)**:
  - Returns false (not error) if target doesn't exist or is not a symlink
  - Reads symlink destination with os.Readlink
  - Compares destination string to source parameter

- **ReadLink(path)**:
  - Wrapper around os.Readlink
  - Returns destination of symlink at path

### Test Coverage
- CreateSymlink happy path: symlink created, points to source
- CreateSymlink creates parent dirs: nested directories created
- CreateSymlink fails if target exists (regular file)
- CreateSymlink fails if target exists (another symlink)
- RemoveSymlink happy path: symlink removed
- RemoveSymlink fails if target doesn't exist
- RemoveSymlink fails if target is regular file (not symlink)
- IsSymlinkTo returns true for correct symlink
- IsSymlinkTo returns false for wrong target
- IsSymlinkTo returns false for missing path
- ReadLink happy path: returns correct destination

### Key Decisions
- File-level symlinking only (no directory folding) — callers responsible for passing file paths
- Pure Go os.Symlink — no shelling out to ln or stow
- No "force" mode — callers must check before calling
- No Windows-specific code paths — os.Symlink is cross-platform
- IsSymlinkTo returns false (not error) for missing/non-symlink paths — caller-friendly API

## Task 4: State File Format + Read/Write (COMPLETED)

### Execution Summary
- State struct: InstalledLink, PackageState, State (map[string]PackageState)
- Functions: DefaultPath(), Load(), Save()
- Test coverage: 9 comprehensive tests covering all requirements
- All tests pass with race detector enabled
- Build verified: `go build ./...` passes
- Diagnostics: 0 errors/warnings
- Commit: feat(state): add state file format and read/write

### Key Implementation Details
- DefaultPath: Uses os.UserConfigDir() for cross-platform support (POSIX: ~/.config/rice/state.json, Windows: %APPDATA%/rice/state.json)
- Load: Returns empty State{} on file not found (not an error); uses json.Unmarshal
- Save: Creates parent directories with os.MkdirAll; uses json.MarshalIndent with 2-space indent; writes with mode 0644
- State is a map[string]PackageState where key is package name (e.g., "nvim", "ghostty")
- InstalledLink tracks source (absolute path in rice repo) and target (absolute path to symlink in $HOME)
- PackageState includes Profile, InstalledLinks slice, and InstalledAt timestamp

### Test Coverage
- TestDefaultPath: Verifies non-empty absolute path in config directory
- TestLoadNonExistentFile: Confirms empty State returned (no error)
- TestLoadValidJSON: Validates correct parsing of valid state file
- TestLoadInvalidJSON: Confirms error on malformed JSON
- TestSaveCreatesParentDirectories: Verifies os.MkdirAll behavior
- TestSaveWritesCorrectJSON: Validates pretty-printed JSON output
- TestRoundTrip: Save then Load returns identical State
- TestSaveEmptyState: Handles empty State correctly
- TestLoadEmptyFile: Handles empty JSON object correctly

### Conventions Established
- State file is authoritative for "what rice installed"
- JSON format with 2-space indentation for readability
- No file locking (documented as "don't run concurrent rice processes")
- No migration logic (v1 only)

## Task 5: Profile Resolution Rules + Validation (COMPLETED)

### Execution Summary
- Resolve() function: Looks up profile by name in manifest.Profiles map
- Error handling: Returns sorted list of available profiles in error message
- Test coverage: 5 table-driven test cases covering all requirements
- All tests pass with race detector enabled
- Build verified: `go build ./...` passes
- Commit: feat(profile): add profile resolution

### Key Implementation Details
- Resolve(m *manifest.Manifest, profileName string) ([]string, error)
- Returns profile.Sources slice as-is (preserves order, no sorting)
- Error message format: "profile %q not defined in package %q; available: %s"
- Available profiles listed alphabetically in error message
- No implicit "common" base — only returns what's in manifest
- Pure in-memory operation (no filesystem access)

### Test Cases
1. Single source: profile with sources = ["."]
2. Multiple sources: profile with sources = ["common", "macbook"] (order preserved)
3. Unknown profile: error contains profile name, package name, and available profiles
4. Empty profiles map: error with no available profiles listed
5. Source order preservation: verifies sources returned in manifest order

### Conventions Applied
- Table-driven test pattern (consistent with manifest tests)
- testify/assert and testify/require for assertions
- Error message includes both profile name and available alternatives
- Alphabetical sorting of available profiles in error (for deterministic output)
