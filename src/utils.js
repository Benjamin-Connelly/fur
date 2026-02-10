// General utilities
const { spawn, execFileSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const ignore = require('ignore');

function parseArgs(argv) {
  const args = {};

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];

    if (arg === '--port' && argv[i + 1]) {
      args.port = parseInt(argv[++i], 10);
    } else if (arg === '--host' && argv[i + 1]) {
      args.host = argv[++i];
    } else if (arg === '--cert' && argv[i + 1]) {
      args.cert = argv[++i];
    } else if (arg === '--key' && argv[i + 1]) {
      args.key = argv[++i];
    } else if (arg === '--open') {
      args.open = true;
    } else if (arg === '--https-only') {
      args.httpsOnly = true;
    } else if (arg === '--no-https') {
      args.noHttps = true;
    } else if (arg === '--no-dirlist') {
      args.noDirlist = true;
    } else if (arg === '--all') {
      args.showAll = true;
    } else if (arg === '--quiet' || arg === '-q') {
      args.quiet = true;
    } else if (arg === '--list' || arg === '-l') {
      args.list = true;
    } else if (arg === '--stop-all') {
      args.stopAll = true;
    } else if (arg === '--stop' && argv[i + 1]) {
      args.stop = parseInt(argv[++i], 10);
    } else if (arg === '--version' || arg === '-v') {
      const pkg = require('../package.json');
      console.log(pkg.version);
      process.exit(0);
    } else if (arg === '--help' || arg === '-h') {
      printHelp();
      process.exit(0);
    }
  }

  return args;
}

function printHelp() {
  console.log(`
lookit - Browse and view code files with syntax highlighting

USAGE:
  lookit [OPTIONS]

OPTIONS:
  --port <number>      Port to listen on (default: 7777) - Lucky number! 🍀
  --host <address>     Host to bind to (default: 127.0.0.1)
  --open               Open browser automatically
  --all                Show hidden files (starting with .)
  --https-only         Fail if TLS certificates are not found
  --no-https           Use HTTP only, skip HTTPS even if certificates exist
  --no-dirlist         Disable directory listings
  --cert <path>        Path to TLS certificate (default: ~/.config/lookit/localhost.pem)
  --key <path>         Path to TLS private key (default: ~/.config/lookit/localhost-key.pem)
  -q, --quiet          Suppress TLS certificate warnings
  -l, --list           List all running lookit instances
  --stop <port>        Stop lookit instance on specific port
  --stop-all           Stop all running lookit instances
  -v, --version        Show version number
  -h, --help           Show this help message

FILE SUPPORT:
  Markdown (.md)       Rendered as HTML with syntax-highlighted code blocks
  Code files           Displayed with syntax highlighting (YAML, JSON, JS, Python, etc.)
  Images/PDFs          Displayed natively in browser
  Other files          Downloaded

EXAMPLES:
  lookit                           # Serve current directory on https://localhost:7777
  lookit --port 8080               # Use port 8080
  lookit --no-https                # Use HTTP only
  lookit --open                    # Auto-open browser
  lookit --https-only              # Require HTTPS, fail if no certs
  lookit --all                     # Show hidden files
  lookit --list                    # Show all running instances
  lookit --stop 7778               # Stop instance on port 7778
  lookit --stop-all                # Stop all lookit instances

PROCESS MANAGEMENT:
  Multiple instances can run simultaneously. Each uses the next available port.
  Use --list to see all running instances, --stop-all to clean up.

TLS SETUP (Ubuntu):
  sudo apt install -y mkcert libnss3-tools
  mkcert -install
  mkdir -p ~/.config/lookit
  mkcert -cert-file ~/.config/lookit/localhost.pem \\
         -key-file ~/.config/lookit/localhost-key.pem \\
         localhost 127.0.0.1 ::1
`);
}

function printCertInstructions() {
  console.log(`📝 To enable HTTPS, install mkcert and generate certificates:\n`);
  console.log(`   1. Install mkcert:`);
  console.log(`      sudo apt install -y mkcert libnss3-tools\n`);
  console.log(`   2. Install the local CA:`);
  console.log(`      mkcert -install\n`);
  console.log(`   3. Generate certificates:`);
  console.log(`      mkdir -p ~/.config/lookit`);
  console.log(`      mkcert -cert-file ~/.config/lookit/localhost.pem \\`);
  console.log(`             -key-file ~/.config/lookit/localhost-key.pem \\`);
  console.log(`             localhost 127.0.0.1 ::1`);
}

/**
 * Check if mkcert is installed and available on PATH.
 */
function hasMkcert() {
  try {
    execFileSync('mkcert', ['-help'], { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

/**
 * Auto-generate TLS certificates with mkcert if available.
 * Returns true if certs exist (or were just created), false otherwise.
 */
function ensureCerts(certPath, keyPath, quiet) {
  if (fs.existsSync(certPath) && fs.existsSync(keyPath)) {
    return true;
  }

  if (!hasMkcert()) {
    if (!quiet) {
      console.warn('⚠️  No TLS certificates found and mkcert is not installed.');
      printCertInstructions();
      console.log('');
    }
    return false;
  }

  // mkcert is available - auto-generate certs
  const certDir = path.dirname(certPath);
  fs.mkdirSync(certDir, { recursive: true });

  try {
    // Ensure local CA is installed (safe to run multiple times)
    execFileSync('mkcert', ['-install'], { stdio: 'pipe' });

    // Generate certificates
    execFileSync('mkcert', [
      '-cert-file', certPath,
      '-key-file', keyPath,
      'localhost', '127.0.0.1', '::1'
    ], { stdio: 'pipe' });

    console.log('🔐 Auto-generated HTTPS certificates with mkcert');
    return true;
  } catch (err) {
    if (!quiet) {
      console.warn(`⚠️  mkcert found but certificate generation failed: ${err.message}`);
      console.warn('   Falling back to HTTP.\n');
    }
    return false;
  }
}

function openBrowser(url) {
  const commands = {
    linux: 'xdg-open',
    darwin: 'open',
    win32: 'start'
  };

  const command = commands[process.platform];
  if (command) {
    spawn(command, [url], { detached: true, stdio: 'ignore' }).unref();
  }
}

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

/**
 * Load .gitignore files from current directory and all parent directories
 * @param {string} dirPath - Directory path to start from
 * @returns {Object} ignore object with combined rules
 */
function loadGitignore(dirPath) {
  const ig = ignore();

  // Always ignore .git directory
  ig.add('.git/');

  let currentDir = path.resolve(dirPath);
  const root = path.parse(currentDir).root;

  // Walk up the directory tree
  while (true) {
    const gitignorePath = path.join(currentDir, '.gitignore');

    try {
      if (fs.existsSync(gitignorePath)) {
        const content = fs.readFileSync(gitignorePath, 'utf8');
        ig.add(content);
      }
    } catch (err) {
      // Silently skip if we can't read the file
    }

    // Stop at root
    if (currentDir === root) {
      break;
    }

    // Move up one directory
    currentDir = path.dirname(currentDir);
  }

  return ig;
}

/**
 * Check if a file should be ignored based on .gitignore rules
 * @param {string} filePath - Absolute path to the file
 * @param {string} basePath - Base directory path
 * @param {Object} ig - ignore object
 * @param {boolean} isDirectory - Whether the path is a directory
 * @returns {boolean} true if file should be ignored
 */
function shouldIgnoreFile(filePath, basePath, ig, isDirectory = false) {
  // Get relative path from base
  let relativePath = path.relative(basePath, filePath);

  // For directories, check both with and without trailing slash
  if (isDirectory) {
    return ig.ignores(relativePath + '/') || ig.ignores(relativePath);
  }

  // Check if ignored
  return ig.ignores(relativePath);
}

module.exports = {
  parseArgs,
  printHelp,
  printCertInstructions,
  ensureCerts,
  openBrowser,
  escapeHtml,
  loadGitignore,
  shouldIgnoreFile
};