// Base HTML template
const { baseStyles } = require('../styles.js');

/**
 * Generate breadcrumb navigation HTML
 * @param {string} urlPath - The current URL path
 * @param {Function} escapeHtml - Function to escape HTML special characters
 * @returns {string} Breadcrumb HTML
 */
function generateBreadcrumb(urlPath, escapeHtml) {
  // Split path into parts and filter out empty strings
  const parts = urlPath.split('/').filter(Boolean);

  // Start with home link
  let breadcrumbHtml = '<a href="/">🏠</a>';

  // Build path incrementally
  let currentPath = '';

  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];
    currentPath += '/' + part;

    // Add separator
    breadcrumbHtml += ' <span class="separator">/</span> ';

    // If this is the last part, show it as current (non-clickable)
    if (i === parts.length - 1) {
      breadcrumbHtml += `<span class="current">${escapeHtml(decodeURIComponent(part))}</span>`;
    } else {
      // Otherwise, make it a clickable link
      breadcrumbHtml += `<a href="${escapeHtml(currentPath)}">${escapeHtml(decodeURIComponent(part))}</a>`;
    }
  }

  return breadcrumbHtml;
}

/**
 * Create base HTML template
 * @param {Object} options - Template options
 * @param {string} options.title - Page title
 * @param {string} options.breadcrumb - Breadcrumb HTML
 * @param {string} options.content - Main content HTML
 * @param {string} [options.extraStyles=''] - Additional CSS styles
 * @param {string} [options.extraHead=''] - Additional head content
 * @returns {string} Complete HTML document
 */
function createBaseTemplate({ title, breadcrumb, content, extraStyles = '', extraHead = '' }) {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${title} - lookit</title>
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css">
  <style>${baseStyles}${extraStyles}</style>
  ${extraHead}
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="breadcrumb">${breadcrumb}</div>
    </div>
    <div class="content">
      ${content}
    </div>
  </div>
</body>
</html>`;
}

module.exports = { createBaseTemplate, generateBreadcrumb };
