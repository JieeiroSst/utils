package postgres

import (
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	instance *PostgresConnect
	once     sync.Once
)

type PostgresConnect struct {
	db *gorm.DB
}

func GetPostgresConnInstance(dns string) *PostgresConnect {
	once.Do(func() {
		db, err := gorm.Open(postgres.Open(dns), &gorm.Config{})
		if err != nil {
			panic(err)
		}
		instance = &PostgresConnect{db: db}
	})
	return instance
}

func NewPostgresConn(dns string) *gorm.DB {
	return GetPostgresConnInstance(dns).db
}
