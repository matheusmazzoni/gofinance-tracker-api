package config

import (
	"github.com/joeshaw/envdecode"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

// Config armazena toda a configuração da aplicação, lida diretamente
// das variáveis de ambiente usando tags 'env'.
type Config struct {
	AppEnv         string `env:"APP_ENV,default=development"`
	ServerPort     string `env:"SERVER_PORT,default=8080"`
	ServerHostName string `env:"SERVER_HOSTNAME,default=localhost"`
	DatabaseURL    string `env:"DATABASE_URL,required"`
	JWTSecretKey   string `env:"JWT_SECRET_KEY,required"`
}

// LoadConfig carrega as configurações das variáveis de ambiente para a struct Config.
// Ele também carrega um arquivo .env se ele existir, ideal para desenvolvimento local.
func (c *Config) Load(logger *zerolog.Logger) error {
	logger.Info().Msg("Loading configuration")

	if err := godotenv.Load(); err != nil {
		logger.Info().Msg("No .env file found, reading from environment")
	}

	return envdecode.Decode(c)
}
