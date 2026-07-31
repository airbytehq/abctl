# abctl external-boundary failure matrix

| Boundary | Force these conditions | Check remaining state |
| --- | --- | --- |
| CLI and config | missing flag/env, invalid version/URL/path, old config, partial write | preserved prior config, clear error, no mutation |
| Filesystem | not found, permission denied, symlink, disk full, rename failure | no truncation, readers closed, safe permissions |
| HTTP and auth | timeout, cancellation, non-2xx, malformed JSON, expired token, refresh failure | body closed, bounded retry, request safely replayed, token not logged |
| Airbyte API | create succeeds then next step fails, delete fails, duplicate name, wrong org | orphan/duplicate prevention, resource ID retained for recovery |
| Docker | daemon absent, permission denied, image missing, interrupted stream | reader closed, no false success, platform error surfaced |
| Kind | existing cluster, create timeout, port collision, delete failure | kubeconfig and cluster state discoverable, retry safe |
| Kubernetes | AlreadyExists, NotFound, conflict, forbidden, watch closes | correct create/update branch, watcher stopped, no partial secret exposure |
| Helm repo/chart | index outage, malformed index, prerelease-only, missing chart, bad archive | no prior external mutation, correct V1/V2 routing, actionable error |
| Helm release | install timeout, Ctrl-C, hook failure, rollback failure | distinguish Helm atomic cleanup from non-Helm resources |
| Release automation | invalid/existing tag, build/archive failure, tap push failure | no misleading completed release, recovery does not overwrite unrelated refs |

For every retry, name the durable state left by attempt one before declaring attempt two safe.
