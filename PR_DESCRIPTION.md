# Add System Resource Validation Before Cluster Creation

## Summary

Enhances the local installation flow with system resource validation to prevent cluster creation on systems that don't meet minimum requirements. Adds comprehensive resource checking for CPU and RAM before proceeding with Airbyte cluster deployment, providing clear error messages and failing fast when resources are insufficient.

## What's New

### Resource Validation Function

The `systemResourcesAvailable()` function validates system resources before cluster creation:

- **CPU validation**: Checks for minimum 2 vCPUs using `runtime.NumCPU()`
- **RAM validation**: Checks for minimum 8 GB RAM using cross-platform `github.com/pbnjay/memory` package
- **Clear error messages**: Provides specific feedback showing actual vs required resources
- **Telemetry integration**: Tracks resource metrics for observability
- **Tracing support**: Includes OpenTelemetry spans for debugging

### Error Handling

New `ErrInsufficientResources` error type with user-friendly guidance:

- Explains minimum requirements clearly
- Provides actionable help message
- Follows existing error pattern for consistency

### Installation Flow Integration

Resource check integrated into the installation workflow:

- Runs after Docker validation and before cluster creation
- Prevents unnecessary cluster creation attempts
- Fails fast with clear error messages
- Maintains all existing functionality

## Command Structure

After this PR, the `abctl local install` command flow:

1. **Docker check** - Validates Docker installation
2. **Resource check** - Validates system resources (2 vCPUs, 8 GB RAM) ✨ NEW
3. **Port check** - Validates port availability
4. **Cluster creation** - Proceeds only if all checks pass

## Resource Validation Flow

1. Checks CPU count using `runtime.NumCPU()`
2. Checks total system memory using cross-platform memory detection
3. Compares against minimum requirements (2 vCPUs, 8 GB RAM)
4. Returns detailed error if resources are insufficient
5. Logs success message with actual resources if validation passes

## Example Error Messages

**Insufficient CPU:**
```
ERROR  Insufficient CPUs: found 1, required 2
error: insufficient system resources: insufficient CPUs (found 1, required 2)
```

**Insufficient RAM:**
```
ERROR  Insufficient RAM: found 4.00 GB, required 8 GB
error: insufficient system resources: insufficient RAM (found 4.00 GB, required 8 GB)
```

**Success:**
```
SUCCESS  System resources check passed: 4 vCPUs, 16.00 GB RAM
```

## Test Coverage

- ✅ `TestSystemResourcesAvailable` - Verifies error handling and error type wrapping
- ✅ All existing tests pass without modification
- ✅ Code compiles successfully
- ✅ Resource check works on systems meeting requirements
- ✅ Error handling verified for insufficient resources

## Requirements

- **Minimum 2 vCPUs** - Validated using `runtime.NumCPU()`
- **Minimum 8 GB RAM** - Validated using cross-platform memory detection

## Backward Compatibility

✅ Fully backward compatible - all existing functionality remains intact. The resource check is additive and doesn't modify any existing behavior.
