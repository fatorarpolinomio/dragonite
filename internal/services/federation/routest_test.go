package federation

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caio-bernardo/dragonite/internal/types"
	"github.com/caio-bernardo/dragonite/internal/util"
	_ "github.com/joho/godotenv/autoload"
)

func MockConfig() types.ServerConfig {
	publicKey, privateKey, _ := ed25519.GenerateKey(nil)
	return types.ServerConfig{
		ServerName: "dragonite.com",
		Version:    "1.0.0",
		KeyID:      "ed25519:1.0.0",
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}

func NewTestHandler() *Handler {
	config := MockConfig()
	return NewHandler(&config)
}

func TestFederationVersion(t *testing.T) {
	h := NewTestHandler()
	server := httptest.NewServer(http.HandlerFunc(h.getVersion))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	expectedName := h.config.ServerName
	expectBody := fmt.Sprintf(`{"server":{"name":"%s","version":"%s"}}`, expectedName, "1.0.0")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != expectBody {
		t.Errorf("expected body %s, got %s", expectBody, string(body))
	}
}

func TestGetKeyServer(t *testing.T) {
	h := NewTestHandler()
	server := httptest.NewServer(http.HandlerFunc(h.getServerKey))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var mockBody ServerKeyResponse
	mockBody.ServerName = h.config.ServerName
	// Validade de 1 ano
	mockBody.ValidUntilTS = time.Now().Add(365 * 24 * time.Hour).UnixMilli()
	publicKey := base64.RawStdEncoding.EncodeToString(h.config.PublicKey)
	mockBody.VerifyKeys = map[string]VerifyKey{
		h.config.KeyID: {
			Key: publicKey,
		},
	}

	// Criptografia
	canonicalJson, _ := util.CanonicalJSON(mockBody)

	signatureBytes := ed25519.Sign(h.config.PrivateKey, canonicalJson)
	signatureBase64 := base64.RawStdEncoding.EncodeToString(signatureBytes)

	// add signature
	mockBody.Signatures = map[string]map[string]string{
		h.config.ServerName: {
			h.config.KeyID: signatureBase64,
		},
	}
	expectBody, _ := json.Marshal(mockBody)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(expectBody) {
		t.Errorf("expected body %s, got %s", expectBody, string(body))
	}
}
