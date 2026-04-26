---
name: vault-cleanup
description: Full maintenance for an opencode-vault — deduplicate, merge, reword, reorganize, and repair notes across the entire vault.
---

# Vault Cleanup

Full vault maintenance skill. Run this explicitly in a dedicated session to clean up, deduplicate, reorganize, and improve the quality of notes across the entire vault.

This skill reads ALL notes thoroughly and performs bulk operations. It uses git worktree isolation for safe concurrent access.

## Vault Location

```
~/.local/share/opencode/vault/
├── vault.git/              # bare clone
├── branches/
│   ├── main/               # permanent worktree (for reference)
│   └── <cleanup-branch>/   # your isolated cleanup worktree
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

### 2. Create Cleanup Worktree

Use a descriptive branch name like `cleanup-dedup`, `cleanup-tags`, `cleanup-full`, or similar.

```bash
git -C ~/.local/share/opencode/vault/vault.git worktree add \
  ~/.local/share/opencode/vault/branches/<branch-name> \
  -b <branch-name> origin/main
```

If the branch already exists (e.g., from a crashed session), clean it up first:

```bash
git -C ~/.local/share/opencode/vault/vault.git worktree remove \
  ~/.local/share/opencode/vault/branches/<branch-name> --force
git -C ~/.local/share/opencode/vault/vault.git branch -D <branch-name>
```

Then retry.

All operations happen in `~/.local/share/opencode/vault/branches/<branch-name>/`.

### 3. Full Secret Scan

Before reading or modifying anything, scan the entire worktree for existing secrets:

```bash
cd ~/.local/share/opencode/vault/branches/<branch-name>
gitleaks detect --no-git --no-banner
```

- If gitleaks is not installed, warn the user: "gitleaks is not installed. Skipping full vault scan. Install with `brew install gitleaks` (macOS) or see https://github.com/gitleaks/gitleaks#installing for other platforms."
- If secrets are detected: **stop and report them to the user immediately**. List the affected files and line numbers. Ask the user how to proceed (remove the secrets, delete the files, etc.) before continuing with any other maintenance work.
- If no secrets detected: proceed.

### 4. Read the Entire Vault

1. Read `README.md` for vault structure and conventions.
2. Read `INDEX.md` for the full catalog.
3. Read **every note** in the vault to build a complete picture of content, tags, structure, and cross-references.

### 5. Maintenance Operations

Perform any combination of the following, committing after each logical change:

#### Deduplication

1. Identify notes that cover the same or heavily overlapping topics.
2. Merge them into a single note — combine the best content from both, preserve all unique information.
3. Delete the redundant note.
4. Update any `[[wikilinks]]` in other notes that pointed to the deleted note.
5. Commit: `vault: merge <note-a> into <note-b>`

#### Rewording and Improvement

1. Improve note clarity, fix grammar, improve structure.
2. Ensure notes are well-organized with clear headings.
3. Commit: `vault: update <title>`

#### Tag Normalization

1. Review all tags across the vault for consistency.
2. Consolidate synonyms (e.g., `js` and `javascript` should be one tag).
3. Remove unused or overly specific tags.
4. Ensure every note has 2-5 relevant tags.
5. Commit: `vault: normalize tags`

#### Broken Wikilink Repair

1. Find all `[[wikilinks]]` across notes.
2. Identify links that point to non-existent notes.
3. Either remove the broken link or create the missing note if it makes sense.
4. Commit: `vault: fix broken wikilinks`

#### Frontmatter Validation

1. Ensure every note has all required frontmatter fields (title, tags, type).
2. Fix any missing or malformed fields.
3. Commit: `vault: fix frontmatter`

#### Folder Reorganization

1. Evaluate whether the current folder structure makes sense.
2. Move misplaced notes to more appropriate folders.
3. Only create or remove top-level folders with user approval.
4. Commit: `vault: reorganize <description>`

#### Note Deletion

1. Identify notes that are empty, obsolete, or no longer useful.
2. Confirm with the user before deleting.
3. Remove the file and clean up any `[[wikilinks]]` pointing to it.
4. Commit: `vault: delete <title>`

### 6. Secret Scan

Before every commit in step 5 (and step 7 below), scan staged files for secrets:

1. Check if gitleaks is available: `command -v gitleaks`
   - If not installed, warn the user: "gitleaks is not installed. Skipping secret scan. Install with `brew install gitleaks` (macOS) or see https://github.com/gitleaks/gitleaks#installing for other platforms." Then proceed with the commit.
2. If available, stage the changes (`git add -A`) then run: `gitleaks detect --staged --no-banner`
3. If secrets are detected: **abort the commit**. Show the gitleaks output and ask the user to remove the secrets before proceeding.
4. If no secrets detected: proceed with the commit.

This check applies to **every** `git commit` in the cleanup workflow.

### 7. Regenerate INDEX.md

After all changes are complete, regenerate `INDEX.md` entirely from note frontmatter:

1. Glob all `.md` files (excluding `INDEX.md` and `README.md`).
2. Read the frontmatter of each note.
3. Produce a markdown table grouped by folder:

```markdown
# Index

Complete catalog of all notes in this vault.

<!-- This file is auto-generated from note frontmatter. Do not edit manually. -->

## <Folder Name>

| Title | Tags | Type | Summary |
|-------|------|------|---------|
| [[note-title]] | tag1, tag2 | note | Brief summary of the note |
```

4. Commit: `vault: regenerate index`

### 8. Merge into Main and Push

Acquire the merge lock:

```bash
mkdir ~/.local/share/opencode/vault/.merge-lock
```

If the lock exists, wait 5 seconds and retry, up to 6 attempts. If still locked, warn the user.

Once locked:

```bash
# Update main
git -C ~/.local/share/opencode/vault/branches/main pull origin main

# Merge cleanup branch
git -C ~/.local/share/opencode/vault/branches/main merge <branch-name>

# If INDEX.md conflicts: regenerate INDEX.md from all note frontmatter, then commit.

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
- **NEVER** delete notes without user confirmation.
- **ALWAYS** regenerate INDEX.md rather than editing it manually.
- **ALWAYS** use the merge lock during the merge+push step.
- **ALWAYS** clean up the session worktree and branch after a successful merge.
- **ALWAYS** commit after each logical change — do not batch unrelated changes into one commit.
- Use `[[wikilinks]]` for cross-references between notes.
- Keep notes **atomic** — one topic per note when possible.
