package telemetry

import "os"

// envVarDNT is the environment variable used to determine if dnt mode is enabled
const envVarDNT = "DO_NOT_TRACK"

// DNT returns the status of the DO_NOT_TRACK flag
// If this flag is enabled, no telemetry data will be collected
func DNT() bool {
	_, ok := os.LookupEnv(envVarDNT)
	return ok
}
