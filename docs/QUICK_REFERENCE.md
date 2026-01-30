# Quick Reference - Security & Workflow Tools

One-page reference for Snyk, Trivy, and Beads.

---

## 🔐 Snyk - Dependency Security

### Status Check
```bash
snyk test                    # Check for vulnerabilities
snyk test --severity-threshold=high  # Only show high/critical
```

### Common Commands
```bash
snyk auth                    # Authenticate (one-time)
snyk test                    # Test current project
snyk monitor                 # Add to dashboard monitoring
snyk code test              # Scan source code
snyk fix                     # Auto-fix vulnerabilities
```

### Quick Workflow
```bash
# Before committing
snyk test

# Before deploying
snyk test --severity-threshold=critical
snyk monitor
```

### Results
✅ **lookit: 0 vulnerabilities found**

---

## 🔍 Trivy - Container & Filesystem Scanner

### Status Check
```bash
trivy fs . --severity HIGH,CRITICAL     # Quick scan
trivy fs . --scanners vuln              # Dependencies only
```

### Common Commands
```bash
trivy fs .                               # Scan filesystem
trivy fs . --severity CRITICAL,HIGH      # Filter severity
trivy image myimage:latest               # Scan Docker image
trivy config .                           # Scan IaC configs
```

### Quick Workflow
```bash
# Before committing
trivy fs . --severity HIGH,CRITICAL

# Before pushing Docker images
docker build -t myapp:latest .
trivy image myapp:latest --severity HIGH,CRITICAL
```

### Database
- Auto-updates on first scan
- Manual update: `trivy image --download-db-only`

### Results
✅ **lookit: 0 vulnerabilities found**

---

## 📋 Beads (bd) - Issue Tracking

### Daily Workflow
```bash
# Morning: See what to work on
bd ready

# Start work: Claim an issue
bd update lookit-abc123 --assignee "Your Name"

# During work: View details
bd show lookit-abc123

# When done: Close the issue
bd close lookit-abc123

# End of day: Sync and push
bd sync
git pull --rebase
git push
```

### Common Commands
```bash
bd ready                     # Show available work
bd create "Title"           # Create new issue
bd show <id>                # View issue details
bd update <id> --assignee "Name"  # Claim issue
bd close <id>               # Close issue
bd list                     # List all issues
bd sync                     # Sync to git
```

### Creating Issues
```bash
# Simple
bd create "Fix the bug"

# With details
bd create "Add feature" \
  --description "Detailed description here" \
  --priority 1 \
  --labels feature,urgent

# Link issues (blockers)
bd update lookit-abc --deps "blocks:lookit-xyz"
```

### Current Issues (lookit)
1. **lookit-ja8** [P1] - Write comprehensive test suite
2. **lookit-gh1** [P2] - Complete directory listing (assigned to you)
3. **lookit-0c0** [P2] - Polish README

### Landing the Plane (End of Session)
```bash
# MANDATORY before ending work session:
bd ready                    # Check open issues
bd close <id>              # Close finished work
bd sync                    # Sync to git
git pull --rebase          # Get latest
git push                   # Push everything
git status                 # Verify clean: "up to date with origin"
```

**Remember:** Work is NOT complete until `git push` succeeds!

---

## 🔄 Integrated Daily Workflow

### Morning
```bash
# 1. Check for work
bd ready

# 2. Claim issue
bd update lookit-abc123 --assignee "Your Name"

# 3. View details
bd show lookit-abc123
```

### During Development
```bash
# Run security checks periodically
snyk test
trivy fs . --severity HIGH,CRITICAL

# Commit work
git add .
git commit -m "Implement feature

Refs: lookit-abc123"
```

### Before Committing
```bash
# Security scan
snyk test --severity-threshold=high
trivy fs . --severity CRITICAL,HIGH

# If clean, commit
git add .
git commit -m "Your message

Refs: lookit-abc123"
```

### End of Session (Landing the Plane)
```bash
# 1. Final security check
snyk test
trivy fs . --severity HIGH,CRITICAL

# 2. Close issue
bd close lookit-abc123

# 3. Sync everything
bd sync
git pull --rebase
git push

# 4. Verify
git status  # Must show: "up to date with origin"
```

---

## 📊 Current Status

### Security (lookit)
- ✅ Snyk: 0 vulnerabilities
- ✅ Trivy: 0 vulnerabilities
- ✅ Dependencies: Clean

### Issues (lookit)
- 📋 3 open issues
- 🔥 1 high priority (testing)
- ✅ 0 closed

### Tools Installed
- ✅ Snyk v1.1302.1 (authenticated)
- ✅ Trivy v0.69.0 (database updated)
- ✅ Beads v0.49.1 (initialized, hooks installed)

---

## 🚨 Quick Troubleshooting

### Snyk not authenticated
```bash
snyk auth
```

### Trivy database outdated
```bash
trivy image --download-db-only
```

### Beads sync issues
```bash
bd doctor          # Check status
bd sync            # Force sync
```

### Git hooks not working
```bash
bd hooks install --force
```

---

## 📚 Full Documentation

- **Complete Setup:** `docs/TOOLS_SETUP_GUIDE.md`
- **Global Setup:** `docs/GLOBAL_DEV_SETUP.md`
- **Agent Instructions:** `AGENTS.md`
- **Project Config:** `CLAUDE.md`

---

## 💡 Tips

**Security Scanning:**
- Run before every commit
- Add to CI/CD pipeline
- Monitor dashboard daily (Snyk)

**Issue Tracking:**
- Create issues for all work
- Always assign issues to yourself
- Use `bd ready` to find next task
- Never end session without `git push`

**Git Hooks:**
- Beads auto-syncs on git operations
- Commits auto-reference issues
- Pre-push validates clean state

---

## 🎯 One-Liner Health Check

```bash
echo "Security:" && snyk test && trivy fs . --severity HIGH,CRITICAL && \
echo "" && echo "Issues:" && bd ready && \
echo "" && echo "Git:" && git status
```

Expected output:
- ✅ Snyk: 0 vulnerabilities
- ✅ Trivy: 0 HIGH/CRITICAL
- 📋 Beads: List of ready work
- 🌳 Git: Clean or changes listed
