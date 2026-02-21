# safer-claude
**100% unsafely vibe coded.**

Run Claude Code on sensitive files without exposing your entire directory. Copies target files to a temp directory, launches Claude there, then syncs changes back — only the files you choose get sent to Anthropic.

Built for editing Obsidian vaults and other directories containing private data.

## How it works

1. `safer-claude <file-or-folder>` copies the target to a temp directory (skipping hidden dirs like `.git`/`.obsidian` and junk files)
2. Records SHA-256 hashes of all copied files as a manifest
3. Writes `.claude/settings.json` in the temp dir to deny file access outside it
4. Launches `claude` in the temp directory
4. After Claude exits, diffs temp files against the manifest to detect edits, new files, and deletions
6. Syncs changes back to the original location with conflict detection (warns if a file was modified both externally and in the session)
7. Cleans up the temp directory unless conflicts remain

## Install

```
go install github.com/philphilphil/safer-claude@latest
```

Or build from source:

```
go build -o safer-claude .
```

## Usage

```
# Edit a single file
safer-claude ~/vault/notes/todo.md

# Edit a folder
safer-claude ~/vault/projects/myproject
```

## Conflict handling

- **Unchanged files** — skipped silently
- **Edited files** — copied back if the original wasn't modified externally
- **New files** — copied to the original directory
- **Deleted files** — original is kept (logged as deleted in session)
- **Conflicts** — if a file was changed both externally and in the session, the temp directory is preserved and the conflict is reported

## Requirements

- Go 1.25+
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`claude` in PATH)
