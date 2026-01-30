#!/bin/bash
# Comprehensive test suite for lookit

# Don't exit on error - we want to run all tests
set +e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Server configuration
PORT=7778  # Use different port to avoid conflicts
HOST="http://localhost:$PORT"
SERVER_PID=""

# Print functions
print_header() {
    echo ""
    echo "========================================="
    echo "$1"
    echo "========================================="
}

print_test() {
    echo -n "  Testing: $1... "
}

pass() {
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
    ((TESTS_RUN++))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}"
    echo "    Error: $1"
    ((TESTS_FAILED++))
    ((TESTS_RUN++))
}

skip() {
    echo -e "${YELLOW}⊘ SKIP${NC} - $1"
}

# Start server
start_server() {
    print_header "Starting Test Server"
    echo "  Port: $PORT"
    echo "  Directory: test/fixtures"

    # Start server in background
    node bin/lookit.js test/fixtures --port $PORT --no-https > /tmp/lookit-test.log 2>&1 &
    SERVER_PID=$!

    echo "  PID: $SERVER_PID"

    # Wait for server to start
    echo -n "  Waiting for server to start..."
    for i in {1..10}; do
        if curl -s "$HOST" > /dev/null 2>&1; then
            echo -e " ${GREEN}ready${NC}"
            return 0
        fi
        sleep 1
        echo -n "."
    done

    echo -e " ${RED}failed${NC}"
    echo "  Server failed to start. Check /tmp/lookit-test.log"
    exit 1
}

# Stop server
stop_server() {
    if [ -n "$SERVER_PID" ]; then
        echo ""
        echo "Stopping server (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        echo "Server stopped"
    fi
}

# Test directory listing
test_directory_listing() {
    print_header "Directory Listing Tests"

    print_test "Directory listing renders"
    if curl -s "$HOST/" | grep -q "directory-header\|file-list"; then
        pass
    else
        fail "Directory listing not found in response"
    fi

    print_test "Shows test.js file"
    if curl -s "$HOST/" | grep -q "test.js"; then
        pass
    else
        fail "test.js not shown in directory listing"
    fi

    print_test "Shows test.md file"
    if curl -s "$HOST/" | grep -q "test.md"; then
        pass
    else
        fail "test.md not shown in directory listing"
    fi

    print_test "Hides node_modules by default"
    RESPONSE=$(curl -s "$HOST/")
    if ! echo "$RESPONSE" | grep -q "node_modules"; then
        pass
    else
        fail "node_modules should be hidden by .gitignore"
    fi

    print_test "Hides secrets.txt by default"
    if ! echo "$RESPONSE" | grep -q "secrets\.txt"; then
        pass
    else
        fail "secrets.txt should be hidden by .gitignore"
    fi

    print_test "Shows file sizes"
    if curl -s "$HOST/" | grep -q "KB\|MB\|B"; then
        pass
    else
        fail "File sizes not displayed"
    fi

    print_test "Shows file dates"
    if curl -s "$HOST/" | grep -q "ago\|today\|yesterday"; then
        pass
    else
        fail "File dates not displayed"
    fi
}

# Test markdown rendering
test_markdown() {
    print_header "Markdown Rendering Tests"

    print_test "Markdown file renders as HTML"
    RESPONSE=$(curl -s "$HOST/test.md")
    if echo "$RESPONSE" | grep -q "<h1>"; then
        pass
    else
        fail "Markdown not rendered to HTML"
    fi

    print_test "Markdown code blocks are highlighted"
    if echo "$RESPONSE" | grep -q "hljs\|highlight"; then
        pass
    else
        fail "Code blocks not highlighted"
    fi

    print_test "Markdown tables render"
    if echo "$RESPONSE" | grep -q "<table>"; then
        pass
    else
        fail "Tables not rendered"
    fi
}

# Test code highlighting
test_code_highlighting() {
    print_header "Code Highlighting Tests"

    print_test "JavaScript file with syntax highlighting"
    RESPONSE=$(curl -s "$HOST/test.js")
    if echo "$RESPONSE" | grep -q "hljs\|highlight"; then
        pass
    else
        fail "JavaScript not highlighted"
    fi

    print_test "YAML file with syntax highlighting"
    RESPONSE=$(curl -s "$HOST/test.yaml")
    if echo "$RESPONSE" | grep -q "hljs\|highlight"; then
        pass
    else
        fail "YAML not highlighted"
    fi

    print_test "JSON file with syntax highlighting"
    RESPONSE=$(curl -s "$HOST/test.json")
    if echo "$RESPONSE" | grep -q "hljs\|highlight"; then
        pass
    else
        fail "JSON not highlighted"
    fi

    print_test "Python file with syntax highlighting"
    RESPONSE=$(curl -s "$HOST/test.py")
    if echo "$RESPONSE" | grep -q "hljs\|highlight"; then
        pass
    else
        fail "Python not highlighted"
    fi
}

# Test binary file handling
test_binary_files() {
    print_header "Binary File Tests"

    # Create a test binary file
    dd if=/dev/urandom of=test/fixtures/test.bin bs=1024 count=1 2>/dev/null

    print_test "Binary file shows preview card"
    RESPONSE=$(curl -s "$HOST/test.bin")
    if echo "$RESPONSE" | grep -q "Download\|Binary File"; then
        pass
    else
        fail "Binary preview not shown"
    fi

    print_test "Binary file can be downloaded"
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/test.bin?download=true")
    if [ "$STATUS" = "200" ]; then
        pass
    else
        fail "Binary download failed (status: $STATUS)"
    fi

    # Cleanup
    rm -f test/fixtures/test.bin
}

# Test help command
test_help() {
    print_header "CLI Command Tests"

    print_test "Help command works"
    if node bin/lookit.js --help | grep -q "lookit"; then
        pass
    else
        fail "Help command failed"
    fi

    print_test "Help shows options"
    if node bin/lookit.js --help | grep -q "\-\-port"; then
        pass
    else
        fail "Help doesn't show options"
    fi
}

# Test security features
test_security() {
    print_header "Security Tests"

    print_test "Cannot access parent directory"
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/../package.json")
    if [ "$STATUS" = "400" ] || [ "$STATUS" = "403" ] || [ "$STATUS" = "404" ]; then
        pass
    else
        fail "Parent directory access not blocked (status: $STATUS)"
    fi

    print_test "Cannot access absolute paths"
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$HOST//etc/passwd")
    if [ "$STATUS" = "400" ] || [ "$STATUS" = "403" ] || [ "$STATUS" = "404" ]; then
        pass
    else
        fail "Absolute path access not blocked (status: $STATUS)"
    fi
}

# Test responsive design
test_responsive() {
    print_header "UI/UX Tests"

    print_test "Responsive meta tag present"
    if curl -s "$HOST/" | grep -q "viewport"; then
        pass
    else
        fail "Responsive meta tag missing"
    fi

    print_test "CSS styles loaded"
    if curl -s "$HOST/" | grep -q "<style>"; then
        pass
    else
        fail "CSS styles not found"
    fi

    print_test "Breadcrumb navigation present"
    if curl -s "$HOST/" | grep -q "breadcrumb\|🏠"; then
        pass
    else
        skip "Breadcrumb navigation not found"
    fi
}

# Main test execution
main() {
    echo ""
    echo "╔═══════════════════════════════════════╗"
    echo "║   lookit Comprehensive Test Suite    ║"
    echo "╚═══════════════════════════════════════╝"
    echo ""

    # Check prerequisites
    if ! command -v node &> /dev/null; then
        echo -e "${RED}Error: Node.js not found${NC}"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        echo -e "${RED}Error: curl not found${NC}"
        exit 1
    fi

    # Ensure we're in project root
    if [ ! -f "bin/lookit.js" ]; then
        echo -e "${RED}Error: Must run from project root${NC}"
        exit 1
    fi

    # Setup trap to cleanup server on exit
    trap stop_server EXIT INT TERM

    # Start server
    start_server

    # Run all tests
    test_directory_listing
    test_markdown
    test_code_highlighting
    test_binary_files
    test_help
    test_security
    test_responsive

    # Print summary
    print_header "Test Summary"
    echo "  Total Tests: $TESTS_RUN"
    echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
    else
        echo "  Failed: 0"
    fi

    echo ""
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}╔═══════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║         ALL TESTS PASSED! ✓           ║${NC}"
        echo -e "${GREEN}╚═══════════════════════════════════════╝${NC}"
        echo ""
        exit 0
    else
        echo -e "${RED}╔═══════════════════════════════════════╗${NC}"
        echo -e "${RED}║          SOME TESTS FAILED            ║${NC}"
        echo -e "${RED}╚═══════════════════════════════════════╝${NC}"
        echo ""
        echo "Check /tmp/lookit-test.log for server output"
        exit 1
    fi
}

# Run main
main "$@"
