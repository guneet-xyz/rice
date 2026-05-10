# Folder-Mode Symlinks

## TL;DR

> **Quick Summary**: Allow `rice.toml` to opt sources into folder-mode (symlink the entire source dir as one symlink) instead of the default file-by-file mode. Use case: tools like nvim or opencode where you want `~/.config/nvim` itself to be a symlink to `<repo>/nvim/common/.config/nvim`, not a tree of file symlinks.
>
> **User decisions baked in**:
> - **Granularity**: Per-source. A source entry is either a string (file-mode, current behavior) or a table `{path = "common", mode = "folder"}`.
> - **Existing target dir**: Always fail as conflict if `~/.config/foo` exists (regardless of empty/non-empty), unless it's a symlink we already own.
> - **Source overlay**: Reject as a planning error if multiple sources in the same profile both touch the same folder-mode subtree (overlay is only valid for file-mode).
>
> **Deliverables**:
> - Schema: `Sources []SourceSpec` where `SourceSpec` accepts both string and table TOML forms (custom UnmarshalTOML)
> - Walking: when a source is folder-mode, walk only one level deep — produce one Op per top-level entry under that source dir; if entry is a dir → emit a folder symlink Op; if a file → emit a file symlink Op
> - Plan: extend `plan.Op` with `IsDir bool` so executor knows whether to create a file or folder symlink (Go's `os.Symlink` is the same call but conflict + uninstall semantics differ)
> - Conflict detection: extend to handle folder targets (target dir existing as anything-other-than-our-symlink = conflict)
> - Symlink package: extend `IsSymlinkTo` for folder symlinks (already works since it uses `os.Readlink`), add explicit folder-aware path
> - State: `InstalledLink` gains `IsDir bool` for correct uninstall semantics
> - Render: `prompt.RenderPlan` distinguishes `CREATE` (file) vs `CREATE-DIR` (folder)
> - Tests: 8+ new tests covering happy paths, conflicts, overlay rejection, uninstall, switch
> - One package migrated as proof: `nvim` to use folder-mode for `.config/nvim/`
> - Docs updated: AGENTS.md schema section + new examples
>
> **Estimated Effort**: Medium (touches 8 internal/ files + tests + 1 dotfile package + docs)
> **Parallel Execution**: YES — 3 waves
> **Critical Path**: Schema (T1) → Plan/Op extension (T2) → Walking (T3) → Conflict (T4) → Executor (T5) → State (T6) → Render (T7) → Tests (T8-T11) → Docs+migration (T12-T13) → Verify+commit (T14)

---

## Context

### Original Request
> "We should give the package option to specify how symlinks should work. Fallback to file based symlinks but it might be a requirement to symlink entire folders."

### Why this matters
Some tools resolve their config dir's identity at startup (nvim, opencode, jetbrains products). If `~/.config/nvim` is a directory containing many file symlinks, each tool sees its real config root as `~/.config/nvim`. But for tools that need to follow symlinks back to a single source (e.g., to find sibling files relative to a known repo path, or to write back), the user may want `~/.config/nvim` itself to be a single symlink pointing to the repo.

Concrete example for nvim:
- File-mode (today): `~/.config/nvim/init.lua` → `<repo>/nvim/common/.config/nvim/init.lua`, `~/.config/nvim/lua/foo.lua` → `<repo>/nvim/common/.config/nvim/lua/foo.lua`, etc. Many symlinks.
- Folder-mode (new): `~/.config/nvim` → `<repo>/nvim/common/.config/nvim`. One symlink. Adding new files under `lua/` requires no rice re-install.

### Research Findings (from explore agent)
- **File walking**: `internal/installer/install.go:136-188` uses `filepath.WalkDir` to enumerate every regular file under each source dir. Skips dirs, skips `rice.toml`, skips symlinks-in-source. Computes `target = targetRoot + rel(sourceDir, file)`. Builds `[]pendingOp{Source, Target}`. Override semantics: later source wins per-target.
- **Schema**: `internal/manifest/schema.go:14-17` — `ProfileDef{Sources []string}` is currently a flat list of strings.
- **Symlink primitives**: `internal/symlink/symlink.go` — `CreateSymlink(source, target)` uses `os.Symlink` which works identically for files and directories on POSIX. `IsSymlinkTo(linkPath, expectedTarget)` uses `os.Readlink`. No code currently distinguishes file vs dir symlinks.
- **Conflict detection**: `internal/installer/conflict.go` — checks if target exists, if it's a symlink we own (via `os.Lstat` + `os.Readlink` comparison). Doesn't distinguish dir vs file targets.
- **State**: `internal/state/state.go:11-15` — `InstalledLink{Source, Target}`. No `IsDir` field. Uninstall iterates and calls a removal helper that uses `os.Remove`. `os.Remove` on a symlink to a dir works (removes the link, not the target dir contents).
- **Plan**: `plan.Op{Kind, Source, Target}` and `plan.Conflict{Target, Source, Reason}`. No `IsDir` distinction.
- **Renderer**: `internal/prompt/prompt.go` — `CREATE  <target>  →  <source>` line per op.
- **Tests**: `testdata/` has fixtures; existing tests in `cmd/rice/cmd/` and `internal/installer/` need to remain passing.
- **No existing package uses anything resembling folder-mode**. All current packages are file-mode by virtue of having no opt-in.
- **No TODO/FIXME** mentions folder symlinks.

### Existing per-source layout (informs schema design)
A source dir like `nvim/common/.config/nvim/` is currently walked file-by-file. With folder-mode, the walker should descend ONE level into the source (`nvim/common/`), then for each entry emit a single Op:
- `nvim/common/.config` (directory) → if folder-mode, this becomes `~/.config` symlink target — TOO BROAD, would clobber entire `.config`.

This reveals a critical design point: **folder-mode needs to descend through directories that already exist on the user's system, only creating symlinks for the leaf directories the user actually wants linked**.

**Refined design**: Folder-mode walks the source tree until it finds a directory whose corresponding target does NOT exist (or only exists as our own symlink), and emits a folder-symlink Op for it. Children of that folder are not enumerated.

**Even simpler refined design**: For folder-mode sources, the user is declaring "treat the leaf directories under this source as folder symlinks." We walk file-by-file BUT when we encounter a directory whose path-from-source-root equals one of the user-declared leaf-dirs, we emit a folder-symlink Op and don't recurse into it.

**Simplest workable design (chosen)**: For folder-mode sources, walk one level deeper than file-mode — instead of walking every file, we walk the source tree until we find any directory `D` such that `D` corresponds to a target path. Then for each LEAF (a file OR a directory whose parent dir in the target ALREADY exists at install time), emit a single Op. Files become file-symlinks, directories become folder-symlinks.

Actually, the cleanest model the user likely wants is the simplest one: **folder-mode means symlink each top-level entry under the source as-is**. So if source is `nvim/common/.config/nvim/`, the Ops are one symlink per top-level child of `.config/nvim/`. But that defeats the purpose — the goal is `~/.config/nvim` IS a symlink.

**FINAL DESIGN (per discussion below)**: A folder-mode source declares: "the entire source dir IS a symlink target." So source `nvim/common/.config/nvim/` produces exactly ONE Op:
- target: `<HOME>/.config/nvim`
- source: `<repo>/nvim/common/.config/nvim`
- kind: folder symlink

This requires the source dir to declare WHERE the link should land. Current sources implicitly do this by mirroring `$HOME` structure. So the single Op's target is computed by stripping the source prefix the same way file-mode does, but stopping at the source root (not walking into it).

So `sources = [{path = "common", mode = "folder"}]` with `common/.config/nvim/init.lua` etc. → ONE op: `~/.config/nvim → <repo>/pkg/common/.config/nvim`.

Wait — that's also wrong. The source is `common/`, not `common/.config/nvim/`. Folder-mode would symlink `common/` itself, producing `<HOME> → <repo>/pkg/common/`, which clobbers $HOME entirely.

**TRUE FINAL DESIGN**: Folder-mode requires the source path to be the exact directory to symlink, not a wrapper that mirrors $HOME structure. So:
- File-mode: source dir mirrors $HOME (e.g., `nvim/common/.config/nvim/init.lua`)
- Folder-mode: source dir IS the leaf dir (e.g., `nvim/common-cfg/`), and the manifest declares the target path (e.g., `target_path = ".config/nvim"`).

This requires NEW SCHEMA: `{path = "common-cfg", mode = "folder", target_path = ".config/nvim"}`.

OR — keep the current "mirror $HOME" convention and have folder-mode auto-detect leaf dirs. Walk until we find a directory whose corresponding target doesn't exist OR exists as our symlink. This is the "auto leaf detection" approach.

**Decision**: To keep things simple and explicit, schema will be:
```toml
[profiles.macbook]
sources = [
  "common",                                                     # file-mode (string form)
  { path = "nvim-cfg", mode = "folder", target = ".config/nvim" }, # folder-mode (table form)
]
```

The folder-mode source dir is symlinked as a unit to `<HOME>/<target>`. The `target` field is required for folder-mode and forbidden for file-mode (file-mode uses package-level `target` + mirrored structure).

---

## Work Objectives

### Core Objective
Add an opt-in folder-mode to the per-source schema, where a source dir is symlinked as a single unit to a declared target path under `$HOME`, instead of being walked file-by-file.

### Concrete Deliverables
1. Schema accepts mixed string/table source entries
2. Walker handles folder-mode sources by emitting exactly ONE Op per source
3. Conflict detection treats existing target dirs as conflicts unless owned
4. Plan/Op carries `IsDir bool`
5. State records `IsDir bool` for correct uninstall (still `os.Remove` since both are symlinks, but logged differently)
6. Renderer shows `CREATE-DIR` vs `CREATE`
7. Multi-source overlay where 2+ sources in same profile target same folder-mode path = planning error
8. nvim package migrated to folder-mode as the canary
9. AGENTS.md schema section documents the new form

### Definition of Done
- `go build ./cli` succeeds
- `go vet ./...` clean
- `gofmt -l .` clean (excluding opencode/, .sisyphus/)
- `go test ./... -race` passes (existing + 8 new tests)
- `nvim/rice.toml` migrated to folder-mode; manual install in tmpdir produces `<HOME>/.config/nvim` as a single symlink
- Backward compatible: every existing rice.toml still parses and behaves identically (no behavior change for file-mode)
- One commit per logical wave (5 commits) OR one squashed commit — TBD by atlas

### Must Have
- Backward compatibility: string-form sources keep working unchanged
- Folder-mode requires explicit `target` field; missing target = manifest validation error
- Conflict detection refuses to overwrite an existing dir at target path
- Same-profile overlay onto folder-mode target = planning error with clear message
- Uninstall removes folder symlinks via `os.Remove` (removes link, not target contents)
- Switch correctly handles folder-mode → file-mode and vice versa across profiles

### Must NOT Have (Guardrails)
- NO recursive directory removal (`os.RemoveAll`) under any circumstance
- NO destructive operations on existing user data
- NO change to how file-mode works (same Ops, same conflicts, same state format aside from new optional field defaulting to false)
- NO new flags on CLI commands
- NO change to `$HOME`/state path resolution
- NO support for deeply nested folder-mode declarations (e.g., glob patterns) — that's a future feature
- NO modification to opencode/, .sisyphus/, hyprland/, waybar/, wofi/, ghostty/, zsh/

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — all verification agent-executable.

### Test Decision
- **Infrastructure exists**: YES (Go testing, testify, fixtures in testdata/)
- **Automated tests**: TDD-leaning (write tests as part of each task) + tests-after for integration
- **Framework**: `go test -race`

### QA Policy
After each implementation wave, the executor must run `go test -race ./...` and `go build ./cli`. Final manual repro:
1. Build binary: `go build -o /tmp/rice ./cli`
2. tmphome: `TH=$(mktemp -d) && HOME=$TH /tmp/rice install nvim --profile common --repo . --state $TH/state.json --yes`
3. Assert: `test -L $TH/.config/nvim` (single symlink), `readlink $TH/.config/nvim` points back into `<repo>/nvim/...`
4. `HOME=$TH /tmp/rice uninstall nvim --state $TH/state.json --yes`
5. Assert: `test ! -e $TH/.config/nvim`

---

## Execution Strategy — Parallel Waves

```
Wave 1 (foundation — sequential within wave, T1 blocks T2 blocks T3):
├── T1: Schema — Sources []SourceSpec with custom UnmarshalTOML       [unspecified-high]
├── T2: Plan/Op extension — Op.IsDir + Conflict.IsDir                 [quick]
└── T3: State — InstalledLink.IsDir                                    [quick]

Wave 2 (core logic — parallel after Wave 1):
├── T4: Walker — folder-mode emits single Op per source               [unspecified-high]
├── T5: Conflict detection — handle existing dir at target            [unspecified-high]
└── T6: Overlay validation — reject same-profile folder-mode collision [quick]

Wave 3 (integration — parallel after Wave 2):
├── T7: Executor — folder symlink creation path                       [quick]
├── T8: Uninstall — folder symlink removal (validates it's a symlink) [quick]
└── T9: Renderer — CREATE-DIR vs CREATE                               [quick]

Wave 4 (tests — parallel):
├── T10: Manifest schema tests (parsing both forms, validation)        [quick]
├── T11: Walker + conflict tests (folder-mode happy + overlay reject)  [unspecified-high]
└── T12: End-to-end install/uninstall/switch tests for folder-mode    [unspecified-high]

Wave 5 (migration + docs + verify — sequential):
├── T13: Migrate nvim/rice.toml + AGENTS.md schema section + README   [quick]
└── T14: Final verify (build/vet/gofmt/test/manual repro) + commit    [quick]

Critical Path: T1 → T2 → T4 → T5 → T7 → T11 → T13 → T14
Max Concurrent: 3 (Waves 2, 3, 4)
```

---

## TODOs

- [x] 1. Extend manifest schema to accept string OR table source entries

  **What to do**:
  - Edit `internal/manifest/schema.go`:
    - Change `ProfileDef.Sources` from `[]string` to `[]SourceSpec`.
    - Define `SourceSpec` struct:
      ```go
      type SourceSpec struct {
          Path   string `toml:"path"`
          Mode   string `toml:"mode"`   // "" or "file" (default) or "folder"
          Target string `toml:"target"` // required when Mode == "folder", relative to package-level Target
      }
      ```
    - Implement `UnmarshalTOML(interface{}) error` on `SourceSpec` so it accepts:
      - bare string → `{Path: <string>, Mode: "file"}`
      - table → all fields populated
    - Use `BurntSushi/toml`'s `UnmarshalerText` or the v2 `Unmarshaler` interface — check what's already in go.mod (BurntSushi/toml v1.3 supports `UnmarshalTOML(interface{}) error` via `toml.Unmarshaler`).
  - Add a new validation step in `manifest.Validate` (find existing validate function — likely in same package):
    - For each profile's sources: if `Mode == "folder"`, `Target` must be non-empty
    - If `Mode == ""`, treat as `"file"`
    - Reject any other value of Mode

  **References**:
  - `internal/manifest/schema.go:14-17` — current ProfileDef
  - Find `func Validate` in `internal/manifest/` and extend it
  - go.mod will confirm BurntSushi/toml version

  **Acceptance Criteria**:
  - [ ] Both forms parse correctly via unit tests (T10)
  - [ ] Backward compat: existing string-only manifests parse with `Mode: "file"`
  - [ ] Validation: folder-mode without Target → returns clear error

  **QA Scenarios**:
  ```
  Scenario: Existing string-form manifests still parse (backward compat)
    Tool: Bash
    Preconditions: T1 implemented; nvim/rice.toml NOT yet migrated (still uses sources = ["."])
    Steps:
      1. go build ./...
      2. go test -race ./internal/manifest/...
    Expected Result: Both commands exit 0; no test failures
    Evidence: .sisyphus/evidence/task-1-parse-backcompat.txt

  Scenario: Folder-mode without target field is rejected at validation time
    Tool: Bash (test will be added in T10, this scenario verifies validation hook fires)
    Preconditions: T1 implemented
    Steps:
      1. Create temp manifest at /tmp/bad.toml with: schema_version=1, name="x", supported_os=["darwin"], target="$HOME", [profiles.common] sources=[{path="a", mode="folder"}]
      2. Run a tiny Go snippet OR rely on T10 unit test that calls manifest.Load + Validate on this file
      3. Expect non-nil error mentioning "target" and "folder"
    Expected Result: Validation error returned with clear message
    Evidence: .sisyphus/evidence/task-1-validation-error.txt
  ```

- [x] 2. Extend plan.Op and plan.Conflict with IsDir field

  **What to do**:
  - Edit `internal/plan/plan.go`:
    - Add `IsDir bool` to `Op` (default false = file-mode link)
    - Add `IsDir bool` to `Conflict` (so renderer/executor know the conflict scope)
  - Update any zero-value tests that compare Op structs (if they break, add `IsDir: false` explicitly)

  **References**:
  - `internal/plan/plan.go` — current Op and Conflict structs

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds after change
  - [ ] Existing tests pass (default false matches old behavior)

  **QA Scenarios**:
  ```
  Scenario: Build + existing tests stay green after Op/Conflict gain IsDir
    Tool: Bash
    Preconditions: T2 implemented
    Steps:
      1. go build ./...
      2. go test -race ./internal/plan/... ./internal/installer/...
    Expected Result: All commands exit 0
    Evidence: .sisyphus/evidence/task-2-build-and-tests.txt
  ```

- [x] 3. Extend state.InstalledLink with IsDir field

  **What to do**:
  - Edit `internal/state/state.go`:
    - Add `IsDir bool \`json:"is_dir,omitempty"\`` to `InstalledLink`
    - The `omitempty` ensures old state.json files (no is_dir) parse as `false` and new state.json files don't bloat with `"is_dir": false` everywhere

  **References**:
  - `internal/state/state.go:11-15` — current InstalledLink

  **Acceptance Criteria**:
  - [ ] Old state.json (without is_dir) loads cleanly
  - [ ] New state.json with is_dir=true serializes/parses correctly

  **QA Scenarios**:
  ```
  Scenario: Old state.json without is_dir loads with IsDir=false default
    Tool: Bash
    Preconditions: T3 implemented
    Steps:
      1. Write fixture /tmp/old-state.json: {"pkg":{"profile":"p","installed_links":[{"source":"/s","target":"/t"}],"installed_at":"2025-01-01T00:00:00Z"}}
      2. Run `go test -race ./internal/state/...` (T10 will add a test that loads this fixture and asserts IsDir==false)
    Expected Result: Test exits 0; no JSON unmarshal error
    Evidence: .sisyphus/evidence/task-3-state-backcompat.txt

  Scenario: New state.json roundtrips with is_dir=true
    Tool: Bash
    Preconditions: T3 implemented
    Steps:
      1. go test -race ./internal/state/... -run TestInstalledLink_IsDirRoundtrip
    Expected Result: Test passes; serialized JSON contains "is_dir":true
    Evidence: .sisyphus/evidence/task-3-state-roundtrip.txt
  ```

- [x] 4. Walker: folder-mode emits ONE Op per source dir

  **What to do**:
  - Edit `internal/installer/install.go`:
    - In `BuildInstallPlan`, after resolving sources, for each `SourceSpec`:
      - If `spec.Mode == "folder"`:
        - Verify source dir exists (`os.Stat`, must be directory)
        - Compute `target = filepath.Join(targetRoot, spec.Target)` where `targetRoot` is the expanded package-level `target` (usually `$HOME`)
        - Verify `withinHome(target, req.HomeDir)` (defense-in-depth, same as today)
        - Compute `absSource` of `<repo>/<pkg>/<spec.Path>`
        - Emit ONE pendingOp `{Source: absSource, Target: target, IsDir: true}`
        - Do NOT walk into the source dir
      - If `spec.Mode == "file"` (or empty): use the existing `filepath.WalkDir` logic unchanged
  - Update the `pendingOp` struct (defined locally in install.go) to include `IsDir bool`
  - Update the loop that converts ops → `plan.Op` (around line 202) to copy `IsDir`

  **References**:
  - `internal/installer/install.go:136-188` — current walker
  - `internal/installer/install.go:198-208` — Op construction

  **Acceptance Criteria**:
  - [ ] Folder-mode source produces exactly 1 Op (verified by test T11)
  - [ ] File-mode behavior unchanged (existing tests still pass)

  **QA Scenarios**:
  ```
  Scenario: Folder-mode source emits exactly one Op
    Tool: Bash
    Preconditions: T4 implemented; T11 fixture testdata/install/folder-pkg/ exists with rice.toml using folder-mode
    Steps:
      1. go test -race ./internal/installer/... -run TestBuildInstallPlan_FolderMode
    Expected Result: Test passes; assertion confirms len(plan.Ops)==1, plan.Ops[0].IsDir==true, target path correct
    Evidence: .sisyphus/evidence/task-4-folder-single-op.txt

  Scenario: File-mode behavior preserved (regression guard)
    Tool: Bash
    Preconditions: T4 implemented
    Steps:
      1. go test -race ./internal/installer/... -run TestBuildInstallPlan
    Expected Result: All pre-existing file-mode tests pass unchanged
    Evidence: .sisyphus/evidence/task-4-filemode-regression.txt
  ```

- [x] 5. Conflict detection: handle existing dir at folder-mode target

  **What to do**:
  - Edit `internal/installer/conflict.go`:
    - Update `DetectConflicts` to handle Op with `IsDir: true`:
      - If target does NOT exist → no conflict
      - If target IS a symlink → use existing `IsSymlinkTo` check; conflict only if it doesn't point to our source
      - If target IS a regular directory (not a symlink) → ALWAYS conflict, reason: "target directory exists (folder-mode requires dir to be absent or our symlink)"
      - If target is a regular file (not a directory or symlink) → conflict, reason: "target exists as file but folder-mode expects directory or our symlink"
    - For file-mode (`IsDir: false`): existing logic unchanged
  - The output `Conflict` should set `IsDir` matching the planned op so renderer can label it

  **References**:
  - `internal/installer/conflict.go` — current DetectConflicts

  **Acceptance Criteria**:
  - [ ] Folder-mode + non-existent target → planned, no conflict
  - [ ] Folder-mode + existing dir → conflict reported (test T11)
  - [ ] Folder-mode + our existing symlink → planned, no conflict (idempotent re-install)
  - [ ] File-mode behavior unchanged

  **QA Scenarios**:
  ```
  Scenario: Conflict matrix for folder-mode targets
    Tool: Bash
    Preconditions: T5 implemented; T11 tests added
    Steps:
      1. go test -race ./internal/installer/... -run TestDetectConflicts_FolderMode
    Expected Result: Test passes; covers 4 cases (absent → no conflict; existing dir → conflict; our symlink → no conflict; foreign symlink → conflict)
    Evidence: .sisyphus/evidence/task-5-conflict-matrix.txt

  Scenario: File-mode conflict logic unchanged
    Tool: Bash
    Preconditions: T5 implemented
    Steps:
      1. go test -race ./internal/installer/... -run TestDetectConflicts
    Expected Result: All pre-existing conflict tests pass
    Evidence: .sisyphus/evidence/task-5-filemode-regression.txt
  ```

- [x] 6. Overlay validation: same-profile folder-mode collisions are errors

  **What to do**:
  - In the walker (T4), after enumerating all Ops across all sources for a profile:
    - For each Op with `IsDir: true`, check if any OTHER Op (from earlier source in the same plan) targets the same path OR a path that is a parent/descendant of this folder-mode target
    - Concretely: for folder-mode op with target `X`, reject if any other planned op has target == X, or target startswith X+"/", or X startswith other_target+"/"
    - Return a planning error with both source paths and the conflicting target
  - This is a manifest-level error (planning), distinct from runtime conflicts

  **References**:
  - Walker code from T4

  **Acceptance Criteria**:
  - [ ] Two folder-mode sources targeting same dir → error
  - [ ] One folder-mode source + one file-mode op overlapping into that dir → error
  - [ ] Two file-mode sources overlapping (existing behavior, last wins) → still works (no error)

  **QA Scenarios**:
  ```
  Scenario: Overlay validation rejects folder-mode collisions
    Tool: Bash
    Preconditions: T6 implemented; T11 fixtures created (folder-pkg-overlay/ with two folder-mode sources hitting same target; folder-pkg-mixed/ with folder-mode + file-mode overlap)
    Steps:
      1. go test -race ./internal/installer/... -run TestBuildInstallPlan_OverlayRejection
    Expected Result: Test passes; both rejection cases produce error mentioning both source paths and the conflicting target
    Evidence: .sisyphus/evidence/task-6-overlay-rejection.txt

  Scenario: File-mode overlay (existing "last wins") still works
    Tool: Bash
    Preconditions: T6 implemented
    Steps:
      1. go test -race ./internal/installer/... -run TestBuildInstallPlan_FileModeOverlay
    Expected Result: Pre-existing overlay test passes (no regression)
    Evidence: .sisyphus/evidence/task-6-filemode-overlay.txt
  ```

- [x] 7. Executor: folder symlink creation

  **What to do**:
  - Edit `internal/installer/install.go` `ExecuteInstallPlan`:
    - For each Op: if `op.IsDir`, ensure parent dir of target exists (`os.MkdirAll(filepath.Dir(target), 0o755)`), then call `symlink.CreateSymlink(op.Source, op.Target)` — same call as file-mode (Go's `os.Symlink` doesn't care about target type)
    - When recording to state: set `InstalledLink.IsDir = op.IsDir`
  - Edit `internal/symlink/symlink.go` if needed: `CreateSymlink` likely needs no change (works for dirs too), but add a comment confirming this and add a test (T11 covers it)

  **References**:
  - `internal/installer/install.go` `ExecuteInstallPlan` (read it to find correct insertion point)
  - `internal/symlink/symlink.go` `CreateSymlink`

  **Acceptance Criteria**:
  - [ ] After install of folder-mode source, target is a symlink (verified via `os.Lstat` mode bits)
  - [ ] Target's `Readlink` returns the absolute source path
  - [ ] State.json records `is_dir: true` for that link

  **QA Scenarios**:
  ```
  Scenario: Folder-mode install creates one symlink + state records IsDir
    Tool: Bash
    Preconditions: T7 implemented; T12 e2e test added
    Steps:
      1. go test -race ./cli/... -run TestInstall_FolderMode_CreatesSingleSymlink
    Expected Result: Test passes; asserts os.Lstat shows symlink mode bit, readlink == abs source path, state.json contains "is_dir":true
    Evidence: .sisyphus/evidence/task-7-folder-install.txt
  ```

- [x] 8. Uninstall: handle folder symlinks correctly

  **What to do**:
  - Edit `internal/installer/uninstall.go`:
    - Removal logic uses `os.Remove` which handles symlinks (file or dir) correctly — no change needed
    - BUT: add safety check before removal: if `link.IsDir` is true, verify the target IS still a symlink (not converted to a real dir). Use `os.Lstat` + check mode bits. If not a symlink, log warning and skip (do NOT recursively delete a real directory)
  - Make this safety check apply to file-mode too (defense in depth — current code may already do this; verify)

  **References**:
  - `internal/installer/uninstall.go`
  - `internal/symlink/symlink.go`

  **Acceptance Criteria**:
  - [ ] Folder symlink uninstalls cleanly (target absent after)
  - [ ] If user replaced symlink with real dir, uninstall warns + skips (does NOT delete real dir)

  **QA Scenarios**:
  ```
  Scenario: Folder symlink uninstalls cleanly
    Tool: Bash
    Preconditions: T8 implemented; T12 e2e test added
    Steps:
      1. go test -race ./cli/... -run TestUninstall_FolderMode_RemovesSymlinkOnly
    Expected Result: Test passes; asserts target gone after uninstall AND original source dir contents intact
    Evidence: .sisyphus/evidence/task-8-folder-uninstall.txt

  Scenario: Refuse to delete real dir if symlink was replaced
    Tool: Bash
    Preconditions: T8 implemented; T12 e2e test added
    Steps:
      1. go test -race ./cli/... -run TestUninstall_FolderMode_SkipsWhenReplacedWithDir
    Expected Result: Test passes; asserts after uninstall the (now-real) dir still exists with its contents AND a warning was emitted
    Evidence: .sisyphus/evidence/task-8-skip-real-dir.txt
  ```

- [x] 9. Renderer: distinguish CREATE-DIR vs CREATE

  **What to do**:
  - Edit `internal/prompt/prompt.go`:
    - In `RenderPlan` (and `RenderSwitchPlan`): when iterating Ops, if `op.IsDir`, print `CREATE-DIR` instead of `CREATE`. Same for `REMOVE-DIR` vs `REMOVE`.
    - In `RenderConflicts`: if `c.IsDir`, label as `CONFLICT-DIR` to make scope clear (or include " (directory)" suffix)

  **References**:
  - `internal/prompt/prompt.go:54-64` (CREATE/REMOVE rendering)
  - `internal/prompt/prompt.go:109-113` (RenderConflicts)

  **Acceptance Criteria**:
  - [ ] Folder-mode plan output contains "CREATE-DIR"
  - [ ] File-mode plan output unchanged ("CREATE")

  **QA Scenarios**:
  ```
  Scenario: Renderer labels folder-mode ops distinctly
    Tool: Bash
    Preconditions: T9 implemented
    Steps:
      1. go test -race ./internal/prompt/... -run TestRenderPlan_FolderMode
    Expected Result: Test passes; output contains "CREATE-DIR" for folder ops and "CREATE" for file ops; existing snapshot tests unchanged
    Evidence: .sisyphus/evidence/task-9-render.txt
  ```

- [x] 10. Manifest schema tests

  **What to do**:
  - Edit/create `internal/manifest/schema_test.go` (or whichever existing test file in the package):
    - Test: parse manifest with all-string sources → `Mode == "file"` for each
    - Test: parse manifest with mixed string + table sources → correct Mode values
    - Test: parse manifest with table source missing `target` for folder-mode → validation error with clear message
    - Test: parse manifest with invalid `mode` value → validation error
    - Test: parse manifest with table source `mode = "file"` and `target` set → ignored or rejected (decide; recommend reject for clarity)

  **Acceptance Criteria**:
  - [ ] All 5 test cases pass

  **QA Scenarios**:
  ```
  Scenario: Manifest schema test suite passes
    Tool: Bash
    Preconditions: T10 implemented
    Steps:
      1. go test -race ./internal/manifest/... -v
    Expected Result: All test cases (string-only parse, mixed parse, missing target → error, invalid mode → error, file-mode + target combo → reject) PASS; verbose output names each
    Evidence: .sisyphus/evidence/task-10-manifest-tests.txt
  ```

- [x] 11. Walker + conflict tests

  **What to do**:
  - Add tests in `internal/installer/install_test.go` (or appropriate _test.go):
    - Test: folder-mode source produces exactly 1 Op with `IsDir: true` and correct target
    - Test: folder-mode + existing target dir → conflict detected
    - Test: folder-mode + our existing symlink at target → no conflict (idempotent)
    - Test: folder-mode + non-existent target → no conflict
    - Test: 2 folder-mode sources targeting same path in same profile → planning error
    - Test: file-mode source overlapping into folder-mode target → planning error
  - Use new fixture under `testdata/install/` if needed — `mypkg-folder/` with `common-cfg/init.lua` etc.

  **Acceptance Criteria**:
  - [ ] All tests pass with `-race`

  **QA Scenarios**:
  ```
  Scenario: Walker + conflict test suite passes
    Tool: Bash
    Preconditions: T11 implemented; new fixture testdata/install/folder-pkg/ created
    Steps:
      1. go test -race ./internal/installer/... -v
    Expected Result: All 6 test cases pass (single Op emit; 4 conflict cases; 2 overlay rejection cases)
    Evidence: .sisyphus/evidence/task-11-installer-tests.txt
  ```

- [x] 12. End-to-end install/uninstall/switch tests for folder-mode

  **What to do**:
  - Add tests in `cli/install_test.go`, `cli/uninstall_test.go`, `cli/switch_test.go`:
    - `TestInstall_FolderMode_CreatesSingleSymlink` — install via cobra, assert target is single symlink, state.json has `is_dir: true`
    - `TestUninstall_FolderMode_RemovesSymlinkOnly` — install then uninstall, assert symlink gone, source dir untouched
    - `TestSwitch_FolderModeToFileMode` — profile A folder-mode, profile B file-mode targeting overlapping path; switch should succeed (uninstall removes folder symlink, install creates file tree)
    - `TestUninstall_FolderMode_SkipsWhenReplacedWithDir` — install, replace symlink with real dir, uninstall warns + skips (no destructive removal)
  - Reuse existing test helpers (`setupTestRepo`, `runInstallCmd`)

  **Acceptance Criteria**:
  - [ ] All 4 tests pass

  **QA Scenarios**:
  ```
  Scenario: End-to-end CLI suite passes
    Tool: Bash
    Preconditions: T12 implemented
    Steps:
      1. go test -race ./cli/... -v -run "FolderMode"
    Expected Result: All 4 tests pass: TestInstall_FolderMode_CreatesSingleSymlink, TestUninstall_FolderMode_RemovesSymlinkOnly, TestSwitch_FolderModeToFileMode, TestUninstall_FolderMode_SkipsWhenReplacedWithDir
    Evidence: .sisyphus/evidence/task-12-cli-e2e.txt
  ```

- [x] 13. Migrate nvim package + update docs

  **Current nvim layout (verified)**:
  - `nvim/rice.toml`: `sources = ["."]`, `target = "$HOME"`
  - `nvim/.config/nvim/` contains the actual nvim config files
  - File-mode today walks `nvim/` and creates per-file symlinks under `~/.config/nvim/`

  **What to do**:
  - Edit `nvim/rice.toml` to use folder-mode pointing at the existing `.config/nvim/` directory:
    ```toml
    [profiles.common]
    sources = [{ path = ".config/nvim", mode = "folder", target = ".config/nvim" }]
    ```
    This means: take the directory at `<repo>/nvim/.config/nvim/` and symlink it to `<HOME>/.config/nvim`.
  - Update `AGENTS.md` "rice.toml Schema" section (around the schema fields table and example):
    - Document that `sources` accepts both string entries (file-mode, current behavior) and table entries `{path = "...", mode = "folder", target = "..."}` (folder-mode)
    - Add a "Folder-mode sources" subsection explaining: when to use it (tools that resolve their own config root), the requirement that `target` be set, and the constraint that folder-mode targets cannot be overlaid by other sources
    - Show the migrated nvim example as the canonical illustration
  - Update README.md ONLY if it currently shows a sources example — quick `grep -n "sources" README.md` to check; skip if no match

  **QA Scenarios**:
  ```
  Scenario: Migrated nvim manifest parses + installs as single symlink
    Tool: Bash
    Preconditions: T1-T9 complete, gofmt/build/vet clean, T10-T12 tests passing
    Steps:
      1. go build -o /tmp/rice ./cli
      2. TH=$(mktemp -d)
      3. HOME=$TH /tmp/rice install nvim --profile common --repo . --state $TH/state.json --yes
      4. test -L $TH/.config/nvim
      5. readlink $TH/.config/nvim  # must equal absolute path to <repo>/nvim/.config/nvim
      6. ls $TH/.config/nvim/  # must list nvim config files (proves symlink resolves)
    Expected Result: Step 4 exit 0, step 5 prints absolute repo path, step 6 lists actual nvim files
    Failure Indicators: Step 4 fails (target is dir not symlink → migration created file-mode tree); step 5 prints relative or wrong path; step 6 empty (broken symlink)
    Evidence: .sisyphus/evidence/task-13-nvim-install.txt (output of all 6 steps)

  Scenario: AGENTS.md documents both source forms
    Tool: Bash (grep)
    Preconditions: AGENTS.md updated
    Steps:
      1. grep -q 'mode = "folder"' AGENTS.md
      2. grep -q '"path"' AGENTS.md
      3. grep -qi 'folder-mode' AGENTS.md
    Expected Result: All three greps exit 0
    Evidence: .sisyphus/evidence/task-13-docs-grep.txt
  ```

  **References**:
  - `nvim/rice.toml` (current: `sources = ["."]`)
  - `nvim/.config/nvim/` (the dir to be symlinked as a unit)
  - `AGENTS.md` schema section (lines ~50-90 — fields table and example)
  - `README.md` (search for "sources" to determine if update needed)

  **Acceptance Criteria**:
  - [ ] nvim/rice.toml migrated; both QA scenarios pass
  - [ ] AGENTS.md schema section documents both string and table forms with example

- [x] 14. Final verify + commit

  **What to do**:
  ```bash
  cd /Users/guneet/rice
  gofmt -w internal/ cli/
  go build ./cli
  go vet ./...
  go test ./... -race
  # Manual repro
  go build -o /tmp/rice ./cli
  TH=$(mktemp -d)
  HOME=$TH /tmp/rice install <folder-mode-pkg> --profile <p> --repo . --state $TH/state.json --yes
  test -L $TH/<expected-target>  # must succeed
  readlink $TH/<expected-target>  # must point into repo
  HOME=$TH /tmp/rice uninstall <folder-mode-pkg> --state $TH/state.json --yes
  test ! -e $TH/<expected-target>  # must succeed
  ```
  - Commit (single squashed commit OR per-wave; recommend single):
    `feat(symlink): add folder-mode option for symlinking entire source dirs as single symlinks`

  **Acceptance Criteria**:
  - [ ] All verification commands pass
  - [ ] Manual repro succeeds (single symlink created, removed cleanly)
  - [ ] Commit SHA reported

  **QA Scenarios**:
  ```
  Scenario: Full verification gate passes
    Tool: Bash
    Preconditions: T1-T13 complete
    Steps:
      1. cd /Users/guneet/rice
      2. gofmt -w internal/ cli/
      3. go build ./cli   # exit 0
      4. go vet ./...     # exit 0, no output
      5. go test ./... -race   # all pass
      6. gofmt -l internal/ cli/   # empty output
    Expected Result: Steps 3-6 all exit 0; step 6 prints nothing
    Evidence: .sisyphus/evidence/task-14-verify-gate.txt (capture all 6 outputs)

  Scenario: End-to-end manual repro on real nvim package
    Tool: Bash
    Preconditions: T13 migrated nvim/rice.toml to folder-mode
    Steps:
      1. go build -o /tmp/rice-folder ./cli
      2. TH=$(mktemp -d)
      3. HOME=$TH /tmp/rice-folder install nvim --profile common --repo /Users/guneet/rice --state $TH/state.json --yes
      4. test -L $TH/.config/nvim   # exit 0
      5. LINK=$(readlink $TH/.config/nvim) && [ "$LINK" = "/Users/guneet/rice/nvim/.config/nvim" ]   # exit 0
      6. ls $TH/.config/nvim/ | head -1   # non-empty
      7. HOME=$TH /tmp/rice-folder uninstall nvim --state $TH/state.json --yes
      8. test ! -e $TH/.config/nvim   # exit 0
      9. test -d /Users/guneet/rice/nvim/.config/nvim   # source still intact, exit 0
    Expected Result: All steps exit 0
    Failure Indicators: Step 4 fails (not a symlink → migration broken); step 5 mismatch (wrong target); step 8 fails (uninstall didn't remove); step 9 fails (uninstall destroyed source — CRITICAL bug)
    Evidence: .sisyphus/evidence/task-14-manual-repro.txt

  Scenario: Commit created with correct message
    Tool: Bash
    Preconditions: All previous scenarios passed
    Steps:
      1. git log -1 --pretty=format:'%s'
    Expected Result: Output matches `feat(symlink): add folder-mode option for symlinking entire source dirs`
    Evidence: .sisyphus/evidence/task-14-commit.txt (include git log -1 --stat)
  ```

---

## Commit Strategy

- **Recommended**: Single commit covering all waves once everything is green
- Message: `feat(symlink): add folder-mode option for symlinking entire source dirs`
- Files: schema.go, plan.go, state.go, install.go, conflict.go, uninstall.go, prompt.go, manifest_test.go, install_test.go, uninstall_test.go, switch_test.go, AGENTS.md, optionally nvim/rice.toml
- Pre-commit: `go test ./... -race && go vet ./... && gofmt -l . | grep -vE '(opencode|.sisyphus)/' | (! grep .)`

---

## Success Criteria

### Verification Commands
```bash
go build ./cli                          # succeeds
go vet ./...                            # clean
go test ./... -race                     # all pass (existing + 8+ new)
gofmt -l internal/ cli/                 # no output

# Backward compat
go test ./internal/manifest/... -run TestParse_StringSources  # existing manifests still parse

# Folder-mode end-to-end
go build -o /tmp/rice ./cli
TH=$(mktemp -d)
HOME=$TH /tmp/rice install <pkg> --profile <p> --repo . --state $TH/state.json --yes
[ -L $TH/<target-path> ] && echo "is symlink"           # PASS
[ "$(readlink $TH/<target>)" = "<expected>" ] && echo OK # PASS
HOME=$TH /tmp/rice uninstall <pkg> --state $TH/state.json --yes
[ ! -e $TH/<target-path> ] && echo "removed"            # PASS
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent (no os.RemoveAll, no behavior change for file-mode, no destructive ops)
- [ ] All tests pass with -race
- [ ] Backward compatible (existing rice.toml files unchanged behavior)
- [ ] Single commit with the specified message
