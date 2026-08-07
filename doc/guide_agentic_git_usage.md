# 🤖 Agentic Git Workflow Guide

Welcome to your AI-assisted, pro-level Git workflow! Since the `pro-git-workflow` skill is installed in your `.agents/skills` directory, I will automatically enforce naming conventions, commit formats, rebase strategies, and pre-flight tests for this repository. 

To make the most of this, here is a cheat sheet of "agentic language" (prompts) you can use to interact with me seamlessly.

---

## 1. Creating a Feature Branch
Instead of typing commands in the terminal yourself, just ask me in plain English. I will ensure the branch name matches the strict `<type>/<description-with-hyphens>` format.

> **Prompt:** *"Create a branch for the Stripe integration."*
> **My Action:** I will execute `git checkout -b feat/stripe-integration`.

> **Prompt:** *"Switch to a new bugfix branch for the nil pointer in auth."*
> **My Action:** I will execute `git checkout -b fix/auth-nil-pointer`.

---

## 2. Committing Work (The "Zero-Tolerance" Flow)
When you ask me to commit, the skill automatically forces me to run `go fmt ./...` and `go test ./...` *before* I propose the git command.

> **Prompt:** *"Commit my current changes. It's for the Stripe webhook handler."*
> **My Action:** 
> 1. Run tests and formatting.
> 2. Automatically generate the Conventional Commit message: `feat(billing): implement stripe webhook handler`
> 3. Propose the `git add .` and `git commit -m "..."` commands for your approval.

---

## 3. Synchronizing and Rebasing
We keep a clean history by rebasing feature branches on top of `main`.

> **Prompt:** *"Pull the latest from main and rebase my branch on top of it."*
> **My Action:** I will run `git fetch --all` and `git rebase origin/main`. 
> 
> *If a conflict occurs:* I will immediately stop, read the conflicted files, present the two conflicting versions of the code to you, and ask for your explicit architectural decision on how to merge them. Once you approve, I will fix the files and run `git rebase --continue`.

---

## 4. Handling Pull Requests (PRs)
I can write world-class PR descriptions using the `gh` CLI tool, populating the Context, Implementation Details, and Audit Checklist automatically.

> **Prompt:** *"Push this branch and create a PR against main."*
> **My Action:** 
> 1. Run `git push -u origin <branch>`.
> 2. Run `gh pr create --title "..." --body "..."` using our strict enterprise PR template. Note: We won't assign reviewers immediately.

---

## 5. Releases and Hotfixes
> **Prompt:** *"Tag the current main branch as v2.0."*
> **My Action:** I will checkout `main`, pull, run `git tag v2.0`, and `git push origin v2.0`.

> **Prompt:** *"Create a hotfix branch for v2.0."*
> **My Action:** I will execute `git checkout -b release/v2.0 v2.0` and push it to origin.

---

## 6. Bypassing the Skill 
If you are doing a quick scratchpad test or prototyping and don't want the strict enterprise rules applied:

> **Prompt:** *"Just do a quick dirty commit with the message 'WIP', ignore the pro git skill."*
> **My Action:** I will skip the tests, skip the naming conventions, and just execute a standard `git commit -am "WIP"`.

---

## ⚙️ Managing Skills (`skills.json`)
I have created a `skills.json` file in your `.agents/` folder. It looks like this:
```json
{
  "entries": [],
  "inherits": [],
  "exclude": []
}
```
If you ever want to globally disable the `pro-git-workflow` skill without deleting its instructions, simply update the file like so:
```json
{
  "exclude": ["pro-git-workflow"]
}
```
