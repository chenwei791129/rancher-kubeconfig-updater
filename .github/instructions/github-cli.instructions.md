---
applyTo: "**"
---

# GitHub CLI Usage

## Global Rules

- **Always use English** for all GitHub CLI operations including issue titles, descriptions, PR comments, and any other text content

## Creating Issues

When creating issues using GitHub CLI:

1. Always run `gh label list` first to check available labels before creating an issue
2. Structure feature issues with these sections:
   - Description: Clear explanation of the feature
   - Implementation Plan: Step-by-step technical approach
   - Benefits: Why this feature is valuable
   - Technical Details: Specific code changes or API modifications
3. Select appropriate labels based on the output from `gh label list`
4. Example command structure:
   ```bash
   gh issue create \
     --title "Clear feature title" \
     --body "Structured description with markdown" \
     --label "enhancement"
   ```
