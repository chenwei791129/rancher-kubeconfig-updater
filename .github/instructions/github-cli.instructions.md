---
applyTo: "**"
---

# GitHub CLI Usage

## Global Rules

- **Always use English** for all GitHub CLI operations including issue titles, descriptions, PR comments, and any other text content

## Best Practices for Long Content

**Always use `--body-file` for issues and comments with long or complex content:**

1. **Create a temporary file** first (e.g., in `.github/` directory)
2. **Write the content** to the file
3. **Use `--body-file`** flag to reference the file

**Why this approach:**
- ✅ Avoids command-line escaping issues with quotes, backticks, and special characters
- ✅ Prevents terminal/shell parsing errors
- ✅ Works reliably across different operating systems (Windows/Linux/Mac)
- ✅ Handles multi-line content, code blocks, and markdown formatting correctly
- ❌ Direct `--body` with complex content often fails due to escaping issues

**Example workflow:**
```bash
# 1. Create content file
echo "Complex markdown content..." > .github/temp-issue.md

# 2. Use --body-file
gh issue create --title "My Title" --body-file .github/temp-issue.md --label "enhancement"
gh issue comment 123 --body-file .github/temp-comment.md
```

## Creating Issues

When creating issues using GitHub CLI:

1. Always run `gh label list` first to check available labels before creating an issue
2. **For long/complex content, use `--body-file`** (see Best Practices above)
3. Structure feature issues with these sections:
   - Description: Clear explanation of the feature
   - Implementation Plan: Step-by-step technical approach
   - Benefits: Why this feature is valuable
   - Technical Details: Specific code changes or API modifications
4. Select appropriate labels based on the output from `gh label list`
5. Example command structure:
   ```bash
   # For simple content:
   gh issue create \
     --title "Clear feature title" \
     --body "Simple one-line description" \
     --label "enhancement"
   
   # For complex content (RECOMMENDED):
   # Create file first, then:
   gh issue create \
     --title "Clear feature title" \
     --body-file .github/issue-content.md \
     --label "enhancement"
   ```

## Adding Comments to Issues

When adding comments to existing issues:

1. **Always use `--body-file` for multi-line or markdown content**
2. Create a temporary file with the comment content
3. Use the file with `gh issue comment` command

Example:
```bash
# 1. Create comment file
cat > .github/temp-comment.md << 'EOF'
## Analysis Results

Here are the findings:
- Point 1
- Point 2

```code
example code block
```
EOF

# 2. Post comment
gh issue comment 24 --body-file .github/temp-comment.md
```
