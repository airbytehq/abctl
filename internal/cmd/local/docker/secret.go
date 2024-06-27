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
	servers := map[string]credential{
		server: newCredential(user, pass, email),
	}
	auths := map[string]map[string]credential{
		"auths": servers,
	}

	return json.Marshal(auths)
}

type credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
}

func newCredential(user, pass, email string) credential {
	return credential{
		Username: user,
		Password: pass,
		Email:    email,
		Auth:     base64.StdEncoding.EncodeToString([]byte(user + ":" + pass)),
	}
}
