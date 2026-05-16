package server

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/caio-bernardo/dragonite/internal/database"
	"github.com/caio-bernardo/dragonite/internal/types"
	"github.com/caio-bernardo/dragonite/internal/util"
	_ "github.com/joho/godotenv/autoload"
)

// AppServer representa o servidor em nível de aplicação
type AppServer struct {
	Config types.ServerConfig

	db database.Service
}

// Cria um novo servidor http
func NewServer() *http.Server {
	keyPair, err := util.GenerateServerKey(os.Getenv("SERVER_NAME"), os.Getenv("VERSION"))
	if err != nil {
		panic(err)
	}

	port, _ := strconv.Atoi(os.Getenv("BACKEND_PORT"))
	config := types.ServerConfig{
		ServerName: os.Getenv("SERVER_NAME"),
		Version:    os.Getenv("VERSION"),
		Port:       port,
		KeyID:      keyPair.Key,
		PublicKey:  keyPair.PubKey,
		PrivateKey: keyPair.PrivKey,
	}
	NewServer := &AppServer{
		Config: config,
		db:     database.New(),
	}

	// servidor http, com endpoints registrados e timeout para operações R/W
	server := http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &server
}
