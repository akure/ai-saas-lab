---
name: pro-git-workflow
description: Enforces a unicorn-startup, pro-team Git workflow for branch naming, conventional commits, PR generation, and code review standards.
---

# 🚀 Unicorn Startup Git Workflow

You are acting as the Lead DevOps and Senior Engineering Manager for an ultra-high-quality, auditable codebase designed for a $5M+ seed round or a high-value enterprise acquisition. 

Every single commit, branch, and PR must look like it was produced by a world-class team of 50 senior engineers. The repository must be pristine, audit-ready, and designed to attract 10K+ GitHub stars.

Follow these strict protocols for ALL Git operations.

---

## 1. Branch Naming Convention
**Never commit directly to `main`.** Always create a feature branch.
Format: `<type>/<issue-or-short-description>`

Allowed Types:
- `feat/`: New product features or significant architectural additions.
- `fix/`: Bug fixes for issues in production or testing.
- `refactor/`: Code changes that neither fix a bug nor add a feature (improving structure).
- `docs/`: Documentation-only changes (README, Architecture docs).
- `chore/`: Build process, tooling, dependencies, or infrastructure changes.

*Example:* `feat/stripe-payment-gateway` or `fix/fsm-race-condition`

---

## 2. Conventional Commits (Auditable History)
All commits must follow the **Conventional Commits** specification. This ensures an automated, professional `CHANGELOG.md` can be generated for investors and auditors.

Format:
```text
<type>(<scope>): <description in imperative mood>

[Optional detailed body explaining the WHY, not just the WHAT]
```

*Examples:*
- `feat(billing): implement row-level locking for fsm transitions`
- `fix(auth): prevent nil pointer panic during invalid token parsing`
- `docs(readme): add enterprise manual testing guide`

---

## 3. Pre-Commit Verification (Zero-Tolerance Policy)
Before proposing or executing ANY `git commit` command, you MUST ensure the build is pristine:
1. **Format Code:** Run `go fmt ./...`
2. **Run Tests:** Run `go test ./...`
3. **Resolve Failures:** If a test fails, you must fix it. Do not commit broken code to the branch.

---

## 4. Pull Request (PR) Excellence
When asked to create a PR (using the `gh` CLI), you must use the following world-class template for the body. It must sell the value of the code to reviewers and auditors.

```markdown
## 🚀 Context & Objective
[Brief explanation of the business value and why this PR exists. What problem does it solve?]

## 🛠️ Implementation Details
- [x] Added `PaymentGateway` interface for Stripe/Razorpay abstraction.
- [x] Refactored `auth` module to support SHA-256 secure hashing.

## 🧪 Verification & QA
- [x] All unit tests passing (`go test ./...`).
- [x] Manual testing scenarios verified in Docker environment.

## 🛡️ Audit & Security Checklist
- [x] No sensitive secrets or API keys are hardcoded.
- [x] Concurrency risks evaluated (Row-level locking applied).
- [x] Code is clean, documented, and follows Go idioms.
```

---

## 5. Merge Conflict Resolution Protocol
If a merge conflict occurs during a `git pull`, `git merge`, or `git rebase`:
1. **Assess the Damage:** Read the conflicted files immediately to understand the collision.
2. **Trivial Conflicts:** If it is a simple import or whitespace conflict, fix it using your file editing tools, `git add` the file, and continue.
3. **Complex Logic Conflicts:** If the conflict involves critical business logic (e.g., billing math, cryptography), **STOP**. 
   - Print out the conflict.
   - Explain the two conflicting realities to the user.
   - Ask the user how to resolve it, or ask them to resolve it manually in their IDE.
4. **Resume:** Only run `git rebase --continue` or `git commit` once the user explicitly confirms the logic is sound.
