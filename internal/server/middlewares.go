package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// reponseWriter é uma estrutura auxiliar pra incluir o statusCode na resposta
type responseWriter struct {
	statusCode int
	http.ResponseWriter
}

type contextKey string

const userClaimsKey contextKey = "userClaims"

type TokenService interface {
	ValidateToken(tokenString string) (*jwt.Token, error)
}

// Middleware que gerencia o cabeçalho de CORS
func (s *AppServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Replace "*" with specific origins if needed
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are required

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Proceed with the next handler
		next.ServeHTTP(w, r)
	})
}

// Middleware para logar o resultado das requisições
func (s *AppServer) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		res := responseWriter{statusCode: http.StatusOK, ResponseWriter: w}

		next.ServeHTTP(&res, r)

		log.Printf("[%s] %s %d %s in %s", r.Method, r.URL.Path, res.statusCode, http.StatusText(res.statusCode), time.Since(now))
	})
}

// Middleware que confere se o usuário possui um token no header "Bearer"
// Confere se o token é válido
func (s *AppServer) TokenBearerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Header de autorização no formato "Bearer <token>"
		authHeader := r.Header.Get("Authorization")

		//Verifica se o token está presente no header
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		//Confere se está no formato correto ("Bearer <token>")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		// Extrai apenas a string do token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		//Delega a validação criptográfica e de federação ao serviço
		token, err := s.tokenService.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			// Cria um novo request com o contexto atualizado
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// WriteHeader é uma implementação personalizada de WriteHeader que armazena o statusCode
func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

type FederatedTokenService struct {
	jwks          keyfunc.Keyfunc
	validIssuers  map[string]bool
	validAudience string
}

func NewFederatedTokenService(jwksURL, audience string, issuers []string) (*FederatedTokenService, error) {
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar JWKS: %w", err)
	}

	validIssuers := make(map[string]bool)
	for _, iss := range issuers {
		validIssuers[iss] = true
	}

	return &FederatedTokenService{
		jwks:          jwks,
		validIssuers:  validIssuers,
		validAudience: audience,
	}, nil
}

func (s *FederatedTokenService) ValidateToken(tokenString string) (*jwt.Token, error) {
	// O keyfunc faz o download, faz o cache e aplica a chave pública correta aqui
	token, err := jwt.Parse(tokenString, s.jwks.Keyfunc)
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("token inválido")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("formato de claims desconhecido")
	}

	// Verifica o Audience (Foi emitido para mim?)
	if aud, ok := claims["aud"].(string); !ok || aud != s.validAudience {
		return nil, errors.New("audience inválido")
	}

	// Verifica o Issuer (Foi emitido por alguém em quem eu confio?)
	if iss, ok := claims["iss"].(string); !ok || !s.validIssuers[iss] {
		return nil, errors.New("issuer não confiável")
	}

	return token, nil
}
