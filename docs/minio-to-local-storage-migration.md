# Airbyte MinIO to Local Storage Migration Runbook

## Overview

This runbook provides step-by-step instructions for migrating Airbyte data from MinIO object storage to local filesystem storage while preserving all job logs, workload data, and connection states. The actions need to be performed before performing an Abctl local install that uses a local physical volume.

## Prerequisites

- `kubectl` configured with access to the Airbyte cluster
- `mc` (MinIO client) installed locally
- Airbyte cluster running with MinIO service accessible
- Port forwarding capability to MinIO service
- Consider taking a backup of your `~/.airbyte/abctl/data/airbyte-minio-pv/` folder.

## Migration Steps

### Step 1: Verify Cluster Status

```bash
# Check that Airbyte is running and MinIO service is available
kubectl get svc -n airbyte-abctl | grep minio
```

Expected output should show `airbyte-minio-svc` service.

### Step 2: Set Up Port Forward to MinIO

```bash
# Forward MinIO service port to localhost
kubectl port-forward -n airbyte-abctl svc/airbyte-minio-svc 9000:9000
```

Leave this running in a separate terminal window.

### Step 3: Configure MinIO Client

```bash
# Configure mc with Airbyte MinIO credentials (credentials are always the same)
mc alias set airbyte-minio http://localhost:9000 minio minio123
```

### Step 4: Verify MinIO Connection

```bash
# Test connection and list available buckets
mc ls airbyte-minio
```

Expected buckets:

- `airbyte-dev-logs/` - Contains job execution logs
- `airbyte-storage/` - Contains workload and sync data
- `state-storage/` - Contains connection state information

### Step 5: Prepare Local Storage Directory

```bash
# Ensure local storage directory exists and has correct permissions
mkdir -p ~/.airbyte/abctl/data/airbyte-local-pv
chmod 777 ~/.airbyte/abctl/data/airbyte-local-pv
```

### Step 6: Migrate Job Logs

```bash
# Mirror job logs from MinIO to local filesystem
mc mirror airbyte-minio/airbyte-dev-logs ~/.airbyte/abctl/data/airbyte-local-pv/job-logging
```

### Step 7: Migrate Workload Data

```bash
# Mirror workload data from MinIO to local filesystem
mc mirror airbyte-minio/airbyte-storage ~/.airbyte/abctl/data/airbyte-local-pv/workload
```

### Step 8: Migrate State Storage (if exists)

```bash
# Mirror state storage from MinIO to local filesystem
mc mirror airbyte-minio/state-storage ~/.airbyte/abctl/data/airbyte-local-pv/state
```

### Step 9: Verify Migration

```bash
# Check that data was successfully migrated
ls -la ~/.airbyte/abctl/data/airbyte-local-pv/

# Count migrated files
find ~/.airbyte/abctl/data/airbyte-local-pv/ -type f | wc -l

# Verify specific data types
find ~/.airbyte/abctl/data/airbyte-local-pv/ -name "*.json" | head -5
find ~/.airbyte/abctl/data/airbyte-local-pv/ -name "*sync*" | head -5
```

### Step 10: Clean Up Port Forward

```bash
# Stop the port forward process (Ctrl+C in the port forward terminal)
```

### Step 11: Move `~/.airbyte/abctl/data/airbyte-minio-pv` to another folder

## Expected Results

After successful migration, you should have:

### Directory Structure

```text
~/.airbyte/abctl/data/airbyte-local-pv/
├── job-logging/          # Migrated from airbyte-dev-logs
│   └── workspace/
│       └── [job-logs]
├── workload/             # Migrated from airbyte-storage
│   └── output/
│       └── [sync-data]
└── state/                # Migrated from state-storage (if exists)
    └── [state-files]
```

### File Types Migrated

- **Job Logs**: `.json` files containing Airbyte job execution logs
- **Workload Data**: Sync outputs and intermediate processing files
- **State Files**: Connection state information for incremental syncs

## Troubleshooting

### Port Forward Issues

```bash
# If port 9000 is busy, use a different port
kubectl port-forward -n airbyte-abctl svc/airbyte-minio-svc 9001:9000

# Then update mc alias
mc alias set airbyte-minio http://localhost:9001 minio minio123
```

### Permission Issues

```bash
# Fix directory permissions if needed
sudo chown -R $(whoami) ~/.airbyte/abctl/data/airbyte-local-pv
chmod -R 755 ~/.airbyte/abctl/data/airbyte-local-pv
```

### Verification Commands

```bash
# Compare file counts between MinIO and local
mc find airbyte-minio/airbyte-dev-logs --type f | wc -l
find ~/.airbyte/abctl/data/airbyte-local-pv/job-logging -type f | wc -l

# Check for specific workspace data
mc ls airbyte-minio/airbyte-storage/output/
ls ~/.airbyte/abctl/data/airbyte-local-pv/workload/output/
```

## Rollback Plan

Copy backup to original `~/.airbyte/abctl/data/airbyte-minio-pv` location.

## Notes

- **Credentials**: MinIO username/password are always `minio`/`minio123`
- **Namespace**: Airbyte components are always in `airbyte-abctl` namespace
- **Service Names**: MinIO service is always `airbyte-minio-svc`
- **Data Integrity**: `mc mirror` preserves file structure and content
- **Incremental**: `mc mirror` can be run multiple times safely (only copies new/changed files)

## Validation Checklist

- [ ] MinIO service is accessible
- [ ] Port forward is working
- [ ] mc client connects successfully
- [ ] All three buckets are accessible
- [ ] Local storage directory exists with correct permissions
- [ ] Job logs migrated successfully
- [ ] Workload data migrated successfully
- [ ] State storage migrated (if applicable)
- [ ] File counts match expectations
- [ ] Sample files are readable and contain expected data
