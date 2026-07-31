---
description: Review an abctl PR or current branch with the shared validated review skill
model: opus
argument-hint: "[pr-url|pr-number]"
---

# Review PR

Use the canonical skill exposed at `.claude/skills/review-pr/SKILL.md` and follow it completely.

<arguments>
$ARGUMENTS
</arguments>

Treat a PR URL or number as the review target. When arguments are empty, review the current branch
against its resolved upstream base.

Claude-specific orchestration:

- Use up to three independent Agent tasks for a non-trivial diff when slots are available.
- Keep the lanes blind and independent; pass each the actual diff, changed-file list, head SHA, and
  canonical prompt contract.
- Run the extraction and validation helpers through `.claude/skills/shared-review-harness/scripts/`.
- Return the report in conversation. Do not post comments or submit a GitHub review unless the user
  explicitly authorizes that separate mutation.
