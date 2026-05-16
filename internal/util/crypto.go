package util

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/caio-bernardo/dragonite/internal/types"
)

func GenerateServerKey(serverName string, version string) (types.ServerKeyPair, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return types.ServerKeyPair{}, fmt.Errorf("failed to generate key: %w", err)
	}
	return types.ServerKeyPair{
		Key:     fmt.Sprintf("ed25519:%s", version),
		PubKey:  pubKey,
		PrivKey: privKey,
	}, nil
}
