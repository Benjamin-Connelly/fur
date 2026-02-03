const { execFileSync } = require('child_process');
const path = require('path');
const fs = require('fs');

// Cache for git data with 5-second TTL
const gitCache = new Map();
const CACHE_TTL = 5000; // 5 seconds
const COMMIT_CACHE_TTL = 30000; // 30 seconds for commit metadata

/**
 * Check if git is installed on the system
 */
function isGitInstalled() {
  try {
    execFileSync('git', ['--version'], { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

/**
 * Walk up directory tree to find .git directory
 * @param {string} dirPath - Starting directory path
 * @returns {string|null} - Path to git repository root, or null if not found
 */
function findGitRoot(dirPath) {
  if (!isGitInstalled()) {
    return null;
  }

  try {
    let currentPath = path.resolve(dirPath);
    const rootPath = path.parse(currentPath).root;

    while (currentPath !== rootPath) {
      const gitPath = path.join(currentPath, '.git');
      if (fs.existsSync(gitPath)) {
        return currentPath;
      }
      currentPath = path.dirname(currentPath);
    }

    return null;
  } catch {
    return null;
  }
}

/**
 * Check if a directory is a git repository
 * @param {string} dirPath - Directory path to check
 * @returns {boolean}
 */
function isGitRepository(dirPath) {
  return findGitRoot(dirPath) !== null;
}

/**
 * Get cached data or execute function if cache expired
 * @param {string} cacheKey - Cache key
 * @param {Function} fetchFn - Function to execute if cache miss
 * @param {number} ttl - Time to live in milliseconds
 * @returns {*} - Cached or fresh data
 */
function getCached(cacheKey, fetchFn, ttl = CACHE_TTL) {
  const cached = gitCache.get(cacheKey);
  const now = Date.now();

  if (cached && (now - cached.timestamp) < ttl) {
    return cached.data;
  }

  try {
    const data = fetchFn();
    gitCache.set(cacheKey, { data, timestamp: now });
    return data;
  } catch (error) {
    // If fetch fails, return stale cache if available
    return cached ? cached.data : null;
  }
}

/**
 * Parse git status --porcelain output
 * @param {string} repoRoot - Repository root path
 * @returns {Map<string, string>|null} - Map of file paths to status codes
 */
function getGitStatus(repoRoot) {
  if (!repoRoot) {
    return null;
  }

  const cacheKey = `status:${repoRoot}`;

  return getCached(cacheKey, () => {
    try {
      const output = execFileSync('git', ['status', '--porcelain'], {
        cwd: repoRoot,
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore']
      });

      const statusMap = new Map();

      output.split('\n').forEach(line => {
        if (!line.trim()) return;

        // Format: "XY filename" where X=staged, Y=unstaged
        const statusCode = line.substring(0, 2);
        const filePath = line.substring(3).trim();

        // Handle renamed files (format: "R  old.txt -> new.txt")
        const actualPath = filePath.includes(' -> ')
          ? filePath.split(' -> ')[1]
          : filePath;

        const fullPath = path.join(repoRoot, actualPath);

        // Determine primary status (prioritize staged over unstaged)
        let status;
        if (statusCode[0] !== ' ' && statusCode[0] !== '?') {
          status = statusCode[0]; // Staged status
        } else if (statusCode[1] !== ' ') {
          status = statusCode[1]; // Unstaged status
        } else if (statusCode === '??') {
          status = '??'; // Untracked
        }

        if (status) {
          statusMap.set(fullPath, status);
        }
      });

      return statusMap;
    } catch {
      return new Map();
    }
  });
}

/**
 * Get git status for a specific file
 * @param {string} filePath - Full path to file
 * @param {string} repoRoot - Repository root path
 * @param {Map<string, string>} statusMap - Pre-fetched status map
 * @returns {string|null} - Status code or null
 */
function getFileGitStatus(filePath, repoRoot, statusMap) {
  if (!statusMap || !repoRoot) {
    return null;
  }

  // Check exact path match
  if (statusMap.has(filePath)) {
    return statusMap.get(filePath);
  }

  // Check if it's a directory with modified files inside
  const isDirectory = fs.statSync(filePath).isDirectory();
  if (isDirectory) {
    for (const [modifiedPath, status] of statusMap.entries()) {
      if (modifiedPath.startsWith(filePath + path.sep)) {
        return 'M'; // Mark directory as modified if any child is modified
      }
    }
  }

  return null;
}

/**
 * Get current git branch name
 * @param {string} repoRoot - Repository root path
 * @returns {string|null} - Branch name or null
 */
function getCurrentBranch(repoRoot) {
  if (!repoRoot) {
    return null;
  }

  const cacheKey = `branch:${repoRoot}`;

  return getCached(cacheKey, () => {
    try {
      const branch = execFileSync('git', ['branch', '--show-current'], {
        cwd: repoRoot,
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore']
      }).trim();

      // If empty, might be in detached HEAD state
      if (!branch) {
        const commitHash = execFileSync('git', ['rev-parse', '--short', 'HEAD'], {
          cwd: repoRoot,
          encoding: 'utf8',
          stdio: ['pipe', 'pipe', 'ignore']
        }).trim();
        return `detached@${commitHash}`;
      }

      return branch;
    } catch {
      return null;
    }
  });
}

/**
 * Get last commit info for a file
 * @param {string} filePath - Full path to file
 * @param {string} repoRoot - Repository root path
 * @returns {string|null} - Formatted commit info or null
 */
function getLastCommit(filePath, repoRoot) {
  if (!repoRoot) {
    return null;
  }

  const cacheKey = `commit:${filePath}`;

  return getCached(cacheKey, () => {
    try {
      const relativePath = path.relative(repoRoot, filePath);

      const output = execFileSync(
        'git',
        ['log', '-1', '--format=%an | %ar | %s', '--', relativePath],
        {
          cwd: repoRoot,
          encoding: 'utf8',
          stdio: ['pipe', 'pipe', 'ignore']
        }
      ).trim();

      return output || null;
    } catch {
      return null;
    }
  }, COMMIT_CACHE_TTL);
}

/**
 * Get repository-wide statistics
 * @param {string} repoRoot - Repository root path
 * @returns {Object|null} - Repo stats object or null
 */
function getRepoStats(repoRoot) {
  if (!repoRoot) {
    return null;
  }

  const cacheKey = `stats:${repoRoot}`;

  return getCached(cacheKey, () => {
    try {
      // Get tracked files count
      const trackedFilesOutput = execFileSync('git', ['ls-files'], {
        cwd: repoRoot,
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore']
      }).trim();
      const trackedFiles = trackedFilesOutput.split('\n').filter(line => line.trim()).length.toString();

      // Get status counts
      const statusOutput = execFileSync('git', ['status', '--porcelain'], {
        cwd: repoRoot,
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore']
      });

      let modified = 0;
      let staged = 0;
      let untracked = 0;

      statusOutput.split('\n').forEach(line => {
        if (!line.trim()) return;
        const statusCode = line.substring(0, 2);

        if (statusCode === '??') {
          untracked++;
        } else if (statusCode[0] !== ' ' && statusCode[0] !== '?') {
          staged++;
        } else if (statusCode[1] !== ' ') {
          modified++;
        }
      });

      // Get total commits
      const totalCommits = execFileSync('git', ['rev-list', '--count', 'HEAD'], {
        cwd: repoRoot,
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore']
      }).trim();

      // Get last commit time
      const lastCommit = execFileSync('git', ['log', '-1', '--format=%ar'], {
        cwd: repoRoot,
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore']
      }).trim();

      return {
        trackedFiles: parseInt(trackedFiles, 10) || 0,
        modified,
        staged,
        untracked,
        totalCommits: parseInt(totalCommits, 10) || 0,
        lastCommit
      };
    } catch {
      return null;
    }
  }, 10000); // 10 second cache for stats
}

/**
 * Batch fetch commit info for multiple files
 * @param {Array<string>} filePaths - Array of file paths
 * @param {string} repoRoot - Repository root path
 * @returns {Map<string, string>} - Map of file paths to commit info
 */
function batchGetLastCommits(filePaths, repoRoot) {
  if (!repoRoot || !filePaths.length) {
    return new Map();
  }

  const commits = new Map();

  // Check cache first
  filePaths.forEach(filePath => {
    const cacheKey = `commit:${filePath}`;
    const cached = gitCache.get(cacheKey);
    const now = Date.now();

    if (cached && (now - cached.timestamp) < COMMIT_CACHE_TTL) {
      commits.set(filePath, cached.data);
    }
  });

  // Fetch uncached files
  const uncachedFiles = filePaths.filter(fp => !commits.has(fp));

  if (uncachedFiles.length === 0) {
    return commits;
  }

  // Batch fetch using single git log command
  try {
    uncachedFiles.forEach(filePath => {
      const commit = getLastCommit(filePath, repoRoot);
      if (commit) {
        commits.set(filePath, commit);
      }
    });
  } catch {
    // Ignore errors, return partial results
  }

  return commits;
}

/**
 * Clear all git cache (useful for testing)
 */
function clearCache() {
  gitCache.clear();
}

module.exports = {
  findGitRoot,
  isGitRepository,
  getGitStatus,
  getFileGitStatus,
  getCurrentBranch,
  getLastCommit,
  getRepoStats,
  batchGetLastCommits,
  clearCache
};
