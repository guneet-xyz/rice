---
name: vault-init
description: One-time setup for opencode-vault — clones a user's vault repo as a bare git repo and creates the main worktree for reading and writing notes.
---

# Vault Init

One-time setup skill for the opencode-vault system. This creates the local vault infrastructure that the other vault skills (vault-search, vault-save, vault-cleanup) depend on.

## Directory Layout

After initialization, the vault is structured as:

```
~/.local/share/opencode/vault/
├── vault.git/              # bare clone of the user's vault repo
├── branches/
│   └── main/               # permanent worktree tracking main branch
└── .merge-lock/            # (created transiently by vault-save/cleanup during merges)
```

## Workflow

### 1. Check for Existing Vault

Check if `~/.local/share/opencode/vault/vault.git` exists.

- If it exists and is a valid bare git repo: warn the user that a vault is already initialized. Ask if they want to re-initialize (which will delete and recreate everything).
- If it does not exist: proceed with setup.

### 2. Ask for Vault Repo URL

Ask the user for their vault git repository URL. This should be a remote git repo (GitHub, GitLab, etc.) that stores their vault notes.

### 3. Clone as Bare Repo

```bash
mkdir -p ~/.local/share/opencode/vault
git clone --bare <repo-url> ~/.local/share/opencode/vault/vault.git
```

### 4. Create Main Worktree

```bash
mkdir -p ~/.local/share/opencode/vault/branches
git -C ~/.local/share/opencode/vault/vault.git worktree add ~/.local/share/opencode/vault/branches/main main
```

If the remote repo is empty (no main branch), create the initial branch:

```bash
git -C ~/.local/share/opencode/vault/vault.git branch main
git -C ~/.local/share/opencode/vault/vault.git worktree add ~/.local/share/opencode/vault/branches/main main
```

### 5. Scaffold Empty Vault (if repo is empty)

If the repo has no commits or no `README.md`, create the initial vault structure in the main worktree:

**README.md**:
```markdown
# Vault

AI-managed knowledge vault. Notes are organized in folders with markdown files.

## Structure

Notes are stored as markdown files with YAML frontmatter. See INDEX.md for a full catalog.

## Conventions

- One topic per note
- Use [[wikilinks]] for cross-references
- All notes must have valid frontmatter (title, tags, type)
```

**INDEX.md**:
```markdown
# Index

Complete catalog of all notes in this vault.

<!-- This file is auto-generated from note frontmatter. Do not edit manually. -->
```

Commit and push:
```bash
cd ~/.local/share/opencode/vault/branches/main
git add -A
```

**Secret scan** before committing:

1. Check if gitleaks is available: `command -v gitleaks`
   - If not installed, warn the user: "gitleaks is not installed. Skipping secret scan. Install with `brew install gitleaks` (macOS) or see https://github.com/gitleaks/gitleaks#installing for other platforms." Then proceed with the commit.
2. If available, run: `gitleaks detect --staged --no-banner`
3. If secrets are detected: **abort the commit**. Show the gitleaks output and ask the user to remove the secrets before proceeding.
4. If no secrets detected: proceed.

```bash
git commit -m "vault: initialize vault"
git push origin main
```

### 6. Verify

Read `INDEX.md` and `README.md` from `~/.local/share/opencode/vault/branches/main` to confirm the vault is accessible.

Report success to the user with the vault location.

## Re-initialization

If the user wants to re-initialize:

1. Remove all worktrees: `git -C ~/.local/share/opencode/vault/vault.git worktree list` and remove each one
2. Remove the entire vault directory: `rm -rf ~/.local/share/opencode/vault`
3. Start from step 3 above

## Important Rules

- **NEVER** store secrets, credentials, or API keys in the vault.
- **NEVER** force push.
- This skill only runs once per machine. The other vault skills handle ongoing operations.
