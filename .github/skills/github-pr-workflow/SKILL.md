---
name: github-pr-workflow
description: "Executes the mandatory LowKey workflow: verify tests, create GitHub issue, create dedicated branch, commit changes, push, and open Pull Request linking to the issue."
user-invocable: true
argument-hint: "Summary of changes to publish as issue and PR"
---

# GitHub PR Workflow Skill

This skill automates the required workflow for shipping any code changes to LowKey.

## Instructions

Whenever changes are ready to be committed or when invoked:

1. **Verify quality gates**:
   ```bash
   go test ./...
   go build -o /dev/null .
   ```

2. **Check or create GitHub issue**:
   ```bash
   gh issue list --repo Ninido/lowkey --state all
   gh issue create --repo Ninido/lowkey --title "<Type>: <Title>" --body "<Body>"
   ```

3. **Ensure dedicated branch**:
   - `feat/issue-<num>-<slug>`
   - `fix/issue-<num>-<slug>`
   - `refactor/issue-<num>-<slug>`
   ```bash
   git checkout -b <branch-name>
   ```

4. **Semantic commit**:
   ```bash
   git add <staged-files>
   git commit -m "<type>: <summary> (closes #<num>)"
   ```

5. **Push and create Pull Request**:
   ```bash
   git push -u origin <branch-name>
   gh pr create --repo Ninido/lowkey --base main --head <branch-name> --title "<type>: <summary>" --body "## Description\n...\n\nCloses #<num>\n\n## Changes\n...\n\n## Verification\n..."
   ```

6. **Provide links** to the created issue and PR in the final response.
