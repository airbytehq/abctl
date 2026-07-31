---
name: review-pr
description: Review an abctl GitHub pull request or the current branch for concrete behavioral regressions using diff-first analysis, Go/CLI/Helm/Kubernetes lifecycle lenses, independent specialist passes, and deterministic changed-line validation. Use when asked to review a PR, review a branch or diff, assess whether an abctl change is safe to merge, find bugs in proposed Go code, or produce an approval/request-changes recommendation. This skill is read-only unless the user separately and explicitly authorizes posting or submitting the review.
---

# Review PR

Review the code that changed, trace the real system path, and report only defects with a concrete
failure scenario. Optimize for credible P0-P2 findings, not comment volume.

Read the repository `AGENTS.md`, this file, and the following resources completely before reviewing:

- [abctl review lenses](references/abctl-review-lenses.md)
- [shared review harness](../shared-review-harness/SKILL.md)
- [edge-path audit](../edge-path-audit/SKILL.md)

## 1. Resolve the review target

- For a PR URL or number, resolve the repository, PR number, base branch, and head SHA. Fetch the PR
  head into a remote-tracking ref without switching the user's branch.
- For a current-branch review, determine the upstream base; default to `origin/main` only when no PR
  base exists.
- Run `git status --short --branch`. Warn if local changes could contaminate a current-branch review.
- Record the head SHA. Before any later GitHub mutation, confirm it has not changed.

## 2. Capture an authoritative review bundle

Create a temporary run directory. Capture:

- the unified diff from the base merge-base to the reviewed head;
- the authoritative changed-file list;
- the diff stat; and
- the reviewed base and head SHAs.

For PR mode, prefer `gh pr diff` for the bundle and fetch the head SHA for tracing. For branch mode,
use `git diff <base>...HEAD`. Include deletions: removing validation, cleanup, or compatibility code can
introduce the regression.

Do not read the PR body, commit messages, comments, or existing reviews yet. Do not let author intent
substitute for observed behavior.

## 3. Choose review depth

- If the diff has fewer than 30 changed lines and at most two files, perform one complete pass.
- Otherwise, use up to three independent specialist agents in one batch when agent slots are
  available. Give each the raw diff, changed-file list, head SHA, one lens group, and the canonical
  prompt from the shared harness. The coordinator covers the remaining lens group.
- If agents are unavailable, perform the same lens groups sequentially.
- Warn when the diff exceeds roughly 3,000 changed lines; continue, but identify areas that could not
  be reviewed with high confidence.

Recommended non-trivial lanes:

1. CLI behavior, configuration, auth, and secrets.
2. External mutations, failure ordering, retry, cancellation, and cleanup.
3. Helm/chart, Kubernetes/Docker, release portability, and test integrity.

Keep the passes independent. Do not tell a specialist about another reviewer's suspected bug.

## 4. Trace beyond the diff without losing scope

Reviewers may inspect unchanged callers, implementations, interfaces, tests, and dependency types
when necessary to prove behavior. Use repository search for every changed public function, interface,
struct field, flag, constant, version classifier, and mutation helper.

Do not report pre-existing problems found during tracing. Anchor the finding to the changed line or
deletion that makes the real caller fail. Use `causal_diff_quote` when the visible failure sits on an
unchanged line.

For behavior governed by Helm, Kind, Kubernetes, Docker, Go, or an Airbyte API, verify uncertain
claims against primary documentation, source, types, a real chart, or a controlled command. Label
inference as inference.

## 5. Validate and synthesize findings

Require every pass to emit the shared harness JSON sentinel block. Run:

```bash
python3 .agents/skills/shared-review-harness/scripts/extract_findings.py \
  --agent-response-files <response-files...> \
  --output <findings.json>

python3 .agents/skills/shared-review-harness/scripts/validate_findings.py \
  <diff.patch> <findings.json> > <validated.json>
```

Independently verify each `anchored` and `causal` finding against the real code path. Keep
`needs_review` separate and exclude it from the verdict unless manually verified. Any extraction
failure makes the review incomplete.

After the blind pass, read the PR description, status checks, and existing review discussion. Use
them to reconcile intended behavior, test claims, and duplicates—not to erase a demonstrated bug.

## 6. Report the result

Lead with findings ordered by severity. For each finding include:

- `[P0]`, `[P1]`, or `[P2]` and a short title;
- changed file and line;
- the concrete failure scenario and affected workflow;
- why the diff causes it;
- a focused fix direction; and
- confidence: `VERIFIED` or `LIKELY`.

Use these verdicts:

- Any P0/P1: `REQUEST CHANGES`.
- Only P2: `APPROVE WITH FINDINGS`.
- No P0-P2: `APPROVE`.
- Extraction failure or materially incomplete coverage: `REVIEW INCOMPLETE`.

Mention verification evidence and meaningful gaps. Do not manufacture praise or low-value nits.

The skill is read-only. Do not post inline comments, submit a review, approve, request changes, edit
code, or create follow-up work unless the user explicitly asks for that separate action. When asked
to post, show the exact draft and proposed review event before mutating GitHub.
