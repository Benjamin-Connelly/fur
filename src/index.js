// lookit server entry point
const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');
const MarkdownIt = require('markdown-it');
const hljs = require('highlight.js');
const mdHighlight = require('markdown-it-highlightjs');
const { parseArgs, printCertInstructions, ensureCerts, openBrowser, escapeHtml } = require('./utils');
const { handleFile } = require('./fileHandler');
const { registerInstance, unregisterInstance, cleanStaleInstances } = require('./instanceManager');
const { handleListCommand, handleStopCommand, handleStopAllCommand } = require('./commands');

// Parse CLI arguments
const args = parseArgs(process.argv.slice(2));

// Handle management commands (don't start server)
if (args.list) {
  handleListCommand();
  process.exit(0);
}

if (args.stopAll) {
  handleStopAllCommand();
  process.exit(0);
}

if (args.stop !== undefined) {
  handleStopCommand(args.stop);
  process.exit(0);
}

// Clean stale instances before starting server
cleanStaleInstances();
const PORT = args.port || 7777;
const HOST = args.host || '127.0.0.1';
const CWD = process.cwd();
const CERT_DIR = path.join(process.env.HOME, '.config', 'lookit');
const CERT_PATH = args.cert || path.join(CERT_DIR, 'localhost.pem');
const KEY_PATH = args.key || path.join(CERT_DIR, 'localhost-key.pem');

// Initialize markdown renderer with syntax highlighting
const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true
}).use(mdHighlight, { hljs });

// Check for TLS certificates (auto-generate with mkcert if available)
const hasCerts = args.noHttps ? false : ensureCerts(CERT_PATH, KEY_PATH, args.quiet);
const useHttps = hasCerts && !args.noHttps;

if (args.httpsOnly && !hasCerts) {
  console.error('❌ Error: --https-only specified but certificates could not be found or generated.\n');
  printCertInstructions();
  process.exit(1);
}

// Track the port we're running on
let runningPort = null;

// Start server with auto-increment port logic
function startServer(port, retriesLeft = 10) {
  const server = useHttps ? createHttpsServer() : createHttpServer();

  server.listen(port, HOST, () => {
    runningPort = port;
    const protocol = useHttps ? 'https' : 'http';
    const url = `${protocol}://${HOST}:${port}`;

    registerInstance(port, CWD, protocol);

    console.log(`👀 lookit - Code Browser`);
    console.log(`📂 Serving: ${CWD}`);
    console.log(`🌐 Address:  ${url}`);
    console.log(`🔒 Security: ${useHttps ? 'HTTPS (TLS)' : 'HTTP (plaintext)'}`);
    if (port === 7777) {
      console.log(`🍀 Lucky port: ${port}`);
    }
    console.log('\nPress Ctrl+C to stop.\n');

    if (args.open) {
      openBrowser(url);
    }
  });

  server.on('error', (err) => {
    if (err.code === 'EADDRINUSE' && retriesLeft > 0) {
      console.log(`⏭️  Port ${port} in use, trying ${port + 1}...`);
      startServer(port + 1, retriesLeft - 1);
    } else if (err.code === 'EADDRINUSE') {
      console.error(`❌ Error: Ports ${PORT}-${port} all in use.`);
      console.error(`   Stop existing instances: lookit --stop-all`);
      console.error(`   Or specify a higher port: lookit --port ${port + 10}`);
      process.exit(1);
    } else {
      console.error(`❌ Server error: ${err.message}`);
      process.exit(1);
    }
  });
}

// Cleanup on exit
process.on('SIGINT', () => {
  console.log('\n👋 Shutting down lookit...');
  if (runningPort) {
    unregisterInstance(runningPort);
  }
  process.exit(0);
});

process.on('SIGTERM', () => {
  if (runningPort) {
    unregisterInstance(runningPort);
  }
  process.exit(0);
});

// Cleanup on uncaught errors
process.on('uncaughtException', (err) => {
  console.error('❌ Uncaught error:', err);
  if (runningPort) {
    unregisterInstance(runningPort);
  }
  process.exit(1);
});

// Start from initial port
startServer(PORT);

function createHttpsServer() {
  const options = {
    key: fs.readFileSync(KEY_PATH),
    cert: fs.readFileSync(CERT_PATH)
  };
  return https.createServer(options, handleRequest);
}

function createHttpServer() {
  return http.createServer(handleRequest);
}

function handleRequest(req, res) {
  const urlPath = decodeURIComponent(req.url.split('?')[0]);
  const filePath = path.join(CWD, urlPath);
  const safePath = path.normalize(filePath);

  // Prevent directory traversal
  if (!safePath.startsWith(CWD)) {
    res.writeHead(403, { 'Content-Type': 'text/plain' });
    res.end('403 Forbidden');
    return;
  }

  // Check if path exists
  if (!fs.existsSync(safePath)) {
    res.writeHead(404, { 'Content-Type': 'text/plain' });
    res.end('404 Not Found');
    return;
  }

  const stat = fs.statSync(safePath);

  if (stat.isDirectory() || stat.isFile()) {
    handleFile(safePath, urlPath, stat, res, { md, hljs, args, CWD, req });
  } else {
    res.writeHead(400, { 'Content-Type': 'text/plain' });
    res.end('400 Bad Request');
  }
}

// Export for testing
module.exports = { handleRequest };