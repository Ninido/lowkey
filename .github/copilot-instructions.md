# LowKey Development & Workflow Guidelines

## Mandatory Git & GitHub Process

**CRITICAL RULE**: Never commit directly to `main`, and NEVER commit any feature, bug fix, or code modification without linking it to a GitHub Issue, a dedicated feature/fix branch, and a Pull Request.

Even if the user prompts directly for a solution, begins coding immediately, or does not mention tickets/branches:
**When the implementation/fix is ready (or before committing), ALWAYS automatically execute this process without waiting to be reminded:**

### 1. Issue First
- Check existing issues via `gh issue list --repo Ninido/lowkey --state all` or GitHub tools.
- If no issue exists covering the task, create a clear, descriptive GitHub issue via:
  ```bash
  gh issue create --repo Ninido/lowkey --title "<Type>: <Descriptive Title>" --body "<Detailed description, proposed changes, validation criteria>"
  ```
- Obtain the issue number (e.g. `#12`).

### 2. Dedicated Feature / Bugfix Branch
- Determine the appropriate branch naming convention:
  - Feature: `feat/issue-<num>-<short-slug>`
  - Bugfix: `fix/issue-<num>-<short-slug>`
  - Refactor / Chore: `refactor/issue-<num>-<short-slug>` or `chore/issue-<num>-<short-slug>`
- Ensure working tree changes are safe.
- Create and switch to the branch:
  ```bash
  git checkout -b <branch-name>
  ```

### 3. Verify Quality & Tests
- Ensure tests pass and the code compiles cleanly:
  ```bash
  go test ./...
  go build -o /dev/null .
  ```
- Never commit broken code or failing tests.

### 4. Semantic Commit
- Stage only relevant files (avoid unwanted artifacts or temporary files).
- Commit with conventional commits referencing the issue:
  ```bash
  git commit -m "<type>: <brief summary> (closes #<num>)"
  ```

### 5. Push & Open Pull Request
- Push the branch to remote:
  ```bash
  git push -u origin <branch-name>
  ```
- Create a Pull Request with a structured description linking to the issue:
  ```bash
  gh pr create --repo Ninido/lowkey --base main --head <branch-name> --title "<type>: <summary>" --body "## Description\n...\n\nCloses #<num>\n\n## Changes\n...\n\n## Verification\n..."
  ```
- Provide the user with the links to the created Issue and Pull Request.

---

## Code & Architecture Conventions

- **Language**: Go 1.27+
- **Terminal UI**: Uses Charmbracelet ecosystem (`huh`, `lipgloss`, `x/term`).
- **Engines**: Located in `pkg/engine/`. All inference engines implement `engine.Engine`.
- **Throttling & Power**: Located in `pkg/osutil/`. Manages OS-level background QoS and duty-cycle throttling.
- **Profiles**: Located in `pkg/profile/`. Persisted user launch setups stored in `~/.lowkey/profiles`.
- **UI & Presentation**: Located in `pkg/ui/`. Keep presentation logic modular and covered by unit tests.
