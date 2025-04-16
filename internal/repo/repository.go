package repo

import (
	"database/sql"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	sqlc "github.com/tscrond/dropper/internal/repo/sqlc"
)

type Repository struct {
	db      *sql.DB
	Queries *sqlc.Queries
}

func NewRepository(db *sql.DB) (*Repository, error) {

	queries := sqlc.New(db)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Println("driver error", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://internal/repo/migrations", "postgres", driver)
	if err != nil {
		log.Println("error creating migrations:", err)
		return nil, err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Println("error running up migration", err)
		return nil, err
	}

	return &Repository{
		db:      db,
		Queries: queries,
	}, nil
}

func (repo *Repository) Close() error {
	if repo != nil {
		return repo.db.Close()
	}
	return nil
}
