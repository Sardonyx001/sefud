---
allowed-tools: Bash(git add:*), Bash(git status:*), Bash(git commit:*)
description: Create a git commit
---

# Emoji Commit Command

## Context

- Current git status: !`git status`
- Current git diff (staged and unstaged changes): !`git diff HEAD`
- Current branch: !`git branch --show-current`
- Recent commits: !`git log --oneline -10`

## Your task

Based on the above changes, create a single git commit following the Conventional Commits format. Ensure that the commit message captures the essence of the changes made in the conversation. make sure to include an emoji (gitmoji) in the commit message to enhance clarity and visual appeal.

```plain
<type>[optional scope]: [gitmoji] <description>

[optional body]

[optional footer(s)]
```

- Breaking changes: Use ! after type/scope or add BREAKING CHANGE: footer
- Common types: feat, fix, docs, style, refactor, test, chore, ci, build, perf
- Scope examples: (api), (ui), (auth), (parser)
- Description: Present tense, lowercase, under 50 chars, no period
- Separate conversation exchanges with ----
- Gitmoji: Use a relevant emoji to represent the change (e.g., :sparkles: for new features, :bug: for fixes, etc.)
- For a list of gitmojis, refer to [gitmoji.dev](https://gitmoji.dev/)
- Examples:
  - `feat(ui): :sparkles: add new button component`
  - `fix(auth): :bug: resolve login issue`
  - `docs(parser): :memo: update README with usage instructions`
