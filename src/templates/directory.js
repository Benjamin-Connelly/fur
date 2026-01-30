// Directory listing template
const { createBaseTemplate, generateBreadcrumb } = require('./base.js');

/**
 * Get file icon emoji based on entry type
 * @param {Object} entry - Directory entry object
 * @param {string} fileType - File type string
 * @returns {string} Icon emoji
 */
function getFileIcon(entry, fileType) {
  // Directory icon
  if (entry.isDirectory) {
    return '📁';
  }

  // File type icons
  const iconMap = {
    'markdown': '📝',
    'code': '💻',
    'image': '🖼️',
    'video': '🎬',
    'audio': '🎵',
    'pdf': '📄',
    'binary': '📦',
    'text': '📄'
  };

  return iconMap[fileType] || '📄';
}

/**
 * Format file size in human-readable format
 * @param {number} bytes - File size in bytes
 * @returns {string} Formatted size string
 */
function formatFileSize(bytes) {
  if (bytes === 0) return '0 B';

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + units[i];
}

/**
 * Format date relative to now
 * @param {Date} date - Date to format
 * @returns {string} Formatted date string
 */
function formatDate(date) {
  const now = new Date();
  const diffMs = now - date;
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffDays === 0) {
    return 'today';
  } else if (diffDays === 1) {
    return 'yesterday';
  } else if (diffDays < 7) {
    return `${diffDays} days ago`;
  } else {
    // Format as "Jan 30, 2026"
    const options = { year: 'numeric', month: 'short', day: 'numeric' };
    return date.toLocaleDateString('en-US', options);
  }
}

/**
 * Create directory listing template
 * @param {Object} options - Template options
 * @param {string} options.dirName - Name of the directory
 * @param {Array} options.entries - Array of directory entries
 * @param {string} options.urlPath - URL path for breadcrumb
 * @param {boolean} options.showAll - Whether to show ignored files
 * @param {Function} options.escapeHtml - Function to escape HTML
 * @returns {string} Complete HTML document
 */
function createDirectoryTemplate({ dirName, entries, urlPath, showAll, escapeHtml }) {
  const breadcrumb = generateBreadcrumb(urlPath, escapeHtml);

  // Sort entries: directories first, then alphabetically
  const sortedEntries = [...entries].sort((a, b) => {
    // Directories before files
    if (a.isDirectory && !b.isDirectory) return -1;
    if (!a.isDirectory && b.isDirectory) return 1;

    // Then alphabetically by name (case-insensitive)
    return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
  });

  // Add parent directory link if not root
  const isRoot = urlPath === '/' || urlPath === '';
  let fileListHtml = '';

  if (!isRoot) {
    fileListHtml += `
      <a href=".." class="file-entry parent-dir">
        <div class="file-icon">📁</div>
        <div class="file-info">
          <div class="file-name">..</div>
        </div>
      </a>
    `;
  }

  // Generate file list HTML
  for (const entry of sortedEntries) {
    const icon = getFileIcon(entry, entry.fileType);
    const sizeStr = entry.isDirectory ? '' : formatFileSize(entry.size);
    const dateStr = formatDate(entry.mtime);

    // Gray out ignored files if they're shown
    const ignoredClass = entry.ignored ? ' ignored' : '';

    fileListHtml += `
      <a href="${escapeHtml(entry.url)}" class="file-entry${ignoredClass}">
        <div class="file-icon">${icon}</div>
        <div class="file-info">
          <div class="file-name">${escapeHtml(entry.name)}</div>
          <div class="file-meta">
            <span class="file-size">${sizeStr}</span>
            ${sizeStr ? '<span class="separator">•</span>' : ''}
            <span class="file-date">${dateStr}</span>
          </div>
        </div>
      </a>
    `;
  }

  const itemCount = entries.length;
  const itemLabel = itemCount === 1 ? 'item' : 'items';

  const content = `
    <div class="directory-header">
      <div class="directory-icon">📁</div>
      <div class="directory-info">
        <div class="directory-name">${escapeHtml(dirName)}</div>
        <div class="directory-meta">${itemCount} ${itemLabel}</div>
      </div>
    </div>
    <div class="file-list">
      ${fileListHtml}
    </div>
  `;

  const extraStyles = `
    .directory-header {
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 1.5rem;
      background: #f8f9fa;
      border-radius: 8px 8px 0 0;
      border-bottom: 2px solid #e9ecef;
    }

    .directory-icon {
      font-size: 2rem;
      line-height: 1;
    }

    .directory-info {
      flex: 1;
    }

    .directory-name {
      font-size: 1.25rem;
      font-weight: 600;
      color: #212529;
      margin-bottom: 0.25rem;
    }

    .directory-meta {
      font-size: 0.875rem;
      color: #6c757d;
    }

    .file-list {
      background: white;
      border-radius: 0 0 8px 8px;
      overflow: hidden;
    }

    .file-entry {
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 1rem 1.5rem;
      text-decoration: none;
      color: inherit;
      border-bottom: 1px solid #e9ecef;
      transition: background-color 0.15s ease;
    }

    .file-entry:last-child {
      border-bottom: none;
    }

    .file-entry:hover {
      background-color: #f8f9fa;
    }

    .file-entry.parent-dir {
      background-color: #f8f9fa;
      font-weight: 500;
    }

    .file-entry.parent-dir:hover {
      background-color: #e9ecef;
    }

    .file-entry.ignored {
      opacity: 0.5;
    }

    .file-icon {
      font-size: 1.5rem;
      line-height: 1;
      flex-shrink: 0;
    }

    .file-info {
      flex: 1;
      min-width: 0;
    }

    .file-name {
      font-size: 1rem;
      font-weight: 500;
      color: #212529;
      margin-bottom: 0.25rem;
      word-wrap: break-word;
    }

    .file-meta {
      display: flex;
      gap: 0.5rem;
      align-items: center;
      font-size: 0.875rem;
      color: #6c757d;
    }

    .file-size {
      font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
      font-size: 0.8125rem;
    }

    .separator {
      color: #dee2e6;
    }

    .file-date {
      color: #6c757d;
    }

    /* Responsive adjustments */
    @media (max-width: 768px) {
      .file-entry {
        padding: 0.875rem 1rem;
      }

      .file-icon {
        font-size: 1.25rem;
      }

      .file-name {
        font-size: 0.9375rem;
      }

      .file-meta {
        font-size: 0.8125rem;
      }
    }
  `;

  return createBaseTemplate({
    title: dirName,
    breadcrumb,
    content,
    extraStyles
  });
}

module.exports = { createDirectoryTemplate, getFileIcon, formatFileSize, formatDate };
