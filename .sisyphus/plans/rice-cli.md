# Rice CLI — Cross-Platform Profile-Aware Dotfile Installer

## TL;DR

> **Quick Summary**: Replace GNU stow + bash install scripts with a Go-based `rice` CLI that installs dotfile packages from this repo onto linux/darwin/windows machines, with per-package profile support (e.g., opencode personal vs work, ghostty macbook vs devstick).
>
> **Deliverables**:
> - `rice` Go CLI with `install`, `switch`, `status`, `doctor`, `uninstall` commands
> - `rice.toml` manifest schema and one manifest per existing package
> - State file at `~/.config/rice/state.json` tracking installed packages and symlinks
> - Migrated existing packages (ghostty restructure, opencode work/personal split, others get manifests)
> - Deleted: `ghostty/install.sh`, empty `profiles/` and `scripts/` dirs
> - New `AGENTS.md` documenting rice conventions for future agents
> - Rewritten `README.md` documenting the new CLI workflow
> - Thorough Go tests using `testdata/` fixtures
>
> **Estimated Effort**: Large
> **Parallel Execution**: YES — 4 waves
> **Critical Path**: Schema → Core symlink/state → Commands → Migration → Docs → Verification

---

## Context

### Original Request
User wants smooth installation of rices on new machines. Specifically motivated by needing an opencode work profile (separate from personal), a quick way to switch between profiles, and accommodating a Windows work machine.

### Interview Summary
**Key Discussions**:
- Existing `ghostty/install.sh` already implements a primitive common+overlay pattern — generalize it into a real tool
- `profiles/personal` and `profiles/work` directories at repo root are stale (created by another agent, never tracked, will be deleted)
- Profile is per-package and freeform (ghostty: macbook/devstick; opencode: personal/work) — no global "identity" concept
- Windows must be a first-class target, not WSL — drives the choice of Go for cross-platform symlinks
- Each package declares its own OS support — `rice install zsh` on Windows is refused
- Opencode skills are FULLY SEPARATE per profile (no shared base) — duplication is acceptable in v1
- `switch` does full re-stow (uninstall everything for that package, reinstall with new profile)

**Research Findings**:
- Repo packages today: ghostty, hyprland, nvim, opencode, waybar, wofi, zsh
- Hyprland/waybar/wofi are linux-only (Wayland)
- ghostty is the only package with existing install.sh; others rely on raw `stow`
- opencode/.agents/skills/ has many skill subdirectories (currently shared)

### Metis Review
**Identified Gaps** (addressed):
- **Switch atomicity**: Resolved — Pre-flight validates all symlink targets are creatable BEFORE removing existing symlinks. Abort on pre-flight failure with no state change.
- **State file authority**: Resolved — `state.json` is authoritative for "what rice installed". Filesystem is authoritative for "what currently exists". `doctor` reconciles drift. Missing state.json = treat as fresh install. Symlink in state but not on disk = reported by `doctor`, no auto-fix.
- **Profile dimension collapse**: Resolved — V1 supports single-axis profiles only. Multi-axis (e.g., macbook+work for one package) is explicitly out of scope and documented in AGENTS.md.
- **rice.toml schema undefined**: Resolved — Schema defined explicitly in this plan with `schema_version`, required/optional fields, and path validation rules. **Profile uses `sources = [...]` (list of folders), not `source = "..."`**, so a profile can compose multiple folders explicitly.
- **No "common" convention magic**: Profile authors explicitly list which folders make up a profile. e.g., `[profiles.macbook] sources = ["common", "macbook"]`. The installer treats `sources` as an ordered list and stows them in order. No special meaning attached to the name "common".
- **Dependencies (binaries on PATH like `bun`, `nvim`, `ripgrep`)**: Explicitly OUT OF SCOPE for v1. No schema field, no doctor check. Documented in AGENTS.md as a v2 candidate. User installs deps via their OS package manager.
- **Conflict semantics for existing symlink to different target**: Treat as conflict (abort), consistent with "abort on any conflict" policy.

---

## Work Objectives

### Core Objective
Build a Go CLI named `rice` that lives in this repo and provides a unified, cross-platform, profile-aware mechanism for installing dotfile packages onto a new machine, replacing the current mix of raw `stow` invocations and bespoke bash install scripts.

### Concrete Deliverables
- `cmd/rice/main.go` — CLI entry point
- `internal/manifest/` — rice.toml parser + validator
- `internal/symlink/` — cross-platform symlink operations
- `internal/state/` — state file management
- `internal/installer/` — install/uninstall/switch orchestration
- `internal/doctor/` — diagnostics
- `internal/profile/` — profile resolution and validation
- `cmd/rice/cmd/` — cobra-style command definitions (install, switch, status, doctor, uninstall)
- `rice.toml` in each existing package directory
- `opencode/personal/.agents/skills/` and `opencode/work/.agents/skills/` (split from current shared layout)
- `ghostty/` restructured with `rice.toml` (no more `install.sh`)
- `AGENTS.md` at repo root
- Rewritten `README.md`
- `go.mod`, `go.sum` at repo root
- `testdata/` directory with fixture rice setups for tests

### Definition of Done
- [ ] `go build -o /tmp/rice ./cmd/rice` succeeds with zero warnings
- [ ] `go test ./...` passes (all tests green)
- [ ] `go vet ./...` passes
- [ ] `/tmp/rice install nvim --profile common` creates expected symlinks (verified via `ls -la`)
- [ ] `/tmp/rice install opencode --profile work` installs work skills, NOT personal skills
- [ ] `/tmp/rice switch opencode personal` swaps to personal skills cleanly
- [ ] `/tmp/rice install zsh` on a Windows test (cross-compiled binary, OS gate honored) refuses with clear error
- [ ] `/tmp/rice doctor` reports state without errors on a clean system
- [ ] `/tmp/rice uninstall opencode` removes all opencode symlinks and updates state.json
- [ ] `ghostty/install.sh` deleted; ghostty installs via `rice install ghostty --profile macbook`
- [ ] `profiles/` dir deleted from repo
- [ ] `AGENTS.md` exists at repo root and documents rice.toml schema, CLI commands, profile model
- [ ] `README.md` no longer mentions `stow`; documents `rice` workflow

### Must Have
- Cross-platform: linux, darwin, windows (one binary per platform via `go build`)
- Pure Go symlinks via `os.Symlink` — no shelling to `stow` or `ln`
- Per-package OS gating enforced from `rice.toml` (refuse install on unsupported OS with clear error)
- Per-package freeform profiles declared in `rice.toml` using `sources = [...]` lists
- State file at `~/.config/rice/state.json` (Windows: `%APPDATA%/rice/state.json`)
- Pre-flight validation on `switch` (no destructive action until all targets verified creatable)
- Conflict policy: abort on any conflict (existing non-symlink, or symlink pointing to a target NOT owned by this rice install)
- **Confirmation prompt before any destructive op** (install/uninstall/switch). Prints full plan (every symlink: source → target, every removal) then `Proceed? [y/N]`. Default on Enter = NO. `--yes` / `-y` flag bypasses for scripting. `doctor` and `status` are read-only and never prompt.
- **Structured logging via `go.uber.org/zap`** with 5 levels: debug, info, warn, error, critical. CRITICAL is a custom zapcore level above Error reserved for "open a github issue" situations. Default level: WARN. Configurable via `--log-level` flag and `RICE_LOG_LEVEL` env var (flag wins). Logs go to STDERR at the configured level. ALSO always written at DEBUG level to a single non-rotating file at `~/.config/rice/logs/rice.log` (Windows: `%APPDATA%/rice/logs/rice.log`) — user manages cleanup. Stdout is reserved for command output (status table, install summary, plan output).
- Thorough Go tests with `testdata/` fixtures
- `AGENTS.md` documenting rice conventions
- Rewritten `README.md`

### Must NOT Have (Guardrails)
- **NO `--force` flag** in v1 — conflicts always abort
- **NO `init` / bootstrap command** in v1 — user clones repo and runs `go run` or `go build` themselves
- **NO `install --all`** in v1 — must specify package
- **NO multi-axis profiles** in v1 — each package has one profile dimension
- **NO automatic backup** of conflicting files — fail loud, don't move user data silently
- **NO shelling to `stow`** — the entire point is to drop that dependency
- **NO global identity concept** — profiles are per-package and freeform
- **NO auto-fix from `doctor`** — report only, never mutate
- **NO TUI / no interactive menus** — strictly stdin-free apart from the SINGLE y/N confirmation prompt before destructive ops. No spinners, no progress bars, no menus, no multi-step wizards.
- **NO publishing to package registries** (homebrew, etc.) in v1
- **NO target paths outside `$HOME` / `%USERPROFILE%`** — manifest validation rejects `/etc`, `/usr`, etc.
- **NO `as any` / silent error swallowing** in Go code — return errors explicitly
- **NO scope creep into other dotfile packages** — only ghostty restructure and opencode split are in scope; nvim/zsh/hyprland/waybar/wofi just get a `rice.toml` added
- **NO implicit "common" profile magic** — `sources = [...]` is explicit; the resolver returns exactly what the manifest declared
- **NO singular `source = "x"` field** — only the plural `sources = [...]` list is accepted
- **NO `[dependencies]` section** in rice.toml v1 — out of scope, documented in AGENTS.md as v2 candidate. Don't sneak in a dependency check anywhere.

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed.

### Test Decision
- **Infrastructure exists**: NO (no Go tests yet, no `go.mod`)
- **Automated tests**: YES (TDD where reasonable, tests-after for trivial glue)
- **Framework**: Standard Go `testing` package, plus `testify/require` for ergonomic assertions
- **Setup is part of Wave 1** (Task 1 below)

### QA Policy
Every implementation task MUST include agent-executed QA scenarios using:
- **Bash + go test**: For unit/integration tests on the Go CLI
- **Bash + compiled binary**: For end-to-end CLI behavior in temp dirs
- **interactive_bash (tmux)**: Only if needed for complex multi-step CLI flows

Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — start immediately):
├── Task 1:  Go module init + project scaffolding + test infra [quick]
├── Task 2:  rice.toml schema definition (Go structs + validation rules) [quick]
├── Task 3:  Cross-platform symlink primitives [quick]
├── Task 4:  State file format + read/write [quick]
├── Task 5:  Profile resolution rules + validation [quick]
├── Task 6:  Delete stale dirs (profiles/, scripts/) + commit baseline [quick]
└── Task 24: Logging package (zap-based, 5 levels including custom CRITICAL) [quick]

Wave 2 (Core orchestration — needs Wave 1):
├── Task 7:  Manifest discovery + parsing (depends: 2) [quick]
├── Task 8:  Package OS gating (depends: 2, 7) [quick]
├── Task 9:  Conflict detection (depends: 3) [quick]
├── Task 25: Plan + confirmation utility (Plan struct, renderer, y/N prompt) (depends: 24) [quick]
├── Task 10: Install orchestrator (depends: 3, 4, 7, 8, 9, 24, 25) [unspecified-high]
├── Task 11: Uninstall orchestrator (depends: 3, 4, 24, 25) [quick]
└── Task 12: Switch orchestrator with pre-flight (depends: 10, 11) [unspecified-high]

Wave 3 (CLI surface — needs Wave 2):
├── Task 13: CLI scaffold + cobra setup + root cmd + log/--yes flags (depends: 1, 24) [quick]
├── Task 14: install command (depends: 10, 13, 25) [quick]
├── Task 15: uninstall command (depends: 11, 13, 25) [quick]
├── Task 16: switch command (depends: 12, 13, 25) [quick]
├── Task 17: status command (depends: 4, 13) [quick]
└── Task 18: doctor command (depends: 4, 7, 13) [quick]

Wave 4 (Migration + docs — needs Wave 3):
├── Task 19: Add rice.toml to nvim, zsh, hyprland, waybar, wofi (depends: 7) [quick]
├── Task 20: Migrate ghostty (delete install.sh, restructure, add rice.toml) (depends: 7) [quick]
├── Task 21: Split opencode skills into personal/work + rice.toml (depends: 7) [unspecified-high]
├── Task 22: Write AGENTS.md (depends: 14-18, 24, 25) [writing]
└── Task 23: Rewrite README.md (depends: 14-18, 24, 25) [writing]

Wave FINAL (Verification — after all implementation):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: End-to-end manual QA — including confirmation flow + log levels (unspecified-high)
└── Task F4: Scope fidelity check (deep)
→ Present results → Wait for explicit user "okay" before completing

Critical Path: 1 → 2 → 7 → 25 → 10 → 12 → 16 → 21 → F1-F4 → user okay
Parallel Speedup: ~3x faster than sequential
Max Concurrent: 7 (Wave 1)
```

### Dependency Matrix (updated)

- **1**: ∅ → 13, 19-21
- **2**: ∅ → 7, 8
- **3**: ∅ → 9, 10, 11
- **4**: ∅ → 10, 11, 17, 18
- **5**: ∅ → 7, 10
- **6**: ∅ → (just cleanup)
- **7**: 2 → 8, 10, 18, 19, 20, 21
- **8**: 2, 7 → 10
- **9**: 3 → 10
- **10**: 3, 4, 7, 8, 9, 24, 25 → 12, 14
- **11**: 3, 4, 24, 25 → 12, 15
- **12**: 10, 11 → 16
- **13**: 1, 24 → 14-18
- **14-16**: 10/11/12, 13, 25 → 22, 23
- **17, 18**: 4, 7, 13 → 22, 23
- **19-21**: 7 → F1-F4
- **22, 23**: 14-18, 24, 25 → F1-F4
- **24**: ∅ → 10, 11, 13, 22, 23, 25
- **25**: 24 → 10, 11, 14-16, 22, 23

### Agent Dispatch Summary

- **Wave 1 (7 tasks)**: All `quick`
- **Wave 2 (7 tasks)**: T10, T12 → `unspecified-high`; rest `quick`
- **Wave 3 (6 tasks)**: All `quick`
- **Wave 4 (5 tasks)**: T21 → `unspecified-high`; T22, T23 → `writing`; rest `quick`
- **FINAL (4 tasks)**: F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. **Go module init + project scaffolding + test infra**

  **What to do**:
  - `go mod init github.com/guneet/rice` at repo root (use whatever module path matches user's GitHub; if unclear, use `rice` as a placeholder local module path)
  - Create directory structure: `cmd/rice/`, `internal/manifest/`, `internal/symlink/`, `internal/state/`, `internal/installer/`, `internal/profile/`, `internal/doctor/`, `testdata/`
  - Add `cmd/rice/main.go` with a stub `func main() { fmt.Println("rice") }`
  - Add `go.mod` dependencies: `github.com/spf13/cobra`, `github.com/BurntSushi/toml`, `github.com/stretchr/testify`, `go.uber.org/zap`
  - Update `.gitignore` to add `/rice` (compiled binary), `/dist/`, `*.test`, `coverage.out`, `.sisyphus/evidence/`
  - Add a placeholder smoke test in `cmd/rice/main_test.go` to confirm `go test ./...` runs

  **Must NOT do**:
  - Don't add any business logic yet — pure scaffolding
  - Don't pull in heavy deps beyond cobra + toml + testify + zap
  - Don't modify existing rice packages (ghostty/, nvim/, etc.)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure scaffolding, no design decisions
  - **Skills**: []
    - No skill needed for go mod init

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 1
  - Blocks: 13, 19-21
  - Blocked By: None

  **References**:
  - Cobra docs: https://github.com/spf13/cobra/blob/main/site/content/user_guide.md — for understanding the cobra-app layout (we don't generate via `cobra-cli`, just import)
  - BurntSushi/toml: https://pkg.go.dev/github.com/BurntSushi/toml — TOML parser
  - testify/require: https://pkg.go.dev/github.com/stretchr/testify/require — fail-fast assertions
  - Standard Go project layout: https://go.dev/doc/modules/layout — `cmd/` and `internal/` conventions

  **Acceptance Criteria**:
  - [ ] `go.mod` exists with module name and the three deps
  - [ ] `go build ./...` succeeds
  - [ ] `go test ./...` runs (the placeholder test passes)
  - [ ] All directories listed above exist (use `.gitkeep` for empty ones)

  **QA Scenarios**:
  ```
  Scenario: Module builds cleanly
    Tool: Bash
    Preconditions: Fresh checkout, no built artifacts
    Steps:
      1. Run `go build -o /tmp/rice ./cmd/rice`
      2. Assert exit code is 0
      3. Assert /tmp/rice exists and is executable
      4. Run `/tmp/rice` and capture output
    Expected Result: Output contains "rice"; exit code 0
    Failure Indicators: Build error, missing binary, panic
    Evidence: .sisyphus/evidence/task-1-build.txt

  Scenario: Test infra works
    Tool: Bash
    Preconditions: Module initialized
    Steps:
      1. Run `go test ./... -v`
      2. Assert exit code is 0
      3. Assert at least one PASS line in output
    Expected Result: Tests run and pass
    Evidence: .sisyphus/evidence/task-1-test.txt
  ```

  **Commit**: YES — `chore: scaffold go module and test infra`
  - Files: `go.mod`, `go.sum`, `cmd/rice/main.go`, `cmd/rice/main_test.go`, `.gitignore`, `.gitkeep` files
  - Pre-commit: `go build ./... && go test ./...`

- [x] 2. **rice.toml schema definition (Go structs + validation rules)**

  **What to do**:
  - In `internal/manifest/schema.go`, define the `Manifest` struct that mirrors the rice.toml schema:
    ```go
    type Manifest struct {
        SchemaVersion int                          `toml:"schema_version"` // required, must be 1 in v1
        Name          string                       `toml:"name"`           // required, must equal directory name
        Description   string                       `toml:"description"`    // optional
        SupportedOS   []string                     `toml:"supported_os"`   // required, non-empty; values: linux|darwin|windows
        Target        string                       `toml:"target"`         // optional, default "$HOME"; supports $HOME, $XDG_CONFIG_HOME placeholders
        ProfileKey    string                       `toml:"profile_key"`    // optional name of the profile dimension (e.g., "machine", "identity"); informational only
        Profiles      map[string]ProfileDef        `toml:"profiles"`       // required, key = profile value (e.g., "macbook"), defines which subdirs to stow
    }

    type ProfileDef struct {
        Sources []string `toml:"sources"` // required, non-empty; ordered list of relative subdir paths to stow when this profile is selected
    }
    ```
  - In `internal/manifest/validate.go`, write `func Validate(m *Manifest) error` that enforces:
    - `SchemaVersion == 1` (else error: "unsupported schema_version: %d")
    - `Name` non-empty
    - `SupportedOS` non-empty, each element ∈ {linux, darwin, windows}
    - At least one profile defined; each profile's `Sources` non-empty
    - Each entry in `Sources` is a relative path (no leading `/`, no `..` segments) — reject absolute paths and parent traversal
    - Sources within a single profile must be unique (reject duplicates like `["common", "common"]`)
    - `Target` (if set) starts with `$HOME`, `$XDG_CONFIG_HOME`, `%USERPROFILE%`, or `%APPDATA%` — reject paths that start with `/etc`, `/usr`, `/var`, `C:\Windows`, etc.
  - Write thorough table-driven tests in `internal/manifest/validate_test.go` covering each validation rule (positive + negative cases)

  **Must NOT do**:
  - Don't implement file I/O / parsing yet (that's Task 7)
  - Don't add hooks (pre/post install commands) — explicitly out of scope for v1
  - Don't add a `[dependencies]` section — out of scope for v1, documented in AGENTS.md as v2
  - Don't allow target paths outside HOME — guardrail
  - Don't allow multiple profile dimensions (multi-axis) — single-key only
  - Don't add backward-compat for a `source = "x"` (singular) field — `sources` (plural list) only. Keeps the parser simple.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Type definitions + table-driven tests, no design ambiguity left
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 1
  - Blocks: 7, 8
  - Blocked By: None (does NOT need Task 1 to be merged — can be developed in parallel and integrated at PR time)

  **References**:
  - BurntSushi/toml struct tag docs: https://pkg.go.dev/github.com/BurntSushi/toml#Decode
  - Existing ghostty layout (`ghostty/common`, `ghostty/macbook`, `ghostty/devstick`) — this is the intended profile model in concrete form. The `common` profile means: stow this dir always; `macbook` means: stow this dir when profile=macbook is selected.
  - User's confirmed decisions: profiles are freeform per package, single-axis, declared in rice.toml

  **Acceptance Criteria**:
  - [ ] `internal/manifest/schema.go` defines `Manifest` and `ProfileDef`
  - [ ] `internal/manifest/validate.go` defines `Validate(m *Manifest) error`
  - [ ] `go test ./internal/manifest/...` passes with ≥10 test cases (positive + negative)

  **QA Scenarios**:
  ```
  Scenario: Valid manifest passes validation
    Tool: Bash
    Preconditions: Task 2 code in place
    Steps:
      1. Run `go test ./internal/manifest/ -run TestValidate_Valid -v`
    Expected Result: All TestValidate_Valid* tests PASS
    Evidence: .sisyphus/evidence/task-2-valid.txt

  Scenario: Invalid manifests are rejected with specific errors
    Tool: Bash
    Preconditions: Task 2 code in place
    Steps:
      1. Run `go test ./internal/manifest/ -run TestValidate_Invalid -v`
    Expected Result: All TestValidate_Invalid* tests PASS, each asserting a specific error substring (e.g., "supported_os must be non-empty", "schema_version", "absolute path not allowed")
    Evidence: .sisyphus/evidence/task-2-invalid.txt

  Scenario: Path traversal rejected
    Tool: Bash
    Preconditions: Task 2 code in place
    Steps:
      1. Run `go test ./internal/manifest/ -run TestValidate_Traversal -v`
    Expected Result: Manifest with `sources = ["../../../etc"]` rejected with "parent traversal not allowed" or similar
    Evidence: .sisyphus/evidence/task-2-traversal.txt
  ```

  **Commit**: YES — `feat(manifest): define rice.toml schema and validation`
  - Files: `internal/manifest/schema.go`, `internal/manifest/validate.go`, `internal/manifest/validate_test.go`
  - Pre-commit: `go test ./internal/manifest/...`

- [x] 3. **Cross-platform symlink primitives**

  **What to do**:
  - Create `internal/symlink/symlink.go` with:
    - `func Create(source, target string) error` — creates symlink from `target` → `source`. Wraps `os.Symlink` with helpful error messages. Ensures parent dir of target exists (`os.MkdirAll` with 0755 on parent).
    - `func Remove(target string) error` — removes a symlink. Verifies target IS a symlink before removing (uses `os.Lstat` + `Mode()&os.ModeSymlink`). Returns specific error if target is not a symlink.
    - `func ReadLink(target string) (string, error)` — wraps `os.Readlink`
    - `func IsSymlinkTo(target, expectedSource string) (bool, error)` — checks if `target` is a symlink whose resolved path equals `expectedSource` (use `filepath.EvalSymlinks` and compare `filepath.Clean`-ed paths)
  - Create `internal/symlink/symlink_test.go` with tests using `t.TempDir()`:
    - Create symlink, read it back, verify
    - Remove symlink, verify gone
    - Remove non-symlink → expect specific error
    - IsSymlinkTo for matching and non-matching cases
  - Create `internal/symlink/symlink_windows_test.go` guarded by `//go:build windows` IF possible to test, otherwise add a `TestMain` skip + comment explaining: "Windows symlink tests require Developer Mode; CI/local runs on linux/darwin verify behavior."

  **Must NOT do**:
  - Don't try to handle Junctions, hard links, copies, or any fallback — symlinks only. If `os.Symlink` fails on Windows due to missing Developer Mode, surface the error verbatim with a hint.
  - Don't auto-create parent dirs beyond ONE level needed for the target — no `MkdirAll` on arbitrary depth without reason
  - Don't follow symlinks during Remove — use `os.Lstat`, not `os.Stat`

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Standard library wrappers with clear contracts
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 1
  - Blocks: 9, 10, 11
  - Blocked By: None

  **References**:
  - `os.Symlink`: https://pkg.go.dev/os#Symlink — note Windows-specific behavior
  - `os.Lstat` vs `os.Stat`: https://pkg.go.dev/os#Lstat — Lstat does NOT follow symlinks
  - Microsoft Developer Mode docs: https://learn.microsoft.com/en-us/windows/apps/get-started/developer-mode-features-and-debugging — for the user-facing error message hint

  **Acceptance Criteria**:
  - [ ] All four functions exist with documented signatures
  - [ ] `go test ./internal/symlink/...` passes on linux + darwin
  - [ ] Tests use `t.TempDir()` (no global state pollution)
  - [ ] Errors from `Remove` on non-symlinks contain the substring "not a symlink"

  **QA Scenarios**:
  ```
  Scenario: Create and read back symlink
    Tool: Bash
    Preconditions: Task 3 code in place
    Steps:
      1. Run `go test ./internal/symlink/ -run TestCreate -v`
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-3-create.txt

  Scenario: Remove refuses non-symlink
    Tool: Bash
    Preconditions: Task 3 code in place
    Steps:
      1. Run `go test ./internal/symlink/ -run TestRemove_NotSymlink -v`
    Expected Result: Test PASSES (asserting Remove returns error containing "not a symlink")
    Evidence: .sisyphus/evidence/task-3-remove-safety.txt
  ```

  **Commit**: YES — `feat(symlink): cross-platform symlink primitives`
  - Files: `internal/symlink/symlink.go`, `internal/symlink/symlink_test.go`
  - Pre-commit: `go test ./internal/symlink/... -race`

- [x] 4. **State file format + read/write**

  **What to do**:
  - Create `internal/state/state.go` with:
    - `type State struct { SchemaVersion int; Packages map[string]PackageState }` (TOML or JSON encoding — use JSON for tooling/diff friendliness)
    - `type PackageState struct { Profile string; InstalledLinks []InstalledLink; InstalledAt time.Time }`
    - `type InstalledLink struct { Target string; Source string }`
    - `func DefaultPath() string` — returns `~/.config/rice/state.json` on linux/darwin, `%APPDATA%/rice/state.json` on windows. Use `os.UserConfigDir()`.
    - `func Load(path string) (*State, error)` — if file missing, returns empty `*State` with SchemaVersion=1, NIL error
    - `func Save(path string, s *State) error` — atomic write (write to `path.tmp`, then `os.Rename`). Creates parent dirs.
    - Methods on `*State`: `SetPackage(name string, ps PackageState)`, `RemovePackage(name string)`, `GetPackage(name string) (PackageState, bool)`
  - Create `internal/state/state_test.go` covering: load missing file → empty state, round-trip save+load preserves data, atomic write doesn't corrupt on simulated failure (call Save, kill mid-write — best-effort), schema version mismatch error

  **Must NOT do**:
  - Don't reconcile state with filesystem here (that's `doctor`'s job)
  - Don't lock the file (no concurrent rice invocations expected; document the limitation in AGENTS.md)
  - Don't store secrets, env vars, or anything beyond installed link mapping

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Clear data model, well-known atomic-write pattern
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 1
  - Blocks: 10, 11, 17, 18
  - Blocked By: None

  **References**:
  - `os.UserConfigDir`: https://pkg.go.dev/os#UserConfigDir — handles XDG/APPDATA correctly
  - Atomic write pattern: write to temp file in same dir, then `os.Rename` (atomic on POSIX; "best effort" on Windows — document)
  - JSON encoding/decoding: standard `encoding/json`

  **Acceptance Criteria**:
  - [ ] `Load` on missing path returns empty state, nil error
  - [ ] `Save` then `Load` round-trips identical data
  - [ ] `DefaultPath()` returns platform-appropriate path
  - [ ] `go test ./internal/state/...` passes

  **QA Scenarios**:
  ```
  Scenario: Round-trip state
    Tool: Bash
    Preconditions: Task 4 code in place
    Steps:
      1. Run `go test ./internal/state/ -run TestRoundTrip -v`
    Expected Result: PASS — state loaded equals state saved
    Evidence: .sisyphus/evidence/task-4-roundtrip.txt

  Scenario: Missing file returns empty state
    Tool: Bash
    Preconditions: Task 4 code in place
    Steps:
      1. Run `go test ./internal/state/ -run TestLoad_Missing -v`
    Expected Result: PASS — empty state, no error
    Evidence: .sisyphus/evidence/task-4-missing.txt
  ```

  **Commit**: YES — `feat(state): state file format and read/write`
  - Files: `internal/state/state.go`, `internal/state/state_test.go`
  - Pre-commit: `go test ./internal/state/...`

- [x] 5. **Profile resolution rules + validation**

  **What to do**:
  - Create `internal/profile/profile.go` with:
    - `func Resolve(m *manifest.Manifest, profile string) ([]string, error)` — given a manifest and a chosen profile name, returns the ordered list of source subdirectories to stow. Algorithm:
      1. Validate `profile` is non-empty (error if not — profiles are now ALWAYS explicit, no implicit "common")
      2. Look up `m.Profiles[profile]` — error if not present (with helpful "valid profiles: [list of keys]")
      3. Return the profile's `Sources` slice as-is (the manifest author already declared the explicit ordering)
    - `func ValidateProfileName(name string) error` — reject empty, reject names containing path separators or spaces or shell meta-chars (allow alphanumeric + `-` + `_`)
  - Create `internal/profile/profile_test.go` covering:
    - Profile with single source `["macbook"]` → returns `["macbook"]`
    - Profile with multi sources `["common", "macbook"]` → returns them in declared order
    - Empty profile name → error
    - Unknown profile → error listing valid keys
    - Invalid profile name "../etc" → ValidateProfileName errors

  **Must NOT do**:
  - Don't read the filesystem here — pure logic on manifest data
  - Don't allow `..` or `/` in profile names
  - Don't infer profiles from environment or hostname (no auto-detection in v1)
  - Don't add any "common" magic — the resolver returns whatever the manifest author declared, no special-cased prepending

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure logic with table-driven tests
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES (depends on Task 2 schema, but only on the type definitions — can be developed against a stub)
  - Parallel Group: Wave 1
  - Blocks: 7, 10
  - Blocked By: None (uses Task 2 types but doesn't need its impl merged first if developed in same branch)

  **References**:
  - Existing ghostty/install.sh — implements the OLD common+platform pattern (which we generalize). In the new model, ghostty's manifest will declare `[profiles.macbook] sources = ["common", "macbook"]` explicitly.
  - Confirmed decision (round 4): profiles use `sources = [...]` lists; no implicit "common" — authors compose explicitly.

  **Acceptance Criteria**:
  - [ ] `Resolve` returns the manifest's declared `sources` list as-is (preserving order)
  - [ ] Empty profile name errors
  - [ ] Unknown profile error includes list of valid profile keys
  - [ ] `ValidateProfileName` rejects unsafe names
  - [ ] `go test ./internal/profile/...` passes

  **QA Scenarios**:
  ```
  Scenario: Resolution returns correct order
    Tool: Bash
    Preconditions: Task 5 code in place
    Steps:
      1. Run `go test ./internal/profile/ -run TestResolve -v`
    Expected Result: PASS — common appears before chosen profile in returned slice
    Evidence: .sisyphus/evidence/task-5-resolve.txt

  Scenario: Bad profile names rejected
    Tool: Bash
    Preconditions: Task 5 code in place
    Steps:
      1. Run `go test ./internal/profile/ -run TestValidateProfileName -v`
    Expected Result: PASS — names with `/`, `..`, spaces, shell metachars rejected
    Evidence: .sisyphus/evidence/task-5-validate.txt
  ```

  **Commit**: YES — `feat(profile): profile resolution and validation`
  - Files: `internal/profile/profile.go`, `internal/profile/profile_test.go`
  - Pre-commit: `go test ./internal/profile/...`

- [x] 6. **Delete stale dirs (profiles/, scripts/) + commit baseline**

  **What to do**:
  - Run `rm -rf profiles/ scripts/` from repo root
  - Verify with `git status` that these were never tracked (should not appear in `git status` after deletion since they were untracked)
  - Verify with `ls` that they are gone
  - This is a single, separate commit so the cleanup is auditable

  **Must NOT do**:
  - Don't delete any other dirs (ghostty/, nvim/, opencode/, etc. are all in active use)
  - Don't delete `.sisyphus/` (contains the plan!)
  - Don't run `git clean -fdx` — too broad, might wipe build artifacts or local IDE files

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Two `rm` commands plus verification
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 1
  - Blocks: None
  - Blocked By: None

  **References**:
  - Confirmed by user: "They're stale created by another agent. Delete them. They were never tracked."

  **Acceptance Criteria**:
  - [ ] `profiles/` directory does not exist
  - [ ] `scripts/` directory does not exist
  - [ ] `git status` does not show those paths

  **QA Scenarios**:
  ```
  Scenario: Stale dirs gone
    Tool: Bash
    Preconditions: Task 6 done
    Steps:
      1. Run `test ! -d profiles && test ! -d scripts && echo OK`
    Expected Result: Output "OK", exit code 0
    Evidence: .sisyphus/evidence/task-6-cleanup.txt
  ```

  **Commit**: NO (untracked dirs being deleted produce no diff). Skip commit; just confirm cleanup in a status report.

- [x] 7. **Manifest discovery + parsing**

  **What to do**:
  - Create `internal/manifest/discover.go` with:
    - `func Discover(repoRoot string) (map[string]*Manifest, error)` — scan immediate subdirectories of `repoRoot`, find `<dir>/rice.toml`, parse via TOML, validate. Skip directories without `rice.toml`. Skip hidden directories (starting with `.`). Returns map keyed by package name (which must equal directory name).
    - `func Load(path string) (*Manifest, error)` — parse a single `rice.toml`, validate, return.
    - On parse error or validation error, wrap with `fmt.Errorf("rice.toml at %s: %w", path, err)` so user sees the offending file
  - Create `internal/manifest/discover_test.go` using `testdata/discover/` fixture:
    - `testdata/discover/foo/rice.toml` (valid)
    - `testdata/discover/bar/rice.toml` (valid)
    - `testdata/discover/baz/` (no rice.toml — should be skipped silently)
    - `testdata/discover/.hidden/rice.toml` (should be skipped)
    - `testdata/discover/broken/rice.toml` (invalid — should cause discover to error)
  - Add at least 5 fixture files

  **Must NOT do**:
  - Don't scan recursively beyond one level — only immediate subdirs
  - Don't load packages from outside `repoRoot`
  - Don't silently swallow parse errors — surface them with file path

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Filesystem scan + TOML parse, well-defined fixture-based testing
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO (needs Task 2 merged for the schema types)
  - Parallel Group: Wave 2
  - Blocks: 8, 10, 18, 19, 20, 21
  - Blocked By: 2

  **References**:
  - Task 2 schema definitions
  - `os.ReadDir`: https://pkg.go.dev/os#ReadDir
  - BurntSushi/toml `DecodeFile`: https://pkg.go.dev/github.com/BurntSushi/toml#DecodeFile

  **Acceptance Criteria**:
  - [ ] `Discover` returns valid manifests, skips dirs without rice.toml, skips hidden dirs
  - [ ] `Discover` returns error on broken rice.toml with file path in error message
  - [ ] `go test ./internal/manifest/... -run TestDiscover` passes

  **QA Scenarios**:
  ```
  Scenario: Discover finds valid manifests
    Tool: Bash
    Preconditions: Task 7 code + testdata/discover/ fixtures in place
    Steps:
      1. Run `go test ./internal/manifest/ -run TestDiscover_Valid -v`
    Expected Result: PASS — returns map with "foo" and "bar" keys, NOT "baz" or ".hidden"
    Evidence: .sisyphus/evidence/task-7-discover.txt

  Scenario: Broken manifest surfaces file path
    Tool: Bash
    Preconditions: Task 7 + broken fixture
    Steps:
      1. Run `go test ./internal/manifest/ -run TestDiscover_Broken -v`
    Expected Result: PASS — error message contains "testdata/discover/broken/rice.toml"
    Evidence: .sisyphus/evidence/task-7-broken.txt
  ```

  **Commit**: YES — `feat(manifest): manifest discovery and parsing`
  - Files: `internal/manifest/discover.go`, `internal/manifest/discover_test.go`, `internal/manifest/testdata/discover/...`
  - Pre-commit: `go test ./internal/manifest/...`

- [x] 8. **Package OS gating**

  **What to do**:
  - Create `internal/installer/os_gate.go` with:
    - `func CheckOS(m *manifest.Manifest, currentOS string) error` — currentOS comes from `runtime.GOOS`. If currentOS not in `m.SupportedOS`, return error: `package "X" does not support OS "Y" (supported: linux, darwin)`
  - Add tests in `internal/installer/os_gate_test.go` for: supported OS passes, unsupported OS errors with helpful message

  **Must NOT do**:
  - Don't try to detect distro / architecture / version — just `runtime.GOOS`
  - Don't allow runtime override flags (e.g., `--ignore-os`) in v1

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Trivial check + test
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO (needs Task 7 merged for the parsed manifest type from Discover)
  - Parallel Group: Wave 2
  - Blocks: 10
  - Blocked By: 2, 7

  **References**:
  - `runtime.GOOS`: https://pkg.go.dev/runtime#pkg-constants
  - User decision: "separate apps should have some control over whether they are supported on that os or not, so we shouldn't even let the user install zsh on windows"

  **Acceptance Criteria**:
  - [ ] `CheckOS` returns nil when currentOS in SupportedOS
  - [ ] `CheckOS` returns descriptive error otherwise
  - [ ] Test coverage includes all 3 OS values + unsupported case

  **QA Scenarios**:
  ```
  Scenario: OS gate enforces support
    Tool: Bash
    Preconditions: Task 8 code in place
    Steps:
      1. Run `go test ./internal/installer/ -run TestCheckOS -v`
    Expected Result: PASS — supported passes, unsupported errors with package name and OS in message
    Evidence: .sisyphus/evidence/task-8-osgate.txt
  ```

  **Commit**: YES — `feat(installer): per-package OS gating`
  - Files: `internal/installer/os_gate.go`, `internal/installer/os_gate_test.go`
  - Pre-commit: `go test ./internal/installer/...`

- [x] 9. **Conflict detection**

  **What to do**:
  - Create `internal/installer/conflict.go` with:
    - `type Conflict struct { Target string; Reason string }` — returned for each conflict found
    - `func DetectConflicts(plannedLinks []PlannedLink) ([]Conflict, error)` where `PlannedLink struct { Target string; Source string }`. For each planned link:
      1. If target does not exist (Lstat fails with `os.IsNotExist`), no conflict.
      2. If target exists and is NOT a symlink, conflict: "target exists and is not a symlink"
      3. If target IS a symlink and resolves to the planned source, no conflict (idempotent re-install)
      4. If target IS a symlink but points elsewhere, conflict: "target is a symlink to a different location: X"
  - Tests in `internal/installer/conflict_test.go` using `t.TempDir()` covering each branch

  **Must NOT do**:
  - Don't auto-resolve any conflict (no backup, no force, no overwrite)
  - Don't follow symlinks when checking existence — use Lstat

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Discrete decision tree, easy to test
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO (depends on Task 3 symlink primitives)
  - Parallel Group: Wave 2
  - Blocks: 10
  - Blocked By: 3

  **References**:
  - Task 3 symlink package
  - `os.Lstat`, `os.IsNotExist`: standard library
  - User decision: "Abort on any conflict"

  **Acceptance Criteria**:
  - [ ] All 4 branches of detection logic covered by tests
  - [ ] Idempotent re-install (same source) returns NO conflict
  - [ ] Symlink to different target returns conflict with offending target in message
  - [ ] `go test ./internal/installer/... -run TestDetectConflicts` passes

  **QA Scenarios**:
  ```
  Scenario: Conflict detection covers all branches
    Tool: Bash
    Preconditions: Task 9 code in place
    Steps:
      1. Run `go test ./internal/installer/ -run TestDetectConflicts -v`
    Expected Result: PASS — all branches asserted
    Evidence: .sisyphus/evidence/task-9-conflicts.txt

  Scenario: Idempotent install returns zero conflicts
    Tool: Bash
    Preconditions: Task 9 code, plus a tmp setup with existing matching symlink
    Steps:
      1. Run `go test ./internal/installer/ -run TestDetectConflicts_Idempotent -v`
    Expected Result: PASS — zero conflicts when target already symlinks to source
    Evidence: .sisyphus/evidence/task-9-idempotent.txt
  ```

  **Commit**: YES — `feat(installer): conflict detection`
  - Files: `internal/installer/conflict.go`, `internal/installer/conflict_test.go`
  - Pre-commit: `go test ./internal/installer/...`

- [ ] 10. **Install orchestrator**

  **What to do**:
  - Create `internal/installer/install.go` with:
    - `type InstallRequest struct { RepoRoot string; PackageName string; Profile string; CurrentOS string; HomeDir string; StatePath string }`
    - `type InstallResult struct { LinksCreated []state.InstalledLink; Conflicts []Conflict }`
    - `func BuildInstallPlan(req InstallRequest) (*plan.Plan, error)` — pure planning, NO filesystem mutation:
      1. Discover manifests at RepoRoot, find req.PackageName (error if missing)
      2. CheckOS against req.CurrentOS (error if not supported)
      3. Resolve profile via `profile.Resolve(manifest, req.Profile)` → ordered list of source subdirs
      4. Walk each source subdir (skipping `rice.toml` files), compute the planned target path (target = HomeDir + relative path within source dir, with `$HOME` placeholder substitution)
      5. DetectConflicts on the planned links
      6. Build and return a `plan.Plan` (defined in Task 25) listing: package name, profile, ordered list of `Op{Kind: Create, Source, Target}`, and any detected conflicts
      7. If conflicts exist, return the plan with conflicts populated AND an error so the caller can format the conflict report. Caller decides whether to render the plan with conflicts shown or just bail.
    - `func ExecuteInstallPlan(p *plan.Plan, statePath string) (*InstallResult, error)` — applies the plan to the filesystem:
      1. For each Op: CreateSymlink. If a Create fails, log ERROR with details, attempt no rollback, return error and SAVE partial state so user can `doctor` / `uninstall` to clean up.
      2. Update state file: set p.PackageName → {Profile, InstalledLinks, InstalledAt: now}
      3. Save state file
      4. Return InstallResult
    - `func Install(req InstallRequest) (*InstallResult, error)` — convenience wrapper that calls BuildInstallPlan then ExecuteInstallPlan with no confirmation. Used in tests. CLI layer (Task 14) calls BuildInstallPlan and ExecuteInstallPlan separately so it can render+confirm in between.
  - Logging: use the package-level logger from Task 24:
    - DEBUG: each file walked, each planned op
    - INFO: "Installing X (profile: Y)" / "Installed X: N symlinks"
    - WARN: skipped files (rice.toml itself, etc.)
    - ERROR: symlink creation failures
  - Tests in `internal/installer/install_test.go` using `testdata/install/` fixture: a fake repo with one package having profiles like `[profiles.macbook] sources = ["common", "macbook"]`; install into `t.TempDir()` and assert symlinks exist with correct targets. Test `BuildInstallPlan` returns expected Op list without touching FS.

  **Must NOT do**:
  - Don't symlink directories — only files (avoids `~/.config` clobber)
  - Don't proceed in BuildInstallPlan to ExecuteInstallPlan automatically — the CLI layer must call them separately to insert the confirmation step. Only the `Install` convenience wrapper composes them.
  - Don't include `rice.toml` files in the planned ops (skip during walk)
  - Don't write state file until ALL symlinks attempted; on partial failure, save what succeeded
  - Don't allow target paths to escape HomeDir (defense in depth — even if manifest is tampered)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Multiple components composed, file-tree walking, error handling at each step. Higher cognitive load than the leaf utilities.
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO (needs all of Wave 1 + early Wave 2)
  - Parallel Group: Wave 2
  - Blocks: 12, 14
  - Blocked By: 3, 4, 7, 8, 9, 24, 25

  **References**:
  - All of Wave 1 (manifest, symlink, state, profile)
  - Task 8 OS gate
  - Task 9 conflict detection
  - Task 24 logger package
  - Task 25 plan package (`plan.Plan`, `plan.Op`)
  - `filepath.Walk` / `filepath.WalkDir`: https://pkg.go.dev/path/filepath#WalkDir
  - Stow's file-symlinking-over-folding behavior — we DON'T do folding (folding = symlinking whole dirs when possible). File-level only for v1.

  **Acceptance Criteria**:
  - [ ] `BuildInstallPlan` returns a plan without touching the filesystem (verified by checking no files appear in tmp HOME after the call)
  - [ ] `ExecuteInstallPlan` creates all expected symlinks
  - [ ] BuildInstallPlan with conflict returns plan with conflicts populated AND error
  - [ ] State file updated with correct profile + InstalledLinks list after Execute
  - [ ] Idempotent: re-running Install with same args succeeds with no changes
  - [ ] Install on unsupported OS returns OS gate error
  - [ ] `rice.toml` files inside source dirs are NOT included in planned ops
  - [ ] `go test ./internal/installer/... -run TestInstall` passes

  **QA Scenarios**:
  ```
  Scenario: Install creates expected symlink tree
    Tool: Bash
    Preconditions: Task 10 + testdata/install/ fixture
    Steps:
      1. Run `go test ./internal/installer/ -run TestInstall_Happy -v`
    Expected Result: PASS — symlinks present at expected target paths, all pointing to expected sources
    Evidence: .sisyphus/evidence/task-10-install.txt

  Scenario: Install aborts on conflict
    Tool: Bash
    Preconditions: Task 10 + fixture with pre-existing conflicting file
    Steps:
      1. Run `go test ./internal/installer/ -run TestInstall_ConflictAborts -v`
    Expected Result: PASS — error returned, NO symlinks created, state.json unchanged
    Evidence: .sisyphus/evidence/task-10-conflict.txt

  Scenario: OS gate honored
    Tool: Bash
    Preconditions: Task 10 + manifest declaring linux only
    Steps:
      1. Run `go test ./internal/installer/ -run TestInstall_OSGate -v`
    Expected Result: PASS — install with currentOS=darwin returns OS error
    Evidence: .sisyphus/evidence/task-10-osgate.txt
  ```

  **Commit**: YES — `feat(installer): install orchestrator`
  - Files: `internal/installer/install.go`, `internal/installer/install_test.go`, `internal/installer/testdata/install/...`
  - Pre-commit: `go test ./internal/installer/... -race`

- [ ] 11. **Uninstall orchestrator**

  **What to do**:
  - In `internal/installer/uninstall.go`:
    - `type UninstallRequest struct { PackageName string; StatePath string }`
    - `func BuildUninstallPlan(req UninstallRequest) (*plan.Plan, error)` — pure planning, NO filesystem mutation: load state, find package's InstalledLinks, build a plan with `Op{Kind: Remove, Target}` for each. If package not in state, return error.
    - `func ExecuteUninstallPlan(p *plan.Plan, statePath string) error` — for each Op: verify it's still our symlink (using `IsSymlinkTo`), Remove it. If a link is missing or has been replaced by something else, log a WARN but continue (don't error — partial cleanup is acceptable; doctor will surface the remaining drift). After processing, remove package entry from state and save.
    - `func Uninstall(req UninstallRequest) error` — convenience wrapper for tests.
  - Logging:
    - DEBUG: each link being removed
    - INFO: "Uninstalling X" / "Uninstalled X: removed N (skipped M due to drift)"
    - WARN: drift cases (link missing, link replaced by file, link points elsewhere)
  - Tests covering: clean uninstall, partial drift (some links manually deleted), package not in state (error), BuildUninstallPlan does not touch FS.

  **Must NOT do**:
  - Don't remove files that are NOT symlinks (defensive: even if state says we own it, if it's now a regular file, don't touch it)
  - Don't remove the parent directory of removed symlinks (avoid `os.Remove` on dirs entirely)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Inverse of install, simpler control flow
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO (depends on state + symlink + logger + plan)
  - Parallel Group: Wave 2
  - Blocks: 12, 15
  - Blocked By: 3, 4, 24, 25

  **References**:
  - Task 3 symlink package (Remove + IsSymlinkTo)
  - Task 4 state package
  - Task 24 logger
  - Task 25 plan package

  **Acceptance Criteria**:
  - [ ] BuildUninstallPlan returns plan without touching FS
  - [ ] ExecuteUninstallPlan removes all package symlinks
  - [ ] Uninstall on missing package returns clear error
  - [ ] Drifted links (replaced by regular files) are skipped with warning, not error
  - [ ] State entry removed after uninstall

  **QA Scenarios**:
  ```
  Scenario: Clean uninstall
    Tool: Bash
    Preconditions: Task 11 + state populated by an Install
    Steps:
      1. Run `go test ./internal/installer/ -run TestUninstall_Happy -v`
    Expected Result: PASS — all symlinks removed, state entry gone
    Evidence: .sisyphus/evidence/task-11-uninstall.txt

  Scenario: Drift tolerated
    Tool: Bash
    Preconditions: Task 11 + state with one link manually replaced by a regular file
    Steps:
      1. Run `go test ./internal/installer/ -run TestUninstall_Drift -v`
    Expected Result: PASS — uninstall completes, regular file untouched, warning emitted
    Evidence: .sisyphus/evidence/task-11-drift.txt
  ```

  **Commit**: YES — `feat(installer): uninstall orchestrator`
  - Files: `internal/installer/uninstall.go`, `internal/installer/uninstall_test.go`
  - Pre-commit: `go test ./internal/installer/... -race`

- [ ] 12. **Switch orchestrator with pre-flight validation**

  **What to do**:
  - In `internal/installer/switch.go`:
    - `type SwitchRequest struct { RepoRoot string; PackageName string; NewProfile string; CurrentOS string; HomeDir string; StatePath string }`
    - `type SwitchPlan struct { Uninstall *plan.Plan; Install *plan.Plan }`
    - `func BuildSwitchPlan(req SwitchRequest) (*SwitchPlan, error)` — pure planning:
      1. Load state. If package not currently installed, error: "package not installed; use install instead"
      2. Build the BuildUninstallPlan for the current profile
      3. Build the BuildInstallPlan for the NEW profile
      4. **Pre-flight conflict check**: simulate conflict detection for the NEW plan assuming the OLD links are gone. If any non-rice file would conflict with the NEW set, return the SwitchPlan with conflicts populated AND an error so the caller can render the conflict report. (Use a temporary ignore-list of the OLD links to exclude them from conflict detection.)
      5. Return the combined SwitchPlan
    - `func ExecuteSwitchPlan(p *SwitchPlan, statePath string) error` — applies in order:
      1. ExecuteUninstallPlan
      2. ExecuteInstallPlan
      3. If install step fails after uninstall succeeded, log ERROR with clear recovery message: "switch left package uninstalled; run `rice install <pkg> --profile X` to recover"
    - `func Switch(req SwitchRequest) error` — convenience wrapper for tests.
  - Logging:
    - INFO: "Switching X from profile A to profile B"
    - DEBUG: pre-flight summary (N old links to remove, M new links to create)
    - ERROR: pre-flight conflicts; install failure after uninstall succeeded
  - Tests covering: happy switch, pre-flight catches conflict (no state change), package not installed error, BuildSwitchPlan does not touch FS.

  **Must NOT do**:
  - Don't proceed past pre-flight if any conflict is detected
  - Don't try to be clever and only diff old/new sets — just full uninstall + reinstall (per user's decision)
  - Don't leave state inconsistent — even on failure, state should reflect filesystem reality at that moment (uninstall already saves state)
  - Don't auto-execute after BuildSwitchPlan — CLI layer (Task 16) inserts confirmation between Build and Execute

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Critical destructive operation needing pre-flight + recovery messaging. Highest blast radius.
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO
  - Parallel Group: Wave 2
  - Blocks: 16
  - Blocked By: 10, 11

  **References**:
  - Tasks 10 (Install) and 11 (Uninstall)
  - Metis gap: switch atomicity. Resolution: pre-flight + clear recovery message. NOT transactional.
  - User decision: "Full re-stow."

  **Acceptance Criteria**:
  - [ ] Pre-flight aborts if NEW profile would conflict
  - [ ] Pre-flight does NOT touch the filesystem
  - [ ] Happy path: switch swaps profile, state.json reflects new profile
  - [ ] Switch on uninstalled package errors clearly
  - [ ] `go test ./internal/installer/... -run TestSwitch` passes

  **QA Scenarios**:
  ```
  Scenario: Switch happy path
    Tool: Bash
    Preconditions: Task 12 + fixture with package installed at profile A
    Steps:
      1. Run `go test ./internal/installer/ -run TestSwitch_Happy -v`
    Expected Result: PASS — profile B's symlinks present, profile A's gone, state shows B
    Evidence: .sisyphus/evidence/task-12-switch.txt

  Scenario: Pre-flight blocks destructive action
    Tool: Bash
    Preconditions: Task 12 + fixture where switching would conflict with a non-rice file at the new target
    Steps:
      1. Run `go test ./internal/installer/ -run TestSwitch_PreflightAbort -v`
    Expected Result: PASS — error returned, profile A still installed (state unchanged, symlinks intact)
    Evidence: .sisyphus/evidence/task-12-preflight.txt
  ```

  **Commit**: YES — `feat(installer): switch with pre-flight validation`
  - Files: `internal/installer/switch.go`, `internal/installer/switch_test.go`
  - Pre-commit: `go test ./internal/installer/... -race`

- [ ] 13. **CLI scaffold + cobra setup + root cmd**

  **What to do**:
  - In `cmd/rice/main.go`: replace stub with `cmd.Execute()` invocation
  - In `cmd/rice/cmd/root.go`: define root cobra command with description, version (read from a const), and global flags:
    - `--repo` (defaults to current working directory; lets tests use temp repos)
    - `--state` (defaults to `state.DefaultPath()`)
    - `--log-level` (string: `debug|info|warn|error|critical`; default `warn`; falls back to `RICE_LOG_LEVEL` env var if flag unset; flag wins over env)
    - `--yes` / `-y` (bool; bypass confirmation prompts on destructive ops; default false)
  - Wire `PersistentPreRunE` to initialize the global `logger` package (Task 24) using the resolved log level — must happen before any subcommand runs
  - In `cmd/rice/cmd/version.go`: add `version` subcommand printing the version
  - Add `cmd/rice/cmd/root_test.go` smoke test invoking `rice version` via cobra's test helpers + a test asserting that `--log-level invalid` errors clearly + a test asserting `RICE_LOG_LEVEL=debug` is honored when no flag is passed

  **Must NOT do**:
  - Don't add `init` command (out of scope)
  - Don't add interactive menus or wizards (the only allowed interactive surface is the y/N confirmation in destructive ops, which is implemented in Tasks 14-16, NOT here)
  - Don't add unrelated global flags (`--verbose`, `--debug`) — `--log-level` covers all verbosity needs

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Cobra boilerplate
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO (needs Task 1 module init AND Task 24 logger)
  - Parallel Group: Wave 3
  - Blocks: 14-18
  - Blocked By: 1, 24

  **References**:
  - Cobra command groups: https://github.com/spf13/cobra/blob/main/site/content/user_guide.md#create-rootcmd
  - Task 24 logger package — root cmd initializes it from --log-level / RICE_LOG_LEVEL

  **Acceptance Criteria**:
  - [ ] `rice version` prints version string
  - [ ] `rice --help` lists commands and global flags (`--repo`, `--state`, `--log-level`, `--yes`)
  - [ ] `rice --log-level invalid` exits non-zero with clear error listing valid values
  - [ ] `RICE_LOG_LEVEL=debug rice version` initializes logger at debug
  - [ ] `--log-level info` overrides `RICE_LOG_LEVEL=debug`

  **QA Scenarios**:
  ```
  Scenario: rice version works
    Tool: Bash
    Preconditions: Task 13 + go build
    Steps:
      1. Build: `go build -o /tmp/rice ./cmd/rice`
      2. Run: `/tmp/rice version`
      3. Assert exit code 0 and stdout matches `^[0-9]+\.[0-9]+\.[0-9]+`
    Expected Result: Version string printed
    Evidence: .sisyphus/evidence/task-13-version.txt
  ```

  **Commit**: YES — `feat(cli): scaffold rice CLI with cobra`
  - Files: `cmd/rice/main.go`, `cmd/rice/cmd/root.go`, `cmd/rice/cmd/version.go`, `cmd/rice/cmd/root_test.go`
  - Pre-commit: `go build ./... && go test ./cmd/...`

- [ ] 14. **install command**

  **What to do**:
  - `cmd/rice/cmd/install.go`: cobra command `rice install <package> --profile <name>`. Flow:
    1. Parse args, build `installer.InstallRequest`
    2. Call `installer.BuildInstallPlan(req)` — get the plan
    3. If plan has conflicts: print conflict report to stderr, exit 1 (no prompt — nothing to confirm)
    4. Render the plan to stdout via `prompt.RenderPlan(p)` (Task 25) — full listing of every symlink to be created
    5. If `--yes` flag (read from root cmd) is true: skip prompt and proceed
    6. Otherwise: call `prompt.Confirm("Proceed?")` — reads from stdin, default NO on Enter, accepts y/yes/Y; anything else aborts with "Cancelled." on stdout, exit 0
    7. If confirmed: call `installer.ExecuteInstallPlan(p, statePath)` — print success summary
    8. On execute error: write to stderr with `Error: ...` prefix, exit 1
  - Output format for plan rendering (Task 25 owns this format; install just uses it):
    ```
    Plan: install nvim (profile: common)
      CREATE  /Users/guneet/.config/nvim/init.lua → /Users/guneet/rice/nvim/common/.config/nvim/init.lua
      CREATE  /Users/guneet/.config/nvim/lua/options.lua → ...
      ... (every symlink listed, no truncation)
    Total: 12 symlinks to create.
    Proceed? [y/N]:
    ```
  - Add `cmd/rice/cmd/install_test.go` testing via `RootCmd.SetArgs(...)` invoking install in a tmp repo+home, with a fake stdin reader providing "y\n" or "n\n" or empty. Also test `--yes` bypass.

  **Must NOT do**:
  - Don't add `--all`, `--force`, `--dry-run` in v1 (explicitly out of scope)
  - Don't write progress bars or animations
  - Don't capture stderr into stdout (preserve stream separation)
  - Don't skip the plan rendering even with `--yes` — user should see what was done in scrollback. `--yes` only skips the prompt, not the plan output.
  - Don't truncate the plan output (no "... and 47 more"). Print every single op.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Thin CLI layer over Task 10
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES with 15-18
  - Parallel Group: Wave 3
  - Blocks: 22, 23
  - Blocked By: 10, 13, 25

  **References**:
  - Task 10 install orchestrator (BuildInstallPlan + ExecuteInstallPlan)
  - Task 13 cobra root + `--yes` flag
  - Task 25 plan + prompt package (RenderPlan, Confirm)

  **Acceptance Criteria**:
  - [ ] `rice install <pkg> --profile <name>` prints plan, prompts y/N, exits 0 on "y"
  - [ ] Bare Enter at prompt cancels (no symlinks created), exits 0 with "Cancelled."
  - [ ] `n` / `no` cancels
  - [ ] `--yes` flag bypasses prompt, still prints plan
  - [ ] Exits 1 with conflict report on conflict (no prompt shown when conflicts exist)
  - [ ] Exits 1 with OS gate error on unsupported OS
  - [ ] `--profile` is required (cobra will error if missing)

  **QA Scenarios**:
  ```
  Scenario: install command happy path (with confirmation)
    Tool: Bash
    Preconditions: Task 14 + tmp fixture repo
    Steps:
      1. Build binary
      2. Run `printf 'y\n' | HOME=$tmpHome /tmp/rice --repo $tmpRepo install foo --profile common` (pipe "y" to confirm the destructive prompt)
      3. Assert exit code 0
      4. Assert stdout contains "Plan: install foo" and "Total: N symlinks"
      5. Assert symlinks exist at expected paths
    Expected Result: Plan printed, "y" confirms, symlinks created, success message on stdout
    Evidence: .sisyphus/evidence/task-14-install.txt

  Scenario: install command --yes bypasses prompt
    Tool: Bash
    Preconditions: Task 14 + tmp fixture repo
    Steps:
      1. Run `HOME=$tmpHome /tmp/rice --repo $tmpRepo install foo --profile common --yes` (no stdin)
      2. Assert exit code 0
      3. Assert stdout contains the plan AND no "[y/N]" prompt text
      4. Assert symlinks exist
    Expected Result: Plan printed without prompt, symlinks created
    Evidence: .sisyphus/evidence/task-14-install-yes.txt

  Scenario: install command bare Enter cancels
    Tool: Bash
    Preconditions: Task 14 + tmp fixture repo
    Steps:
      1. Run `printf '\n' | HOME=$tmpHome /tmp/rice --repo $tmpRepo install foo --profile common`
      2. Assert exit code 0
      3. Assert stdout contains "Cancelled."
      4. Assert NO symlinks were created
    Expected Result: Default-NO honored, no FS changes
    Evidence: .sisyphus/evidence/task-14-install-cancel.txt

  Scenario: install command surfaces conflict (no prompt)
    Tool: Bash
    Preconditions: Task 14 + tmp fixture with pre-existing conflict
    Steps:
      1. Pre-create a regular file at the would-be target path
      2. Run `HOME=$tmpHome /tmp/rice --repo $tmpRepo install foo --profile common < /dev/null` (no stdin needed; no prompt should appear when conflicts exist)
      3. Assert exit code 1
      4. Assert stderr contains "conflict" and the offending path
      5. Assert stdout does NOT contain "[y/N]"
    Expected Result: Clean abort, no symlinks created, no prompt shown
    Evidence: .sisyphus/evidence/task-14-install-conflict.txt
  ```

  **Commit**: YES — `feat(cli): install command`
  - Files: `cmd/rice/cmd/install.go`, `cmd/rice/cmd/install_test.go`
  - Pre-commit: `go build ./... && go test ./cmd/...`

- [ ] 15. **uninstall command**

  **What to do**:
  - `cmd/rice/cmd/uninstall.go`: `rice uninstall <package>`. Flow:
    1. Build `UninstallRequest`, call `installer.BuildUninstallPlan(req)`
    2. Render plan via `prompt.RenderPlan(p)` — full listing of every symlink to be removed
    3. If `--yes`: skip prompt. Otherwise call `prompt.Confirm("Proceed?")`. Default NO on Enter.
    4. If confirmed: call `installer.ExecuteUninstallPlan(p, statePath)`
    5. Print summary: `Done. N symlinks removed (M skipped due to drift).`
  - Plan output format (rendered via Task 25):
    ```
    Plan: uninstall nvim
      REMOVE  /Users/guneet/.config/nvim/init.lua
      REMOVE  /Users/guneet/.config/nvim/lua/options.lua
      ... (every link listed)
    Total: 12 symlinks to remove.
    Proceed? [y/N]:
    ```
  - Add corresponding test covering: confirm yes, confirm no (Enter), --yes bypass, package not in state.

  **Must NOT do**:
  - Don't add `--purge` or anything that touches files outside the recorded symlinks
  - Don't truncate the plan output
  - Don't skip plan rendering when `--yes` is set

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Thin CLI layer over Task 11
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 3
  - Blocks: 22, 23
  - Blocked By: 11, 13, 25

  **References**: Task 11, Task 13, Task 25

  **Acceptance Criteria**:
  - [ ] `rice uninstall <pkg>` prints plan, prompts y/N, exits 0 on success
  - [ ] Bare Enter cancels with no FS changes
  - [ ] `--yes` bypasses prompt, still prints plan
  - [ ] Exits 1 with clear error if package not in state
  - [ ] Drift skips reported in summary

  **QA Scenarios**:
  ```
  Scenario: uninstall removes symlinks (with confirmation)
    Tool: Bash
    Preconditions: Task 15 + an installed package via install command
    Steps:
      1. Run install in tmp env (with --yes)
      2. Run `printf 'y\n' | rice uninstall foo` (pipe "y" to confirm)
      3. Assert exit 0
      4. Assert stdout contains "Plan: uninstall foo"
      5. Assert symlinks gone, state.json no longer contains "foo"
    Expected Result: Clean uninstall after confirmation
    Evidence: .sisyphus/evidence/task-15-uninstall.txt

  Scenario: uninstall --yes bypass
    Tool: Bash
    Preconditions: Task 15 + installed package
    Steps:
      1. Run `rice uninstall foo --yes` (no stdin)
      2. Assert exit 0
      3. Assert plan printed but no "[y/N]" text
      4. Assert symlinks gone
    Expected Result: Plan shown, prompt skipped, uninstall succeeds
    Evidence: .sisyphus/evidence/task-15-uninstall-yes.txt

  Scenario: uninstall bare Enter cancels
    Tool: Bash
    Preconditions: Task 15 + installed package
    Steps:
      1. Run `printf '\n' | rice uninstall foo`
      2. Assert exit 0, stdout contains "Cancelled."
      3. Assert symlinks STILL EXIST and state.json STILL contains "foo"
    Expected Result: Default-NO honored
    Evidence: .sisyphus/evidence/task-15-uninstall-cancel.txt
  ```

  **Commit**: YES — `feat(cli): uninstall command`
  - Files: `cmd/rice/cmd/uninstall.go`, `cmd/rice/cmd/uninstall_test.go`
  - Pre-commit: `go build ./... && go test ./cmd/...`

- [ ] 16. **switch command**

  **What to do**:
  - `cmd/rice/cmd/switch.go`: `rice switch <package> <newProfile>`. Flow:
    1. Build `SwitchRequest`, call `installer.BuildSwitchPlan(req)`
    2. If pre-flight conflict: print conflict report to stderr, exit 1 (no prompt)
    3. Render combined SwitchPlan via `prompt.RenderSwitchPlan(p)` showing BOTH the uninstall ops AND the install ops
    4. If `--yes`: skip prompt. Otherwise call `prompt.Confirm(...)`. Default NO.
    5. If confirmed: call `installer.ExecuteSwitchPlan(p, statePath)`
    6. On post-uninstall install failure: exit 1 with recovery instructions
  - Plan output format:
    ```
    Plan: switch opencode (work → personal)
      Uninstall (profile: work):
        REMOVE  /Users/guneet/.config/opencode/agents.toml
        ...
      Install (profile: personal):
        CREATE  /Users/guneet/.config/opencode/agents.toml → ...
        ...
    Total: 8 symlinks to remove, 12 to create.
    Proceed? [y/N]:
    ```
  - Test covering: confirm yes/no, --yes, pre-flight conflict (no prompt), happy switch.

  **Must NOT do**:
  - Don't accept multiple `<package> <profile>` pairs in v1 (one at a time)
  - Don't add `--dry-run`
  - Don't truncate the plan output

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Thin CLI layer over Task 12
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 3
  - Blocks: 22, 23
  - Blocked By: 12, 13, 25

  **References**: Task 12, Task 13, Task 25

  **Acceptance Criteria**:
  - [ ] `rice switch <pkg> <profile>` prints combined plan, prompts y/N, exits 0 on success
  - [ ] Bare Enter cancels with no FS changes
  - [ ] `--yes` bypasses prompt, still prints plan
  - [ ] Pre-flight failure exits 1 with no state change AND no prompt
  - [ ] Recovery message printed if install step fails post-uninstall

  **QA Scenarios**:
  ```
  Scenario: switch flips profile (with confirmation)
    Tool: Bash
    Preconditions: Task 16 + fixture with multi-profile package installed
    Steps:
      1. Install package with profile=A using `--yes`
      2. Run `printf 'y\n' | rice switch <pkg> B` (pipe "y" to confirm)
      3. Assert exit 0
      4. Assert stdout contains both "Uninstall (profile: A):" and "Install (profile: B):"
      5. Assert state.json shows profile=B
      6. Assert symlinks point into the B subdir of the package source
    Expected Result: Profile swapped cleanly after confirmation
    Evidence: .sisyphus/evidence/task-16-switch.txt

  Scenario: switch --yes bypass
    Tool: Bash
    Preconditions: Task 16 + fixture installed at profile A
    Steps:
      1. Run `rice switch <pkg> B --yes` (no stdin)
      2. Assert exit 0, plan printed, no "[y/N]" text
      3. Assert state.json shows profile=B
    Expected Result: Combined plan shown, prompt skipped
    Evidence: .sisyphus/evidence/task-16-switch-yes.txt

  Scenario: switch bare Enter cancels
    Tool: Bash
    Preconditions: Task 16 + fixture installed at profile A
    Steps:
      1. Run `printf '\n' | rice switch <pkg> B`
      2. Assert exit 0, "Cancelled." on stdout
      3. Assert state.json STILL shows profile=A
      4. Assert symlinks STILL point into A subdir
    Expected Result: Default-NO honored, no FS or state changes
    Evidence: .sisyphus/evidence/task-16-switch-cancel.txt

  Scenario: switch pre-flight conflict skips prompt
    Tool: Bash
    Preconditions: Task 16 + fixture where profile B would conflict with a non-rice file
    Steps:
      1. Pre-create a non-rice file at one of profile B's planned target paths
      2. Run `rice switch <pkg> B < /dev/null`
      3. Assert exit 1
      4. Assert stderr contains conflict report
      5. Assert stdout does NOT contain "[y/N]" (no prompt when conflicts exist)
      6. Assert state.json STILL shows profile=A
    Expected Result: Conflict aborts before any prompt or FS change
    Evidence: .sisyphus/evidence/task-16-switch-conflict.txt
  ```

  **Commit**: YES — `feat(cli): switch command`
  - Files: `cmd/rice/cmd/switch.go`, `cmd/rice/cmd/switch_test.go`
  - Pre-commit: `go build ./... && go test ./cmd/...`

- [ ] 17. **status command**

  **What to do**:
  - `cmd/rice/cmd/status.go`: `rice status`. Reads state file, prints table of installed packages with profile and symlink count. If state file missing, prints "no packages installed".
  - Output:
    ```
    PACKAGE    PROFILE    LINKS    INSTALLED AT
    ghostty    macbook    3        2026-05-10 17:00
    nvim       common     42       2026-05-10 17:01
    opencode   work       18       2026-05-10 17:02
    ```
  - Test with empty state and populated state.

  **Must NOT do**:
  - Don't fetch filesystem state (that's `doctor`'s job — `status` is purely about what we recorded)
  - Don't add `--json` output in v1 (defer until needed)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Read state, format table
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 3
  - Blocks: 22, 23
  - Blocked By: 4, 13

  **References**: Task 4, Task 13. Tab-aligned formatting via `text/tabwriter` stdlib.

  **Acceptance Criteria**:
  - [ ] `rice status` prints table when packages installed
  - [ ] Prints "no packages installed" when state missing/empty
  - [ ] Exits 0 in both cases

  **QA Scenarios**:
  ```
  Scenario: status with packages
    Tool: Bash
    Preconditions: Task 17 + populated state
    Steps:
      1. Install at least 2 packages
      2. Run `rice status`
      3. Assert exit 0
      4. Assert stdout contains both package names + their profiles
    Expected Result: Table printed
    Evidence: .sisyphus/evidence/task-17-status.txt

  Scenario: status with no state
    Tool: Bash
    Preconditions: Task 17, empty HOME
    Steps:
      1. Run `rice status` with no state file
    Expected Result: Exit 0, prints "no packages installed"
    Evidence: .sisyphus/evidence/task-17-empty.txt
  ```

  **Commit**: YES — `feat(cli): status command`
  - Files: `cmd/rice/cmd/status.go`, `cmd/rice/cmd/status_test.go`
  - Pre-commit: `go build ./... && go test ./cmd/...`

- [ ] 18. **doctor command**

  **What to do**:
  - `cmd/rice/cmd/doctor.go`: `rice doctor`. Performs:
    1. Verify state file exists and parses
    2. For each package in state, for each InstalledLink: check the symlink exists and points to the recorded source. Report drift (missing link, replaced by file, points elsewhere)
    3. Discover manifests at --repo. Report packages in state but no longer in repo.
    4. On Windows, check whether `os.Symlink` is likely to work: try creating a symlink in a tmp dir. If it fails, print message: "Windows symlinks require Developer Mode. Enable: Settings → Privacy & security → For developers → Developer Mode."
    5. Verify all rice.toml files in repo parse + validate (regardless of install state).
  - Output: `OK` lines for healthy, `WARN`/`ERROR` lines for drift, exit 0 if all OK, exit 1 if any ERROR.
  - **Doctor MUST NOT mutate anything**. Read-only diagnostic.
  - Test cases for: clean state, drifted state, missing rice.toml, broken rice.toml.

  **Must NOT do**:
  - Don't auto-fix anything (per Metis guardrail)
  - Don't run network checks
  - Don't check for app installations (vim, ghostty binary, etc.) — out of scope

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Read-only checks composing existing pieces
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES
  - Parallel Group: Wave 3
  - Blocks: 22, 23
  - Blocked By: 4, 7, 13

  **References**: Tasks 4, 7, 13. `os.UserConfigDir()` for the Windows symlink probe location (use a tmp subdir).

  **Acceptance Criteria**:
  - [ ] `rice doctor` exits 0 on a clean system
  - [ ] Exits 1 if any link is drifted/missing
  - [ ] Reports Windows Developer Mode hint when symlink probe fails
  - [ ] Never mutates filesystem

  **QA Scenarios**:
  ```
  Scenario: doctor on clean install
    Tool: Bash
    Preconditions: Task 18 + a fresh successful install
    Steps:
      1. Install a package
      2. Run `rice doctor`
      3. Assert exit 0, "OK" lines present, no "ERROR"
    Expected Result: Clean report
    Evidence: .sisyphus/evidence/task-18-doctor-clean.txt

  Scenario: doctor detects drift
    Tool: Bash
    Preconditions: Task 18 + install + manually delete one symlink
    Steps:
      1. Install a package
      2. `rm` one of the created symlinks
      3. Run `rice doctor`
      4. Assert exit 1, "ERROR" or "WARN" line referencing the missing symlink
    Expected Result: Drift surfaced
    Evidence: .sisyphus/evidence/task-18-doctor-drift.txt
  ```

  **Commit**: YES — `feat(cli): doctor command`
  - Files: `cmd/rice/cmd/doctor.go`, `cmd/rice/cmd/doctor_test.go`
  - Pre-commit: `go build ./... && go test ./cmd/... -race`

- [ ] 19. **Add rice.toml to nvim, zsh, hyprland, waybar, wofi**

  **What to do**:
  - For each of these 5 packages, add a `rice.toml` at `<package>/rice.toml`. Most are single-profile (just one profile named "common" with `sources = ["."]` to stow the package dir itself). Use the schema from Task 2.
  - **Investigate first**: `ls -la <package>/` for each to determine current layout. They are likely flat (no subdirs) — manifest will use `sources = ["."]` so installer walks the package directory directly (excluding `rice.toml`).
  - The installer (Task 10) must skip `rice.toml` files when walking. Add a test for this in Task 10's tests.
  - Concrete manifests:
    ```toml
    # nvim/rice.toml
    schema_version = 1
    name = "nvim"
    description = "Neovim configuration"
    supported_os = ["linux", "darwin", "windows"]
    target = "$HOME"
    [profiles.common]
    sources = ["."]
    ```
    Same shape for `zsh` (linux+darwin), `hyprland` (linux), `waybar` (linux), `wofi` (linux). Each has exactly one profile named `common` with `sources = ["."]`.

  **Must NOT do**:
  - Don't restructure these packages' contents (only add the manifest)
  - Don't add OSes that the package can't actually support (e.g., zsh on Windows is excluded; Windows has its own shell)
  - Don't invent profiles for packages that don't need them — single "common" profile is fine

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Five small TOML files
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES with 20, 21
  - Parallel Group: Wave 4
  - Blocks: F1-F4
  - Blocked By: 7

  **References**:
  - Task 2 (schema), Task 7 (manifest discovery)

  **Acceptance Criteria**:
  - [ ] All 5 packages have a `rice.toml` that parses cleanly via `manifest.Load`
  - [ ] `rice install nvim --profile common` works in a clean tmp env (after wiring; smoke-test in F3)
  - [ ] OS gating: `rice install zsh --profile common` on `windows` errors with the OS gate message

  **QA Scenarios**:
  ```
  Scenario: Manifests parse
    Tool: Bash
    Preconditions: Task 19
    Steps:
      1. Run `go test ./internal/manifest/ -run TestLoadAllRepoManifests -v` (loads every rice.toml in the repo)
    Expected Result: PASS — all 5 manifests load
    Evidence: .sisyphus/evidence/task-19-manifests.txt
  ```

  **Commit**: YES — `feat(packages): add rice.toml to nvim, zsh, hyprland, waybar, wofi`
  - Files: `nvim/rice.toml`, `zsh/rice.toml`, `hyprland/rice.toml`, `waybar/rice.toml`, `wofi/rice.toml`
  - Pre-commit: `go test ./internal/manifest/...`

- [ ] 20. **Migrate ghostty (delete install.sh, restructure, add rice.toml)**

  **What to do**:
  - Read current `ghostty/install.sh` to understand the common+overlay logic. Confirm: it copies `ghostty/common/` then overlays `ghostty/<machine>/`.
  - The directory structure is already aligned with our profile model: `ghostty/common/`, `ghostty/macbook/`, `ghostty/devstick/`. Each contains `.config/ghostty/...` files.
  - Add `ghostty/rice.toml`:
    ```toml
    schema_version = 1
    name = "ghostty"
    description = "Ghostty terminal configuration"
    supported_os = ["linux", "darwin"]
    target = "$HOME"
    [profiles.macbook]
    sources = ["common", "macbook"]
    [profiles.devstick]
    sources = ["common", "devstick"]
    ```
  - Delete `ghostty/install.sh` (no longer needed — `rice install ghostty --profile macbook` replaces it).
  - Verify: walking `ghostty/common/` then `ghostty/macbook/` produces the same target file set as the install.sh would have. If install.sh did anything beyond plain copy (env var substitution, conditional skips), DOCUMENT in the commit body and call out as a behavior diff.

  **Must NOT do**:
  - Don't change the contents of any config files in common/ or macbook/ or devstick/ — pure restructure
  - Don't introduce a new profile (no "linux" or "default") — only macbook and devstick exist
  - Don't add `windows` to supported_os — ghostty isn't shipping on Windows for this user

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: One manifest, one delete; well-scoped
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES with 19, 21
  - Parallel Group: Wave 4
  - Blocks: F1-F4
  - Blocked By: 7

  **References**:
  - Existing `ghostty/install.sh` (for behavior parity check)
  - Task 2 schema

  **Acceptance Criteria**:
  - [ ] `ghostty/rice.toml` parses
  - [ ] `ghostty/install.sh` deleted (`test ! -f ghostty/install.sh`)
  - [ ] BuildInstallPlan for ghostty/macbook produces ops covering all files in `common/` + `macbook/`
  - [ ] If two sources have the same relative file: conflict surfaces (per Task 9 within-package conflict rules)

  **QA Scenarios**:
  ```
  Scenario: ghostty install plan covers both source layers
    Tool: Bash
    Preconditions: Task 20
    Steps:
      1. Run `go test ./internal/installer/ -run TestGhosttyMacbookPlan -v`
    Expected Result: PASS — plan ops match union of common/ and macbook/ files
    Evidence: .sisyphus/evidence/task-20-ghostty.txt

  Scenario: install.sh removed
    Tool: Bash
    Preconditions: Task 20
    Steps:
      1. Run `test ! -f ghostty/install.sh && echo OK`
    Expected Result: prints OK
    Evidence: .sisyphus/evidence/task-20-no-install-sh.txt
  ```

  **Commit**: YES — `refactor(ghostty): replace install.sh with rice.toml manifest`
  - Files: `ghostty/rice.toml`, deletes `ghostty/install.sh`
  - Pre-commit: `go test ./internal/manifest/... ./internal/installer/...`

- [ ] 21. **Split opencode skills into personal/work profiles + rice.toml**

  **What to do**:
  - **Goal**: Make `opencode` a multi-profile package so `rice switch opencode personal|work` flips between two skill sets.
  - Current state: `opencode/.agents/skills/` and `opencode/.config/opencode/` live at the top of the package.
  - **Restructure**:
    - Create `opencode/common/.config/opencode/` and move config files there that are shared between personal and work (the user has confirmed: "Fully separate skill sets per profile" — interpret strictly: SKILLS are split, but base opencode CONFIG can be shared via a `common` source if it makes sense; if the user wants total separation, common stays empty and both personal+work duplicate. ASK in F3 if ambiguous).
    - Create `opencode/personal/.agents/skills/` and move ALL current skills there (the user's existing personal skill set).
    - Create `opencode/work/.agents/skills/` initially empty (just a `.gitkeep`) — to be populated by user later.
    - Add `opencode/rice.toml`:
      ```toml
      schema_version = 1
      name = "opencode"
      description = "opencode AI assistant configuration"
      supported_os = ["linux", "darwin", "windows"]
      target = "$HOME"
      [profiles.personal]
      sources = ["common", "personal"]
      [profiles.work]
      sources = ["common", "work"]
      ```
  - **Decision point during execution**: If "common" config files have any personal-only content (API keys, personal model preferences), MOVE them out of common into personal. The executing agent must `git diff` the moved content and inspect for things that shouldn't be shared.
  - Update `.gitignore` if any moved file paths were previously ignored relative to the old layout.

  **Must NOT do**:
  - Don't lose any existing skill files — `find opencode/.agents/skills -type f | wc -l` BEFORE the move must equal `find opencode/personal/.agents/skills -type f | wc -l` AFTER
  - Don't put work skills in personal or vice versa — they are SEPARATE per user decision
  - Don't commit any secrets if they exist in current opencode config — sanitize before move
  - Don't delete the old top-level `opencode/.agents/` and `opencode/.config/` directories until the move is verified by file count diff

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Real content reorganization with risk of lost files. Higher rigor than the other migration tasks.
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES with 19, 20
  - Parallel Group: Wave 4
  - Blocks: F1-F4
  - Blocked By: 7

  **References**:
  - User decision (round 1): "Fully separate skill sets per profile"
  - Task 2 schema, Task 5 profile resolution
  - Existing `opencode/` directory layout

  **Acceptance Criteria**:
  - [ ] `opencode/rice.toml` parses
  - [ ] `opencode/personal/.agents/skills/` contains all original skills (file count matches pre-migration count)
  - [ ] `opencode/work/.agents/skills/` exists (with `.gitkeep`)
  - [ ] Old top-level `opencode/.agents/` and `opencode/.config/` directories DO NOT exist post-migration
  - [ ] `rice install opencode --profile personal` (smoke-tested in F3) creates symlinks from personal+common into `~/.agents/skills/` and `~/.config/opencode/`
  - [ ] `rice switch opencode work` produces a clean plan in dry-form (work has no skills yet, just empty .gitkeep — that's fine)

  **QA Scenarios**:
  ```
  Scenario: File count preserved
    Tool: Bash
    Preconditions: Task 21 + record `BEFORE_COUNT=$(find opencode/.agents/skills -type f | wc -l)` BEFORE migration
    Steps:
      1. After migration: `AFTER_COUNT=$(find opencode/personal/.agents/skills -type f | wc -l)`
      2. Assert `BEFORE_COUNT == AFTER_COUNT`
    Expected Result: Counts match
    Evidence: .sisyphus/evidence/task-21-file-count.txt

  Scenario: Manifest parses and profiles resolve
    Tool: Bash
    Preconditions: Task 21
    Steps:
      1. Run `go test ./internal/profile/ -run TestOpencodeProfilesResolve -v`
    Expected Result: PASS — both `personal` and `work` resolve to ordered source lists
    Evidence: .sisyphus/evidence/task-21-profiles.txt
  ```

  **Commit**: YES — `refactor(opencode): split into personal/work profiles with rice.toml`
  - Files: `opencode/rice.toml`, restructured `opencode/{common,personal,work}/...`, removed top-level `opencode/.agents/` and `opencode/.config/`
  - Pre-commit: `go test ./internal/manifest/... ./internal/profile/...`

- [ ] 22. **Write AGENTS.md documenting rice conventions**

  **What to do**:
  - Create `/Users/guneet/rice/AGENTS.md` (repo root) with sections:
    - **Repository purpose**: dotfile/rice configs for multiple machines/profiles, managed by the `rice` Go CLI
    - **Build & test**: `go build ./cmd/rice`, `go test ./...`, `go vet ./...`. Mention `-race` for installer tests.
    - **Project layout**: `cmd/rice/`, `internal/{manifest,symlink,state,profile,installer,doctor,logger,plan,prompt}`, package directories at repo root each with `rice.toml`, `testdata/` for fixture-based tests.
    - **rice.toml schema**: full reference with example, including `schema_version = 1`, `name`, `description`, `supported_os`, `target`, `profile_key`, `[profiles.<name>] sources = [...]` blocks. Document path constraints (no absolute, no `..`, must be under `$HOME`). Show a multi-source profile example (e.g., ghostty's `sources = ["common", "macbook"]`) and a single-source example (e.g., nvim's `sources = ["."]`).
    - **Profile model**: per-package, freeform values, single-axis only in v1. Each profile explicitly declares its `sources` list — there is NO implicit "common" base. If a manifest author wants a base+overlay pattern (like ghostty), they list it: `sources = ["common", "macbook"]`. The installer walks each source in order; later sources can shadow earlier ones if files collide (but that produces a conflict error per Task 9 conflict policy — within a single package, two sources cannot place a symlink at the same target).
    - **OS support**: each package declares `supported_os = ["linux", "darwin", "windows"]` (subset). Installer enforces.
    - **State file**: `~/.config/rice/state.json` (POSIX) / `%APPDATA%/rice/state.json` (Windows). Authoritative for "what rice installed". Rice does not lock the file; do not run multiple rice processes concurrently.
    - **CLI commands**: `install`, `switch`, `status`, `doctor`, `uninstall`, `version`. Each with example invocation and brief semantics.
    - **Confirmation flow (destructive ops)**: `install`, `uninstall`, `switch` print the full plan (every symlink: source → target, every removal) then prompt `Proceed? [y/N]`. Default on bare Enter = NO. Accepts `y` / `yes` (case-insensitive) to proceed. `--yes` / `-y` flag bypasses the prompt for scripts/CI but the plan is still printed for audit. `status` and `doctor` are read-only and never prompt.
    - **Logging**: structured logging via `go.uber.org/zap`. Five levels:
      - `debug` — verbose internal traces
      - `info` — high-level operation milestones
      - `warn` — recoverable edge cases handled internally
      - `error` — edge cases the user must fix
      - `critical` — bizarre internal state; please open a github issue
      Console output goes to STDERR at the configured level (default `warn`). Stdout is reserved for command output (status table, install plan, summaries) so it can be piped/parsed without log noise. Configure via `--log-level` flag or `RICE_LOG_LEVEL` env var (flag wins). Independently of console level, ALL logs are always written at DEBUG level (JSON format) to a single non-rotating file at `~/.config/rice/logs/rice.log` (POSIX) or `%APPDATA%/rice/logs/rice.log` (Windows). The file is append-only; rice never rotates or trims it — manage cleanup yourself (e.g., `rm` it periodically or pipe through logrotate).
    - **Conflict policy**: rice always aborts on conflict. No `--force` exists. Resolve by removing/moving the conflicting file manually.
    - **Switch atomicity**: pre-flight validates the new profile, then full uninstall + reinstall. If install step fails post-uninstall, recovery message printed; user must re-run `rice install`.
    - **Windows requirements**: Developer Mode enabled. `rice doctor` checks this.
    - **Out of scope (v1) — explicitly deferred to v2 or later**:
      - Multi-axis profiles (e.g., a single package having both machine AND identity profiles)
      - Init / bootstrap command
      - `--all` flag (install everything)
      - `--force` flag (override conflicts)
      - Hooks (pre/post install commands)
      - Secrets / env management
      - Package dependencies (one package depending on another)
      - JSON output mode for `status`
      - Doctor auto-fix
      - Log rotation

  **Must NOT do**:
  - Don't include CLI flag reference for flags that don't exist (`--force`, `--all`, `--dry-run`)
  - Don't document the old stow-based workflow except in a brief "migration from previous bash setup" note
  - Don't make this a tutorial — it's a reference doc for AI agents

  **Recommended Agent Profile**:
  - **Category**: `writing`
    - Reason: Pure documentation
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES with 23
  - Parallel Group: Wave 4
  - Blocks: F1-F4
  - Blocked By: 14, 15, 16, 17, 18, 24, 25 (need final CLI surface, logger, and confirmation utility to document accurately)

  **References**:
  - This entire plan
  - Existing AGENTS.md skill loading conventions for opencode (don't conflict with those)

  **Acceptance Criteria**:
  - [ ] AGENTS.md exists at repo root
  - [ ] All 6 CLI commands documented
  - [ ] rice.toml schema documented with example
  - [ ] Out-of-scope section lists every deferred feature
  - [ ] Logging + confirmation flow documented

  **QA Scenarios**:
  ```
  Scenario: AGENTS.md sections present
    Tool: Bash
    Preconditions: Task 22
    Steps:
      1. Run `grep -E '^## ' AGENTS.md | sort -u | wc -l` — assert >= 8 sections
      2. Run `grep -c 'rice install' AGENTS.md` — assert >= 1
      3. Run `grep -c 'log-level' AGENTS.md` — assert >= 1
      4. Run `grep -c 'Proceed?' AGENTS.md` — assert >= 1
    Expected Result: All assertions pass
    Evidence: .sisyphus/evidence/task-22-agents-md.txt
  ```

  **Commit**: YES — `docs: add AGENTS.md for rice CLI conventions`
  - Files: `AGENTS.md`
  - Pre-commit: none

- [ ] 23. **Rewrite README.md for rice CLI workflow**

  **What to do**:
  - Replace the current stow-based README with a rice-CLI-focused one. Sections:
    - **What is this**: Personal dotfile repo managed by `rice`, a small Go CLI inspired by GNU Stow.
    - **Install rice**: `go install ./cmd/rice` or `go build -o rice ./cmd/rice`.
    - **Quick start**: clone repo, `cd` into it, `rice install nvim --profile common`, `rice install ghostty --profile macbook`, `rice install opencode --profile personal`.
    - **Switching profiles**: `rice switch opencode work`.
    - **Logging & confirmation**: brief mention with link to AGENTS.md for full reference. Show example: `rice --log-level debug install ghostty --profile macbook` and `rice install ghostty --profile macbook --yes`.
    - **Available packages table**: package name, supported OSes, available profiles, brief description. Cover all 7 packages (ghostty, hyprland, nvim, opencode, waybar, wofi, zsh).
    - **Adding a new package**: short steps — create dir, add `rice.toml`, `rice install <name> --profile <p>`.
    - **Migration note**: one short paragraph mentioning "previously this repo used GNU stow + bash install scripts; the new `rice` CLI replaces both".
    - Link to AGENTS.md for full schema/conventions reference.

  **Must NOT do**:
  - Don't keep the old stow setup instructions
  - Don't document internal implementation details (those live in AGENTS.md)
  - Don't add badges/CI fluff in v1

  **Recommended Agent Profile**:
  - **Category**: `writing`
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES with 22
  - Parallel Group: Wave 4
  - Blocks: F1-F4
  - Blocked By: 14, 15, 16, 17, 18, 24, 25

  **References**:
  - Current `README.md` (for migration note context)
  - AGENTS.md from Task 22 (cross-link)

  **Acceptance Criteria**:
  - [ ] README.md does not contain the word "stow" except in the migration note paragraph
  - [ ] Quick-start section shows building `rice` and using each command
  - [ ] Packages table lists all 7 packages with their OS+profile support
  - [ ] References AGENTS.md for deep details
  - [ ] Logging + confirmation example present

  **QA Scenarios**:
  ```
  Scenario: README updated
    Tool: Bash
    Preconditions: Task 23 done
    Steps:
      1. Run `grep -c 'stow' README.md` — assert <= 1 (only migration note)
      2. Run `grep -c '^## ' README.md` — assert >=5 sections
      3. Run `grep -E 'rice install|rice switch' README.md` — assert at least one example each
      4. Run `grep -c 'log-level\|--yes' README.md` — assert >= 1
    Expected Result: All assertions pass
    Evidence: .sisyphus/evidence/task-23-readme.txt
  ```

  **Commit**: YES — `docs: rewrite README for rice CLI workflow`
  - Files: `README.md`
  - Pre-commit: none

- [x] 24. **Logging package (zap-based, 5 levels including custom CRITICAL)**

  **What to do**:
  - Create `internal/logger/logger.go` exposing a package-level logger plus init function:
    - `type Level int8` mirroring zapcore: `DebugLevel`, `InfoLevel`, `WarnLevel`, `ErrorLevel`, `CriticalLevel`. CriticalLevel is a custom zapcore.Level set to `zapcore.ErrorLevel + 1` (so it's strictly above Error).
    - `func ParseLevel(s string) (Level, error)` — parses `debug|info|warn|error|critical`, case-insensitive. Errors with a clear message listing valid values.
    - `func Init(consoleLevel Level, logFilePath string) error` — sets up a zap logger with a `zapcore.NewTee(consoleCore, fileCore)`:
      - consoleCore: writes to STDERR (NOT stdout — stdout is reserved for command output) at `consoleLevel`. Human-readable encoder (zapcore.NewConsoleEncoder) with timestamp, level, message, fields. Color when `os.Stderr` is a TTY, plain otherwise.
      - fileCore: writes to `logFilePath` at DebugLevel (always full debug). JSON encoder (zapcore.NewJSONEncoder) for machine grep-ability. The file is opened with `os.O_APPEND | os.O_CREATE | os.O_WRONLY`, mode 0644. Parent dir created via `os.MkdirAll(filepath.Dir(logFilePath), 0755)` if missing. NO ROTATION — user manages cleanup.
    - Package-level functions: `Debug(msg string, fields ...zap.Field)`, `Info`, `Warn`, `Error`, `Critical(msg string, fields ...zap.Field)`. `Critical` writes at the custom level AND includes a hardcoded field `github_issue_url = "https://github.com/guneet/rice/issues/new"` to nudge the user to file an issue.
    - `func DefaultLogPath() string` — returns `~/.config/rice/logs/rice.log` on POSIX, `%APPDATA%/rice/logs/rice.log` on Windows. Mirrors the state.DefaultPath logic from Task 4.
    - `func Sync()` — calls underlying logger's Sync(). Called from root cmd's PersistentPostRun.
  - Export a `var L *zap.Logger` package-level so other packages can do `logger.L.Debug(...)` if they want raw zap, but the wrapper functions are the recommended API.
  - Tests in `internal/logger/logger_test.go`:
    - ParseLevel happy + error cases
    - Init creates the log file and parent dir
    - Logging at WARN does NOT write to stderr buffer at INFO level
    - Logging at WARN DOES write to file (because file is always Debug)
    - Critical logs include the github_issue_url field
    - Test using `t.TempDir()` for log file path

  **Must NOT do**:
  - Don't add log rotation (user manages cleanup per their decision)
  - Don't write logs to stdout — stdout is reserved for command output
  - Don't make `Init` panic on file open failure — return error so root cmd can decide
  - Don't add log shipping, syslog, or remote sinks
  - Don't add per-package loggers / sublogger trees in v1 — flat package logger is enough

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Bounded, well-documented zap setup
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: YES (no deps)
  - Parallel Group: Wave 1
  - Blocks: 10, 11, 13, 22, 23, 25
  - Blocked By: None

  **References**:
  - zap docs: https://pkg.go.dev/go.uber.org/zap
  - Custom zapcore levels: https://pkg.go.dev/go.uber.org/zap/zapcore#Level — extend by defining a const above ErrorLevel
  - Tee pattern: https://pkg.go.dev/go.uber.org/zap/zapcore#NewTee
  - User decision (round 5): default WARN, levels debug/info/warn/error/critical with semantics they specified
  - User decision: single log file no rotation, debug level always to file

  **Acceptance Criteria**:
  - [ ] `ParseLevel("debug")` etc. returns expected level
  - [ ] `ParseLevel("verbose")` returns error listing valid values
  - [ ] `Init(WarnLevel, tmpFile)` creates the file and its parent dir
  - [ ] At WarnLevel console, `Info("x")` does NOT appear in captured stderr
  - [ ] At WarnLevel console, `Warn("x")` DOES appear in captured stderr
  - [ ] At any console level, `Debug("x")` is written to the log FILE
  - [ ] `Critical("boom")` includes the `github_issue_url` field in the file output
  - [ ] CRITICAL log level is strictly higher than ERROR (verify via `Critical` writes when console=ErrorLevel)
  - [ ] `go test ./internal/logger/... -race` passes

  **QA Scenarios**:
  ```
  Scenario: Default WARN level filters INFO
    Tool: Bash
    Preconditions: Task 24 + binary built
    Steps:
      1. Run `rice version 2> stderr.txt` (default level = WARN)
      2. Internally version cmd does `logger.Info("running version cmd")`
      3. Assert `stderr.txt` does NOT contain "running version cmd"
      4. Assert `~/.config/rice/logs/rice.log` (or test override) DOES contain it
    Expected Result: Console respects WARN filter, file always captures DEBUG+
    Evidence: .sisyphus/evidence/task-24-default-level.txt

  Scenario: --log-level debug surfaces everything
    Tool: Bash
    Preconditions: Task 24 + Task 13 root cmd
    Steps:
      1. Run `rice --log-level debug version 2> stderr.txt`
      2. Assert `stderr.txt` contains the DEBUG line
    Expected Result: DEBUG visible on stderr
    Evidence: .sisyphus/evidence/task-24-debug-level.txt

  Scenario: CRITICAL includes github issue link
    Tool: Bash
    Preconditions: Task 24 unit test fixture
    Steps:
      1. Run `go test ./internal/logger/ -run TestCritical_IncludesIssueURL -v`
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-24-critical.txt
  ```

  **Commit**: YES — `feat(logger): add zap-based logger with 5 levels including custom CRITICAL`
  - Files: `internal/logger/logger.go`, `internal/logger/logger_test.go`, `go.mod`, `go.sum`
  - Pre-commit: `go test ./internal/logger/... -race`

- [ ] 25. **Plan + confirmation prompt utility**

  **What to do**:
  - Create `internal/plan/plan.go`:
    - `type OpKind int` with `OpCreate` and `OpRemove` constants
    - `type Op struct { Kind OpKind; Source string; Target string }` (Source unused for Remove)
    - `type Plan struct { PackageName string; Profile string; Ops []Op; Conflicts []Conflict }` (Conflict imported from installer or moved here — pick one home; prefer here so installer can import plan without cycle)
    - `func (p *Plan) IsEmpty() bool` — true if no Ops AND no Conflicts
  - Create `internal/prompt/prompt.go`:
    - `func RenderPlan(w io.Writer, p *plan.Plan)` — writes the human-readable plan output described in Tasks 14/15. Format:
      ```
      Plan: install <pkg> (profile: <name>)
        CREATE  <target> → <source>
        ...
      Total: N symlinks to create.
      ```
      For uninstall: `Plan: uninstall <pkg>` and `REMOVE  <target>`. For switch: see RenderSwitchPlan below. NEVER truncate. Align columns with `text/tabwriter`.
    - `func RenderSwitchPlan(w io.Writer, uninstall *plan.Plan, install *plan.Plan)` — combined output showing both phases under headers `Uninstall (profile: X):` and `Install (profile: Y):`.
    - `func RenderConflicts(w io.Writer, conflicts []plan.Conflict)` — `CONFLICT  <target>: <reason>` lines.
    - `func Confirm(in io.Reader, out io.Writer, message string) (bool, error)` — writes `<message> [y/N]: ` to `out`, reads ONE line from `in`. Returns true ONLY if input is `y` or `yes` (case-insensitive, trimmed). Empty input (Enter), `n`, `no`, anything else returns false. Uses `bufio.NewReader(in).ReadString('\n')`. EOF returns `(false, nil)` (treat as cancellation, not error).
  - Tests in `internal/prompt/prompt_test.go` covering:
    - RenderPlan with empty Ops
    - RenderPlan with mixed CREATE/REMOVE ordering preserved
    - RenderPlan with 100+ ops — no truncation
    - Confirm with `"y\n"`, `"Y\n"`, `"yes\n"`, `"YES\n"` → true
    - Confirm with `"\n"` (bare Enter), `"n\n"`, `"no\n"`, `"q\n"`, `"asdf\n"` → false
    - Confirm with EOF → (false, nil)
    - RenderConflicts formatting

  **Must NOT do**:
  - Don't make `Confirm` accept default-yes — default is ALWAYS NO per user
  - Don't truncate plan output even for huge plans
  - Don't add fancy interactive selection (multi-choice menu) — strictly y/N
  - Don't read more than one line from `in` — exactly one ReadString call
  - Don't write the prompt to stdout if `out` is a different writer; respect the passed writer

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure formatting + stdin reading, well-bounded
  - **Skills**: []

  **Parallelization**:
  - Can Run In Parallel: NO (depends on logger conceptually, though plan package itself has no logger dep)
  - Parallel Group: Wave 2 (early)
  - Blocks: 10, 11, 14-16, 22, 23
  - Blocked By: 24

  **References**:
  - Task 24 logger
  - User decision (round 5): full plan listing, default NO on Enter, y/N prompt, --yes bypass at CLI layer (not in this util)
  - `bufio.Reader.ReadString`: https://pkg.go.dev/bufio#Reader.ReadString
  - `text/tabwriter`: https://pkg.go.dev/text/tabwriter

  **Acceptance Criteria**:
  - [ ] `Plan` type holds Ops + Conflicts
  - [ ] `RenderPlan` produces aligned output, no truncation at 100+ ops
  - [ ] `Confirm` returns true ONLY for y/yes (case-insensitive)
  - [ ] `Confirm` returns false on Enter (default NO)
  - [ ] `Confirm` returns (false, nil) on EOF
  - [ ] `go test ./internal/plan/... ./internal/prompt/... -race` passes

  **QA Scenarios**:
  ```
  Scenario: Confirm defaults to NO on bare Enter
    Tool: Bash
    Preconditions: Task 25 unit tests
    Steps:
      1. Run `go test ./internal/prompt/ -run TestConfirm_BareEnter -v`
    Expected Result: PASS — empty input → false
    Evidence: .sisyphus/evidence/task-25-confirm-default.txt

  Scenario: RenderPlan does not truncate
    Tool: Bash
    Preconditions: Task 25
    Steps:
      1. Run `go test ./internal/prompt/ -run TestRenderPlan_NoTruncation -v`
    Expected Result: PASS — all 100 ops appear in output
    Evidence: .sisyphus/evidence/task-25-no-truncate.txt

  Scenario: Confirm accepts y/yes/Y/YES
    Tool: Bash
    Preconditions: Task 25
    Steps:
      1. Run `go test ./internal/prompt/ -run TestConfirm_PositiveInputs -v`
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-25-confirm-yes.txt
  ```

  **Commit**: YES — `feat(plan,prompt): add Plan type, renderer, and y/N confirmation`
  - Files: `internal/plan/plan.go`, `internal/plan/plan_test.go`, `internal/prompt/prompt.go`, `internal/prompt/prompt_test.go`
  - Pre-commit: `go test ./internal/plan/... ./internal/prompt/... -race`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)


> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
>
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read `.sisyphus/plans/rice-cli.md` end-to-end. For each "Must Have": verify implementation exists by reading the relevant Go file or running the relevant `rice` command. For each "Must NOT Have": grep the codebase for forbidden patterns (e.g., `--force`, `os/exec.*stow`, multi-axis profile code, target paths outside HOME) — reject with file:line if found. Check evidence files exist in `.sisyphus/evidence/`.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./...`, `go vet ./...`, `go test ./... -race`, and `gofmt -l .` (must produce no output). Review all changed Go files for: silent error swallowing (`_ = err`), panic in non-main code, package-level mutable globals, missing error wrapping, magic strings that should be constants, AI slop patterns (excessive comments, generic names like `data`/`result`/`item`/`temp`).
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | gofmt [clean/dirty] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **End-to-End Manual QA** — `unspecified-high`
  In a clean tmpdir acting as $HOME: build the binary, then exercise the full lifecycle WITH confirmation flow: `rice install ghostty --profile macbook` (verify plan printed, prompt shown, "y" proceeds) → `rice status` → `rice switch ghostty devstick` (verify combined plan, prompt) → `rice install opencode --profile work --yes` (verify plan printed, NO prompt, proceeds) → `rice switch opencode personal` (at prompt: bare Enter, verify cancellation with no FS change) → re-run with "y" → `rice doctor` (read-only, NO prompt) → `rice status` (read-only, NO prompt) → `rice uninstall ghostty`. Test failure modes: install on unsupported OS (simulate via GOOS), install with conflict (pre-create file at target — no prompt should appear, conflict report only), invalid profile name, missing manifest. Test logging: run with default level → assert no DEBUG/INFO on stderr but log file populated; run with `--log-level debug` → assert debug visible on stderr; run with `RICE_LOG_LEVEL=info rice install ...` → assert info visible; verify log file at `~/.config/rice/logs/rice.log` accumulates across all runs (no rotation). Save terminal output + log file samples to `.sisyphus/evidence/final-qa/`.
  Output: `Lifecycle [PASS/FAIL] | Confirmation flow [N/N scenarios] | Logging [N/N levels verified] | Failure modes [N/N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task in this plan: read "What to do", read actual `git diff` for that task's files. Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT Have" guardrails are honored. Detect: code in packages that weren't supposed to be touched (only ghostty + opencode should have content changes; others only get a new rice.toml), CLI commands that weren't planned (no init, no --all, no --force), feature creep into doctor (no auto-fix code).
  Output: `Tasks [N/N compliant] | Out-of-scope changes [CLEAN/N issues] | Guardrails [N/N honored] | VERDICT`

---

## Commit Strategy

Atomic commits per task. Conventional commits format. No commit until that task's tests/QA pass.

- **1**: `chore: scaffold go module and test infra` — go.mod, go.sum, .gitignore, README test section
- **2**: `feat(manifest): define rice.toml schema and validation` — internal/manifest/
- **3**: `feat(symlink): cross-platform symlink primitives` — internal/symlink/
- **4**: `feat(state): state file format and read/write` — internal/state/
- **5**: `feat(profile): profile resolution and validation` — internal/profile/
- **6**: `chore: delete stale profiles and scripts dirs` — repo cleanup
- **7**: `feat(manifest): manifest discovery and parsing` — internal/manifest/
- **8**: `feat(installer): per-package OS gating` — internal/installer/
- **9**: `feat(installer): conflict detection` — internal/installer/
- **10**: `feat(installer): install orchestrator` — internal/installer/
- **11**: `feat(installer): uninstall orchestrator` — internal/installer/
- **12**: `feat(installer): switch with pre-flight validation` — internal/installer/
- **13**: `feat(cli): scaffold rice CLI with cobra` — cmd/rice/
- **14**: `feat(cli): install command` — cmd/rice/cmd/install.go
- **15**: `feat(cli): uninstall command` — cmd/rice/cmd/uninstall.go
- **16**: `feat(cli): switch command` — cmd/rice/cmd/switch.go
- **17**: `feat(cli): status command` — cmd/rice/cmd/status.go
- **18**: `feat(cli): doctor command` — cmd/rice/cmd/doctor.go
- **19**: `chore: add rice.toml to existing packages` — nvim, zsh, hyprland, waybar, wofi
- **20**: `refactor(ghostty): migrate to rice.toml, delete install.sh` — ghostty/
- **21**: `refactor(opencode): split skills into personal/work profiles` — opencode/
- **22**: `docs: add AGENTS.md documenting rice conventions` — AGENTS.md
- **23**: `docs: rewrite README for rice CLI workflow` — README.md

---

## Success Criteria

### Verification Commands
```bash
# Build
go build -o /tmp/rice ./cmd/rice
echo $? # Expected: 0

# Tests
go test ./... -race
echo $? # Expected: 0

# Vet & format
go vet ./...; gofmt -l . # Expected: empty output

# CLI smoke (in a tmpdir HOME)
RICE_HOME=$(mktemp -d) HOME=$RICE_HOME /tmp/rice install nvim --profile common
ls -la $RICE_HOME/.config/nvim # Expected: symlink to repo nvim/common/.config/nvim

# OS gating
GOOS=windows go build -o /tmp/rice.exe ./cmd/rice # Expected: 0
# (binary refuses zsh install when run on Windows; confirmed via cross-compile test)

# Stale dirs gone
test ! -d profiles && test ! -d scripts && test ! -f ghostty/install.sh
echo $? # Expected: 0
```

### Final Checklist
- [ ] All "Must Have" items present and verified
- [ ] All "Must NOT Have" items absent (verified by grep)
- [ ] All Go tests pass with `-race`
- [ ] AGENTS.md exists and is comprehensive
- [ ] README.md does not mention `stow`
- [ ] All 4 final verification agents APPROVED
- [ ] User explicitly said "okay" after seeing final verification results
