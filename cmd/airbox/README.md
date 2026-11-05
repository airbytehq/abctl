# Airbox CLI

Airbox is a command-line tool for managing Airbyte dataplanes on Kubernetes.

## Installation

Build from source:
```bash
go build -o airbox ./cmd/airbox
```

## Quick Start

```bash
# 1. Configure airbox with your Airbyte instance
airbox config init

# 2. Authenticate with Airbyte
airbox auth login

# 3. Install a dataplane on your Kubernetes cluster
airbox install dataplane
```

## Configuration

### Environment Variables

- `AIRBOXCONFIG`: Override the default config file location (default: `~/.airbyte/airbox/config`)
- `AIRBYTE_CLIENT_ID`: OAuth2 client ID for authentication
- `AIRBYTE_CLIENT_SECRET`: OAuth2 client secret for authentication

### Initialization

Initialize your airbox configuration:
```bash
airbox config init
```

This will prompt you to:
- What Airbyte control plane are you connecting to (Cloud or Enterprise)

For **Cloud**: Automatically configures cloud.airbyte.com URLs
For **Enterprise**: You'll be prompted for:
- Your Airbyte instance URL (e.g., https://airbyte.yourcompany.com)
- API host is automatically set to your URL + "/api"
- Edition is automatically set to "enterprise"
- Context name defaults to "enterprise"

Both Cloud and Enterprise use OAuth2 authentication only.

## Authentication

**OAuth2 Authentication (Both Cloud and Enterprise):**

Set your OAuth2 client credentials as environment variables:

```bash
export AIRBYTE_CLIENT_ID="your-oauth-client-id"
export AIRBYTE_CLIENT_SECRET="your-oauth-client-secret"
```

Login to authenticate with Airbyte:
```bash
airbox auth login
```

This will:
1. Use OAuth2 client credentials from environment variables
2. Authenticate via OAuth2 client credentials flow
3. Save access tokens locally in YAML format
4. Prompt you to select an organization context (if not already set)

Logout and clear stored credentials:
```bash
airbox auth logout
```

This removes your saved authentication tokens from local storage.

Switch between Airbyte organization contexts:
```bash
airbox auth switch-organization
```

## Dataplane Management

### Install Dataplane

Install a dataplane on your Kubernetes cluster:
```bash
airbox install dataplane
```

This interactive command will:
1. Prompt for organization and region selection
2. Ask for a dataplane name
3. Create a new Kind cluster (default)
4. Register the dataplane with Airbyte Cloud/Enterprise
5. Deploy the dataplane using Helm with the provided credentials
6. Configure the dataplane to connect to your Airbyte control plane

### Install components on one or more clusters

Pass repeatable Helm overlays and map each component's `cluster` to a kubeconfig
context. Later files override earlier files; maps merge recursively and lists
replace earlier lists:

```bash
airbox install dataplane -n airbyte \
  --values components.values.yaml --values production.values.yaml \
  --cluster-context primary=gke-production \
  --cluster-context agents=gke-agents
```

Airbox performs one Helm install/upgrade per enabled component cluster or previously used cluster,
using the dataplane name as the release name and setting `targetCluster` for each
release. Unused context mappings do not cause additional deployments. Keep related
`egress.allowComponents` peers in the same cluster. Set
`componentsNamespace` in the overlays if the default `airbyte-agents` namespace is
not suitable. Explicit context mappings use existing clusters rather than creating
Kind. Without mappings, a single-cluster installation uses Kind by default, or the
current kubeconfig context with `--with-kind-cluster=false`.

After all Helm operations succeed, enabled components with a `publicIngress.host`
are registered at `https://<host>` according to this mapping:

| Chart component | Control-plane component |
| --- | --- |
| `cloudMcp` | `cloud_mcp` |
| `mtxApi` | `mtx_api` |
| `aguiFrontend` | `agui` |

All other components, including `aguiBackend`, `directConnect` and `sourceConnect`,
are installed without registration, with one informational line per component.
Configure `AGUI_BASE_URL` on the frontend to point at its backend's public URL.

For Cloud MCP, Airbox writes the registration response to `cloud-mcp-oidc` in the
components namespace, with `clientId` and `clientSecret` keys, and restarts the
release's Cloud MCP deployment. Cloud MCP's Helm operation does not wait for
readiness or use atomic rollback: the pod cannot start until registration creates
the Secret. Other clusters retain the normal Helm wait and atomic behavior. A
registration or Secret-write failure is reported; rerunning the same installation
retries registration and rotates the Cloud MCP secret again.

To upgrade, select the same organization, dataplane name and namespace and supply
the desired overlays again. Airbox retains the dataplane identity, cluster mappings
and registration intents in `AIRBOXCONFIG`. The file contains dataplane client
credentials in addition to the existing login credentials and is written with
mode `0600`; do not share or commit it. Saved cluster mappings are reused unless
explicitly supplied; changing a saved context or release namespace requires manual
uninstallation first.

Set a component's `enabled` to `false`, remove it from the overlays, or remove its
public host to unregister it on the next successful install. Previously used
clusters are upgraded too, removing their disabled component resources. Omitting
all overlays on a saved component installation disables all its components.
Cleanup is checkpointed so an API failure can be retried. Multi-cluster operations
are not transactional: a failure can leave earlier clusters upgraded; rerun with
the same values to reconcile them. Use a chart release containing the component
framework; `--chart-version` can pin that release.

### Get Resources

List dataplanes:
```bash
airbox get dataplane
```

Get specific dataplane by ID:
```bash
airbox get dataplane <dataplane-id>
```

Output in YAML format:
```bash
airbox get dataplane -o yaml
```

### Delete Dataplane

Remove a dataplane registration:
```bash
airbox delete dataplane <dataplane-id>
```

Note: This only removes the registration from Airbyte. You must manually uninstall the Helm release from Kubernetes.

## Global Options

- `--help, -h`: Show help for any command

## Examples

Complete workflow:
```bash
# Set up authentication credentials
export AIRBYTE_CLIENT_ID="your-client-id"
export AIRBYTE_CLIENT_SECRET="your-client-secret"

# Initialize configuration
airbox config init

# Authenticate
airbox auth login

# Install the dataplane (handles both registration and deployment)
airbox install dataplane

# Verify installation
airbox get dataplane
```

## Uninstalling a Dataplane

### For Kind-based installations

Kind clusters created by airbox follow the naming pattern: `airbox-<dataplane-name>`

```bash
# List all Kind clusters to find yours
kind get clusters

# Example output:
# airbox-test-dp
# airbox-dev-dp
# airbox-demo-dp

# Delete the dataplane registration from Airbyte
airbox delete dataplane <dataplane-id>

# Delete the Kind cluster (using the full cluster name with airbox- prefix)
kind delete cluster --name airbox-test-dp
```

### For existing Kubernetes clusters

```bash
# Delete the dataplane registration from Airbyte
airbox delete dataplane <dataplane-id>

# Uninstall the Helm release
helm uninstall <dataplane-name> -n <namespace>
```

## Running Syncs

After installing a dataplane, create connections in the Airbyte web UI using the same workspace and region as your dataplane. When you trigger a sync, you'll see pods created in your Kubernetes cluster:

```bash
# Watch pods to see sync activity
watch kubectl get po

Every 2.0s: kubectl get po

NAME                                         READY   STATUS      RESTARTS   AGE
my-dataplane-airbyte-data-plane-567f98c6d9   1/1     Running     0          5m12s
replication-job-15-attempt-0                 0/3     Completed   0          2m18s
source-postgres-discover-15-0-xyz123         0/2     Completed   0          2m45s
```

The dataplane handles all job scheduling within your cluster while reporting status to the Airbyte control plane.

## Configuration File

Airbox stores configuration in `~/.airbyte/airbox/config` (YAML format).

After running `airbox config init` and `airbox auth login`, your config file will look like this:

```yaml
current-context: cloud
credentials:
  access_token: [REDACTED]
  token_type: Bearer
  expires_at: 2025-09-08T18:06:30.147961-07:00
contexts:
  - name: cloud
    context:
      airbyteApiHost: https://api.airbyte.com
      airbyteUrl: https://cloud.airbyte.com
      organizationId: [REDACTED]
      edition: cloud
      auth:
        clientId: [REDACTED]
        clientSecret: [REDACTED]
        type: oauth2
```

The config file contains:
- **current-context**: Active context name
- **credentials**: OAuth2 access tokens and expiry (added after login)
- **contexts**: Named configurations for different Airbyte instances
  - **airbyteApiHost**: API endpoint URL
  - **airbyteUrl**: Web UI URL
  - **organizationId**: Selected organization UUID (added after login)
  - **edition**: Deployment edition (cloud/enterprise/community)
  - **auth**: OAuth2 client credentials

## Requirements

- Go 1.21+
- Access to Airbyte Cloud or Enterprise
- Kubernetes cluster (for dataplane installation)
- Helm 3.x (for dataplane installation)
- Kind (optional, for local development)
