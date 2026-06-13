# AGENTS.md — fur build instructions

## Task Tracking

Use `bd` (Beads) for all task tracking. Before starting any work:

1. Run `bd ready` to see what tasks are available
2. Run `bd update <id> --claim` to claim a task before working on it
3. Run `bd update <id> --status done` when a task is complete
4. Never work on a task that is blocked by an open dependency

Do not create new tasks unless explicitly told to. Work only the tasks
assigned in the epic for the current stage.

## Build Rules

- No CGO. Pure Go only. Must cross-compile cleanly.
- No external web frameworks. stdlib `net/http` only.
- All errors handled explicitly. No panics.
- Idiomatic Go throughout -- exported types where appropriate.
- After every task, run `go build ./...` and confirm it compiles.
- After every epic, run `go test ./...` and fix all failures before
  marking the epic done.

## Project

GitHub: github.com/Benjamin-Connelly/fur
Language: Go
Binary name: fur
Config dir: ~/.config/fur/

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
