# Rice CLI — Architectural Decisions

## Wave 1 (Foundation — parallel, no deps)
- T1: Go module init + scaffolding (deps: cobra, toml, testify, zap)
- T2: rice.toml schema structs + validation
- T3: Cross-platform symlink primitives
- T4: State file format + read/write
- T5: Profile resolution rules
- T6: Delete stale dirs (profiles/, scripts/)
- T24: Logger package (zap, 5 levels, custom CRITICAL)

## Wave 2 (Core — needs Wave 1)
- T7: Manifest discovery + parsing (needs T2)
- T8: OS gating (needs T2, T7)
- T9: Conflict detection (needs T3)
- T25: Plan + confirmation prompt utility (needs T24)
- T10: Install orchestrator (needs T3,4,7,8,9,24,25)
- T11: Uninstall orchestrator (needs T3,4,24,25)
- T12: Switch orchestrator (needs T10,T11)

## Wave 3 (CLI — needs Wave 2)
- T13: Root cmd + cobra + --log-level + --yes flags (needs T1,T24)
- T14: install command (needs T10,T13,T25)
- T15: uninstall command (needs T11,T13,T25)
- T16: switch command (needs T12,T13,T25)
- T17: status command (needs T4,T13)
- T18: doctor command (needs T4,T7,T13)

## Wave 4 (Migration + Docs — needs Wave 3)
- T19: rice.toml for nvim/zsh/hyprland/waybar/wofi
- T20: Migrate ghostty (delete install.sh, add rice.toml)
- T21: Split opencode personal/work
- T22: AGENTS.md
- T23: README.md rewrite
