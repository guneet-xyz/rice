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
