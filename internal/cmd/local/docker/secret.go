package docker

import (
	"encoding/base64"
	"encoding/json"
)

// Secret generates a docker registry secret that can be stored as a k8s secret
// and used to pull images from docker hub (or any other registry) in an
// authenticated manner.
// The format if the []byte in string form will be
//
//	{
//	  "auths": {
//	    "[SERVER]": {
//	      "username": "[USER]",
//	      "password": "[PASS]",
//	      "email": "[EMAIL]",
//	      "auth"; "[base64 encoding of 'user:pass']"
//	    }
//	  }
//	}
func Secret(server, user, pass, email string) ([]byte, error) {
	// map of the server to the credentials
	return json.Marshal(map[string]any{
		"auths": map[string]any{
			server: map[string]any{
				"username": user,
				"password": pass,
				"email":    email,
				"auth":     base64.StdEncoding.EncodeToString([]byte(user + ":" + pass)),
			},
		},
	})
}
