package handler

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authdb "github.com/panduputragit/gym/backend/app/auth-service/internal/db/gen"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/response"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/token"
)

func (h *Handler) AdminLogout(c *gin.Context) {
	payload, ok := h.requireAdminToken(c)
	if !ok {
		return
	}

	session, err := h.query.GetAdminSession(c.Request.Context(), payload.ID)
	if errors.Is(err, sql.ErrNoRows) {
		response.Unauthorized(c, "session not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to fetch session")
		return
	}
	if session.RevokedAt.Valid {
		response.Unauthorized(c, "session already revoked")
		return
	}

	adminID, _ := uuid.Parse(payload.UserID)
	if err := h.query.RevokeAdminSession(c.Request.Context(), authdb.RevokeAdminSessionParams{
		TokenID:   payload.ID,
		RevokedBy: uuid.NullUUID{UUID: adminID, Valid: true},
	}); err != nil {
		response.InternalError(c, "failed to revoke session")
		return
	}

	response.OK(c, gin.H{"message": "logged out successfully"})
}

// requireAdminToken extracts and validates the Bearer token, checks role == "admin".
func (h *Handler) requireAdminToken(c *gin.Context) (*token.Payload, bool) {
	tokenStr := bearerToken(c.GetHeader("Authorization"))
	if tokenStr == "" {
		response.Unauthorized(c, "missing token")
		return nil, false
	}

	payload, err := h.token.VerifyToken(tokenStr)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return nil, false
	}

	if payload.Role != "admin" {
		response.Unauthorized(c, "forbidden")
		return nil, false
	}

	session, err := h.query.GetAdminSession(c.Request.Context(), payload.ID)
	if err != nil {
		response.Unauthorized(c, "session not found")
		return nil, false
	}
	if session.RevokedAt.Valid {
		response.Unauthorized(c, "session has been revoked")
		return nil, false
	}

	return payload, true
}
