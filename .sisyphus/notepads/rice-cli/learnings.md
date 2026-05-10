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

## Task 25: Plan + Confirmation Prompt Utility (COMPLETED)

### Execution Summary
- internal/plan/plan.go: OpKind, Op, Conflict, Plan types with IsEmpty() method
- internal/plan/plan_test.go: 4 test cases covering IsEmpty() logic
- internal/prompt/prompt.go: RenderPlan, RenderSwitchPlan, RenderConflicts, Confirm functions
- internal/prompt/prompt_test.go: 9 test functions with 30+ test cases
- All tests pass with race detector: `go test ./internal/plan/... ./internal/prompt/... -race`
- Build verified: `go build ./...` passes
- Commit: feat(plan,prompt): add Plan type, renderer, and y/N confirmation

### Key Implementation Details
- OpKind: enum-style (OpCreate=0, OpRemove=1)
- Op: Kind, Source (empty for remove), Target fields
- Conflict: Target, Source, Reason fields
- Plan: PackageName, Profile, Ops, Conflicts fields
- RenderPlan: detects install vs uninstall from first Op kind; uses text/tabwriter for alignment
- RenderSwitchPlan: prints uninstall phase, then install phase, combined total
- RenderConflicts: simple line-by-line format
- Confirm: default NO on bare Enter; accepts "y"/"yes" (case-insensitive); returns (false, nil) on EOF

### Test Coverage
- plan.IsEmpty(): 4 cases (empty, with ops, with conflicts, with both)
- RenderPlan: empty install, create ops, remove ops, 100 ops (no truncation)
- RenderSwitchPlan: both phases present with correct totals
- RenderConflicts: correct format
- Confirm: y/Y/yes/YES (true), bare enter/n/N/no/NO/random (false), EOF (false, nil), error handling

### Conventions Applied
- text/tabwriter for column alignment (2-space padding)
- bufio.NewReader for line reading
- strings.TrimSpace + strings.ToLower for input normalization
- io.EOF treated as (false, nil) per spec
- Docstrings follow Go conventions for exported types/functions

## Task 10: Install Orchestrator (COMPLETED)

### Execution Summary
- internal/installer/install.go: InstallRequest, InstallResult, BuildInstallPlan, ExecuteInstallPlan, Install
- internal/installer/install_test.go: 11 tests covering build (no FS touch), multi-source, single-source, rice.toml skip, conflict detection, execute symlinks, state update, OS gate, unknown package, unknown profile, idempotency, full flow
- testdata/install/mypkg/ fixture with rice.toml + 3 source files (root, common, macbook)
- All tests pass with -race

### Key Design Choices
- Build/Execute split: BuildInstallPlan is pure (no FS writes), ExecuteInstallPlan applies. CLI inserts confirmation between them.
- Multi-source override: when same target appears in multiple sources, LATER source wins (replaces in ops list keyed by target).
- Idempotency: in ExecuteInstallPlan, if CreateSymlink fails AND IsSymlinkTo confirms target already points to our source, treat as success.
- Partial failure: save partial state (created links so far) before returning error, so doctor/uninstall can clean up.
- Defense-in-depth: withinHome() check ensures target paths cannot escape HomeDir via path traversal.
- $HOME / %USERPROFILE% expansion in m.Target.
- rice.toml files in source trees are skipped (logged as WARN).
- Symlinks in source trees are skipped (we only manage real files).

### Gotchas
- sources=["."] walks the entire package root including subdirs like common/, macbook/. The fixture design must account for this if "common" profile is intended to be minimal.
- DetectConflicts already handles idempotent symlinks (existing symlink to our source = NOT a conflict), so re-running install with same args naturally succeeds.

## Task 12: Switch Orchestrator (COMPLETED)

### Execution Summary
- internal/installer/switch.go: SwitchRequest, SwitchPlan, BuildSwitchPlan, ExecuteSwitchPlan, Switch
- internal/installer/switch_test.go: 7 tests covering not-installed, happy path, missing profile, foreign-file conflict, ignoreTargets reuse, execute happy, convenience wrapper
- All tests pass with -race
- Commit: 539c6df

### Key Implementation Details
- BuildSwitchPlan does NOT touch FS — load state, build uninstall+install plans, re-run DetectConflicts with ignoreTargets from uninstall ops
- BuildInstallPlan may return (plan, err) on conflicts; we keep the plan and re-check, then clear stale conflicts
- ExecuteSwitchPlan: uninstall first, then install; on install failure logs recovery message ("rice install <pkg> --profile <profile>") and returns error
- Profile fixture caveat: common (sources=["."]) walks subdirs, producing targets like ~/common/.config/mypkg/base.toml — does NOT overlap with macbook profile (which produces ~/.config/mypkg/base.toml). Fixed pre-flight reuse test by switching macbook→macbook with a manually-placed foreign file at an old target.

### Test Pattern: ignoreTargets validation
- To validate that old-link reuse is NOT a conflict: install profile X, replace one symlink with a foreign file, BuildSwitchPlan to profile X again. Without ignoreTargets this would conflict; with it, it must be empty conflicts.

## Task 13: CLI Scaffold + Cobra Setup (COMPLETED)

### Execution Summary
- Replaced cmd/rice/cmd/root.go with full cobra root command
- Created cmd/rice/cmd/version.go subcommand
- Created cmd/rice/cmd/root_test.go with 5 smoke tests
- All tests pass with race detector: `go test ./cmd/... -race`
- Binary builds and version command works: `go build -o /tmp/rice ./cmd/rice && /tmp/rice version`
- Commit: feat(cli): add cobra root cmd with --log-level, --yes, --repo, --state flags

### Key Implementation Details
- Root command uses PersistentPreRunE to initialize logger before any subcommand runs
- PersistentPostRun calls logger.Sync() to flush logs
- Log level resolution: flag > env var (RICE_LOG_LEVEL) > default (warn)
- Global flags: --repo, --state, --log-level, --yes (-y)
- Version constant: "0.1.0" defined in root.go
- Version subcommand: simple RunE that prints "rice version X.Y.Z"

### Testing Patterns
- Cobra tests require resetting global flag variables between test runs
- Created resetRootCmd() helper to reset flagRepo, flagState, flagLogLevel, flagYes
- Tests verify: version output, invalid log level error, help text, env var resolution, flag override
- SetArgs() must be called before Execute() to set command-line arguments
- SetOut/SetErr work for help output but not for command output (fmt.Println goes to stdout)

### Cobra Integration Notes
- github.com/spf13/cobra v1.10.2 added to go.mod
- Cobra automatically adds --help flag and generates usage text
- PersistentFlags are inherited by all subcommands
- Command.Execute() returns error on failure (non-zero exit handled by main.go)
- Global flag variables must be declared at package level for init() to bind them

### Lessons Learned
- Cobra's global singleton rootCmd requires careful test isolation
- Flag values persist across test runs unless explicitly reset
- Logger initialization in PersistentPreRunE happens before subcommand runs
- Version constant should be defined in root.go for easy access by subcommands

## Task 14: rice install command (COMPLETED)

### Execution Summary
- cmd/rice/cmd/install.go: cobra subcommand wiring BuildInstallPlan -> RenderPlan -> Confirm -> ExecuteInstallPlan
- cmd/rice/cmd/install_test.go: 5 tests covering --yes, stdin y/n, missing args, --profile flag
- Per-test temp repo + fake $HOME via t.Setenv("HOME", ...) ensures isolation
- All tests pass with -race; lsp diagnostics clean

### Gotchas Discovered
- prompt.RenderPlan signature is RenderPlan(io.Writer, *plan.Plan) — task spec showed RenderPlan(plan)
- prompt.Confirm signature is Confirm(io.Reader, io.Writer, string) — task spec showed Confirm(msg)
- installer.ExecuteInstallPlan returns (*InstallResult, error) — task spec showed single error return
- profile.Resolve does NOT auto-detect from hostname for empty profile string; tests must pass --profile explicitly
- Cobra's SetIn/SetOut/SetErr persist across rootCmd.Execute() calls — must restore in test helper to avoid pollution between tests
- runtime.GOOS used directly; HomeDir resolved via os.UserHomeDir() (overridable in tests via HOME env var)

## Task 15: rice uninstall command (COMPLETED)

### Execution Summary
- cmd/rice/cmd/uninstall.go: cobra command, mirrors install.go pattern
- prompt.RenderPlan auto-detects uninstall via OpRemove kind → "Plan: uninstall <pkg>" header
- ExecuteUninstallPlan returns single error (not (_, error) like ExecuteInstallPlan)
- Tests reuse runInstallCmd helper from install_test.go (works for any subcommand)
- State setup in tests done by running `install --yes` first via the same rootCmd
- All 5 tests pass with -race
- Commit: feat(cli): add uninstall command

## Task 16: rice switch command (COMPLETED)
- Pattern matches install.go/uninstall.go: BuildPlan → Render → Confirm (unless --yes) → Execute
- prompt.RenderSwitchPlan(w, uninstallPlan, installPlan) — renders both phases + combined total
- installer.SwitchRequest takes RepoRoot, PackageName, NewProfile, CurrentOS, HomeDir, StatePath
- ExecuteSwitchPlan(sp, statePath) returns single error; no count return (unlike ExecuteInstallPlan)
- Tests reuse runInstallCmd helper from install_test.go (it just dispatches via root)
- All 6 tests pass with -race

## Task 17: rice status command (COMPLETED)

### Execution Summary
- cmd/rice/cmd/status.go: reads state.Load(flagState), iterates packages, checks each link via symlink.IsSymlinkTo
- Optional positional arg filters to a single package; unknown filter prints nothing (no error)
- Empty state prints "No packages installed."
- Uses "OK" / "BROKEN" markers (avoiding non-ASCII to keep test assertions simple)
- 5 test cases: no-packages, healthy, filter, broken (symlink to wrong source), filter-unknown
- All `go test ./cmd/... -race` pass; `go build ./cmd/rice` succeeds
- Commit: feat(cli): add status command

### Gotchas
- symlink.IsSymlinkTo signature is (target, source string) (bool, error) — must handle both bool and error
- state.PackageState field is `InstalledLinks` (not `Links` as task spec example suggested)
- state.Load already returns empty State on ErrNotExist (no need for os.IsNotExist check in caller)

## Task 18: rice doctor (COMPLETED)

### Summary
- cmd/rice/cmd/doctor.go: read-only health checker (state file, symlink integrity, repo accessibility)
- cmd/rice/cmd/doctor_test.go: 5 test cases (no state, healthy, missing link, replaced link, bad repo)
- state.Load returns (State{}, nil) when file missing — no error needed for absent state
- IsSymlinkTo signature: (target, source string) (bool, error) — note target FIRST
- doctor returns non-zero error when issues > 0; "All checks passed." on clean
- Commit: feat(cli): add doctor command
