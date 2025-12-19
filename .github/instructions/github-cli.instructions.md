---
applyTo: "**"
---

# GitHub CLI Usage

## ⚠️ CRITICAL: Language Requirements

**ALL GitHub CLI operations MUST use English only**

This is a **MANDATORY** requirement that applies to:
- ✅ Issue titles
- ✅ Issue descriptions and body content
- ✅ Pull request titles and descriptions
- ✅ Comments on issues and PRs
- ✅ Commit messages
- ✅ Any text content that will be posted to GitHub

**Before creating any file for GitHub CLI:**
1. ⚠️ **STOP and verify the content is in English**
2. If content is in another language (e.g., Chinese/中文), **translate it to English first**
3. Only proceed after translation is complete

**Why this is critical:**
- GitHub is an international platform
- English ensures accessibility for all contributors
- Maintains consistency across the project
- Required by project guidelines

## Global Rules

- **Always use English** for all GitHub CLI operations including issue titles, descriptions, PR comments, and any other text content
- Check existing labels with `gh label list` before creating issues
- Use `--body-file` for all long or complex content (see Best Practices below)

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

1. **⚠️ VERIFY: All content must be in English** (see Language Requirements above)
2. Always run `gh label list` first to check available labels before creating an issue
3. **For long/complex content, use `--body-file`** (see Best Practices above)
4. Structure feature issues with these sections:
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
⚠️ VERIFY: Comment content must be in English** (see Language Requirements above)
2. **Always use `--body-file` for multi-line or markdown content**
3. Create a temporary file with the comment content in **English**
4. Create a temporary file with the comment content
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
gh 
## Pre-Flight Checklist

Before executing any `gh` command that posts content:

- [ ] ⚠️ **Is all content in English?** (Issue title, body, comments)
- [ ] Did you run `gh label list` to check available labels?
- [ ] Are you using `--body-file` for complex/long content?
- [ ] Have you reviewed the content for typos and clarity?

**If any item is unchecked, STOP and fix it before proceeding.**```
