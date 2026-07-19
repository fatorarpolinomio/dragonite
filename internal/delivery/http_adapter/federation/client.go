package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"io"

	"github.com/caio-bernardo/dragonite/internal/domain"
	"github.com/caio-bernardo/dragonite/internal/usecase"
)

type Client struct {
	httpClient *http.Client
}

func NewFederationClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) QueryRemoteProfile(ctx context.Context, remoteServerName string, userID string) (*domain.Profile, error) {
	// Chamada direta para o servidor remoto sem o cabeçalho X-Matrix
	url := fmt.Sprintf("http://%s/_matrix/federation/v1/query/profile?user_id=%s", remoteServerName, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no servidor remoto: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("servidor remoto retornou erro: %d", resp.StatusCode)
	}

	var profile domain.Profile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("falha ao decodificar profile remoto: %v", err)
	}

	return &profile, nil
}

// FetchRemoteMedia faz o download de uma mídia de outro servidor via Server-to-Server API
// Retorna o leitor do corpo (io.ReadCloser), o Content-Type, o nome do arquivo e um erro.
func (c *Client) FetchRemoteMedia(ctx context.Context, serverName string, mediaID string) (io.ReadCloser, string, string, error) {
	// Padrão da API de federação para download de mídia
	url := fmt.Sprintf("http://%s/_matrix/federation/v1/media/download/%s", serverName, mediaID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("falha ao conectar no servidor remoto: %w", err)
	}

	// Se o servidor remoto não encontrar a imagem, precisamos devolver o erro específico
	// para que o seu media/routes.go traduza isso corretamente para um HTTP 404 do Element.
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, "", "", usecase.ErrMediaNotFound
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", "", fmt.Errorf("servidor remoto retornou erro: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	// O Matrix especifica que o filename pode vir no header Content-Disposition,
	// mas para imagens de perfil e anexos simples, podemos deixar vazio e o Element lida bem.
	filename := ""

	// Devolvemos o resp.Body diretamente. O seu Handler de rotas fará o io.Copy e fechará a conexão!
	return resp.Body, contentType, filename, nil
}
