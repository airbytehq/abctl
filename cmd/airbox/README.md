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

Initialize your airbox configuration:
```bash
airbox config init
```

This will interactively prompt you for configuration details such as:
- Airbyte API host
- Authentication server URL
- OIDC client ID
- Context name

The exact prompts may vary based on your deployment type (Cloud vs Enterprise).

## Authentication

**Prerequisites:** Your user account needs the **instance admin** role to manage dataplanes.

Create application credentials in the Airbyte web UI:
1. Go to **Settings > Application**
2. Click **"Create an application"**
3. Copy the generated Client ID and Client Secret

Set your application credentials as environment variables:
```bash
export AIRBYTE_CLIENT_ID="your-client-id"
export AIRBYTE_CLIENT_SECRET="your-client-secret"
```

Login to authenticate with Airbyte:
```bash
airbox auth login
```

This will:
1. Use application credentials (AIRBYTE_CLIENT_ID and AIRBYTE_CLIENT_SECRET environment variables)
2. Save your credentials locally
3. Prompt you to select an organization context

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

### Delete Dataplane

Remove a dataplane registration:
```bash
airbox delete dataplane my-dataplane
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
airbox delete dataplane test-dp

# Delete the Kind cluster (using the full cluster name with airbox- prefix)
kind delete cluster --name airbox-test-dp
```

### For existing Kubernetes clusters

```bash
# Delete the dataplane registration from Airbyte  
airbox delete dataplane <dataplane-name>

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

Airbox stores configuration in `~/.airbyte/airbox/config.yaml`:
- Authentication credentials
- Context settings
- Organization and workspace IDs

## Requirements

- Go 1.21+
- Access to Airbyte Cloud or Enterprise
- Kubernetes cluster (for dataplane installation)
- Helm 3.x (for dataplane installation)
- Kind (optional, for local development)
