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
