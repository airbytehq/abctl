# Review priority rubric

Priority describes demonstrated impact, not how strongly the reviewer feels.

## P0 — critical

Immediate widespread harm: credential disclosure, destructive data loss, arbitrary code execution,
release compromise, or a default core workflow unusable for essentially all users. Request changes.

## P1 — high

Likely serious user or operator failure: broken install/uninstall/auth, durable resource corruption,
unsafe destructive behavior, chart/API compatibility break, repeatable orphaning with significant
cleanup, or a release that cannot complete or recover safely. Request changes.

## P2 — medium

A concrete bounded regression: a dependency failure leaves recoverable partial state, an edge input
misroutes behavior, retry/cancellation is unsafe in a limited path, one supported platform breaks, or
a test gap demonstrably masks a named product bug. Approve with findings unless the shown blast
radius warrants P1.

## Not findings

Do not report style, naming, formatting, DRY, comment preferences, generic missing tests, speculative
hardening, or pre-existing problems. Mention verification limitations separately when useful.

## Confidence

- `VERIFIED`: proven from the diff plus traced code, test, primary source, or controlled reproduction.
- `LIKELY`: the execution path is concrete, but one external premise could not be directly exercised.

Anything weaker stays out of the actionable findings.
