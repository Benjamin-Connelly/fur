# Tool Setup Guide: Snyk, Trivy, and Beads

Complete setup instructions for your security and workflow tools.

---

## 🔐 Snyk Setup

### 1. Authenticate with Snyk

```bash
# Login to Snyk (opens browser for authentication)
snyk auth

# Or use an API token
snyk auth <your-api-token>
```

**Get your token:**
1. Visit https://app.snyk.io/account
2. Generate an API token
3. Use `snyk auth <token>`

### 2. Test Your Project

```bash
# Test for vulnerabilities
snyk test

# Test and show all vulnerabilities (not just fixable)
snyk test --all-projects

# Test with JSON output
snyk test --json > snyk-results.json
```

### 3. Monitor Your Project

```bash
# Add project to Snyk dashboard for continuous monitoring
snyk monitor

# Monitor with project name
snyk monitor --project-name=lookit
```

### 4. Snyk Code (Static Analysis)

```bash
# Scan source code for security issues
snyk code test

# Scan and fix
snyk code test --fix
```

### 5. Configuration File (Optional)

Create `.snyk` file in project root:

```yaml
# .snyk configuration
version: v1.25.0

# Ignore specific vulnerabilities
ignore:
  'SNYK-JS-MINIMIST-559764':
    - '*':
        reason: 'Not used in production'
        expires: '2026-12-31'

# Exclude paths from scanning
exclude:
  global:
    - test/**
    - docs/**

# Language-specific settings
language-settings:
  javascript:
    ignore-dev-dependencies: true
```

### 6. CI/CD Integration

```yaml
# .github/workflows/security.yml
name: Security Scan

on: [push, pull_request]

jobs:
  snyk:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Snyk
        uses: snyk/actions/node@master
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
        with:
          args: --severity-threshold=high
```

---

## 🔍 Trivy Setup

### 1. Update Vulnerability Database

```bash
# Download latest vulnerability database
trivy image --download-db-only

# Or update during first scan (automatic)
trivy image node:18
```

### 2. Scan Your Project

```bash
# Scan filesystem (current directory)
trivy fs .

# Scan specific directory
trivy fs /path/to/project

# Scan with severity filtering
trivy fs . --severity CRITICAL,HIGH

# Output as JSON
trivy fs . --format json > trivy-results.json
```

### 3. Scan Docker Images

```bash
# Scan a local image
trivy image myapp:latest

# Scan during build (before pushing)
docker build -t myapp:latest .
trivy image myapp:latest

# Scan remote image
trivy image node:18-alpine
```

### 4. Configuration File (Optional)

Create `trivy.yaml`:

```yaml
# trivy.yaml
severity:
  - CRITICAL
  - HIGH
  - MEDIUM

format: table

exit-code: 1  # Fail on findings

vulnerability:
  type:
    - os
    - library

# Ignore unfixed vulnerabilities
ignore-unfixed: true

# Skip files/directories
skip-files:
  - "test/**"
  - "docs/**"

# Timeout
timeout: 5m
```

Use it:
```bash
trivy fs . --config trivy.yaml
```

### 5. Ignore Specific Vulnerabilities

Create `.trivyignore`:

```
# .trivyignore
# Ignore specific CVEs
CVE-2023-12345
CVE-2024-67890

# With expiration (YYYY-MM-DD)
CVE-2023-11111 exp:2026-12-31

# With reason
CVE-2023-22222  # Not exploitable in our use case
```

### 6. CI/CD Integration

```yaml
# .github/workflows/security.yml
name: Security Scan

on: [push, pull_request]

jobs:
  trivy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Run Trivy filesystem scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'CRITICAL,HIGH'
          exit-code: '1'

      # If you build Docker images
      - name: Build image
        run: docker build -t ${{ github.repository }}:${{ github.sha }} .

      - name: Run Trivy image scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'image'
          image-ref: '${{ github.repository }}:${{ github.sha }}'
          severity: 'CRITICAL,HIGH'
          exit-code: '1'
```

### 7. Scan Node.js Dependencies

```bash
# Scan package-lock.json or yarn.lock
trivy fs . --scanners vuln

# Show only npm vulnerabilities
trivy fs . --scanners vuln --vuln-type library
```

---

## 📋 Beads (bd) Setup

### 1. Initialize in Project (Already Done)

```bash
# If not already initialized
bd init

# This creates:
# - AGENTS.md (workflow instructions)
# - .beads/ directory (database)
# - .gitattributes (merge driver)
```

### 2. Run Onboarding

```bash
# Interactive tutorial
bd onboard
```

### 3. Configure Git Hooks (Already Done)

```bash
# Install recommended hooks
bd hooks install

# Hooks installed:
# - post-merge: Sync after pulling
# - pre-push: Sync before pushing
# - post-checkout: Sync after branch switch
# - prepare-commit-msg: Add issue refs
# - pre-commit: Validate before commit
```

### 4. Create Your First Issues

```bash
# Quick start guide
bd quickstart

# Create an issue manually
bd create "Implement directory listing" \
  --description "Add .gitignore support and file icons" \
  --status ready

# Create with tags
bd create "Add tests" --tags testing,priority:high

# Create with dependencies
bd create "Deploy to production" --blocked-by lookit-abc123
```

### 5. Daily Workflow

```bash
# Start of day: See what's ready
bd ready

# Claim an issue
bd update lookit-abc123 --status in_progress

# View issue details
bd show lookit-abc123

# Work on the issue...
# (write code, commit changes)

# Close when done
bd close lookit-abc123

# Sync with git (hooks do this automatically)
bd sync
```

### 6. Advanced Features

```bash
# Search issues
bd search "directory listing"

# Filter by status
bd list --status ready
bd list --status in_progress

# Show all issues
bd list

# Create issue from template
bd create --template bug "Fix markdown rendering"

# Link issues
bd update lookit-abc123 --blocks lookit-def456

# Set priority
bd update lookit-abc123 --priority high

# Add tags
bd update lookit-abc123 --tags bug,urgent
```

### 7. Configuration

```bash
# View current config
bd config

# Set up remote sync (optional)
bd config set sync.remote origin
bd config set sync.branch bd-sync

# Configure issue prefix
bd config set issue.prefix lookit

# Set default status for new issues
bd config set issue.default_status ready
```

### 8. Integration with Git

Issues are automatically referenced in commits:

```bash
# Commit with issue reference
git commit -m "Implement directory listing

Refs: lookit-abc123"

# The pre-commit hook can add this automatically
# based on current branch or in-progress issues
```

### 9. Team Collaboration (Optional)

```bash
# Push issues to shared branch
bd config set sync.branch bd-issues
bd sync
git push origin bd-issues

# Team members pull issues
git pull origin bd-issues
bd sync
```

### 10. Beads + Claude Integration

The AGENTS.md file tells Claude to:

1. Use `bd ready` to find work
2. Update issue status when starting work
3. Close issues when complete
4. Always `bd sync && git push` before ending session

**Landing the Plane workflow:**
```bash
# End of session (mandatory steps)
bd ready                    # Check for open issues
bd close <issue-id>         # Close completed work
bd sync                     # Sync to git
git pull --rebase          # Get latest
git push                   # Push everything
git status                 # Verify clean state
```

---

## 🔄 Integrated Workflow

### Pre-Development (One Time)

```bash
# 1. Authenticate Snyk
snyk auth

# 2. Test Snyk setup
snyk test

# 3. Update Trivy database
trivy image --download-db-only

# 4. Run bd onboarding
bd onboard
```

### Starting New Work

```bash
# 1. See available work
bd ready

# 2. Claim an issue
bd update lookit-abc123 --status in_progress

# 3. Create branch (optional)
git checkout -b feature/directory-listing
```

### During Development

```bash
# Run security scans periodically
npm audit
snyk test
trivy fs .

# Commit work
git add .
git commit -m "Add directory listing

Refs: lookit-abc123"
```

### Finishing Work

```bash
# 1. Final security check
snyk test --all-projects
trivy fs . --severity HIGH,CRITICAL

# 2. Close issue
bd close lookit-abc123

# 3. Sync and push (landing the plane)
bd sync
git pull --rebase
git push
git status  # Must be clean and up to date
```

### CI/CD Pipeline

```yaml
# .github/workflows/ci.yml
name: CI/CD

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install dependencies
        run: npm install

      - name: Run tests
        run: npm test

      - name: Snyk security scan
        uses: snyk/actions/node@master
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
        with:
          args: --severity-threshold=high

      - name: Trivy filesystem scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'CRITICAL,HIGH'
          exit-code: '1'

      - name: Check beads sync
        run: |
          if [ -d ".beads" ]; then
            git diff --exit-code .beads/issues.jsonl || \
              echo "::warning::Beads not synced"
          fi
```

---

## 📊 Monitoring & Reports

### Daily Security Dashboard

```bash
#!/bin/bash
# security-check.sh

echo "=== Security Scan Report ==="
echo ""

echo "📦 NPM Audit:"
npm audit --production
echo ""

echo "🔍 Snyk Scan:"
snyk test --severity-threshold=high
echo ""

echo "🛡️  Trivy Scan:"
trivy fs . --severity CRITICAL,HIGH
echo ""

echo "✅ Scan complete!"
```

Make executable and run daily:
```bash
chmod +x security-check.sh
./security-check.sh
```

### Weekly Issue Review

```bash
#!/bin/bash
# weekly-review.sh

echo "=== Weekly Issue Review ==="
echo ""

echo "📋 Open Issues:"
bd list --status ready,in_progress
echo ""

echo "✅ Closed This Week:"
bd list --status closed --since 7d
echo ""

echo "⚡ Blocked Issues:"
bd list --blocked
```

---

## 🎯 Quick Reference

### Snyk Commands
```bash
snyk auth                    # Authenticate
snyk test                    # Test for vulnerabilities
snyk monitor                 # Add to dashboard
snyk code test              # Scan source code
snyk fix                     # Auto-fix vulnerabilities
```

### Trivy Commands
```bash
trivy fs .                   # Scan filesystem
trivy image <name>          # Scan Docker image
trivy fs . --severity HIGH  # Filter by severity
trivy config .              # Scan IaC configs
```

### Beads Commands
```bash
bd ready                     # Show ready work
bd create "<title>"         # Create issue
bd update <id> --status X   # Update issue
bd show <id>                # View details
bd close <id>               # Close issue
bd sync                     # Sync with git
bd list                     # List all issues
```

---

## 🚨 Troubleshooting

### Snyk Issues

**Problem:** `snyk: command not found`
```bash
# Verify installation
npm list -g snyk

# Reinstall if needed
npm install -g snyk
```

**Problem:** Authentication fails
```bash
# Clear auth and retry
rm ~/.config/snyk/snyk-config.json
snyk auth
```

### Trivy Issues

**Problem:** Database download fails
```bash
# Use alternative mirror
trivy image --download-db-only --db-repository ghcr.io/aquasecurity/trivy-db
```

**Problem:** Slow scans
```bash
# Skip Java archives (if not needed)
trivy fs . --skip-files "**/*.jar"

# Use cache
trivy fs . --cache-dir ~/.cache/trivy
```

### Beads Issues

**Problem:** Hooks not working
```bash
# Reinstall hooks
bd hooks install --force

# Check hook status
ls -la .git/hooks/
```

**Problem:** Sync conflicts
```bash
# View conflicts
bd doctor

# Force sync
bd sync --force
```

---

## ✅ Verification Checklist

Run these to verify everything is set up:

```bash
# ✓ Snyk authenticated
snyk test --help >/dev/null 2>&1 && echo "✓ Snyk ready" || echo "✗ Snyk not authenticated"

# ✓ Trivy database current
trivy image --download-db-only >/dev/null 2>&1 && echo "✓ Trivy ready" || echo "✗ Trivy database missing"

# ✓ Beads initialized
[ -d ".beads" ] && echo "✓ Beads initialized" || echo "✗ Beads not initialized"

# ✓ Git hooks installed
[ -f ".git/hooks/pre-push" ] && echo "✓ Git hooks installed" || echo "✗ Git hooks missing"
```

Expected output:
```
✓ Snyk ready
✓ Trivy ready
✓ Beads initialized
✓ Git hooks installed
```
