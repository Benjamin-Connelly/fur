// Code file template
const { createBaseTemplate, generateBreadcrumb } = require('./base.js');

/**
 * Map file extension to highlight.js language identifier
 * @param {string} ext - File extension (with or without leading dot)
 * @returns {string|null} Language identifier or null if not found
 */
function getLanguageFromExtension(ext) {
  // Normalize extension (remove leading dot, lowercase)
  const normalized = ext.toLowerCase().replace(/^\./, '');

  const languageMap = {
    // JavaScript/TypeScript
    'js': 'javascript',
    'jsx': 'javascript',
    'ts': 'typescript',
    'tsx': 'typescript',
    'mjs': 'javascript',
    'cjs': 'javascript',

    // Python
    'py': 'python',
    'pyw': 'python',
    'pyx': 'python',
    'pyi': 'python',

    // Ruby
    'rb': 'ruby',
    'rake': 'ruby',
    'gemspec': 'ruby',

    // Go
    'go': 'go',

    // Rust
    'rs': 'rust',

    // Java/Kotlin/Scala
    'java': 'java',
    'kt': 'kotlin',
    'kts': 'kotlin',
    'scala': 'scala',

    // C/C++/C#
    'c': 'c',
    'h': 'c',
    'cpp': 'cpp',
    'hpp': 'cpp',
    'cc': 'cpp',
    'cxx': 'cpp',
    'cs': 'csharp',

    // Shell
    'sh': 'bash',
    'bash': 'bash',
    'zsh': 'bash',
    'fish': 'fish',

    // Web
    'html': 'html',
    'htm': 'html',
    'css': 'css',
    'scss': 'scss',
    'sass': 'sass',
    'less': 'less',

    // Config/Data
    'json': 'json',
    'jsonc': 'json',
    'yaml': 'yaml',
    'yml': 'yaml',
    'toml': 'toml',
    'xml': 'xml',

    // Other languages
    'php': 'php',
    'swift': 'swift',
    'r': 'r',
    'm': 'objectivec',
    'sql': 'sql',
    'pl': 'perl',
    'lua': 'lua',

    // Markdown-like
    'md': 'markdown',
    'mdx': 'markdown',
    'txt': 'plaintext',
    'log': 'plaintext',
    'csv': 'plaintext',
    'tsv': 'plaintext',

    // Build/Config
    'dockerfile': 'dockerfile',
    'makefile': 'makefile',
    'cmake': 'cmake',
    'gradle': 'gradle',

    // Misc
    'vue': 'vue',
    'svelte': 'html',
    'astro': 'html',
    'graphql': 'graphql',
    'proto': 'protobuf'
  };

  return languageMap[normalized] || null;
}

/**
 * Create code file template with syntax highlighting
 * @param {Object} options - Template options
 * @param {string} options.fileName - Name of the file
 * @param {string} options.code - Code content
 * @param {string} options.urlPath - URL path for breadcrumb
 * @param {string} options.language - Programming language
 * @param {Function} options.escapeHtml - Function to escape HTML
 * @returns {string} Complete HTML document
 */
function createCodeTemplate({ fileName, code, urlPath, language, escapeHtml }) {
  const breadcrumb = generateBreadcrumb(urlPath, escapeHtml);

  // Get language display name (capitalize first letter)
  const languageDisplay = language
    ? language.charAt(0).toUpperCase() + language.slice(1)
    : 'Plain Text';

  // Language badge colors
  const languageColors = {
    'javascript': '#f7df1e',
    'typescript': '#3178c6',
    'python': '#3776ab',
    'ruby': '#cc342d',
    'go': '#00add8',
    'rust': '#dea584',
    'java': '#007396',
    'kotlin': '#7f52ff',
    'scala': '#dc322f',
    'c': '#555555',
    'cpp': '#00599c',
    'csharp': '#239120',
    'bash': '#4eaa25',
    'fish': '#4eaa25',
    'html': '#e34c26',
    'css': '#1572b6',
    'scss': '#cc6699',
    'sass': '#cc6699',
    'less': '#1d365d',
    'json': '#000000',
    'yaml': '#cb171e',
    'toml': '#9c4221',
    'xml': '#005a9c',
    'php': '#777bb4',
    'swift': '#ffac45',
    'r': '#276dc3',
    'objectivec': '#438eff',
    'sql': '#e38c00',
    'perl': '#39457e',
    'lua': '#000080',
    'markdown': '#083fa1',
    'plaintext': '#666666',
    'dockerfile': '#384d54',
    'makefile': '#427819',
    'cmake': '#064f8c',
    'gradle': '#02303a',
    'vue': '#42b883',
    'graphql': '#e10098',
    'protobuf': '#346ad1'
  };

  const languageColor = languageColors[language] || '#666666';

  const content = `
    <div class="file-header">
      <div class="file-icon">💻</div>
      <div class="file-info">
        <div class="file-name">${escapeHtml(fileName)}</div>
        <div class="file-meta">
          <span class="language-badge" style="background-color: ${languageColor};">${escapeHtml(languageDisplay)}</span>
        </div>
      </div>
    </div>
    <div class="code-container">
      <pre><code class="hljs ${escapeHtml(language || '')}">${code}</code></pre>
    </div>
  `;

  const extraStyles = `
    .file-header {
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 1.5rem;
      background: #f8f9fa;
      border-radius: 8px 8px 0 0;
      border-bottom: 2px solid #e9ecef;
    }

    .file-icon {
      font-size: 2rem;
      line-height: 1;
    }

    .file-info {
      flex: 1;
    }

    .file-name {
      font-size: 1.25rem;
      font-weight: 600;
      color: #212529;
      margin-bottom: 0.5rem;
    }

    .file-meta {
      display: flex;
      gap: 0.5rem;
      align-items: center;
    }

    .language-badge {
      display: inline-block;
      padding: 0.25rem 0.75rem;
      border-radius: 12px;
      font-size: 0.75rem;
      font-weight: 600;
      color: white;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }

    .code-container {
      background: #0d1117;
      border-radius: 0 0 8px 8px;
      overflow: hidden;
    }

    .code-container pre {
      margin: 0;
      padding: 1.5rem;
      overflow-x: auto;
    }

    .code-container code {
      display: block;
      font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
      font-size: 0.9rem;
      line-height: 1.6;
      color: #c9d1d9;
    }

    /* Override highlight.js styles for better visibility */
    .hljs {
      background: transparent !important;
      padding: 0 !important;
    }
  `;

  return createBaseTemplate({
    title: fileName,
    breadcrumb,
    content,
    extraStyles
  });
}

module.exports = { createCodeTemplate, getLanguageFromExtension }
