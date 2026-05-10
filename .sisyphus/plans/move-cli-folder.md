# Move CLI to /cli at repo root

## TL;DR

> **Quick Summary**: Flatten `cmd/rice/main.go` + `cmd/rice/cmd/*.go` into a single top-level `cli/` package. Removes the awkward `cmd/rice/cmd` double-nesting. New build command becomes `go build ./cli`.
>
> **Deliverables**:
> - All `.go` files from `cmd/rice/` and `cmd/rice/cmd/` moved to `cli/`
> - Single `package cli` (renamed from `package cmd` + `package main`); main.go remains `package main`
> - All imports updated: `github.com/guneet/rice/cmd/rice/cmd` → `github.com/guneet/rice/cli`
> - `cmd/` directory removed
> - README.md and AGENTS.md updated to reference `./cli` instead of `./cmd/rice`
>
> **Estimated Effort**: Quick (mechanical move + 1 import path update + doc tweaks)
> **Parallel Execution**: NO — sequential (file moves must use `git mv` cleanly, then verify)
> **Critical Path**: git mv files → fix package declarations → fix import → update docs → verify → commit

---

## Context

### Original Request
> "Can you move all the cli stuff to a cli folder?"
> User chose: "Move to /cli at repo root" — flatten everything, single package.

### Current Layout
```
cmd/
└── rice/
    ├── main.go              (package main; imports .../cmd/rice/cmd)
    └── cmd/
        ├── root.go          (package cmd)
        ├── install.go       (package cmd)
        ├── install_test.go  (package cmd)
        ├── switch.go        (package cmd)
        ├── switch_test.go   (package cmd)
        ├── uninstall.go     (package cmd)
        ├── uninstall_test.go
        ├── status.go        (package cmd)
        ├── status_test.go
        ├── doctor.go        (package cmd)
        ├── doctor_test.go
        ├── version.go       (package cmd)
        └── root_test.go     (package cmd)
```

### Target Layout
```
cli/
├── main.go              (package main; imports own siblings via package cli? NO — main.go stays in same package as commands? Decided: split.)
├── root.go              (package cli)
├── install.go           (package cli)
├── install_test.go      (package cli)
├── switch.go            (package cli)
├── switch_test.go       (package cli)
├── uninstall.go         (package cli)
├── uninstall_test.go    (package cli)
├── status.go            (package cli)
├── status_test.go       (package cli)
├── doctor.go            (package cli)
├── doctor_test.go       (package cli)
├── version.go           (package cli)
└── root_test.go         (package cli)
```

**Decision on main.go**: Two valid options:
- (A) Single `package main` for everything — simplest, matches user's "all in one folder" intent, build with `go build ./cli`.
- (B) Split into `package main` (main.go) + `package cli` (commands) in the same dir — Go disallows two packages in one directory.

Therefore (A) wins: ALL files in `cli/` use `package main`. Build target: `go build ./cli`. Tests: `go test ./cli/...` works because Go test recognizes `package main` testing.

**However**: Existing tests reference exported helpers between files within the same package — that continues to work in `package main`. But `cobra` package init pattern using `var rootCmd` etc. continues to work.

**Caveat for tests**: `package main` tests run fine with `go test`. No issue.

### Files Touching Old Path
- `cmd/rice/main.go` — has `import "github.com/guneet/rice/cmd/rice/cmd"` → will be deleted/moved
- `README.md:21` — `go build -o rice ./cmd/rice` → update to `./cli`
- `AGENTS.md:21` — directory tree comment mentions `cmd/rice/` → update

No other Go files import `cmd/rice/cmd` (verified via grep).

---

## Work Objectives

### Core Objective
Reorganize CLI source files into a single `cli/` directory at the repo root, eliminating the `cmd/rice/cmd` double-nesting.

### Concrete Deliverables
- New directory: `cli/` containing all 13 .go files (12 from `cmd/rice/cmd/` + main.go newly written or moved)
- Old `cmd/` directory fully deleted
- All package declarations changed to `package main` in cli/
- main.go body changed from importing external cmd package to direct `Execute()` call (since Execute is now in same package)
- README.md updated: `./cmd/rice` → `./cli`
- AGENTS.md updated: directory tree shows `cli/` not `cmd/rice/`
- Single commit using `git mv` to preserve history

### Definition of Done
- `go build ./cli` produces a working `rice` binary
- `go test ./... -race` passes (all existing tests including the 2 conflict-display tests added in last commit)
- `go vet ./...` clean
- `gofmt -l cli/` produces no output
- `cmd/` directory does not exist
- README.md and AGENTS.md no longer mention `cmd/rice`
- Manual smoke test: `go build -o /tmp/rice ./cli && /tmp/rice --help` shows the cobra help text
- One commit with message `refactor: move cli sources to /cli at repo root`

### Must Have
- `git mv` used for moves (preserves blame/history)
- All tests still pass
- Binary still builds and runs

### Must NOT Have (Guardrails)
- NO duplicate `package` declarations (every cli/*.go must say `package main`)
- NO leftover files in `cmd/` (directory fully removed)
- NO behavior changes — pure mechanical refactor
- NO modification to `internal/` packages (they don't reference cmd/rice/cmd)
- NO modification to dotfile packages, .sisyphus/, testdata/
- NO new flags or features
- NO renaming of any exported symbol

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — all verification is agent-executable.

### Test Decision
- **Infrastructure exists**: YES (existing Go test suite)
- **Automated tests**: Tests-after — no NEW tests needed; existing tests must continue to pass
- **Framework**: `go test -race`

---

## TODOs

- [x] 1. git mv files from cmd/rice/cmd → cli/ and cmd/rice/main.go → cli/main.go

  **What to do**:
  - From repo root `/Users/guneet/rice`:
    ```bash
    mkdir -p cli
    git mv cmd/rice/main.go cli/main.go
    git mv cmd/rice/cmd/root.go cli/root.go
    git mv cmd/rice/cmd/root_test.go cli/root_test.go
    git mv cmd/rice/cmd/install.go cli/install.go
    git mv cmd/rice/cmd/install_test.go cli/install_test.go
    git mv cmd/rice/cmd/switch.go cli/switch.go
    git mv cmd/rice/cmd/switch_test.go cli/switch_test.go
    git mv cmd/rice/cmd/uninstall.go cli/uninstall.go
    git mv cmd/rice/cmd/uninstall_test.go cli/uninstall_test.go
    git mv cmd/rice/cmd/status.go cli/status.go
    git mv cmd/rice/cmd/status_test.go cli/status_test.go
    git mv cmd/rice/cmd/doctor.go cli/doctor.go
    git mv cmd/rice/cmd/doctor_test.go cli/doctor_test.go
    git mv cmd/rice/cmd/version.go cli/version.go
    ```
  - Then remove the now-empty `cmd/` tree:
    ```bash
    rmdir cmd/rice/cmd cmd/rice cmd
    ```
  - Verify with `git status` — should show 13 renames + 0 deletions of files (only empty dirs gone).

  **Must NOT do**:
  - Don't use plain `mv` (loses git history detection)
  - Don't leave any `cmd/rice` directories behind

  **References**:
  - `cmd/rice/main.go` — entrypoint
  - `cmd/rice/cmd/*.go` — 12 command files

  **Acceptance Criteria**:
  - [ ] `cli/` contains 13 .go files
  - [ ] `cmd/` does not exist
  - [ ] `git status` shows the moves as renames (R)

- [x] 2. Update package declarations and imports

  **What to do**:
  - Change `package cmd` → `package main` in all 12 moved command files (root, install, switch, uninstall, status, doctor, version + their _test.go files)
  - `cli/main.go` is already `package main` — but its body needs to change: it currently imports `github.com/guneet/rice/cmd/rice/cmd` and calls `cmd.Execute()`. Since `Execute` will now be in the SAME package (`package main`), change main.go to:
    ```go
    package main

    func main() {
        Execute()
    }
    ```
    (Drop the import statement entirely.)
  - Verify `Execute` is exported (capital E) in root.go — it should already be, since it was previously called from outside the package.

  **Must NOT do**:
  - Don't change `internal/` packages — they don't import cmd
  - Don't change any function signatures
  - Don't add new imports

  **References**:
  - `cli/main.go` (post-move) — must drop the `cmd/rice/cmd` import
  - `cli/root.go` (post-move) — verify `func Execute()` is exported

  **Acceptance Criteria**:
  - [ ] All `cli/*.go` files start with `package main`
  - [ ] `cli/main.go` has no imports referring to `cmd/rice` or external cli packages
  - [ ] `grep -r "github.com/guneet/rice/cmd" .` returns nothing (excluding .sisyphus/)

- [x] 3. Update documentation

  **What to do**:
  - `README.md`: replace `./cmd/rice` with `./cli` (one occurrence at line 21)
  - `AGENTS.md`: update the directory tree section. Replace lines that show `cmd/rice/` and `cmd/` subtree with `cli/` showing the flat layout. Approximately:
    ```
    ├── cli/                  # CLI entrypoint (main.go) and cobra commands
    │   ├── main.go
    │   ├── root.go
    │   ├── install.go        # install, uninstall, switch, status, doctor, version
    │   └── ...
    ```
  - Search AGENTS.md and README.md for any other mention of `cmd/rice` and update if found.

  **References**:
  - `README.md:21`
  - `AGENTS.md:21`

  **Acceptance Criteria**:
  - [ ] `grep -n "cmd/rice" README.md AGENTS.md` returns nothing

- [x] 4. Verify, format, and commit

  **What to do**:
  ```bash
  cd /Users/guneet/rice
  gofmt -w cli/
  go build ./cli
  go vet ./...
  go test ./... -race
  go build -o /tmp/rice ./cli && /tmp/rice --help
  ```
  All must succeed. Then commit:
  ```bash
  git add -A
  git commit -m "refactor: move cli sources to /cli at repo root

  Flatten cmd/rice/main.go + cmd/rice/cmd/*.go into a single top-level cli/
  package. Removes the awkward cmd/rice/cmd double-nesting.

  - All command files moved via git mv (preserves history)
  - Package renamed: cmd → main (single package in cli/)
  - main.go simplified: no longer imports external cmd package
  - README and AGENTS docs updated to reference ./cli"
  ```
  Report the commit SHA.

  **Acceptance Criteria**:
  - [ ] Build succeeds
  - [ ] `--help` output shows cobra help with all 5 subcommands (install, uninstall, switch, status, doctor)
  - [ ] All tests pass with -race
  - [ ] gofmt clean
  - [ ] Single commit with the specified message

---

## Commit Strategy

- **Single commit**: `refactor: move cli sources to /cli at repo root`
- All file moves + package rename + main.go update + doc updates in one atomic commit
- Pre-commit verification: `go build ./cli && go test ./... -race && gofmt -l cli/ | (! grep .)`

---

## Success Criteria

### Verification Commands
```bash
go build ./cli                          # produces working binary
go vet ./...                            # clean
go test ./... -race                     # all pass (including conflict tests)
gofmt -l cli/                           # no output
test ! -d cmd                           # cmd/ is gone
grep -r "cmd/rice" --include='*.go' .   # no Go files reference old path
grep "cmd/rice" README.md AGENTS.md     # docs updated, no matches

# Smoke test
go build -o /tmp/rice ./cli && /tmp/rice --help
# Expected: cobra help with install/uninstall/switch/status/doctor subcommands
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass with -race
- [ ] One commit with the specified message
- [ ] Git history preserved (renames detected)
