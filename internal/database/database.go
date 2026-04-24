package database

import (
	"embed"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/pressly/goose/v3"
	"github.com/yokkkoso/ongaku/internal/config_manager"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type Database struct {
	*gorm.DB

	Client *bot.Client
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

func InitDatabase(client *bot.Client) (*Database, error) {
	config := config_manager.GetConfigManager().Get()

	db, err := gorm.Open(
		postgres.Open(
			fmt.Sprintf(
				"host=%s port=%s user=%s password=%s dbname=%s",
				config.Database.Host,
				config.Database.Port,
				config.Database.User,
				config.Database.Password,
				config.Database.DBName,
			),
		),
		&gorm.Config{
			Logger: gormLogger.Default.LogMode(gormLogger.Silent),
		},
	)

	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()

	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(6)
	sqlDB.SetMaxOpenConns(45)
	sqlDB.SetConnMaxLifetime(3600 * time.Second)

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		panic(err)
	}

	return &Database{
		DB:     db,
		Client: client,
	}, nil
}
