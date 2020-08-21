package store

import (
	"github.com/jmoiron/sqlx"
)

// Postgres is the concrete implementation of Store backed by PostgreSQL
type Postgres struct {
	db *sqlx.DB
}

// New returns a new Postgres struct using the given DB
func New(db *sqlx.DB) *Postgres {
	return &Postgres{
		db: db,
	}
}
