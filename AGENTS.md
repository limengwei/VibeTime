# Agent Instructions

## Task Tracking

This project does **not** use bd/beads or any issue tracker. Detailed git
commit messages are the single source of record for what changed and why.

**Never** run `bd init` or any `bd` command, never let tooling re-create
`.beads/`, `.claude/`, `CLAUDE.md`, or beads instruction blocks in this
file — the project removed all of that (git `core.hooksPath` pointing
into `.beads/hooks` was how it kept coming back; it is unset).

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

Shell commands like `cp`, `mv`, and `rm` may be aliased to include `-i` (interactive) mode on some systems, causing the agent to hang indefinitely waiting for y/n input.

**Use these forms instead:**
```bash
# Force overwrite without prompting
cp -f source dest           # NOT: cp source dest
mv -f source dest           # NOT: mv source dest
rm -f file                  # NOT: rm file

# For recursive operations
rm -rf directory            # NOT: rm -r directory
cp -rf source dest          # NOT: cp -r source dest
```

**Other commands that may prompt:**
- `scp` - use `-o BatchMode=yes` for non-interactive
- `ssh` - use `-o BatchMode=yes` to fail instead of prompting
- `apt-get` - use `-y` flag
- `brew` - use `HOMEBREW_NO_AUTO_UPDATE=1` env var

## Session Completion

**When ending a work session**, complete the steps below. Work is NOT
complete until `git push` succeeds.

1. **Run quality gates** (if code changed) — tests, linters, builds
2. **Push to remote**:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
3. **Verify** — all changes committed AND pushed

**Critical rules:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing — that leaves work stranded locally
- If push fails, resolve and retry until it succeeds
