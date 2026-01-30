// File handling utilities
const path = require('path');
const fs = require('fs').promises;
const { isBinaryFile } = require('isbinaryfile');

// File extension sets
const CODE_EXTENSIONS = new Set([
  // JavaScript/TypeScript
  '.js', '.jsx', '.ts', '.tsx', '.mjs', '.cjs',
  // Python
  '.py', '.pyw', '.pyx', '.pyi',
  // Ruby
  '.rb', '.rake', '.gemspec',
  // Go
  '.go',
  // Rust
  '.rs',
  // Java/Kotlin/Scala
  '.java', '.kt', '.kts', '.scala',
  // C/C++/C#
  '.c', '.h', '.cpp', '.hpp', '.cc', '.cxx', '.cs',
  // Shell
  '.sh', '.bash', '.zsh', '.fish',
  // Web
  '.html', '.htm', '.css', '.scss', '.sass', '.less',
  // Config/Data
  '.json', '.jsonc', '.yaml', '.yml', '.toml', '.xml',
  // Other languages
  '.php', '.swift', '.r', '.m', '.sql', '.pl', '.lua',
  // Markdown-like (non-.md)
  '.txt', '.log', '.csv', '.tsv',
  // Build/Config
  '.dockerfile', '.makefile', '.cmake', '.gradle',
  // Misc
  '.vue', '.svelte', '.astro', '.graphql', '.proto'
]);

const MARKDOWN_EXTENSIONS = new Set(['.md', '.mdx']);

const IMAGE_EXTENSIONS = new Set([
  '.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.ico', '.bmp'
]);

const VIDEO_EXTENSIONS = new Set([
  '.mp4', '.webm', '.ogg', '.mov', '.avi', '.mkv'
]);

const AUDIO_EXTENSIONS = new Set([
  '.mp3', '.wav', '.ogg', '.m4a', '.flac', '.aac'
]);

const PDF_EXTENSIONS = new Set(['.pdf']);

/**
 * Detect the type of file based on extension and stats
 * @param {string} filePath - Path to the file
 * @param {fs.Stats} stats - File stats object
 * @returns {Promise<string>} - File type string
 */
async function detectFileType(filePath, stats) {
  // Check if directory first
  if (stats.isDirectory()) {
    return 'directory';
  }

  // Get file extension
  const ext = path.extname(filePath).toLowerCase();

  // Check known extensions
  if (MARKDOWN_EXTENSIONS.has(ext)) {
    return 'markdown';
  }

  if (CODE_EXTENSIONS.has(ext)) {
    return 'code';
  }

  if (IMAGE_EXTENSIONS.has(ext)) {
    return 'image';
  }

  if (VIDEO_EXTENSIONS.has(ext)) {
    return 'video';
  }

  if (AUDIO_EXTENSIONS.has(ext)) {
    return 'audio';
  }

  if (PDF_EXTENSIONS.has(ext)) {
    return 'pdf';
  }

  // Check if binary using isbinaryfile
  try {
    const isBinary = await isBinaryFile(filePath);
    if (isBinary) {
      return 'binary';
    }
  } catch (err) {
    // If we can't determine, assume text
    console.error(`Error checking if binary: ${err.message}`);
  }

  // Default to text
  return 'text';
}

/**
 * Route file to appropriate handler based on type
 * @param {string} filePath - Absolute path to the file
 * @param {string} urlPath - URL path requested
 * @param {fs.Stats} stats - File stats object
 * @param {http.ServerResponse} res - Response object
 * @param {Object} context - Context object with md, hljs, args, CWD, req
 */
async function handleFile(filePath, urlPath, stats, res, context) {
  const fileType = await detectFileType(filePath, stats);

  switch (fileType) {
    case 'directory':
      return handleDirectory(filePath, urlPath, stats, res, context);

    case 'markdown':
      return handleMarkdown(filePath, urlPath, stats, res, context);

    case 'code':
      return handleCode(filePath, urlPath, stats, res, context);

    case 'image':
    case 'video':
    case 'audio':
    case 'pdf':
      return handleStatic(filePath, urlPath, stats, res, context, fileType);

    case 'binary':
      return handleBinary(filePath, urlPath, stats, res, context);

    case 'text':
    default:
      return handleCode(filePath, urlPath, stats, res, context);
  }
}

/**
 * Handler stubs - to be implemented in future tasks
 */

async function handleDirectory(filePath, urlPath, stats, res, context) {
  const { createDirectoryTemplate } = require('./templates/directory.js');
  const { escapeHtml, loadGitignore, shouldIgnoreFile } = require('./utils.js');

  // Check if directory listing is disabled
  if (context.args.noDirlist) {
    res.writeHead(403, { 'Content-Type': 'text/plain' });
    res.end('Directory listing is disabled');
    return;
  }

  try {
    // Read directory contents
    const files = await fs.readdir(filePath);

    // Load .gitignore rules
    const ig = loadGitignore(filePath);

    // Process entries
    const entries = [];

    for (const file of files) {
      const fullPath = path.join(filePath, file);
      const fileUrl = path.posix.join(urlPath, file);

      try {
        const fileStats = await fs.stat(fullPath);
        const isDirectory = fileStats.isDirectory();

        // Check if file is ignored
        const ignored = shouldIgnoreFile(fullPath, filePath, ig, isDirectory);

        // Skip ignored files unless --all flag is set
        if (ignored && !context.args.showAll) {
          continue;
        }

        // Detect file type
        const fileType = await detectFileType(fullPath, fileStats);

        entries.push({
          name: file,
          url: fileUrl,
          isDirectory,
          size: fileStats.size,
          mtime: fileStats.mtime,
          fileType,
          ignored
        });
      } catch (err) {
        // Skip files we can't stat
        console.error(`Error reading ${file}: ${err.message}`);
      }
    }

    // Get directory name
    const dirName = path.basename(filePath) || '/';

    // Generate HTML using directory template
    const html = createDirectoryTemplate({
      dirName,
      entries,
      urlPath,
      showAll: context.args.showAll || false,
      escapeHtml
    });

    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(html);
  } catch (err) {
    res.writeHead(500, { 'Content-Type': 'text/plain' });
    res.end(`Error reading directory: ${err.message}`);
  }
}

async function handleMarkdown(filePath, urlPath, stats, res, context) {
  const { createMarkdownTemplate } = require('./templates/markdown.js');
  const { escapeHtml } = require('./utils.js');

  try {
    // Read the markdown file content
    const data = await fs.readFile(filePath, 'utf8');

    // Render markdown to HTML
    const html = context.md.render(data);

    // Get the file name
    const fileName = path.basename(filePath);

    // Generate HTML using the markdown template
    const fullHtml = createMarkdownTemplate({
      fileName,
      html,
      urlPath,
      escapeHtml
    });

    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(fullHtml);
  } catch (err) {
    res.writeHead(500, { 'Content-Type': 'text/plain' });
    res.end(`Error reading file: ${err.message}`);
  }
}

async function handleCode(filePath, urlPath, stats, res, context) {
  const { createCodeTemplate, getLanguageFromExtension } = require('./templates/code.js');
  const { escapeHtml } = require('./utils.js');

  try {
    // Read the file content
    const code = await fs.readFile(filePath, 'utf8');

    // Get the file extension and determine language
    const ext = path.extname(filePath);
    const fileName = path.basename(filePath);
    const language = getLanguageFromExtension(ext);

    let highlightedCode;

    // Try to highlight with the detected language
    if (language) {
      try {
        const result = context.hljs.highlight(code, { language });
        highlightedCode = result.value;
      } catch (err) {
        // If language-specific highlighting fails, fall back to auto-detection
        console.error(`Error highlighting with language ${language}, falling back to auto: ${err.message}`);
        const result = context.hljs.highlightAuto(code);
        highlightedCode = result.value;
      }
    } else {
      // No language detected, use auto-detection
      const result = context.hljs.highlightAuto(code);
      highlightedCode = result.value;
    }

    // Generate HTML using the code template
    const html = createCodeTemplate({
      fileName,
      code: highlightedCode,
      urlPath,
      language: language || 'plaintext',
      escapeHtml
    });

    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(html);
  } catch (err) {
    res.writeHead(500, { 'Content-Type': 'text/plain' });
    res.end(`Error reading file: ${err.message}`);
  }
}

async function handleStatic(filePath, urlPath, stats, res, context, fileType) {
  // Map file types to MIME types
  const mimeTypes = {
    // Images
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.jpeg': 'image/jpeg',
    '.gif': 'image/gif',
    '.webp': 'image/webp',
    '.svg': 'image/svg+xml',
    '.ico': 'image/x-icon',
    '.bmp': 'image/bmp',
    // Video
    '.mp4': 'video/mp4',
    '.webm': 'video/webm',
    '.ogg': 'video/ogg',
    '.mov': 'video/quicktime',
    '.avi': 'video/x-msvideo',
    '.mkv': 'video/x-matroska',
    // Audio
    '.mp3': 'audio/mpeg',
    '.wav': 'audio/wav',
    '.m4a': 'audio/mp4',
    '.flac': 'audio/flac',
    '.aac': 'audio/aac',
    // PDF
    '.pdf': 'application/pdf'
  };

  const ext = path.extname(filePath).toLowerCase();
  const mimeType = mimeTypes[ext] || 'application/octet-stream';

  try {
    const content = await fs.readFile(filePath);
    res.writeHead(200, {
      'Content-Type': mimeType,
      'Content-Length': content.length
    });
    res.end(content);
  } catch (err) {
    res.writeHead(500, { 'Content-Type': 'text/plain' });
    res.end(`Error reading file: ${err.message}`);
  }
}

async function handleBinary(filePath, urlPath, stats, res, context) {
  const { createBinaryTemplate } = require('./templates/binary.js');
  const { escapeHtml } = require('./utils.js');

  // Check for download query parameter
  const url = new URL(context.req.url, `http://${context.req.headers.host}`);
  const shouldDownload = url.searchParams.get('download') === 'true';

  if (shouldDownload) {
    // Serve file for download
    try {
      const content = await fs.readFile(filePath);
      const fileName = path.basename(filePath);

      res.writeHead(200, {
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': `attachment; filename="${fileName}"`,
        'Content-Length': content.length
      });
      res.end(content);
    } catch (err) {
      res.writeHead(500, { 'Content-Type': 'text/plain' });
      res.end(`Error reading file: ${err.message}`);
    }
  } else {
    // Show preview card with metadata
    try {
      const fileName = path.basename(filePath);
      const fileSize = stats.size;
      const modified = stats.mtime;

      const html = createBinaryTemplate({
        fileName,
        filePath,
        fileSize,
        modified,
        urlPath,
        escapeHtml
      });

      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      res.end(html);
    } catch (err) {
      res.writeHead(500, { 'Content-Type': 'text/plain' });
      res.end(`Error generating preview: ${err.message}`);
    }
  }
}

module.exports = {
  handleFile,
  detectFileType
};
