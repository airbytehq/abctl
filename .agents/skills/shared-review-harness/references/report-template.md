# PR review report template

Lead with actionable findings. Omit empty sections except the verdict and verification.

```markdown
## Findings

- **[P2] Short title** — `path/file.go:42`
  Concrete failure scenario, why the diff causes it, affected workflow, and focused fix.
  Confidence: VERIFIED.

## Verdict

**APPROVE WITH FINDINGS**

One-sentence reason for the verdict.

## Verification

- Checks and controlled commands that actually ran.
- External contracts or real charts inspected.
- Material boundaries that remain untested.

## Unvalidated findings

- Entries from `needs_review`, with the reason they could not be anchored or verified.
```

Verdict rules:

- Extraction failure or materially incomplete coverage: `REVIEW INCOMPLETE`.
- Any P0/P1: `REQUEST CHANGES`.
- Only P2: `APPROVE WITH FINDINGS`.
- No P0-P2: `APPROVE`.

Do not post this report or submit a GitHub review without separate explicit authorization.
