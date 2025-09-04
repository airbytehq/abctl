package mock

// ValidAirboxConfig is a valid airbox config JSON for testing
const ValidAirboxConfig = `{
	"current-context": "default",
	"contexts": [{
		"name": "default",
		"context": {
			"airbyteApiHost": "https://api.example.com/v1",
			"organizationId": "org-123",
			"oidcClientId": "client-123"
		}
	}],
	"user": {
		"accessToken": "token",
		"expiresAt": "2030-01-01T00:00:00Z"
	}
}`