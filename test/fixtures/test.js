// Test JavaScript file with syntax highlighting
const express = require('express');
const app = express();

/**
 * Calculate the factorial of a number
 * @param {number} n - The number to calculate factorial for
 * @returns {number} The factorial result
 */
function factorial(n) {
  if (n <= 1) return 1;
  return n * factorial(n - 1);
}

// Middleware to parse JSON
app.use(express.json());

// Route handler
app.get('/api/factorial/:number', (req, res) => {
  const num = parseInt(req.params.number, 10);

  if (isNaN(num) || num < 0) {
    return res.status(400).json({
      error: 'Invalid number',
      message: 'Please provide a non-negative integer'
    });
  }

  const result = factorial(num);
  res.json({ number: num, factorial: result });
});

// Start server
const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});

// Export for testing
module.exports = { factorial };
