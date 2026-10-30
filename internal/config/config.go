package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HttpPort     string `env:"HTTP_PORT" env-default:"8080"`
	PostgresDB   string `env:"POSTGRES_DB" env-default:"family_expenses"`
	PostgresUser string `env:"POSTGRES_USER" env-default:"postgres"`
	PostgresPass string `env:"POSTGRES_PASS" env-default:"postgres"`
	PostgresHost string `env:"POSTGRES_HOST" env-default:"localhost"`
	TgApiKey     string `env:"TG_API_KEY"`
	LLMURL       string `env:"LLM_URL" env-default:"https://openrouter.ai/api/v1"`
	LLMModel     string `env:"LLM_MODEL" env-default:"qwen/qwen3.8-27b:free"`
	LLMAPIKey    string `env:"LLM_API_KEY" env-required:"true"`
}

func ReadConfig() (*Config, error) {
	cfg := new(Config)

	_, err := os.Stat(".env")

	if err == nil {
		err = cleanenv.ReadConfig(".env", cfg)
	} else if errors.Is(err, os.ErrNotExist) {
		err = cleanenv.ReadEnv(cfg)
	}

	if err != nil {
		slog.Error("Failed to read config", "error", err)
		return nil, err
	}

	return cfg, nil
}
