package types

import (
	"crypto/ed25519"
	"net/http"
)

// Config for the server
type ServerConfig struct {
	ServerName string
	Version    string
	Port       int
	KeyID      string
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// Middleware is a function type that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// ServerKeyPair holds the crypto key pair for a server
type ServerKeyPair struct {
	Key     string
	PubKey  ed25519.PublicKey
	PrivKey ed25519.PrivateKey
}
