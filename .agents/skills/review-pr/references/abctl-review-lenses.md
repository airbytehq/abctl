# abctl PR review lenses

Apply only the lenses relevant to the changed files, but always perform caller census, behavior
parity, failure ordering, and test falsification.

## Caller and contract census

For each changed function signature, interface method, struct field, flag, constant, version helper,
configuration shape, chart value, or output field:

1. Search every caller and consumer.
2. Distinguish real uses from tests, generated mocks, and same-named symbols.
3. Verify unchanged callers still satisfy the new contract.
4. Check both command entry points when behavior is shared by `abctl` and `airbox`.
5. Anchor any stale-caller finding to the changed producer or contract line.

## CLI behavior and compatibility

- Kong flag tags: defaults, XOR groups, required/existing-file constraints, hidden flags, env vars.
- Validation before mutation: versions, URLs, host/port, paths, values, secret files, organization and
  region selection.
- Error and exit behavior: actionable context, preserved causes, cancellation, no false success.
- Output compatibility: JSON/YAML fields, credentials output, status text used by scripts.
- Configuration compatibility: current context, credentials, auth provider, unknown/older config.
- Repeated invocation: install over existing state, uninstall then reinstall, login refresh, switching
  organizations, rerunning after partial failure.

## External mutation and lifecycle ledger

List every mutation in order across:

- local config and credential files;
- Docker images and containers;
- Kind clusters and kubeconfig;
- Kubernetes namespaces, volumes, claims, Secrets, ConfigMaps, Services, and Ingresses;
- Helm repositories and releases; and
- Airbyte regions and dataplane records.

After each mutation, trace failure of the next operation. Check cleanup, recovery instructions,
idempotency, name reuse, retry behavior, and whether the user can safely rerun. Confirm validation and
read-only network resolution happen before mutations when possible. Treat `Atomic` as Helm-release
cleanup only.

Check context propagation, bounded waits, Ctrl-C, goroutine/watcher shutdown, and resource closure.

## Helm and chart contracts

- Empty, explicit, malformed, missing, prerelease, and `v`-prefixed versions.
- Stable latest-version selection and repository-index ordering.
- V1/V2 repository routing and warnings using the same classification.
- Relative versus absolute chart URLs and URL escaping.
- Local path and direct URL metadata resolution.
- Values schema differences across supported chart generations.
- Ingress routing, image manifest discovery, chart hook resources, release name, namespace, timeout,
  wait, and atomic behavior.
- Repository outage, malformed index, missing entry, multiple URLs, and unavailable archive.

Use real repository indexes, `helm show chart`, `helm show values`, `helm template`, or upstream source
when the behavior is not guaranteed by code. Do not equate rendering with a working installation.

## Kubernetes, Docker, filesystem, and portability

- Kubernetes AlreadyExists/NotFound/conflict behavior and update resource versions.
- Namespace and cluster selection; stale or wrong kubeconfig/context.
- Kind creation/deletion, node image compatibility, port binding, volume mounts, and image loading.
- Docker daemon absent/unavailable, permissions, platform/architecture, auth, and streaming readers.
- Files missing, unreadable, permission denied, partially written, symlinked, or platform-specific.
- Linux/macOS/Windows paths, executable suffixes, archives, permissions, and release artifacts.

## Authentication, secrets, and HTTP

- Token refresh serialization, retry count, request replay, and refresh failure.
- Response status, body closure, decode errors, timeouts, and context cancellation.
- API base URL joining and path/query escaping.
- Config file permissions and preservation of unrelated contexts/credentials.
- Secrets or tokens in logs, errors, telemetry, generated values, command output, or fixtures copied
  into production behavior.
- External create/delete calls scoped to the intended organization and resource ID.

## Go correctness

- Error wrapping with `%w`; no swallowed or replaced root causes.
- Goroutine, watcher, timer, response body, file, archive, and lock lifetime.
- Data races from mutable globals, test seams, caches, or shared authentication state.
- Nil/empty handling at boundaries; zero values with real semantic meaning.
- Slice/map aliasing, mutation during iteration, and nondeterministic ordering in user output.
- Timeouts, retry loops, polling intervals, and cancellation-safe blocking calls.

## Test falsification

Ask: what is the simplest broken implementation that still passes these tests?

- Mocks asserting values returned by the same mock are not independent evidence.
- Verify call order and forbidden later calls after a failure.
- Cover empty, malformed, absent, conflict, timeout, cancellation, retry, and partial-success paths.
- When package globals are overridden, restore them with `t.Cleanup` and avoid parallel execution.
- Confirm tests exercise the correct chart generation, namespace, repository, API path, and platform.
- State when a real Helm/Kind/Docker/API boundary remains untested.

## Release and CI

- `go.mod` Go version versus CI and release setup versions.
- Build/test/vet/format claims versus workflows that actually run them.
- GoReleaser targets, archive names, checksums, tag validation, prerelease behavior, and Homebrew update.
- Workflow permissions, untrusted interpolation, fork behavior, ref/checkout correctness, and partial
  release failure after a tag has been created.
- Generated mocks stay synchronized with interfaces.
