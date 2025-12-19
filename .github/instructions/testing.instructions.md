---
applyTo: "**/*.go, **/*_test.go"
---

# Testing Guidelines

## Test Execution Priority

When running tests for Go projects, follow this priority order:

### 1. **Primary: Use IDE Testing Tools** (Recommended)

**Always prefer using the IDE's built-in testing capabilities:**

- Use the `runTests` tool provided by the IDE
- This provides better integration, faster execution, and richer feedback
- Supports targeted test execution (specific files or test functions)
- Provides detailed test results with pass/fail counts

**Example usage:**
```
Run tests for specific files:
- runTests with files parameter

Run specific test functions:
- runTests with testNames parameter

Run with coverage:
- runTests with mode="coverage"
```

**Benefits:**
- ✅ Faster test execution
- ✅ Better error reporting and formatting
- ✅ Integrated with IDE debugging capabilities
- ✅ Supports test filtering and targeting
- ✅ Real-time test status updates

### 2. **Fallback: Terminal Execution**

**Only use terminal commands when IDE tools are unavailable or insufficient:**

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/rancher/...

# Run specific test function
go test -v -run TestFunctionName ./path/to/package

# Run tests with coverage
go test -cover ./...

# Run tests with detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Best Practices

1. **Always run tests after making code changes** to ensure nothing is broken
2. **Run targeted tests first** during development (faster feedback loop)
3. **Run full test suite** before committing or creating pull requests
4. **Use table-driven tests** for multiple similar test cases
5. **Write meaningful test names** that describe what is being tested
6. **Include both positive and negative test cases**

## Test Organization

- Place test files next to the code they test
- Use `_test.go` suffix for test files
- Use descriptive test function names: `TestFunctionName_Scenario`
- Group related tests using subtests with `t.Run()`

## When to Use Each Approach

| Scenario | Recommended Tool |
|----------|------------------|
| Development (quick feedback) | IDE `runTests` tool |
| Debugging specific tests | IDE `runTests` tool |
| CI/CD pipeline | Terminal `go test` |
| Coverage analysis | IDE `runTests` with coverage mode |
| Running all tests locally | Either (IDE preferred) |
| Testing specific packages | IDE `runTests` tool |
