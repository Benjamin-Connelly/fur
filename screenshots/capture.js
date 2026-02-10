const { chromium } = require('playwright');
const path = require('path');

const outDir = __dirname;
const BASE = 'http://localhost:7777';

const viewport = { width: 1280, height: 800 };

async function capture() {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport,
    deviceScaleFactor: 2,
  });

  const shots = [
    {
      name: 'hero-directory',
      url: `${BASE}/`,
      description: 'Root directory listing with git badges and dark theme',
    },
    {
      name: 'markdown-rendering',
      url: `${BASE}/README.md`,
      description: 'GitHub-style markdown rendering',
    },
    {
      name: 'code-highlighting',
      url: `${BASE}/src/index.js`,
      description: 'Syntax-highlighted code view',
    },
    {
      name: 'git-directory',
      url: `${BASE}/src`,
      description: 'Directory listing with per-file git status',
    },
  ];

  for (const shot of shots) {
    const page = await context.newPage();
    await page.goto(shot.url, { waitUntil: 'networkidle' });
    await page.waitForTimeout(500);

    const filePath = path.join(outDir, `${shot.name}.png`);
    await page.screenshot({ path: filePath, fullPage: false });
    console.log(`captured: ${shot.name}.png - ${shot.description}`);
    await page.close();
  }

  await browser.close();
  console.log(`\nAll screenshots saved to ${outDir}`);
}

capture().catch(err => {
  console.error('Screenshot capture failed:', err.message);
  process.exit(1);
});
