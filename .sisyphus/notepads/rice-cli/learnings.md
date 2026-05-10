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
