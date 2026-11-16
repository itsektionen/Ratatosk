package auth

import (
	"context"
	"database/sql"

	"github.com/itsektionen/ratatosk/internal/auth/authdb"
	"github.com/itsektionen/ratatosk/internal/config"
)

type Auth struct {
	cfg config.Config
	q   *authdb.Queries
}

func New(ctx context.Context, cfg config.Config, db *sql.DB) (*Auth, error) {
	return &Auth{cfg: cfg, q: authdb.New(db)}, nil
}
