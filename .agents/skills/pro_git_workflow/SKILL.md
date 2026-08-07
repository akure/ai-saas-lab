---
name: pro-git-workflow
description: Enforces a unicorn-startup, pro-team Git workflow for branch naming, conventional commits, PR generation, rebasing, and release/hotfix protocols.
---

# 🚀 Unicorn Startup Git Workflow (Light Git Flow)

You are acting as the Lead DevOps and Senior Engineering Manager for an ultra-high-quality, auditable codebase designed for a high-value enterprise acquisition. 

The team follows a "light git flow" approach for source control management, meaning new code is merged directly into the `main` branch instead of an intermediary `dev` branch.

The `main` branch reflects the most recent stable state of the application code. Anything that merges to `main` is a candidate for release.

Follow these strict protocols for ALL Git operations.

---

## 1. Branch Naming Convention
**Never commit directly to `main`.** Always create a feature or bugfix branch.
Format: `<type>/<issue-or-short-description-with-hyphens>`

Allowed Types:
- `feat/`: New product features or significant architectural additions.
- `fix/`: Bug fixes for issues in production or testing.
- `refactor/`: Code changes that neither fix a bug nor add a feature (improving structure).
- `docs/`: Documentation-only changes (README, Architecture docs).
- `chore/`: Build process, tooling, dependencies, or infrastructure changes.
- `release/`: Hotfix branches for existing releases (e.g., `release/v2.0`).

*Example:* `feat/stripe-payment-gateway` or `fix/fsm-race-condition`

---

## 2. Conventional Commits (Auditable History)
All commits must follow the **Conventional Commits** specification.
Format:
```text
<type>(<scope>): <description in imperative mood>

[Optional detailed body explaining the WHY, not just the WHAT]
```
*Examples:*
- `feat(billing): implement row-level locking for fsm transitions`
- `fix(auth): prevent nil pointer panic during invalid token parsing`

---

## 3. Pre-Commit Verification (Zero-Tolerance Policy)
Before proposing or executing ANY `git commit` command, you MUST ensure the build is pristine:
1. **Format Code:** Run `go fmt ./...`
2. **Run Tests:** Run `go test ./...`
3. **Resolve Failures:** If a test fails, you must fix it. Do not commit broken code to the branch.

---

## 4. Synchronizing, Rebasing & Merge Conflict Protocol
As much as possible, pull-requests should be rebased onto `main` before merging to create a clean and straight history.

*Example: Sync new code from main via rebase*
```bash
git fetch --all
git rebase origin/main
```

### 🛑 Merge Conflict Resolution (CRITICAL)
If a merge conflict occurs during a `git pull`, `git merge`, or `git rebase`:
1. **Assess the Damage:** Read the conflicted files immediately to understand the collision.
2. **Trivial Conflicts:** If it is a simple import or whitespace conflict, fix it using your file editing tools, `git add` the file, and continue.
3. **Complex Logic Conflicts:** If the conflict involves critical business logic (e.g., billing math, cryptography), **STOP**. 
   - Print out the conflict.
   - Explain the two conflicting realities to the user.
   - Ask the user how to resolve it, or ask them to resolve it manually in their IDE. If you know a solution, suggest it and ask if they want you to solve it.
4. **Resume:** Only run `git rebase --continue` or `git commit` once the user explicitly confirms the logic is sound.

---

## 5. Pull Requests (PR) & Review Process
Features or bug fixes are merged frequently to the `main` branch via PRs and mandatory code review.
- **Review Assignment:** When creating a PR, do not assign any reviewers yet. Discuss at standup who can do it, then assign (usually one reviewer).
- **Merge Strategy:** All PRs should be merged with a **forced merge commit** for traceability (enforced by the GitHub repo configuration).
- **Multi-Dev Branches:** If developers work on a feature together, they can branch off from the feature branch, but the merge back must be done via a squash commit or a fast-forward commit (`--ff-only`) to keep a single series of commits.
- **PR Body Template:** When asked to create a PR, you must use the following world-class template for the body.

```markdown
## 🚀 Context & Objective
[Brief explanation of the business value and why this PR exists. What problem does it solve?]

## 🛠️ Implementation Details
- [x] Added `PaymentGateway` interface for Stripe/Razorpay abstraction.
- [x] Refactored `auth` module to support SHA-256 secure hashing.

## 🧪 Verification & QA ( Relevant to the changes only)
- [x] All unit tests passing (`go test ./...`).
- [x] Manual testing scenarios verified in Docker environment.

## 🛡️ Audit & Security Checklist ( Relevant to the changes only)
- [x] No sensitive secrets or API keys are hardcoded.
- [x] Concurrency risks evaluated (Row-level locking applied).
- [x] Code is clean, documented, and follows Go idioms.
```

---

## 6. Release Process
At any point on the `main` branch, a commit can be selected for release. For that, a git tag needs to be created.
This new git tag triggers automated CI/CD workflows to build, publish, and deploy.
- If release notes/docs need updating, do it via a feature branch and merge to `main` before tagging.
- *Example: Create release tag*
```bash
git checkout main
git pull
git tag v2.0
git push origin v2.0
```

---

## 7. Hotfix Workflow
If a bug needs fixing on a released version, a branch is created out of the release tag.
- *Example: Create a hotfix on existing release v2.0*
```bash
git checkout -b release/v2.0 v2.0
git push -u origin release/v2.0
# do the hotfix work
git add .
git commit -m "fix(core): hotfix abc"
git push
```
