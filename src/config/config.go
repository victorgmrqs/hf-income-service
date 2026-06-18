package config

import (
	"errors"
	"os"

	"github.com/spf13/viper"
)

// Config agrega toda a configuração do serviço, carregada de variáveis de
// ambiente (e opcionalmente de um arquivo .env). Ver .env.example.
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Transaction   TransactionConfig
	Observability ObservabilityConfig
}

type ServerConfig struct {
	Port        string
	Env         string
	MetricsPort string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// TransactionConfig aponta para o hf-transaction-service (consumido pelo saldo/metas).
type TransactionConfig struct {
	URL string
}

type ObservabilityConfig struct {
	ServiceName   string
	SamplingRatio float64
	OTLPEndpoint  string
	OTLPHeaders   string
}

// Load lê a configuração das variáveis de ambiente. O arquivo .env é opcional:
// sua ausência não é erro (útil em CI/containers, onde as vars vêm do ambiente).
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.SetDefault("APP_PORT", "8081")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_METRICS_PORT", "9090")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("DB_NAME", "hf-income")
	viper.SetDefault("TRANSACTION_SERVICE_URL", "http://localhost:8080")
	viper.SetDefault("OTEL_SERVICE_NAME", "hf-income-service")
	viper.SetDefault("OTEL_SAMPLING_RATIO", 1.0)

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return nil, err
		}
	}

	return &Config{
		Server: ServerConfig{
			Port:        viper.GetString("APP_PORT"),
			Env:         viper.GetString("APP_ENV"),
			MetricsPort: viper.GetString("APP_METRICS_PORT"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
		},
		Transaction: TransactionConfig{
			URL: viper.GetString("TRANSACTION_SERVICE_URL"),
		},
		Observability: ObservabilityConfig{
			ServiceName:   viper.GetString("OTEL_SERVICE_NAME"),
			SamplingRatio: viper.GetFloat64("OTEL_SAMPLING_RATIO"),
			OTLPEndpoint:  viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"),
			OTLPHeaders:   viper.GetString("OTEL_EXPORTER_OTLP_HEADERS"),
		},
	}, nil
}
