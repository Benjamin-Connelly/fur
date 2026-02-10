# Security Scan Report - 2026-02-02

## Summary

- **Images Scanned:** 0 (no Docker images)
- **Dependency Vulnerabilities:** 0 (npm audit clean)
- **Code Vulnerabilities Found:** 1 Critical
- **Outdated Dependencies:** 2 (non-security)
- **Total Code Lines:** 3,445

## Status: FAIL (1 Critical Vulnerability)

### FAIL Criteria
❌ 1 Critical (command injection vulnerability)
✅ 0 High
✅ 0 Medium

---

## Critical Vulnerabilities Found

### 1. CVE-CUSTOM-001: Command Injection in Git Handler (CRITICAL)

**File:** `src/gitHandler.js:227`
**Severity:** CRITICAL
**CVSS Score:** 9.8 (Critical)
**CWE:** CWE-78 (OS Command Injection)

**Description:**
The `getLastCommit()` function constructs a shell command using string interpolation with user-controlled input (file path) without proper sanitization. This creates a command injection vulnerability where an attacker could execute arbitrary shell commands.

**Vulnerable Code:**
```javascript
const output = execSync(
  `git log -1 --format="%an | %ar | %s" -- "${relativePath}"`,
  {
    cwd: repoRoot,
    encoding: 'utf8',
    stdio: ['pipe', 'pipe', 'ignore']
  }
).trim();
```

**Attack Vector:**
An attacker could create a file with a malicious filename containing shell metacharacters:
- Filename: `test"; rm -rf /; echo "pwned.txt`
- When Git log is executed, the command becomes:
  ```bash
  git log -1 --format="%an | %ar | %s" -- "test"; rm -rf /; echo "pwned.txt"
  ```
- This would execute `rm -rf /` on the system

**Impact:**
- Remote Code Execution (RCE)
- Complete system compromise
- Data exfiltration
- Denial of Service

**Exploitability:** High (easy to exploit, no authentication required)

**Affected Functions:**
- `getLastCommit()` - Line 227

**Remediation:**
Replace string interpolation with array-based command execution:

```javascript
const { execFileSync } = require('child_process');

const output = execFileSync(
  'git',
  ['log', '-1', '--format=%an | %ar | %s', '--', relativePath],
  {
    cwd: repoRoot,
    encoding: 'utf8',
    stdio: ['pipe', 'pipe', 'ignore']
  }
).trim();
```

**Status:** UNFIXED - Requires immediate remediation

---

## Dependency Security Status

### npm audit Results
```
found 0 vulnerabilities
```

All production dependencies are free from known CVEs:
- ✅ markdown-it: ^14.0.0
- ✅ markdown-it-highlightjs: ^4.1.0
- ✅ highlight.js: ^11.9.0
- ✅ isbinaryfile: ^5.0.2
- ✅ ignore: ^5.3.0

---

## Outdated Dependencies (Non-Security)

### 1. ignore (5.3.2 → 7.0.5)
**Current:** 5.3.2
**Latest:** 7.0.5
**Breaking:** Yes (major version bump)
**Security Impact:** None
**Recommendation:** Review changelog before upgrading (v7 is major version change)

### 2. isbinaryfile (5.0.7 → 6.0.0)
**Current:** 5.0.7
**Latest:** 6.0.0
**Breaking:** Yes (major version bump)
**Security Impact:** None
**Recommendation:** Review changelog before upgrading

---

## Additional Security Findings

### Positive Security Practices Found

1. **Path Traversal Protection** (src/index.js:154-159)
   ```javascript
   // Prevent directory traversal
   if (!safePath.startsWith(CWD)) {
     res.writeHead(403, { 'Content-Type': 'text/plain' });
     res.end('403 Forbidden');
     return;
   }
   ```
   ✅ **Good:** Uses path normalization and prefix checking

2. **HTML Escaping** (src/utils.js:128-137)
   ```javascript
   function escapeHtml(text) {
     const map = {
       '&': '&amp;',
       '<': '&lt;',
       '>': '&gt;',
       '"': '&quot;',
       "'": '&#039;'
     };
     return text.replace(/[&<>"']/g, m => map[m]);
   }
   ```
   ✅ **Good:** Prevents XSS attacks in rendered HTML

3. **Binary File Detection** (src/fileHandler.js:95-96)
   ```javascript
   const isBinary = await isBinaryFile(filePath);
   ```
   ✅ **Good:** Uses dedicated library to avoid serving malicious content as text

4. **Localhost Binding by Default** (src/index.js:36)
   ```javascript
   const HOST = args.host || '127.0.0.1';
   ```
   ✅ **Good:** Defaults to localhost, preventing accidental exposure

### Minor Security Concerns

1. **Shell Option in Git Command** (src/gitHandler.js:257)
   ```javascript
   const trackedFiles = execSync('git ls-files | wc -l', {
     cwd: repoRoot,
     encoding: 'utf8',
     shell: true,  // ⚠️ Allows shell interpretation
     stdio: ['pipe', 'pipe', 'ignore']
   });
   ```
   **Severity:** LOW
   **Issue:** Uses `shell: true` unnecessarily
   **Recommendation:** Replace with array-based commands

2. **Browser Command Execution** (src/utils.js:124)
   ```javascript
   spawn(command, [url], { detached: true, stdio: 'ignore' }).unref();
   ```
   **Severity:** LOW
   **Issue:** URL parameter could contain special characters
   **Current Status:** SAFE (URL is constructed by the application, not user input)
   **Recommendation:** No immediate action needed, but add URL validation as defense-in-depth

---

## Secret Scanning Results

✅ No hardcoded credentials found
✅ No API keys detected
✅ No private keys in source code
✅ .env files are test fixtures only (not tracked in git)

Files checked:
- test/fixtures/.env (test fixture, ignored content)
- test/fixtures/dir-test/.env (test fixture, "SECRET_KEY=should_be_ignored")

---

## Security Best Practices Compliance

| Practice | Status | Notes |
|----------|--------|-------|
| Dependency scanning | ✅ PASS | 0 vulnerabilities |
| Secret scanning | ✅ PASS | No secrets detected |
| Input validation | ⚠️ PARTIAL | Path traversal protected, but command injection exists |
| Output encoding | ✅ PASS | HTML escaping implemented |
| Least privilege | ✅ PASS | Localhost-only by default |
| Secure defaults | ✅ PASS | HTTPS preferred, safe fallback |
| Error handling | ✅ PASS | Errors caught and logged safely |

---

## Recommendations

### Immediate (Critical - Fix within 24 hours)

1. **Fix Command Injection in gitHandler.js** (CVE-CUSTOM-001)
   - Replace `execSync` with `execFileSync` for all git commands
   - Use array-based arguments instead of string interpolation
   - Validate all file paths before passing to shell commands

### High Priority (Fix within 7 days)

2. **Remove `shell: true` from all execSync calls**
   - src/gitHandler.js:260 - Replace piped command with separate calls
   - Prevents potential shell injection vectors

### Medium Priority (Fix within 30 days)

3. **Upgrade outdated dependencies**
   - ignore: 5.3.2 → 7.0.5 (test compatibility first)
   - isbinaryfile: 5.0.7 → 6.0.0 (test compatibility first)

4. **Add URL validation to openBrowser()**
   - Validate URL format before passing to spawn()
   - Defense-in-depth measure

### Long-term

5. **Implement security testing**
   - Add integration tests for path traversal protection
   - Add unit tests for HTML escaping
   - Add fuzzing tests for file name handling

6. **Add security documentation**
   - Document security considerations in README.md
   - Add SECURITY.md with vulnerability reporting process
   - Add security testing to CI/CD pipeline

7. **Consider security headers**
   - Add Content-Security-Policy headers
   - Add X-Frame-Options headers
   - Add X-Content-Type-Options headers

---

## Risk Assessment

**Overall Risk Level:** HIGH (due to critical command injection)

**Pre-Production Deployment:** ❌ BLOCKED
**Production Deployment:** ❌ BLOCKED

**Mitigation Required:**
- Fix CVE-CUSTOM-001 (command injection) before any production use
- Test fix thoroughly with malicious file names
- Add regression tests to prevent reintroduction

---

## Scan Details

### Tools Used
- npm audit (dependency vulnerabilities)
- Manual code review (security anti-patterns)
- Pattern matching (secrets, shell commands)
- CWE mapping (vulnerability classification)

### Coverage
- ✅ All JavaScript source files (3,445 lines)
- ✅ All dependency declarations
- ✅ All environment files
- ✅ All shell command executions
- ✅ All HTTP request handlers

### Excluded
- Test files (test/*.js)
- Node modules (node_modules/*)
- Documentation (docs/*, *.md)

---

## Conclusion

The lookit project has generally good security practices including path traversal protection, HTML escaping, and localhost-only defaults. However, a **critical command injection vulnerability** in the git handler must be fixed immediately before this application can be safely deployed.

The fix is straightforward (replace string-based commands with array-based execution) and should take less than 1 hour to implement and test.

**Next Steps:**
1. Apply fix for CVE-CUSTOM-001
2. Run regression tests
3. Re-scan to verify fix
4. Consider security testing in CI/CD

---

**Report Generated:** 2026-02-02
**Scan Duration:** ~5 minutes
**Scanner:** Claude Security Agent v1.0
