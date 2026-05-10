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

## Task 24: Logging Package (COMPLETED)

### Execution Summary
- Added go.uber.org/zap v1.28.0 dependency
- Implemented internal/logger/logger.go with:
  - Level type with 5 levels: Debug, Info, Warn, Error, Critical
  - CriticalLevel = ErrorLevel + 1 (custom level above Error)
  - ParseLevel() for case-insensitive level parsing with error messages
  - Init() with tee architecture: console (STDERR) + file (JSON, always DEBUG)
  - DefaultLogPath() using os.UserConfigDir() with fallback
  - Package-level functions: Debug, Info, Warn, Error, Critical
  - Critical() auto-appends github_issue_url field
  - Sync() for flushing logger
  - L initialized as zap.NewNop() for safe pre-Init usage
- Implemented comprehensive test suite (14 tests):
  - ParseLevel: case-insensitive, whitespace handling, error messages
  - Level ordering: CriticalLevel > ErrorLevel
  - File creation and parent directory creation
  - File always at DebugLevel regardless of console level
  - Console filtering respects level threshold
  - Critical includes github_issue_url field
  - DefaultLogPath returns absolute path with rice/logs/rice.log
  - JSON format validation for file logs
  - All tests pass with -race flag

### Key Implementation Details
- Console encoder: zap.NewDevelopmentEncoderConfig() for human-readable output
- File encoder: zap.NewProductionEncoderConfig() for JSON format
- Tee core: zapcore.NewTee(consoleCore, fileCore) for dual output
- File open: os.O_APPEND|os.O_CREATE|os.O_WRONLY with 0644 mode
- Parent dir creation: os.MkdirAll with 0755 mode
- Critical level logging: L.Log(zapcore.Level(CriticalLevel), ...)

### Test Coverage
- 15 test cases covering all public functions
- Stderr capture for console filtering verification
- JSON parsing for file output validation
- Temp directories for file creation tests
- Error cases: invalid paths, invalid levels
- Edge cases: empty strings, whitespace, case variations

### Conventions Established
- Logger package is safe to use before Init() (nop logger)
- Console output always to STDERR (stdout reserved for command output)
- File always at DEBUG level for complete audit trail
- Critical logs always include github issue URL for bug reporting
- DefaultLogPath follows XDG conventions with Windows fallback

## Task 7: Manifest Discovery + Parsing (COMPLETED)

### Execution Summary
- Implemented Load(dir string) function: reads rice.toml, parses with BurntSushi/toml, validates
- Implemented Discover(repoRoot string) function: walks one level deep, collects valid manifests
- Created comprehensive test suite with 9 test cases covering all scenarios
- Test fixtures: testdata/manifest/ (with bad/ for error testing) and testdata/manifest_valid/ (for discovery tests)
- All tests pass with race detector enabled
- Build verified: `go build ./...` passes
- Commit: feat(manifest): add manifest discovery and TOML parsing

### Key Implementation Details
- Load: Returns error if file missing, TOML parse fails, or validation fails
- Discover: One-level-deep walk only (packages are direct children of repoRoot)
- Discover: Silently skips directories without rice.toml
- Discover: Returns error if rice.toml found but fails parse/validate
- Discover: Skips non-directory entries at repoRoot level
- Used runtime.Caller() in tests to resolve testdata paths correctly

### Test Coverage
- Load happy path: valid manifest parsing
- Load file not found: error handling
- Load invalid TOML: parse error handling
- Load validation failure: validation error handling
- Discover finds all valid manifests: multi-manifest discovery
- Discover returns error on invalid manifest: error propagation
- Discover skips dirs without rice.toml: selective discovery
- Discover empty repository: empty result handling
- Discover multi-profile manifest: complex manifest handling

### Conventions Established
- Testdata organized by scenario (manifest/ for error cases, manifest_valid/ for success cases)
- Test helper functions for path resolution (getTestdataDir, getTestdataManifestDir)
- Error messages include context (directory path, manifest name, etc.)

## Task 8: Package OS Gating (COMPLETED)

### Execution Summary
- CheckOS function: Validates currentOS against m.SupportedOS
- Error format: "package %q does not support %s; supported: %s"
- Defensive check: Handles empty SupportedOS (shouldn't happen post-Validate)
- Test coverage: 10 table-driven test cases covering all scenarios
- All tests pass with race detector enabled
- Build verified: `go build ./...` passes
- Commit: feat(manifest): add OS gating check

### Key Implementation Details
- CheckOS accepts currentOS as parameter (not calling runtime.GOOS internally)
- Returns nil if currentOS is in SupportedOS list
- Returns descriptive error with supported OS list if not found
- Defensive: checks for empty SupportedOS and returns appropriate error
- Test cases: single OS match, multi-OS match, unsupported OS, empty SupportedOS

### Test Coverage
- linux package on linux → nil
- darwin package on darwin → nil
- linux-only package on windows → error with "windows" and "linux"
- multi-OS package (linux+darwin) on darwin → nil
- multi-OS package (linux+darwin) on windows → error with "linux, darwin"
- empty SupportedOS → error (defensive)
- windows package on windows → nil
- all three OSes supported on linux → nil

## Task 9: Conflict Detection (COMPLETED)

### Execution Summary
- Created internal/installer/conflict.go with Conflict type and DetectConflicts function
- Created internal/installer/conflict_test.go with 8 comprehensive test cases
- All tests pass with race detector enabled: `go test ./internal/installer/... -race`
- Build verified: `go build ./...` passes
- Commit: feat(installer): add conflict detection

### Key Implementation Details
- Conflict struct: Target, Source, Reason fields with Error() method
- PlannedLink struct: Source, Target pair for symlink creation
- DetectConflicts function:
  - Checks each planned link for conflicts
  - Skips targets in ignoreTargets map (for switch pre-flight)
  - Returns empty slice if no conflicts
  - Idempotent: symlink already pointing to source = no conflict
  - Detects: regular files, directories, symlinks pointing elsewhere
  - Uses symlink.IsSymlinkTo() for idempotency check
  - Uses os.Readlink() to get other symlink destinations

### Test Coverage
- No conflicts when targets don't exist
- No conflict when target is already our symlink (idempotent)
- Conflict when target is a regular file → "existing file"
- Conflict when target is a directory → "existing directory"
- Conflict when target is a symlink pointing elsewhere → "symlink points to <path>"
- ignoreTargets: targets in map are skipped even if they would conflict
- Multiple conflicts returned in one call
- Conflict.Error() method formats error message correctly

### Design Decisions
- Read-only detection: no modifications to filesystem
- No "force" mode: conflicts are always reported
- Idempotent by design: existing correct symlinks don't block
- ignoreTargets parameter enables switch pre-flight to exclude old links
- Error handling: filesystem errors treated as conflicts (conservative)
