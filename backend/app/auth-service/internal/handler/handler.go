package handler

import (
	"database/sql"
	"strings"

	authdb "github.com/panduputragit/gym/backend/app/auth-service/internal/db/gen"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/token"
)

type Handler struct {
	query *authdb.Queries
	token *token.Maker
}

func New(db *sql.DB, tokenMaker *token.Maker) *Handler {
	return &Handler{
		query: authdb.New(db),
		token: tokenMaker,
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
