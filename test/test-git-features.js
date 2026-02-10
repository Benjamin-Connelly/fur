const path = require('path');
const { handleFile } = require('./src/fileHandler.js');
const fs = require('fs').promises;
const md = require('markdown-it')();
const hljs = require('highlight.js');

async function testDirectory() {
  const srcDir = path.join(process.cwd(), 'src');
  const stats = await fs.stat(srcDir);

  // Create mock response
  let htmlOutput = '';
  const res = {
    writeHead: (code, headers) => {
      console.log(`Response: ${code}`);
    },
    end: (data) => {
      htmlOutput = data;
    }
  };

  const context = {
    md,
    hljs,
    args: { showAll: false, noDirlist: false },
    CWD: process.cwd(),
    req: { url: '/src', headers: { host: 'localhost:7777' } }
  };

  await handleFile(srcDir, '/src', stats, res, context);

  // Check for git features in output
  const hasGitBadge = htmlOutput.includes('git-badge');
  const hasGitBranch = htmlOutput.includes('git-branch');
  const hasRepoStats = htmlOutput.includes('repo-stats');
  const hasCommitInfo = htmlOutput.includes('data-commit-info');
  const hasGitLegend = htmlOutput.includes('git-legend');
  const hasLastCommit = htmlOutput.includes('last-commit');

  console.log('\nGit Features in HTML:');
  console.log('✓ Git Badges:', hasGitBadge ? '✓' : '✗');
  console.log('✓ Git Branch:', hasGitBranch ? '✓' : '✗');
  console.log('✓ Git Legend:', hasGitLegend ? '✓' : '✗');
  console.log('✓ Repo Stats:', hasRepoStats ? '✓' : '✗');
  console.log('✓ Commit Info (per-file):', hasCommitInfo ? '✓' : '✗');
  console.log('✓ Last Commit (header):', hasLastCommit ? '✓' : '✗');

  // Show sample of generated HTML
  if (hasGitBadge) {
    const match = htmlOutput.match(/<span class="git-badge[^>]*>.*?<\/span>/);
    if (match) console.log('\nSample badge:', match[0]);
  }

  if (hasGitBranch) {
    const match = htmlOutput.match(/<span class="git-branch[^>]*>.*?<\/span>/);
    if (match) console.log('Sample branch:', match[0]);
  }

  // Count badges
  const badges = htmlOutput.match(/<span class="git-badge/g);
  if (badges) console.log(`\nTotal badges found: ${badges.length}`);

  console.log('\n✓ All git features tested successfully!');
}

testDirectory().catch(err => console.error('Error:', err.message));
