---
name: validate-changes
description: Reviews the latest changes in this repository - the last commit plus any uncommitted work - for repo integrity, style, and the simplicity principle from AGENTS.md, and runs the configured checks. Use whenever the user asks to validate, review, check, or sanity-check changes, a diff, or the last commit, and before committing or pushing, even if they do not say "code review".
---

# Validate changes

Read-only review of recent changes against the rules in `AGENTS.md`. Report findings only - do not edit, stage, or commit anything unless the user asks for it afterwards.

## 1. Scope

The default scope is the last commit plus uncommitted work. If the user names a ref or range, use it instead of `HEAD~1`.

```bash
git status --short
git log -1 --stat
git diff HEAD~1
```

- `git diff` does not show untracked files. Read the ones listed by `git status --short` directly.
- Judge only what the change introduced. Problems in untouched lines go under "Outside scope" in the report.

## 2. Rules

Read `AGENTS.md` in full before judging. It is the source of truth and it evolves, so this skill does not copy its rules. For each changed file, read enough of the file and its neighbors to judge whether the change matches the existing style.

## 3. Checks

Always run `git diff --check HEAD~1`, then the checks that the "Validation" section of `AGENTS.md` prescribes for the changed paths.

A check that cannot run (missing Makefile target, tool not installed, service not created yet) is "not run" - neither a pass nor a finding. Report it with the reason and do not install tools or create targets to make it run.

`[complete here: checks not covered by AGENTS.md, e.g. Docker Compose config or migration validation]`

## 4. Review

**Integrity** - the repo stays coherent after the change:

- New and moved paths fit "Architecture and repository layout" in `AGENTS.md`, or the change updates that section.
- Things that must move together do: a new env variable has an `.env.example` entry, a new service appears in `docker-compose.yml` and the docs, a renamed Makefile target is renamed where it is referenced.
- No secrets, populated `.env` files, or generated artifacts are tracked, and `.gitkeep` files follow the rule.
- No user-written comment was deleted, except a TODO that the change implements.
- The change does only what it claims, with no unrelated edits riding along.
- Redirects stay temporary (`307`/`302`). A permanent redirect is cached by clients, so later clicks silently skip telemetry.
- `[complete here: telemetry invariants once the telemetry design exists, e.g. events every redirect must emit]`

**Style** - back every finding with the quoted `AGENTS.md` rule:

- Conventions for the file type from "Code style and formatting conventions".
- Non-ASCII dashes in added lines (check untracked files the same way):
  ```bash
  git diff HEAD~1 | grep -e $'^+.*\xe2\x80\x93' -e $'^+.*\xe2\x80\x94'
  ```
- Where `AGENTS.md` is silent, consistency with neighboring files.
- Do not restate what a linter already reported - point to the check output instead.
- `[complete here: commit message convention, if one is adopted]`

**Simplicity** - for each added piece ask what would break if it were removed:

- Abstractions with a single caller, wrappers, options nothing reads, features nothing requested.
- Boilerplate and comments that restate the code.
- Dependencies or tooling config added where the defaults would do.

Name the simpler alternative concretely instead of a general "simplify this".

## 5. Verify

Before reporting, re-check each finding against the actual file, not only the diff hunk: the line is right, the change introduced it, and you can quote the broken rule or name what concretely breaks. Drop anything that fails this. A false positive costs more trust than a missed nit.

## 6. Report

Use this template and omit empty sections:

```markdown
## Validation: <range>

**Verdict:** pass | pass with fixes | fail

### Checks
- `<command>` - pass | fail | not run (<reason>)

### Findings

#### Blocking
- `path:line` - what is wrong. Rule: "<quoted rule>". Fix: <concrete fix>.

#### Should fix

#### Nit

### Outside scope
- <pre-existing issue noticed along the way>
```

- **Blocking**: a failing check, broken behavior, a tracked secret, a deleted user comment, a broken layout or out-of-sync file. Any of these makes the verdict `fail`.
- **Should fix**: a clear `AGENTS.md` rule violation or unjustified complexity. These make it `pass with fixes`.
- **Nit**: optional polish. Never affects the verdict.

If there are no findings, say so in one line and still list the checks.
