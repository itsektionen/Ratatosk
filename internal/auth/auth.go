package auth

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/itsektionen/ratatosk/internal/auth/authdb"
	"github.com/itsektionen/ratatosk/internal/config"
)

type Auth struct {
	cfg  config.Config
	q    *authdb.Queries
	oidc *oidcClient // nil if OIDC is not configured
}

func New(ctx context.Context, cfg config.Config, db *sql.DB) (*Auth, error) {
	oc, err := newOIDC(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("oidc: %w", err)
	}
	return &Auth{cfg: cfg, q: authdb.New(db), oidc: oc}, nil
}
