# Global Development Environment Setup

This document tracks the global development tools installed on this system.

## Installation Date
2026-01-30

---

## Security & Vulnerability Scanning

### Trivy (v0.69.0)
**Purpose:** Container and filesystem vulnerability scanner
**Location:** `~/.local/bin/trivy`

```bash
# Scan a Docker image
trivy image myimage:latest

# Scan local filesystem
trivy fs /path/to/project

# Scan with specific severity
trivy image --severity CRITICAL,HIGH myimage:latest
```

### Snyk (v1.1302.1)
**Purpose:** Open source security and dependency scanning
**Location:** Global npm package

```bash
# Test project for vulnerabilities
snyk test

# Monitor project
snyk monitor

# Fix vulnerabilities
snyk fix

# Auth (first time)
snyk auth
```

---

## Task & Issue Tracking

### Beads (bd) (v0.49.1)
**Purpose:** Git-integrated issue tracking
**Location:** `~/.local/bin/bd`
**Project:** Initialized in lookit

```bash
# Common commands
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git

# Quick start
bd quickstart
bd onboard
```

**Files Created:**
- `AGENTS.md` - Instructions for Claude and other agents
- `.beads/` - Local issue database
- `.gitattributes` - Merge driver config

**Git Hooks Installed:**
- post-merge
- pre-push
- post-checkout
- prepare-commit-msg
- pre-commit

---

## CLI Utilities (Already Installed)

### GitHub CLI (gh)
**Purpose:** GitHub operations from command line
**Location:** `/usr/bin/gh`

### jq
**Purpose:** JSON processor
**Location:** `/usr/bin/jq`

### fzf
**Purpose:** Fuzzy finder
**Location:** `~/src/3rd-party/tui/.fzf/bin/fzf`

### bat
**Purpose:** cat with syntax highlighting
**Location:** System package (v0.25.0)

```bash
bat README.md  # View with syntax highlighting
```

---

## Search & Navigation (Newly Installed)

### ripgrep (rg) (v14.1.1)
**Purpose:** Fast recursive grep alternative
**Location:** System package

```bash
# Search for pattern
rg "pattern"

# Search specific file types
rg "pattern" -t js

# Search with context
rg "pattern" -C 3
```

### fd-find (fd) (v10.3.0)
**Purpose:** Fast find alternative
**Location:** System package

```bash
# Find files by name
fd "pattern"

# Find by type
fd -t f "pattern"  # files only
fd -t d "pattern"  # directories only

# Find and execute
fd "pattern" -x echo {}
```

---

## Node.js Global Packages

### Prettier (Latest)
**Purpose:** Code formatter

```bash
prettier --write "**/*.{js,json,md}"
```

### TypeScript (Latest)
**Purpose:** TypeScript compiler

```bash
tsc --version
```

### ts-node (Latest)
**Purpose:** TypeScript execution

```bash
ts-node script.ts
```

### nodemon (Latest)
**Purpose:** Auto-restart Node apps

```bash
nodemon app.js
```

### npm-check-updates (Latest)
**Purpose:** Update package.json dependencies

```bash
ncu  # Check for updates
ncu -u  # Update package.json
```

---

## Additional Recommended Tools

### Not Yet Installed (Optional)

**Docker Security:**
- `hadolint` - Dockerfile linter
- `dive` - Docker image layer explorer

**Code Quality:**
- `shellcheck` - Shell script linter
- `yamllint` - YAML linter

**Performance:**
- `hyperfine` - Benchmark command-line tools

**Install commands:**
```bash
# Hadolint
sudo wget -O /usr/local/bin/hadolint https://github.com/hadolint/hadolint/releases/latest/download/hadolint-Linux-x86_64
sudo chmod +x /usr/local/bin/hadolint

# Dive
wget https://github.com/wagoodman/dive/releases/latest/download/dive_0.12.0_linux_amd64.deb
sudo apt install ./dive_0.12.0_linux_amd64.deb

# Others via apt
sudo apt install shellcheck yamllint hyperfine
```

---

## Matt's Resources Summary

Explored `/home/bconnelly/Documents/LLMs/matts/` and found useful reference materials:

### Agents
- **Autonomous Testing Agent** - Comprehensive test maintenance agent
  - Fixes failing tests systematically
  - Maintains high coverage (>80%)
  - Runs in Docker environments
  - Logs progress every 30 minutes

### Guides
- **agents-workflow.md** - bd (beads) workflow and session completion
- **docker-guidelines.md** - Docker best practices
- **python-style.md** - Python coding conventions
- **retro-process.md** - Retrospective process

### Personas
- **fastapi-persona.md** - Pydantic-powered API architecture
- **frontend-architecture.md** - Next.js and TypeScript expertise
- **senior-python-api-architect.md** - OpenAPI schema instrumentation

### Reference
- **api-security.md** - Multi-layered security with API Gateway and Redis
- **claude-guidelines.md** - Project-specific rules and autonomous mode
- **testing-guide.md** - Testing strategies and best practices

**Key Patterns from Matt's Setup:**
- Uses `bd` (beads) for issue tracking
- Strong focus on security scanning (trivy, CVE tracking)
- Autonomous testing agent pattern
- Landing the plane workflow (git push before ending session)
- Structured documentation (docs/, agents/, personas/)

---

## Environment Verification

Run this to verify your setup:

```bash
echo "=== Security Tools ===" && \
trivy --version && \
snyk --version && \
echo "" && \
echo "=== Task Management ===" && \
bd --version && \
echo "" && \
echo "=== CLI Utilities ===" && \
gh --version && \
jq --version && \
fzf --version && \
bat --version && \
rg --version && \
fd --version && \
echo "" && \
echo "=== Node.js Tools ===" && \
prettier --version && \
tsc --version && \
ts-node --version && \
nodemon --version && \
ncu --version
```

---

## Next Steps for lookit

1. **Start using bd for task tracking:**
   ```bash
   bd quickstart
   bd ready
   ```

2. **Run security scans:**
   ```bash
   snyk test
   trivy fs .
   ```

3. **Set up remote (if not done):**
   ```bash
   git remote add origin <your-repo-url>
   git push -u origin master
   ```

4. **Consider adding CI/CD with security scanning:**
   - GitHub Actions with trivy and snyk
   - Automated testing on push
   - Vulnerability alerts

---

## Integration with CLAUDE.md

The AGENTS.md file created by `bd init` works alongside your CLAUDE.md:

**CLAUDE.md:** Global "cu" (continue unattended) mode + project values
**AGENTS.md:** Beads workflow + landing the plane instructions

Both files are read by Claude Code and provide complementary guidance.
