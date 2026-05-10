# Source Spec: Table-Only Schema

## TL;DR

> **Quick Summary**: Make every `rice.toml` source a `{path, mode, target}` table — drop the bare-string form. Each source carries its own absolute `target`; remove the package-level `target` field entirely.
>
> **Deliverables**:
> - Schema: `SourceSpec.UnmarshalTOML` drops string branch; `Manifest.Target` field removed
> - Validator: `target` required for both `file` and `folder` modes; env vars expanded
> - Walker: uses per-source `target` as destination root (no more package-level target)
> - 12 rice.toml files migrated (7 real packages + 5 testdata fixtures)
> - 3 Go test files with inline TOML fixtures migrated
> - `internal/profile.Resolve` (legacy `[]string` form) deleted; only `ResolveSpecs` remains
> - AGENTS.md schema section rewritten
> - All tests green; manual install/uninstall repro of one real package
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 4 waves
> **Critical Path**: Task 1 (schema) → Task 2 (validator) → Task 3 (walker + Manifest.Target removal) → Task 4-7 (parallel migrations) → Task 11 (final verify + commit)

---

## Context

### Original Request
"Can we make Source schema be just that object with path mode and target."

Follow-up locked decisions:
- Migrate all packages in this repo (option 1a)
- Require `mode` explicitly on every source (option 2b)
- No schema_version bump (option 3b)
- `target` required on both modes; for file-mode it's the absolute destination root
- Package-level `target` field deleted entirely (per Metis follow-up — cleanest schema)
- Old format gets default BurntSushi/toml decode error (no custom message)
- Tests only, no QA scenarios

### Interview Summary
**Key Discussions**:
- Started from completed `folder-mode-symlinks` boulder (commit `65c4ff0`) where `SourceSpec` accepted both string and table forms
- User wants to simplify: tables only, every field required
- Identified that `target` semantics for file-mode needed locking down — settled on "per-source absolute destination root"
- Metis surfaced critical blocker: current validator REJECTS `target` on file-mode sources (validate.go:48-58). This must be inverted.
- Metis-driven decision: delete package-level `target` entirely so there's only ONE source of truth for destination paths

**Research Findings**:
- 12 rice.toml files in repo (7 real packages, 5 testdata fixtures, 3 manifest fixtures with bad/valid variants)
- 3 Go test files contain inline TOML fixtures: `cli/install_test.go`, `internal/installer/install_test.go`, `internal/manifest/load_test.go`
- `profile.Resolve` (legacy string form) has only test-file callers; safe to delete
- `m.Target` is used in `internal/installer/install.go:102-106` to derive `targetRoot`; this becomes per-source after change
- `os.ExpandEnv` is currently called on `m.Target`; must move to per-source target

### Metis Review
**Identified Gaps** (addressed):
- Validator currently REJECTS `target` on file-mode → Task 2 inverts the rule
- Per-source `target` semantics needed locking → defined as "absolute destination root with env var expansion"
- Package-level `target` becomes meaningless if every source has its own → deleted entirely (Task 3)
- `cli/install_test.go` line 41 second inline fixture flagged → Task 9 covers it
- `profile.Resolve` deletion safe but breaks profile_test.go → Task 3 updates tests

---

## Work Objectives

### Core Objective
Replace bare-string source form with table-only `{path, mode, target}`; remove package-level `target` field. Migrate every existing rice.toml in the repo. Tests green.

### Concrete Deliverables
- `internal/manifest/schema.go`: `SourceSpec.UnmarshalTOML` accepts tables only; `Manifest.Target` field removed
- `internal/manifest/validate.go`: `target` required + env-expanded for both file and folder modes
- `internal/installer/install.go`: walker uses per-source `target` as root
- `internal/profile/profile.go`: `Resolve` deleted; `ResolveSpecs` remains
- `internal/profile/profile_test.go`: rewritten to test `ResolveSpecs` only
- 12 migrated rice.toml files
- 3 migrated Go test files with inline TOML
- `AGENTS.md`: schema section rewritten
- All `go build ./...` + `go test -race -count=1 ./...` green
- One manual install/uninstall repro (zsh or ghostty) confirming behavior preserved
- Single git commit

### Definition of Done
- [ ] `go build ./...` succeeds
- [ ] `go test -race -count=1 ./...` all packages PASS
- [ ] `gofmt -l internal/ cli/` reports nothing
- [ ] `go vet ./...` clean
- [ ] No string source form anywhere in codebase (`grep -rn 'sources = \[\"' .` returns no .toml/.go files except docs/comments)
- [ ] No `Manifest.Target` field anywhere (`grep -rn '\.Target' internal/ cli/` shows only `SourceSpec.Target` references)
- [ ] Manual repro: install one real package → expected symlinks created → uninstall → expected removal
- [ ] One git commit with message `refactor(schema): require table-form sources with explicit mode and target`

### Must Have
- Every `rice.toml` in repo uses table form for every source
- Every source specifies all three fields: `path`, `mode`, `target`
- Validator rejects empty `target` for either mode
- Validator rejects empty `path` for either mode
- Validator rejects `mode` other than "file" or "folder"
- Per-source `target` undergoes `os.ExpandEnv`
- File-mode walker lays files under `<expanded-source-target>/<relative-path>`
- Folder-mode walker creates one symlink at `<expanded-source-target>` (unchanged)
- All existing tests updated to new format and passing

### Must NOT Have (Guardrails)
- NO custom migration error message for old format (use default TOML decode error per user)
- NO `schema_version = 2` bump
- NO backward-compat code path for bare-string sources
- NO backward-compat code path for `Manifest.Target` field
- NO QA scenarios in tasks (tests only per user); the Final Verification Gate is the single integration check
- NO README rewrite — schema example blocks in README.md may be updated as part of Task 11, but the rest of README content stays unchanged
- NO touching other unrelated packages, refactors, or "while we're here" cleanup

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — all verification is agent-executed via Go test suite + gofmt/vet/build + one final manual repro of an installed package.
> Per user request, no per-task QA scenarios — Go tests are the verification.

### Test Decision
- **Infrastructure exists**: YES (Go testing + race detector)
- **Automated tests**: YES — tests-after pattern (each task updates/adds tests as it changes code)
- **Framework**: Go `testing` with `-race`
- **No QA scenarios per task** (user-confirmed); the final verify wave includes one manual install/uninstall repro using the binary

### Verification Commands
```bash
go build -o /tmp/rice ./cli                    # must succeed
gofmt -l internal/ cli/                        # must be empty
go vet ./...                                   # must be clean
go test -race -count=1 ./...                   # all packages pass
grep -rn 'sources = \["' --include='*.toml' .  # must be empty
grep -rn 'sources = \["' --include='*.go' .    # must be empty (except comments)
```

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (sequential foundation — must finish before anything else):
└── Task 1: Update SourceSpec schema (drop string branch, remove Manifest.Target)

Wave 2 (after Task 1 — validator + walker + profile):
├── Task 2: Update validator (target required for both modes, env expansion)
└── Task 3: Update walker + delete profile.Resolve + drop Manifest.Target consumers

Wave 3 (after Wave 2 — parallel migrations of fixtures and packages):
├── Task 4: Migrate real packages — group A (zsh, ghostty, hyprland)
├── Task 5: Migrate real packages — group B (waybar, wofi, opencode)
├── Task 6: Migrate nvim package (already folder-mode; remove package target)
├── Task 7: Migrate testdata/install fixtures (folder-pkg, folder-overlay-pkg, mypkg)
├── Task 8: Migrate testdata/manifest + testdata/manifest_valid fixtures
├── Task 9: Migrate inline TOML in cli/install_test.go
└── Task 10: Migrate inline TOML in internal/installer/install_test.go + internal/manifest/load_test.go

Wave 4 (after Wave 3 — final):
└── Task 11: Update AGENTS.md + final verify + manual repro + commit

Critical Path: Task 1 → Task 2 → Task 3 → (Tasks 4-10 parallel) → Task 11
Parallel Speedup: ~50% faster than sequential
Max Concurrent: 7 (Wave 3)
```

### Dependency Matrix

- **Task 1**: blocked by — none; blocks — 2, 3, 9, 10
- **Task 2**: blocked by — 1; blocks — 11
- **Task 3**: blocked by — 1; blocks — 11
- **Task 4-10**: blocked by — 1, 2, 3; blocks — 11
- **Task 11**: blocked by — all

### Agent Dispatch Summary

- **Wave 1** (1 task): Task 1 → `quick`
- **Wave 2** (2 tasks): Task 2 → `quick`, Task 3 → `quick`
- **Wave 3** (7 tasks parallel): Tasks 4-10 → `quick` each
- **Wave 4** (1 task): Task 11 → `quick`

---

## TODOs

- [x] 1. Update SourceSpec schema (drop string branch, remove Manifest.Target)

  **What to do**:
  - In `internal/manifest/schema.go`:
    - Remove the bare-string branch from `SourceSpec.UnmarshalTOML`. The `data interface{}` switch must reject anything that's not a `map[string]interface{}` with an error like `fmt.Errorf("source must be a table with path, mode, and target fields")`.
    - Remove the `Target` field from the `Manifest` struct (the package-level `target = "$HOME"` is going away).
    - Update the doc comment block (lines 22-29) to show only the table form.

  **Must NOT do**:
  - Do NOT add a custom error message advising users to migrate (per user — generic decode error is fine; we're just rejecting non-table input cleanly).
  - Do NOT keep a back-compat string branch.
  - Do NOT change `SourceSpec` field order or add new fields.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `internal/manifest/schema.go:22-100` — current `SourceSpec.UnmarshalTOML` (table + string branches)
  - `internal/manifest/schema.go:Manifest` — struct definition with `Target string` field
  - BurntSushi/toml v1.6.0 `Unmarshaler` interface: `UnmarshalTOML(data interface{}) error`; data is `string` or `map[string]interface{}`

  **Acceptance Criteria**:
  - [ ] String branch removed from `SourceSpec.UnmarshalTOML`
  - [ ] `Manifest.Target` field removed from struct
  - [ ] Doc comment updated to show only table form
  - [ ] `go build ./internal/manifest/...` may FAIL at this point (consumers downstream); that's expected — they're fixed in Task 2/3
  - [ ] `go vet ./internal/manifest/` clean

  **QA Scenarios**:
  ```
  Scenario: String-form sources are rejected
    Tool: Bash (go test)
    Preconditions: Task 1 changes applied
    Steps:
      1. cd /Users/guneet/rice
      2. Verify: grep -c 'case string' internal/manifest/schema.go
      3. Verify: grep -c '\bTarget\s*string\b' internal/manifest/schema.go
    Expected Result:
      - Step 2 returns 0 (no string branch in schema.go)
      - Step 3 returns 1 (only SourceSpec.Target remains; Manifest.Target gone)
    Failure Indicators: grep -c >0 for string branch; grep -c >1 for Target string
    Evidence: .sisyphus/evidence/task-1-schema-grep.txt

  Scenario: Manifest package vets clean post-change
    Tool: Bash (go vet)
    Preconditions: Task 1 applied
    Steps:
      1. cd /Users/guneet/rice && go vet ./internal/manifest/ 2>&1
    Expected Result: Empty output, exit code 0
    Failure Indicators: Any output, non-zero exit
    Evidence: .sisyphus/evidence/task-1-vet.txt
  ```

  **Commit**: NO (groups with Task 11)

- [x] 2. Update validator (target required for both modes, env expansion)

  **What to do**:
  - In `internal/manifest/validate.go`, find the source validation block (around lines 40-60):
    - REMOVE the rule that rejects `target` on file-mode sources
    - For BOTH `mode == "file"` and `mode == "folder"`: require `source.Target != ""` with error `source %q: target field is required`
    - Keep the rule that rejects unknown modes
    - Keep the rule that rejects empty `path`
  - In `validate.go`, after collecting the source target, run `os.ExpandEnv(source.Target)` either at validation time (for the path returned in errors) OR document that expansion happens at install time. Check how `m.Target` was previously expanded — probably at install time in `install.go:102-106`. Match that pattern: leave expansion to the walker/installer, only validate it's non-empty here.
  - Since `Manifest.Target` is gone, also remove any validation rules that reference `m.Target`.

  **Must NOT do**:
  - Do NOT add validation for "absolute path" — env vars like `$HOME` are not absolute pre-expansion, so we can't reject relative paths at this layer.
  - Do NOT add tilde (`~`) expansion — only `os.ExpandEnv`.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `internal/manifest/validate.go:48-58` — current rule that rejects file-mode target (the one being inverted)
  - `internal/manifest/validate.go` (full file) — for `m.Target` references to remove
  - `internal/manifest/validate_test.go` — existing test cases to update

  **Acceptance Criteria**:
  - [ ] file-mode rejection of `target` REMOVED
  - [ ] `target == ""` rejected for both modes
  - [ ] `path == ""` rejected (preserve existing behavior)
  - [ ] Unknown `mode` rejected (preserve existing behavior)
  - [ ] No references to removed `m.Target` field
  - [ ] `validate_test.go` updated: change "file-mode rejects target" cases to "file-mode rejects empty target" and "file-mode accepts target"
  - [ ] `go test -race ./internal/manifest/...` PASSES

  **QA Scenarios**:
  ```
  Scenario: Validator accepts file-mode with target
    Tool: Bash (go test)
    Preconditions: Task 2 applied
    Steps:
      1. cd /Users/guneet/rice && go test -race -run 'TestValidate' ./internal/manifest/ -v 2>&1 | tee /tmp/t2-validate.log
      2. grep -c 'PASS' /tmp/t2-validate.log
      3. grep -c 'FAIL' /tmp/t2-validate.log
    Expected Result: Step 2 >= 1, Step 3 == 0
    Failure Indicators: Any FAIL line; "target field is only valid for folder-mode" still in error messages
    Evidence: .sisyphus/evidence/task-2-validate.log

  Scenario: Validator rejects empty target on either mode
    Tool: Bash (go test)
    Preconditions: Task 2 applied; new test cases added covering empty-target rejection
    Steps:
      1. cd /Users/guneet/rice && go test -race -run 'TestValidate.*[Ee]mpty.*[Tt]arget' ./internal/manifest/ -v 2>&1
    Expected Result: At least one matching test runs and PASSES
    Failure Indicators: 0 matching tests, or any FAIL
    Evidence: .sisyphus/evidence/task-2-empty-target.log
  ```

  **Commit**: NO (groups with Task 11)

- [x] 3. Update walker, delete profile.Resolve, drop Manifest.Target consumers

  **What to do**:
  - In `internal/installer/install.go`:
    - The current walker uses `targetRoot` derived from `m.Target` at lines 102-106 and `filepath.Join(targetRoot, rel)` at line 192. Change this so each source uses its own `target`:
      - In the file-mode branch: `targetRoot := os.ExpandEnv(spec.Target)`; then `target := filepath.Join(targetRoot, rel)`.
      - In the folder-mode branch: `targetPath := os.ExpandEnv(spec.Target)` (no relative join — folder-mode target IS the destination).
    - Remove any reference to `m.Target` from the install function and its helpers.
  - In `internal/installer/uninstall.go` and `internal/installer/switch.go`: check for `m.Target` references and remove if present (state.json holds final paths so uninstall shouldn't need it; verify).
  - In `internal/profile/profile.go`: DELETE the legacy `Resolve` function (returning `[]string`). Keep only `ResolveSpecs`.
  - In `internal/profile/profile_test.go`: rewrite tests that called `Resolve` to call `ResolveSpecs` instead. Test cases that constructed manifests with `[]string` sources need updated fixtures using `[]SourceSpec`.
  - In `cli/`: search for `m.Target` or `manifest.Target` and remove (likely in install.go, uninstall.go, switch.go cobra command files); package-level target is gone.

  **Must NOT do**:
  - Do NOT modify `state.InstalledLink` schema (final paths are already absolute).
  - Do NOT change the walker's directory-creation behavior.
  - Do NOT modify the conflict-detection logic in `internal/installer/conflict.go` (it operates on resolved paths, not source targets).

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `internal/installer/install.go:97` — calls `profile.ResolveSpecs(m, profileName)`
  - `internal/installer/install.go:102-106` — `targetRoot` derivation from `m.Target` (the line being relocated per-source)
  - `internal/installer/install.go:192` — `filepath.Join(targetRoot, rel)` (uses targetRoot)
  - `internal/profile/profile.go` — `Resolve` (delete) and `ResolveSpecs` (keep)
  - `internal/profile/profile_test.go` — tests to update
  - `cli/install.go`, `cli/uninstall.go`, `cli/switch.go` — check for `m.Target` consumers

  **Acceptance Criteria**:
  - [ ] Walker uses `os.ExpandEnv(spec.Target)` per-source for both modes
  - [ ] No references to `m.Target` anywhere in `internal/installer/` or `cli/`
  - [ ] `profile.Resolve` deleted
  - [ ] `profile_test.go` updated to test `ResolveSpecs` only
  - [ ] `go build ./...` SUCCEEDS (full codebase compiles)
  - [ ] `grep -rn '\.Target' internal/ cli/ | grep -v 'SourceSpec\|spec\.Target\|source\.Target'` returns nothing meaningful

  **QA Scenarios**:
  ```
  Scenario: Full codebase compiles after walker + profile changes
    Tool: Bash (go build)
    Preconditions: Task 3 applied (walker, profile.Resolve deletion, m.Target removal)
    Steps:
      1. cd /Users/guneet/rice && go build ./... 2>&1 | tee /tmp/t3-build.log
      2. echo "exit=$?"
    Expected Result: Empty build output, exit=0
    Failure Indicators: Any compile error; references to removed Manifest.Target or profile.Resolve
    Evidence: .sisyphus/evidence/task-3-build.log

  Scenario: profile.Resolve is fully removed
    Tool: Bash (grep)
    Preconditions: Task 3 applied
    Steps:
      1. cd /Users/guneet/rice && grep -rn 'func Resolve(' internal/profile/ 2>&1
      2. grep -rn 'profile\.Resolve(' internal/ cli/ 2>&1
    Expected Result: Both grep calls return empty (no func definition, no callers)
    Failure Indicators: Any match for old Resolve function or its callers
    Evidence: .sisyphus/evidence/task-3-resolve-grep.txt

  Scenario: Installer + profile tests pass
    Tool: Bash (go test)
    Preconditions: Task 3 applied
    Steps:
      1. cd /Users/guneet/rice && go test -race -count=1 ./internal/profile/ ./internal/installer/ 2>&1 | tail -20
    Expected Result: All packages report PASS
    Failure Indicators: Any FAIL line, panic, or build failure
    Evidence: .sisyphus/evidence/task-3-tests.log
  ```

  **Commit**: NO (groups with Task 11)

- [x] 4. Migrate real packages — group A (zsh, ghostty, hyprland)

  **What to do**:
  - For each package below, rewrite `<pkg>/rice.toml`:
    - Remove the package-level `target = "..."` line
    - Convert every `sources = [...]` to table form with each source carrying `target = "$HOME"` (or whatever the original package target was)
  - **zsh** (`zsh/rice.toml`): currently `target = "$HOME"`, `sources = ["."]` → `[profiles.common] sources = [{path = ".", mode = "file", target = "$HOME"}]`; remove top-level `target = "$HOME"` line.
  - **ghostty** (`ghostty/rice.toml`): currently `target = "$HOME"`, three profiles use `["common"]`, `["common", "macbook"]`, `["common", "devstick"]`. Convert each string to `{path = "common", mode = "file", target = "$HOME"}` etc. Remove top-level `target`.
  - **hyprland** (`hyprland/rice.toml`): `target = "$HOME"`, `sources = ["."]` → `[{path = ".", mode = "file", target = "$HOME"}]`; remove top-level `target`.
  - Verify each migrated file parses: `go test -race ./internal/manifest/... ./internal/installer/...`

  **Must NOT do**:
  - Do NOT change `name`, `description`, `supported_os`, `schema_version`, or `profile_key` fields.
  - Do NOT change source `path` values — only restructure the form.
  - Do NOT touch any other packages.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `zsh/rice.toml`, `ghostty/rice.toml`, `hyprland/rice.toml` — files to migrate

  **Acceptance Criteria**:
  - [ ] `zsh/rice.toml` uses table form, no package-level target
  - [ ] `ghostty/rice.toml` uses table form, no package-level target, all three profiles migrated
  - [ ] `hyprland/rice.toml` uses table form, no package-level target
  - [ ] `grep -n 'target = "' zsh/rice.toml ghostty/rice.toml hyprland/rice.toml` shows only per-source targets

  **QA Scenarios**:
  ```
  Scenario: Group A packages have no package-level target and use table-form sources
    Tool: Bash (grep)
    Preconditions: Task 4 applied
    Steps:
      1. cd /Users/guneet/rice
      2. for f in zsh/rice.toml ghostty/rice.toml hyprland/rice.toml; do
           awk '/^\[/{p=1} !p && /^target *=/{print FILENAME":"NR":"$0}' "$f"
         done
      3. for f in zsh/rice.toml ghostty/rice.toml hyprland/rice.toml; do
           grep -E '^sources = \["' "$f" && echo "BAD: $f"
         done
    Expected Result: Step 2 empty (no top-level target line); Step 3 empty (no string-form sources)
    Failure Indicators: Any line printed
    Evidence: .sisyphus/evidence/task-4-grep.txt

  Scenario: Group A files parse via manifest loader
    Tool: Bash (go run snippet via test)
    Preconditions: Task 4 applied; Tasks 1-3 done
    Steps:
      1. cd /Users/guneet/rice && go test -race -count=1 ./internal/manifest/ -v 2>&1 | tail -20
    Expected Result: PASS
    Failure Indicators: Any FAIL or parse error
    Evidence: .sisyphus/evidence/task-4-manifest-tests.log
  ```

  **Commit**: NO (groups with Task 11)

- [x] 5. Migrate real packages — group B (waybar, wofi, opencode)

  **What to do**:
  - **waybar** (`waybar/rice.toml`): `target = "$HOME"`, `sources = ["."]` → `[{path = ".", mode = "file", target = "$HOME"}]`; remove top-level target.
  - **wofi** (`wofi/rice.toml`): same as waybar pattern.
  - **opencode** (`opencode/rice.toml`): `target = "$HOME"`, two profiles `personal` and `work` each `sources = ["personal"]` / `sources = ["work"]` → convert to `[{path = "personal", mode = "file", target = "$HOME"}]` etc. Remove top-level target.

  **Must NOT do**:
  - Do NOT change `name`, `description`, `supported_os`, `schema_version` fields.
  - Do NOT change source `path` values.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `waybar/rice.toml`, `wofi/rice.toml`, `opencode/rice.toml`

  **Acceptance Criteria**:
  - [ ] `waybar/rice.toml`, `wofi/rice.toml`, `opencode/rice.toml` all use table form, no package-level target
  - [ ] `grep -n 'target = "' waybar/rice.toml wofi/rice.toml opencode/rice.toml` shows only per-source targets

  **QA Scenarios**:
  ```
  Scenario: Group B packages migrated correctly
    Tool: Bash (grep)
    Preconditions: Task 5 applied
    Steps:
      1. cd /Users/guneet/rice
      2. for f in waybar/rice.toml wofi/rice.toml opencode/rice.toml; do
           awk '/^\[/{p=1} !p && /^target *=/{print FILENAME":"NR}' "$f"
         done
      3. for f in waybar/rice.toml wofi/rice.toml opencode/rice.toml; do
           grep -E '^sources = \["' "$f" && echo "BAD: $f"
         done
    Expected Result: Both empty
    Failure Indicators: Any line printed
    Evidence: .sisyphus/evidence/task-5-grep.txt
  ```

  **Commit**: NO (groups with Task 11)

- [x] 6. Migrate nvim package (already folder-mode; remove package target)

  **What to do**:
  - In `nvim/rice.toml`: the source line is already correct (`sources = [{path = ".config/nvim", mode = "folder", target = ".config/nvim"}]`). However:
    - Currently `target` on the source is `".config/nvim"` (relative). With package-level target gone, this must become `"$HOME/.config/nvim"` (absolute, env-var form).
    - Remove the package-level `target = "$HOME"` line.

  **Must NOT do**:
  - Do NOT change `name`, `description`, `supported_os`, `schema_version`.
  - Do NOT change `path` value.
  - Do NOT change `mode` value.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `nvim/rice.toml` — file to migrate

  **Acceptance Criteria**:
  - [ ] Package-level `target = "$HOME"` line removed
  - [ ] Source `target` rewritten to `"$HOME/.config/nvim"`
  - [ ] File parses (validated by Task 11's full test run)

  **QA Scenarios**:
  ```
  Scenario: nvim/rice.toml has absolute target and no package-level target
    Tool: Bash (grep)
    Preconditions: Task 6 applied
    Steps:
      1. cd /Users/guneet/rice
      2. awk '/^\[/{p=1} !p && /^target *=/{print FILENAME":"NR}' nvim/rice.toml
      3. grep -E 'target = "\$HOME/.config/nvim"' nvim/rice.toml
    Expected Result: Step 2 empty; Step 3 finds 1 match (the per-source target)
    Failure Indicators: Step 2 non-empty; Step 3 0 matches
    Evidence: .sisyphus/evidence/task-6-nvim-grep.txt
  ```

  **Commit**: NO (groups with Task 11)

- [x] 7. Migrate testdata/install fixtures

  **What to do**:
  - **`testdata/install/folder-pkg/rice.toml`**: package `target = "$HOME"`, source `target = ".config/myfolder"`. Migration: remove package target; source becomes `target = "$HOME/.config/myfolder"`.
  - **`testdata/install/folder-overlay-pkg/rice.toml`**: package `target = "$HOME"`, two folder-mode sources both targeting `.config/dup`. Migration: remove package target; both sources become `target = "$HOME/.config/dup"`.
  - **`testdata/install/mypkg/rice.toml`**: package `target = "$HOME"`, profile `common` has `sources = ["."]`, profile `macbook` has `sources = ["common", "macbook"]`. Migrate all three sources to file-mode tables with `target = "$HOME"`. Remove package target.

  **Must NOT do**:
  - Do NOT change `name`, `description`, `supported_os`.
  - Do NOT change source `path` values.
  - Do NOT add new sources or profiles.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `testdata/install/folder-pkg/rice.toml`
  - `testdata/install/folder-overlay-pkg/rice.toml`
  - `testdata/install/mypkg/rice.toml`

  **Acceptance Criteria**:
  - [ ] All three files migrated
  - [ ] `go test -race ./internal/installer/...` PASSES
  - [ ] `go test -race ./cli/...` PASSES (cli tests use these fixtures)

  **QA Scenarios**:
  ```
  Scenario: testdata/install fixtures parse and tests pass
    Tool: Bash (go test)
    Preconditions: Task 7 applied; Tasks 1-3 done
    Steps:
      1. cd /Users/guneet/rice && go test -race -count=1 ./internal/installer/ ./cli/ 2>&1 | tail -30
    Expected Result: All packages PASS
    Failure Indicators: Any FAIL; "cannot unmarshal string" panic; missing target errors
    Evidence: .sisyphus/evidence/task-7-tests.log

  Scenario: All three fixtures use table form
    Tool: Bash (grep)
    Preconditions: Task 7 applied
    Steps:
      1. cd /Users/guneet/rice
      2. for f in testdata/install/folder-pkg/rice.toml testdata/install/folder-overlay-pkg/rice.toml testdata/install/mypkg/rice.toml; do
           grep -E '^sources = \["' "$f" && echo "BAD: $f"
         done
    Expected Result: Empty
    Failure Indicators: Any line printed
    Evidence: .sisyphus/evidence/task-7-grep.txt
  ```

  **Commit**: NO (groups with Task 11)

- [x] 8. Migrate testdata/manifest + testdata/manifest_valid fixtures

  **What to do**:
  - **`testdata/manifest_valid/ghostty/rice.toml`**: package `target = "$HOME/.config/ghostty"`, two profiles using `["common", "macbook"]` and `["common", "devstick"]`. Migrate each source to `{path = "X", mode = "file", target = "$HOME/.config/ghostty"}`. Remove package target.
  - **`testdata/manifest_valid/nvim/rice.toml`**: package `target = "$HOME/.config/nvim"`, two profiles using `["common", "linux"]` and `["common", "darwin"]`. Migrate each source with `target = "$HOME/.config/nvim"`. Remove package target.
  - **`testdata/manifest/ghostty/rice.toml`**: identical to manifest_valid version — same migration.
  - **`testdata/manifest/nvim/rice.toml`**: identical to manifest_valid version — same migration.
  - **`testdata/manifest/bad/rice.toml`**: this is a deliberately-invalid fixture (`schema_version = 99`). It currently has `target = "$HOME/.config/bad"` and `sources = ["common"]`. The test asserts loading fails on schema version, not source format. Migrate sources anyway to keep the file syntactically valid post-schema-change (use `target = "$HOME/.config/bad"`). Remove package target. The schema_version 99 mismatch should remain the failure cause.

  **Must NOT do**:
  - Do NOT change `schema_version` in any of these (especially `bad/`).
  - Do NOT change `name`, `description`, `supported_os`, `profile_key`.
  - Do NOT change source `path` values.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `testdata/manifest_valid/ghostty/rice.toml`
  - `testdata/manifest_valid/nvim/rice.toml`
  - `testdata/manifest/ghostty/rice.toml`
  - `testdata/manifest/nvim/rice.toml`
  - `testdata/manifest/bad/rice.toml`

  **Acceptance Criteria**:
  - [ ] All five files migrated; package-level target removed in each
  - [ ] `bad/rice.toml` still has `schema_version = 99`
  - [ ] `go test -race ./internal/manifest/...` PASSES

  **QA Scenarios**:
  ```
  Scenario: All five manifest fixtures parse via tests
    Tool: Bash (go test)
    Preconditions: Task 8 applied; Tasks 1-3 done
    Steps:
      1. cd /Users/guneet/rice && go test -race -count=1 ./internal/manifest/ -v 2>&1 | tail -40
    Expected Result: All PASS; bad/rice.toml load test still fails on schema_version mismatch (test asserts that)
    Failure Indicators: Unexpected FAIL; bad/rice.toml fails on TOML parse rather than schema check
    Evidence: .sisyphus/evidence/task-8-tests.log

  Scenario: schema_version=99 preserved in bad fixture
    Tool: Bash (grep)
    Preconditions: Task 8 applied
    Steps:
      1. grep '^schema_version' /Users/guneet/rice/testdata/manifest/bad/rice.toml
    Expected Result: `schema_version = 99`
    Failure Indicators: Any other version
    Evidence: .sisyphus/evidence/task-8-bad-version.txt
  ```

  **Commit**: NO (groups with Task 11)

- [x] 9. Migrate inline TOML in cli/install_test.go

  **What to do**:
  - In `cli/install_test.go`:
    - Line ~38: `sources = [{path = "cfg", mode = "folder", target = ".config/folderpkg"}]` — already table form, but `target` is relative. Update to `target = "$HOME/.config/folderpkg"` and remove any `target = "$HOME"` on the manifest. Re-verify the test asserts still hold (the assertion checks the resolved symlink path).
    - Line ~41: `sources = ["cfg"]` (file-mode, bare string) → `sources = [{path = "cfg", mode = "file", target = "$HOME"}]`. Remove package-level `target = "$HOME"` from this fixture too.
    - Line ~79: `sources = ["common"]` → `sources = [{path = "common", mode = "file", target = "$HOME"}]`.
    - Line ~82: `sources = ["common", "macbook"]` → table form for both with `target = "$HOME"`.
    - For each fixture in this file, remove the package-level `target = "..."` line.
  - Re-run `go test -race ./cli/...` and update any assertion that compares to the old resolved-path computation. Most assertions check the FINAL state.json or filesystem, so they should pass without change.

  **Must NOT do**:
  - Do NOT add new test cases.
  - Do NOT remove existing test cases.
  - Do NOT change test names or assertions UNLESS the migration breaks them.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `cli/install_test.go` lines 38, 41, 79, 82 — inline TOML fixtures

  **Acceptance Criteria**:
  - [ ] All inline TOML fixtures use table form
  - [ ] No `target = "$HOME"` on manifest scope (only on sources)
  - [ ] `go test -race ./cli/...` PASSES

  **QA Scenarios**:
  ```
  Scenario: cli tests pass with migrated inline fixtures
    Tool: Bash (go test)
    Preconditions: Task 9 applied
    Steps:
      1. cd /Users/guneet/rice && go test -race -count=1 ./cli/ -v 2>&1 | tail -40
    Expected Result: All PASS
    Failure Indicators: Any FAIL; "cannot unmarshal string" in test output
    Evidence: .sisyphus/evidence/task-9-cli-tests.log

  Scenario: No string-form sources in cli/install_test.go
    Tool: Bash (grep)
    Preconditions: Task 9 applied
    Steps:
      1. grep -nE 'sources = \["[^{]' /Users/guneet/rice/cli/install_test.go
    Expected Result: Empty
    Failure Indicators: Any line printed
    Evidence: .sisyphus/evidence/task-9-grep.txt
  ```

  **Commit**: NO (groups with Task 11)

- [x] 10. Migrate inline TOML in internal/installer/install_test.go and internal/manifest/load_test.go

  **What to do**:
  - In `internal/installer/install_test.go`:
    - Find all inline TOML fixtures (search for `sources = [` and any TOML strings written via `WriteFile` or `bytes`/`heredoc`). The Metis review noted line 117 has a comment about `sources = ["."]`; check if real fixtures exist.
    - Migrate each to table form. Remove package-level `target` lines.
  - In `internal/manifest/load_test.go`:
    - Line ~110: `os.WriteFile(ricePath, []byte("schema_version = 99\nname = \"bad\"\nsupported_os = [\"linux\"]\nprofile_key = \"os\"\n[profiles.default]\nsources = [\"common\"]"), 0644)` — this writes a bad-schema fixture. Update the `sources = [\"common\"]` portion to table form: `sources = [{path = \"common\", mode = \"file\", target = \"$HOME\"}]`. The test still asserts schema_version failure.
    - Line ~141: `sources = ["common"]` in another test fixture → `sources = [{path = "common", mode = "file", target = "$HOME"}]`. Remove any package-level `target` from that fixture.
  - Re-run `go test -race ./internal/installer/... ./internal/manifest/...`.

  **Must NOT do**:
  - Do NOT change test names, assertions, or fail-expectations UNLESS migration breaks them.
  - Do NOT add or remove test cases.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: none

  **References**:
  - `internal/installer/install_test.go` (search for `sources` and inline TOML)
  - `internal/manifest/load_test.go` lines 110, 141

  **Acceptance Criteria**:
  - [ ] All inline TOML fixtures in both files use table form
  - [ ] No package-level `target = "..."` in inline TOML except where intentional for testing
  - [ ] `go test -race ./internal/installer/... ./internal/manifest/...` PASSES

  **QA Scenarios**:
  ```
  Scenario: Installer + manifest tests pass with migrated inline fixtures
    Tool: Bash (go test)
    Preconditions: Task 10 applied
    Steps:
      1. cd /Users/guneet/rice && go test -race -count=1 ./internal/installer/ ./internal/manifest/ -v 2>&1 | tail -40
    Expected Result: All PASS
    Failure Indicators: Any FAIL; "cannot unmarshal string" panic
    Evidence: .sisyphus/evidence/task-10-tests.log

  Scenario: No string-form sources in inline test fixtures
    Tool: Bash (grep)
    Preconditions: Task 10 applied
    Steps:
      1. grep -nE 'sources = \\\["[^{]' /Users/guneet/rice/internal/installer/install_test.go /Users/guneet/rice/internal/manifest/load_test.go
    Expected Result: Empty
    Failure Indicators: Any line printed
    Evidence: .sisyphus/evidence/task-10-grep.txt
  ```

  **Commit**: NO (groups with Task 11)

- [ ] 11. Update AGENTS.md, run final verify gate, manual repro, single commit

  **What to do**:
  - In `/Users/guneet/rice/AGENTS.md`:
    - "rice.toml Schema" section — REWRITE the example block to show table-only form:
      ```toml
      schema_version = 1
      name = "ghostty"
      description = "Ghostty terminal emulator configuration"
      supported_os = ["linux", "darwin"]

      [profiles.common]
      sources = [
        {path = "common", mode = "file", target = "$HOME"},
      ]

      [profiles.macbook]
      sources = [
        {path = "common", mode = "file", target = "$HOME"},
        {path = "macbook", mode = "file", target = "$HOME"},
      ]
      ```
    - Fields table — REMOVE the `target` row (it's gone from package level). UPDATE the `profiles.<name>.sources` row to describe the required table form: "List of source tables, each with required `path`, `mode`, and `target` fields."
    - Add a new "Source Spec" subsection explaining the three required fields:
      - `path`: relative to package directory
      - `mode`: `"file"` (walk and lay each file under target) or `"folder"` (single symlink at target)
      - `target`: absolute destination root (env vars expanded with `os.ExpandEnv`)
    - Remove or update the existing "Folder-mode sources" subsection so it no longer talks about it being optional/special — it's just one mode option.
    - Update "Adding a New Dotfile Package" example accordingly.

  - In `/Users/guneet/rice/README.md`:
    - Lines 78-94 have a stale `rice.toml` example with package-level `target = "$HOME"` and bare-string sources. UPDATE the example to use the new table form (matching the AGENTS.md update).
    - Lines 112-113 have another stale example showing string-form sources in a `workmac` profile. UPDATE to table form.

  - Run final verification gate:
    ```bash
    cd /Users/guneet/rice
    gofmt -l internal/ cli/
    go vet ./...
    go build -o /tmp/rice ./cli
    go test -race -count=1 ./...
    grep -rn 'sources = \["' --include='*.toml' .
    grep -rn 'sources = \["[^{]' --include='*.go' .
    ```
    All must pass / be empty.

  - Manual repro (one real package, e.g. zsh):
    ```bash
    TH=$(mktemp -d)
    HOME=$TH /tmp/rice install zsh --profile common --repo /Users/guneet/rice --state $TH/state.json --yes
    ls -la $TH/.zshrc 2>/dev/null && echo "INSTALLED OK"
    HOME=$TH /tmp/rice uninstall zsh --state $TH/state.json --yes
    test ! -e $TH/.zshrc && echo "REMOVED OK"
    ```

  - Stage and commit ALL changes with message: `refactor(schema): require table-form sources with explicit mode and target`

  **Must NOT do**:
  - Do NOT push.
  - Do NOT amend any prior commit.
  - Do NOT skip the final verify gate.

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `git-master`

  **References**:
  - `/Users/guneet/rice/AGENTS.md` — schema section to rewrite
  - All previously-modified files for the commit

  **Acceptance Criteria**:
  - [ ] AGENTS.md schema section rewritten
  - [ ] README.md schema examples updated (lines ~78-94 and ~112-113)
  - [ ] `gofmt -l internal/ cli/` empty
  - [ ] `go vet ./...` clean
  - [ ] `go build -o /tmp/rice ./cli` succeeds
  - [ ] `go test -race -count=1 ./...` ALL packages PASS
  - [ ] `grep -rn 'sources = \["' --include='*.toml' .` empty
  - [ ] `grep -rn 'sources = \["[^{]' --include='*.go' .` empty
  - [ ] Manual repro: zsh installs and uninstalls cleanly
  - [ ] Single commit created with specified message; SHA reported

  **QA Scenarios**:
  ```
  Scenario: Full verification gate passes
    Tool: Bash
    Preconditions: All tasks 1-11 applied
    Steps:
      1. cd /Users/guneet/rice
      2. gofmt -l internal/ cli/ 2>&1 | tee /tmp/t11-fmt.log
      3. go vet ./... 2>&1 | tee /tmp/t11-vet.log
      4. go build -o /tmp/rice ./cli 2>&1 | tee /tmp/t11-build.log
      5. go test -race -count=1 ./... 2>&1 | tee /tmp/t11-tests.log
      6. grep -rn 'sources = \["' --include='*.toml' . 2>&1 | tee /tmp/t11-toml-grep.log
      7. grep -rn 'sources = \\\["[^{]' --include='*.go' . 2>&1 | tee /tmp/t11-go-grep.log
    Expected Result:
      - Step 2: empty
      - Step 3: empty
      - Step 4: empty (success)
      - Step 5: every package shows "ok"; zero FAIL
      - Step 6: empty
      - Step 7: empty
    Failure Indicators: Any non-empty step (except step 5 which prints "ok" lines)
    Evidence: .sisyphus/evidence/task-11-verify-gate.log

  Scenario: Manual install/uninstall of zsh succeeds end-to-end
    Tool: Bash
    Preconditions: Verification gate passed; /tmp/rice binary built
    Steps:
      1. TH=$(mktemp -d) && echo "TH=$TH"
      2. HOME=$TH /tmp/rice install zsh --profile common --repo /Users/guneet/rice --state $TH/state.json --yes 2>&1 | tee /tmp/t11-install.log
      3. ls -la $TH/.zshrc 2>&1 | tee /tmp/t11-after-install.log
      4. test -L $TH/.zshrc && echo "IS SYMLINK"
      5. HOME=$TH /tmp/rice uninstall zsh --state $TH/state.json --yes 2>&1 | tee /tmp/t11-uninstall.log
      6. test ! -e $TH/.zshrc && echo "REMOVED"
      7. test -f /Users/guneet/rice/zsh/.zshrc && echo "SOURCE INTACT"  # adjust filename to whatever zsh package contains
    Expected Result:
      - Step 2 exit 0; install plan shows expected ops
      - Step 3-4: symlink exists
      - Step 5 exit 0
      - Step 6: REMOVED printed
      - Step 7: SOURCE INTACT printed (rice repo files untouched)
    Failure Indicators: Any non-zero exit; symlink missing after install; symlink remaining after uninstall; source files modified
    Evidence: .sisyphus/evidence/task-11-manual-repro.log

  Scenario: Single commit created with correct message
    Tool: Bash (git)
    Preconditions: All changes staged and committed
    Steps:
      1. cd /Users/guneet/rice && git log -1 --pretty=format:'%s'
      2. git log -1 --pretty=format:'%H'
    Expected Result:
      - Step 1: `refactor(schema): require table-form sources with explicit mode and target`
      - Step 2: 40-char hex SHA
    Failure Indicators: Different message; no commit; multiple commits since plan start
    Evidence: .sisyphus/evidence/task-11-commit.txt
  ```

  **Commit**: YES (final commit)
  - Message: `refactor(schema): require table-form sources with explicit mode and target`
  - Files: ALL changes from Tasks 1-11
  - Pre-commit: full verify gate above

---

## Final Verification Gate

- [ ] **Final**: Run full verification gate

  **Verification commands (must all succeed)**:
  ```bash
  cd /Users/guneet/rice
  gofmt -l internal/ cli/                                              # must be empty
  go vet ./...                                                          # must be clean
  go build -o /tmp/rice ./cli                                          # must succeed
  go test -race -count=1 ./...                                         # all green
  grep -rn 'sources = \["' --include='*.toml' .                        # empty
  grep -rn 'sources = \["[^{]' --include='*.go' .                      # empty (no string-form fixtures)
  grep -rn '\bm\.Target\b\|manifest\.Target\b' internal/ cli/          # empty (Manifest.Target removed)
  ```

  **Manual repro** (one real package, e.g. zsh):
  ```bash
  TH=$(mktemp -d)
  HOME=$TH /tmp/rice install zsh --profile common --repo /Users/guneet/rice --state $TH/state.json --yes
  ls -la $TH/.zshrc 2>/dev/null && echo "INSTALLED OK"
  HOME=$TH /tmp/rice uninstall zsh --state $TH/state.json --yes
  test ! -e $TH/.zshrc && echo "REMOVED OK"
  ```

  **Single commit**:
  - Message: `refactor(schema): require table-form sources with explicit mode and target`
  - Use `git add -A` then `git commit -m "..."`
  - Report commit SHA

---

## Commit Strategy

Single commit at the end (Task 11). Message:
```
refactor(schema): require table-form sources with explicit mode and target

- SourceSpec.UnmarshalTOML now requires {path, mode, target} table form;
  bare-string sources are no longer accepted
- Manifest.Target package-level field removed; every source carries its
  own absolute target (with os.ExpandEnv)
- Validator: target required for both file and folder modes
- Walker uses per-source target as destination root
- profile.Resolve (legacy []string) deleted; only ResolveSpecs remains
- All rice.toml files in repo migrated
- AGENTS.md schema section rewritten
```

---

## Success Criteria

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass with `-race`
- [ ] gofmt + vet clean
- [ ] Build succeeds
- [ ] No string-form sources anywhere
- [ ] No `Manifest.Target` references anywhere
- [ ] Manual repro of one real package succeeds
- [ ] Single commit created with specified message
