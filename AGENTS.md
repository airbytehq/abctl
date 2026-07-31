# AGENTS.md — abctl engineering and review standards

These instructions apply to the entire repository. `abctl` is a Go CLI that orchestrates
Docker, Kind, Kubernetes, Helm, local files, browsers, and authenticated Airbyte APIs. Review
and implementation work must treat those boundaries as real stateful systems, not as ordinary
in-process helpers.

## Scope and change discipline

- Keep each change focused on the requested behavior. Do not mix opportunistic refactors with a
  feature or fix.
- Prefer the smallest diff that solves the problem and preserves existing CLI behavior.
- Treat command flags, output formats, configuration files, chart values, and persisted Kubernetes
  resources as compatibility contracts.
- Do not hand-edit generated mocks under `internal/**/mock/` or other files carrying generated-code
  headers. Update the source interface and run `make mocks`.

## Stateful operation safety

- Validate and resolve all inputs that can fail without side effects before creating or mutating
  external resources.
- For flows that touch more than one system, write down the mutation order. After every mutation,
  ask what remains if the next step fails, times out, or is cancelled.
- Make retries and repeated invocations safe. Detect existing Kind clusters, Helm releases,
  Kubernetes resources, configuration, and Airbyte API records before blindly creating duplicates.
- Add cleanup or a clear recovery path for partial state. Helm `Atomic` protects the Helm release;
  it does not clean up Kind clusters, API records, files, or resources created outside Helm.
- Propagate `context.Context` through network, Kubernetes, Docker, and long-running operations.
  Preserve cancellation and use bounded waits where the dependency can hang.

## Helm and chart compatibility

- Preserve V1/V2 chart routing deliberately. Repository choice, warning behavior, values generation,
  ingress routing, and image discovery must classify versions consistently, including prereleases.
- Treat chart names, versions, repository URLs, metadata, and values schemas as external contracts.
  Verify uncertain behavior against the actual chart or repository index rather than inferring it.
- Default-version resolution must select a stable release and fail clearly when the repository is
  unavailable, malformed, or missing the expected chart.
- When changing generated values, verify both relevant chart generations and distinguish
  `helm template` success from an end-to-end working installation.

## Secrets, authentication, and configuration

- Never log access tokens, refresh tokens, client secrets, passwords, Docker credentials, or raw
  Kubernetes Secret data.
- Preserve authentication retry and refresh semantics, including request-body replay and concurrent
  refresh behavior.
- Treat configuration writes as durable state: retain unrelated contexts and credentials, use safe
  file permissions, and avoid leaving truncated files after a failed write.
- Validate user-supplied paths, URLs, versions, hosts, ports, values, and secret files at the CLI or
  I/O boundary before downstream mutation.

## Go correctness

- Close response bodies, files, archive readers, watchers, and other resources on every path.
- Do not leak goroutines or watchers after command completion, cancellation, or early return.
- Wrap errors with actionable operation context while preserving the original cause with `%w`.
- Avoid package-level mutable test seams when parallel tests can race. Restore overridden globals
  with `t.Cleanup` and do not mark those tests parallel.
- Keep Linux, macOS, and Windows behavior in mind for paths, permissions, archives, executables, and
  release artifacts.

## Test integrity

- Test the behavioral contract, not values copied from the mock setup.
- Cover empty, malformed, absent, error, timeout/cancellation, retry, and partial-success paths when
  they are relevant to the changed behavior.
- For mutation flows, assert call ordering and assert that later calls do not occur after failure.
- A mocked Helm/Kubernetes/API unit test does not prove a real chart can be fetched, rendered, or
  installed. State that boundary explicitly and add focused integration evidence when risk warrants.

## Verification

Use the narrowest relevant checks while iterating, then run the repository checks before declaring
the work complete:

```bash
make fmt
make build
make test
make vet
git diff --check
```

CI currently runs `make build` and `make test` on Linux. Do not claim that CI covers vet, the race
detector, real Kind/Helm installation, or the release platform matrix unless the workflow changes.

## Pull-request reviews

- Use `.agents/skills/review-pr/SKILL.md` for PR or branch reviews.
- Review the authoritative diff independently before reading the PR description, commits, or review
  discussion. Reconcile that metadata only after the initial bug hunt.
- Report only concrete defects introduced by changed lines or deletions. Inspect unchanged callers
  and dependencies when needed, but anchor every actionable finding to the causal diff.
- P0/P1 findings block approval. P2 findings are non-blocking unless their demonstrated blast radius
  warrants promotion. Style-only observations are not findings.
- Never post comments, approve, or request changes without the user's explicit authorization.
