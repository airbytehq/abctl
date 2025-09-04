package abctl

// Config holds configuration extracted from Airbyte installations.
// May be moved to a more specific package as the architecture evolves.
type Config struct {
	AirbyteAPIHost string // From AIRBYTE_API_HOST env var
	AirbyteURL     string // From AIRBYTE_URL env var
	AirbyteAuthURL string // From AB_AIRBYTE_AUTH_IDENTITY_PROVIDER_OIDC_ENDPOINTS_AUTHORIZATION_SERVER_ENDPOINT env var
	OIDCClientID   string // From AB_AIRBYTE_AUTH_IDENTITY_PROVIDER_OIDC_CLIENT_ID env var
	Edition        string // Airbyte edition: "community", "enterprise", etc.

	// Organization Context is set by selection from API result.
	OrganizationID string
}
