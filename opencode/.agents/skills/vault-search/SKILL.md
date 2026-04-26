---
name: vault-search
description: Search and retrieve information from an opencode-vault — read-only access to an AI-managed, git-tracked markdown knowledge vault.
---

# Vault Search

Read-only skill for searching and retrieving information from the opencode-vault. Use this in any session where you need to look up notes, research past findings, or check what's already been documented.

**This skill never writes, commits, or pushes.**

## Vault Location

```
~/.local/share/opencode/vault/
├── vault.git/              # bare clone
├── branches/
│   └── main/               # permanent worktree — read from here
```

## Boot Check

Before any operation:

1. Verify `~/.local/share/opencode/vault/vault.git` exists and is a bare git repo.
2. Verify `~/.local/share/opencode/vault/branches/main` exists.
3. If either is missing, tell the user: "Vault not found. Please run the vault-init skill to set up your vault."

## Workflow

### 1. Pull Latest

Update the main worktree to get the latest notes:

```bash
git -C ~/.local/share/opencode/vault/branches/main pull origin main
```

If pull fails (e.g., network issues), warn the user that results may be stale, then proceed with what's available locally.

### 2. Read Vault Context

Read from `~/.local/share/opencode/vault/branches/main`:

1. Read `README.md` to understand the vault's folder structure and conventions.
2. Read `INDEX.md` to get the full catalog of notes — titles, tags, types, and summaries.

### 3. Search the Index

Scan `INDEX.md` for relevant notes by matching:
- Tags
- Titles
- Summary keywords
- Folder names

Identify candidate notes from the index before reading full files.

### 4. Read Specific Notes

Once you've identified relevant notes from the index, read their full content from the main worktree.

### 5. Fallback: Grep

If the index doesn't surface what you need, use grep to search note contents directly:

```bash
grep -r "<search-term>" ~/.local/share/opencode/vault/branches/main --include="*.md" -l
```

### 6. Return Findings

Return findings to the user with:
- File paths (relative to the vault root)
- Relevant excerpts
- Related notes via `[[wikilinks]]` if applicable

## Frontmatter Reference

Notes have this frontmatter structure (for understanding search results):

```yaml
---
title: <Note Title>
tags: [<tag1>, <tag2>]
type: <note|template|project|canvas>
---
```

## Important Rules

- **NEVER** write, modify, commit, or push anything. This skill is strictly read-only.
- Always pull before searching to ensure fresh results.
- If the user wants to save new information based on search findings, tell them to use the vault-save skill.
