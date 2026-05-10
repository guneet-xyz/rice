# Fix: Display conflict details when rice install/switch detect conflicts

## TL;DR

> **Quick Summary**: When `rice install` or `rice switch` detects conflicts, the error says `"conflicts detected: 1"` but never lists which files conflict. Conflicts ARE captured in `plan.Conflicts` and `RenderConflicts()` already exists — they're just never invoked because the CLI discards the plan on error.
>
> **Deliverables**:
> - `RenderPlan` displays a "Conflicts (N):" section when `p.Conflicts` is non-empty
> - `cmd/rice/cmd/install.go` renders the plan even when `BuildInstallPlan` returns a conflict error
> - `cmd/rice/cmd/switch.go` renders the plan even when `BuildSwitchPlan` returns a conflict error
> - 2 new tests asserting conflict details appear in stdout
>
> **Estimated Effort**: Quick (~20 LOC across 5 files)
> **Parallel Execution**: NO - sequential (single small fix)
> **Critical Path**: Edit prompt.go → Edit install.go + switch.go → Add tests → Verify → Commit

---

## Context

### Original Request
> "During installation, it says I have conflicts. But it won't list what files are causing those conflicts."

### Reproduction (confirmed)
```sh
go build -o /tmp/rice ./cmd/rice
TH=$(mktemp -d) && mkdir -p "$TH/.config/ghostty" && echo manual > "$TH/.config/ghostty/config"
HOME="$TH" /tmp/rice install ghostty --profile common --repo . --state "$TH/state.json" --yes
# → Error: build plan: conflicts detected: 1
# → (no mention of WHICH file conflicts)
```

### Root Cause Analysis
- `internal/installer/install.go:217` returns `(p, error)` where `p.Conflicts` is fully populated
- `internal/installer/switch.go:120` returns `(sp, error)` where `sp.Install.Conflicts` is populated
- `internal/prompt/prompt.go:109` already has `RenderConflicts(w, conflicts)` printing `CONFLICT  <target>: <reason>`
- `internal/prompt/prompt.go:29 RenderPlan` does NOT include conflicts even though `plan.Plan` has a `Conflicts` field
- `cmd/rice/cmd/install.go:45-48` returns immediately on `BuildInstallPlan` error, never calling `RenderPlan(p)` — conflicts are silently discarded
- `cmd/rice/cmd/switch.go` has the same defect

---

## Work Objectives

### Core Objective
Surface conflict details (target path + reason) in CLI output before exiting with non-zero status, so users can fix or investigate the conflicting files.

### Concrete Deliverables
- `internal/prompt/prompt.go`: `RenderPlan` appends a Conflicts section when `p.Conflicts` is non-empty
- `internal/prompt/prompt.go`: `RenderSwitchPlan` appends a Conflicts section for both phases
- `cmd/rice/cmd/install.go`: render plan even when `BuildInstallPlan` returns error
- `cmd/rice/cmd/switch.go`: render plan even when `BuildSwitchPlan` returns error
- `cmd/rice/cmd/install_test.go`: new test `TestInstall_ShowsConflictDetails`
- `cmd/rice/cmd/switch_test.go`: new test `TestSwitch_ShowsConflictDetails`

### Definition of Done
- `go build ./...` succeeds
- `go vet ./...` clean
- `gofmt -l .` produces no output for changed files
- `go test ./... -race` passes (all existing + 2 new tests)
- Manual repro shows `CONFLICT  <full-target-path>: <reason>` line(s) BEFORE the error
- Non-zero exit code preserved on conflict
- One commit with message `fix(cli): display conflict details when install/switch detect conflicts`

### Must Have
- Conflict target path appears in stdout
- Conflict reason appears in stdout
- Error still returned (non-zero exit)
- Existing tests still pass

### Must NOT Have (Guardrails)
- NO new flags (no `--force`, no `--show-conflicts`)
- NO change to `BuildInstallPlan`/`BuildSwitchPlan` return semantics
- NO change to conflict detection logic
- NO suppression of error — conflicts still cause non-zero exit
- NO modification to opencode/, .sisyphus/, or other dotfile packages

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — all verification agent-executable.

### Test Decision
- **Infrastructure exists**: YES (Go testing, testify, existing _test.go patterns)
- **Automated tests**: Tests-after (add 2 new tests verifying conflict display)
- **Framework**: `go test -race`

---

## TODOs

- [x] 1. Add Conflicts section to RenderPlan and RenderSwitchPlan

  **What to do**:
  - Edit `internal/prompt/prompt.go`
  - In `RenderPlan` (around line 73, after the `Total:` line): if `len(p.Conflicts) > 0`, print `\nConflicts (%d):\n` then call `RenderConflicts(w, p.Conflicts)`
  - In `RenderSwitchPlan` (around line 103, after the `Total:` line): same pattern, but check both `uninstall.Conflicts` and `install.Conflicts` (if either non-empty, print combined Conflicts section). In practice only `install.Conflicts` will be populated, but defend against both.

  **Must NOT do**:
  - Don't change `RenderConflicts` itself
  - Don't change the existing output format for the operations table

  **References**:
  - `internal/prompt/prompt.go:29-73` — existing `RenderPlan`
  - `internal/prompt/prompt.go:76-104` — existing `RenderSwitchPlan`
  - `internal/prompt/prompt.go:109-113` — existing `RenderConflicts`
  - `internal/plan/plan.go` — `Plan{Conflicts []Conflict}`, `Conflict{Target, Source, Reason}`

  **Acceptance Criteria**:
  - [ ] `gofmt -l internal/prompt/prompt.go` produces no output
  - [ ] Existing `prompt_test.go` tests still pass

- [x] 2. Render plan even on conflict error in install.go

  **What to do**:
  - Edit `cmd/rice/cmd/install.go`
  - Change the block at lines 45-48 to:
    ```go
    p, err := installer.BuildInstallPlan(req)
    if p != nil {
        prompt.RenderPlan(cmd.OutOrStdout(), p)
    }
    if err != nil {
        return fmt.Errorf("build plan: %w", err)
    }
    ```
  - Note: when err != nil, `RenderPlan` runs first (so conflicts visible), then error returned. Confirm and execute paths only run when err == nil.

  **Must NOT do**:
  - Don't render the plan twice in the success path (move the existing `prompt.RenderPlan` call — only call it via the new pre-error block)
  - Wait — re-read: existing code at line 50 calls `prompt.RenderPlan` on success. The fix should NOT double-render. Solution: remove the line-50 call (now redundant since line 46 always renders if p != nil). Verify success path still renders plan exactly once.

  **References**:
  - `cmd/rice/cmd/install.go:28-67` — current implementation
  - `internal/installer/install.go:200-218` — confirms `p` is non-nil with conflicts populated even on conflict error

  **Acceptance Criteria**:
  - [ ] On success: plan rendered exactly once before confirm prompt
  - [ ] On conflict: plan + Conflicts section rendered, then error returned with non-zero exit
  - [ ] `gofmt -l cmd/rice/cmd/install.go` clean

- [x] 3. Render plan even on conflict error in switch.go

  **What to do**:
  - Edit `cmd/rice/cmd/switch.go`
  - Apply the same pattern: render `RenderSwitchPlan(out, sp.Uninstall, sp.Install)` if `sp != nil`, then check error
  - Avoid double-rendering on success path

  **References**:
  - `cmd/rice/cmd/switch.go` — current implementation
  - `internal/installer/switch.go:100-120` — confirms `sp` non-nil on pre-flight conflict

  **Acceptance Criteria**:
  - [ ] On success: switch plan rendered exactly once
  - [ ] On conflict: switch plan + Conflicts section rendered, then error returned

- [x] 4. Add test: TestInstall_ShowsConflictDetails

  **What to do**:
  - Edit `cmd/rice/cmd/install_test.go` (read existing tests for the pattern first)
  - New test: set up tmpdir as HOME, copy `testdata/install/mypkg` into a tmp repo, pre-create a regular file at the target path that mypkg's install would create (use the manifest to know the target), run `runInstall` via cobra (or call the command directly with stdout captured), assert:
    - Returned error is non-nil
    - Captured stdout contains the substring `CONFLICT`
    - Captured stdout contains the conflicting target path

  **References**:
  - `cmd/rice/cmd/install_test.go` — existing test patterns
  - `testdata/install/mypkg/` — existing fixture
  - `internal/prompt/prompt.go:111` — format `CONFLICT  <target>: <reason>`

  **Acceptance Criteria**:
  - [ ] `go test -race ./cmd/rice/cmd/ -run TestInstall_ShowsConflictDetails` passes

- [x] 5. Add test: TestSwitch_ShowsConflictDetails

  **What to do**:
  - Edit `cmd/rice/cmd/switch_test.go`
  - Pattern: install pkg with profile A, then pre-create a foreign file at a target the new profile B uses but A doesn't, run switch B --yes, assert stdout contains `CONFLICT` and the target path, error is non-nil
  - If existing test fixtures don't have a profile pair with disjoint targets, simpler approach: install with profile A, manually rm the symlink (so state still records it), pre-create a regular file at that path, run switch to profile B with `--yes` — pre-flight should detect the foreign file as conflict for the install phase
  - Consult existing `switch_test.go` for the right fixture/pattern

  **References**:
  - `cmd/rice/cmd/switch_test.go` — existing patterns
  - `internal/installer/switch.go` — pre-flight conflict detection logic

  **Acceptance Criteria**:
  - [ ] `go test -race ./cmd/rice/cmd/ -run TestSwitch_ShowsConflictDetails` passes

- [x] 6. Verify, format, and commit

  **What to do**:
  - Run `gofmt -w internal/prompt/prompt.go cmd/rice/cmd/install.go cmd/rice/cmd/switch.go cmd/rice/cmd/install_test.go cmd/rice/cmd/switch_test.go`
  - Run `go build ./...`
  - Run `go vet ./...`
  - Run `go test ./... -race`
  - Manual repro:
    ```sh
    go build -o /tmp/rice ./cmd/rice
    TH=$(mktemp -d) && mkdir -p "$TH/.config/ghostty" && echo manual > "$TH/.config/ghostty/config"
    HOME="$TH" /tmp/rice install ghostty --profile common --repo /Users/guneet/rice --state "$TH/state.json" --yes 2>&1
    ```
  - Assert output contains: `CONFLICT` and the path `$TH/.config/ghostty/config`
  - Save output to `.sisyphus/evidence/fix-conflict-display-repro.txt`
  - `git add -A && git commit -m "fix(cli): display conflict details when install/switch detect conflicts"`
  - Report the commit SHA back

  **Acceptance Criteria**:
  - [ ] Build/vet/tests/gofmt all clean
  - [ ] Manual repro shows CONFLICT line with target path
  - [ ] Single commit created with the specified message
  - [ ] Commit SHA reported

---

## Commit Strategy

- **Single commit**: `fix(cli): display conflict details when install/switch detect conflicts`
- Files: `internal/prompt/prompt.go`, `cmd/rice/cmd/install.go`, `cmd/rice/cmd/switch.go`, `cmd/rice/cmd/install_test.go`, `cmd/rice/cmd/switch_test.go`
- Pre-commit: `go test ./... -race && gofmt -l . | grep -v opencode/ | grep -v .sisyphus/ | (! grep .)`

---

## Success Criteria

### Verification Commands
```bash
go build ./...                          # succeeds
go vet ./...                            # clean
go test ./... -race                     # all pass including 2 new tests
gofmt -l internal/ cmd/                 # no output

# Manual repro
TH=$(mktemp -d) && mkdir -p "$TH/.config/ghostty" && echo manual > "$TH/.config/ghostty/config"
HOME="$TH" /tmp/rice install ghostty --profile common --repo /Users/guneet/rice --state "$TH/state.json" --yes
# Expected: output contains "CONFLICT" + the target path BEFORE the error
# Expected: exit code != 0
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass with -race
- [ ] One commit with the specified message
