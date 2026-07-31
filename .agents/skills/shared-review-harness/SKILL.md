---
name: shared-review-harness
description: Provide the structured findings contract and deterministic extraction, changed-line anchoring, causal anchoring, severity, and verdict rules used by abctl's review-pr skill. Use when running or extending the abctl PR review workflow, constructing specialist review prompts, validating reviewer output against a unified diff, or diagnosing REVIEW INCOMPLETE and unvalidated findings.
---

# Shared Review Harness

Use one findings contract across every review lane. Model output is advisory; the deterministic
helpers prove only that a finding is tied to the diff, not that its reasoning is correct.

Read these references completely when running or changing the harness:

- [agent prompt template](references/agent-prompt-template.md)
- [priority rubric](references/priority-rubric.md)
- [report template](references/report-template.md)

## Pipeline

```text
review responses -> extract_findings.py -> validate_findings.py -> coordinator verification
     text              JSON array          anchored/causal/needs_review
```

## Extract findings

```bash
python3 scripts/extract_findings.py \
  --agent-response-files <response...> \
  --output <findings.json>
```

The extractor requires exactly one `BEGIN_FINDINGS_JSON` / `END_FINDINGS_JSON` pair per response and
a JSON array inside it. Missing, duplicate, malformed, empty, or unreadable output becomes an
`extraction_failed: true` marker. The marker is intentional: it prevents a broken reviewer response
from silently looking like a clean review.

## Validate findings

```bash
python3 scripts/validate_findings.py <diff.patch> <findings.json>
```

The validator emits:

- `anchored`: `diff_quote` matches the cited changed line in the cited file and side;
- `causal`: `causal_diff_quote` matches a changed line in the cited file when the failure is visible
  on unchanged code; and
- `needs_review`: malformed, unanchored, out-of-diff, and extraction-failure entries.

Leading `+` and `-` in quotes enforce the diff side. Quotes without a marker are side-agnostic.
Whitespace is normalized, but quotes shorter than eight normalized characters do not anchor.

## Coordinator contract

1. Pass the actual diff and exact changed-file list to each reviewer.
2. Preserve the prompt schema and sentinel lines from the template.
3. Run extraction and validation before reporting a verdict.
4. Verify every anchored and causal finding against the real execution path.
5. Exclude `needs_review` from counts unless the coordinator manually proves and upgrades it.
6. Return `REVIEW INCOMPLETE` when any extraction failure remains.
7. Delete temporary review bundles after the review unless the user asks to retain them.

Do not use the validator as a truth oracle. A perfectly anchored false claim is still a false finding.
