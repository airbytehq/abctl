---
name: edge-path-audit
description: Adversarially trace abctl failure, retry, cancellation, and cleanup paths. Use before declaring changes complete or when reviewing code that parses CLI/config input, reads or writes files, performs HTTP/auth operations, invokes Docker/Kind/Kubernetes/Helm, mutates Airbyte API resources, runs goroutines or watchers, resolves charts, or changes release automation. Also use when asked what could go wrong, whether a flow fails safely, or whether retries and partial failures are handled.
---

# Edge Path Audit

Trace each changed operation under adverse conditions instead of stopping after the happy path
passes. Read [the abctl failure matrix](references/abctl-failure-matrix.md) when the change crosses an
external boundary.

## Audit method

For every changed function or branch:

1. Name its inputs, durable state, external calls, resources, and outputs.
2. List read-only operations and mutations in execution order.
3. At each step, force the next step to fail, time out, or receive cancellation.
4. Record what state remains and whether retry, cleanup, or recovery is safe.
5. Search every entry point and caller; a guard on one command path is not a guard on all paths.
6. Turn meaningful adverse paths into tests or explicitly document why the risk is accepted.

## Required adverse conditions

- Absent, empty, whitespace-only, malformed, semantically contradictory, and unsupported input.
- Dependency unavailable, unauthorized, permission denied, non-2xx, malformed response, and timeout.
- Wrong or stale kubeconfig, context, repository index, chart reference, config, cached token, and Git
  ref.
- Existing resource, missing resource, update conflict, partial previous attempt, and second attempt.
- Cancellation immediately before and after each external mutation.
- Cleanup failure after the primary failure.
- Oversized input, slow stream, truncated output, and platform-specific path/archive behavior.
- Concurrent token refresh, watcher shutdown, shared mutable global, and goroutine completion.

## Fail-safe rules

- Perform fallible validation and resolution before mutation when possible.
- Do not turn a failed guard, lookup, or verification step into success.
- Preserve the original error and distinguish the failed operation.
- Do not claim rollback covers resources outside the rollback mechanism.
- Prefer idempotent create-or-update behavior or explicit recovery over blind replay.
- Close resources and stop goroutines on every return path.

## Output

Return a compact ledger:

```text
step -> mutation/state -> forced failure -> remaining state -> retry/cleanup behavior
```

Identify concrete defects separately with a changed-line anchor, affected workflow, and focused fix.
Do not report generic hypotheticals without tracing the real path.
