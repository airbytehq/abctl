package oidc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCEChallenge holds PKCE challenge and verifier
type PKCEChallenge struct {
	Verifier  string
	Challenge string
	Method    string
}

// GeneratePKCEChallenge creates a new PKCE challenge
func GeneratePKCEChallenge() (*PKCEChallenge, error) {
	// Generate random verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	
	// Create challenge using SHA256
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	
	return &PKCEChallenge{
		Verifier:  verifier,
		Challenge: challenge,
		Method:    "S256",
	}, nil
}