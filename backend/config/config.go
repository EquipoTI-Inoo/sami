package config

import "os"

type Config struct {
	Env             string
	Port            string
	JWTSecret       string
	CORSOrigin      string
	DBPath          string
	AnthropicAPIKey string
}

func Load() Config {
	return Config{
		Env:       getEnv("ENV", "development"),
		Port:      getEnv("PORT", "3000"),
		JWTSecret: getEnv("JWT_SECRET", ""),
		// Comma-separated list — see routes.Setup for why.
		CORSOrigin:      getEnv("CORS_ORIGIN", "http://localhost:5173"),
		DBPath:          getEnv("DB_PATH", "./data/coachia.db"),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
