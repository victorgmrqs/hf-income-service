package database

import (
	"fmt"

	"github.com/victorgmrqs/hf-income-service/src/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect abre uma conexão GORM com o PostgreSQL a partir da configuração.
// O AutoMigrate das entidades é feito posteriormente (T02+), quando elas existirem.
func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}
