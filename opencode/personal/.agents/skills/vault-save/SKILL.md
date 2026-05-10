---
name: vault-save
description: Save notes to an opencode-vault — create or update notes with git worktree isolation for safe concurrent access across multiple sessions.
---

# Vault Save

Skill for creating and updating notes in the opencode-vault. Uses git worktree isolation so multiple sessions can save concurrently without conflicts.

## Vault Location

```
~/.local/share/opencode/vault/
├── vault.git/              # bare clone
├── branches/
│   ├── main/               # permanent worktree (for reference reads)
│   └── <session-branch>/   # your isolated session worktree
```

## Boot Check

Before any operation:

1. Verify `~/.local/share/opencode/vault/vault.git` exists and is a bare git repo.
2. Verify `~/.local/share/opencode/vault/branches/main` exists.
3. If either is missing, tell the user: "Vault not found. Please run the vault-init skill to set up your vault."

## Workflow

### 1. Fetch Latest

```bash
git -C ~/.local/share/opencode/vault/vault.git fetch origin
```

### 2. Create Session Worktree

Choose a short, human-readable branch name that describes the session's topic (e.g., `react-auth-research`, `kubernetes-networking`, `python-testing-patterns`). No dates — keep it concise.

```bash
git -C ~/.local/share/opencode/vault/vault.git worktree add \
  ~/.local/share/opencode/vault/branches/<branch-name> \
  -b <branch-name> origin/main
```

If the branch name already exists (e.g., from a crashed session), clean it up first:

```bash
git -C ~/.local/share/opencode/vault/vault.git worktree remove \
  ~/.local/share/opencode/vault/branches/<branch-name> --force
git -C ~/.local/share/opencode/vault/vault.git branch -D <branch-name>
```

Then retry the worktree creation.

All subsequent file operations happen in `~/.local/share/opencode/vault/branches/<branch-name>/`.

### 3. Read Vault Context

From the session worktree:

1. Read `README.md` for vault structure and conventions.
2. Read `INDEX.md` for the full note catalog.

### 4. Duplicate Check

Before creating a new note, search `INDEX.md` for notes with similar titles, tags, or topics.

- If a closely related note exists, **update it** instead of creating a new one.
- If a loosely related note exists, consider whether the new content belongs there or warrants its own note.

### 5. Create or Update Notes

**Creating a new note:**

1. Determine the correct folder from the vault's existing structure.
2. Create the file with required frontmatter:

```yaml
---
title: <Note Title>
tags: [<tag1>, <tag2>]
type: <note|template|project|canvas>
---
```

3. Write content in standard markdown. Use `[[wikilinks]]` to reference other vault notes.

**Updating an existing note:**

1. Read the current note content.
2. Make changes.

**Tag discipline:** before inventing a new tag, check what tags already exist in `INDEX.md`. Reuse existing tags for consistency. Only create new tags when no existing tag fits.

### 6. Regenerate INDEX.md

After all note changes are complete, regenerate `INDEX.md` entirely from note frontmatter:

1. Glob all `.md` files in the session worktree (excluding `INDEX.md` and `README.md`).
2. Read the frontmatter of each note.
3. Produce a markdown table grouped by folder, with columns: Title (as `[[wikilink]]`), Tags, Type, Summary (5-15 words).
4. Write the regenerated content to `INDEX.md`, preserving the header:

```markdown
# Index

Complete catalog of all notes in this vault.

<!-- This file is auto-generated from note frontmatter. Do not edit manually. -->

## <Folder Name>

| Title | Tags | Type | Summary |
|-------|------|------|---------|
| [[note-title]] | tag1, tag2 | note | Brief summary of the note |
```

### 7. Secret Scan and Commit

```bash
cd ~/.local/share/opencode/vault/branches/<branch-name>
git add -A
```

**Secret scan** before committing:

1. Check if gitleaks is available: `command -v gitleaks`
   - If not installed, warn the user: "gitleaks is not installed. Skipping secret scan. Install with `brew install gitleaks` (macOS) or see https://github.com/gitleaks/gitleaks#installing for other platforms." Then proceed with the commit.
2. If available, run: `gitleaks detect --staged --no-banner`
3. If secrets are detected: **abort the commit**. Show the gitleaks output and ask the user to remove the secrets before proceeding.
4. If no secrets detected: proceed.

```bash
git commit -m "vault: add <title>"
```

Use appropriate commit messages:
- `vault: add <title>` — new note
- `vault: update <title>` — modified note
- `vault: add and update notes` — multiple changes in one session

### 8. Merge into Main and Push

Acquire the merge lock to prevent concurrent merge collisions:

```bash
# Acquire lock (atomic — fails if already locked)
mkdir ~/.local/share/opencode/vault/.merge-lock
```

If the lock already exists, wait 5 seconds and retry, up to 6 attempts (30 seconds total). If still locked, warn the user and abort the merge (their changes are safe on the session branch).

Once locked:

```bash
# Update main worktree
git -C ~/.local/share/opencode/vault/branches/main pull origin main

# Merge session branch into main
git -C ~/.local/share/opencode/vault/branches/main merge <branch-name>

# If INDEX.md conflicts: accept the version from the session branch,
# then regenerate INDEX.md from all note frontmatter, amend the merge commit.

# Push
git -C ~/.local/share/opencode/vault/branches/main push origin main
```

Release the lock:

```bash
rmdir ~/.local/share/opencode/vault/.merge-lock
```

### 9. Cleanup

```bash
git -C ~/.local/share/opencode/vault/vault.git worktree remove \
  ~/.local/share/opencode/vault/branches/<branch-name>
git -C ~/.local/share/opencode/vault/vault.git branch -d <branch-name>
```

## Frontmatter Schema

Every note MUST have these fields:

```yaml
---
title: <Note Title>
tags: [<tag1>, <tag2>]
type: <note|template|project|canvas>
---
```

| Field      | Type   | Description                                              |
|------------|--------|----------------------------------------------------------|
| `title`    | string | Human-readable title                                     |
| `tags`     | list   | 2-5 tags. Reuse existing tags from INDEX.md first.       |
| `type`     | string | Note category (see values above)                         |

## Important Rules

- **NEVER** store secrets, credentials, or API keys in the vault.
- **NEVER** force push.
- **ALWAYS** regenerate INDEX.md rather than editing it manually.
- **ALWAYS** use the merge lock during the merge+push step.
- **ALWAYS** clean up the session worktree and branch after a successful merge.
- Use `[[wikilinks]]` for cross-references between notes.
- Keep notes **atomic** — one topic per note when possible.
- **Prefer updating** existing notes over creating new ones to avoid duplication.
- Do not create new top-level folders without user approval.
