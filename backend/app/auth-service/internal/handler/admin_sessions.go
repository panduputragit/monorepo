package handler

import (
	"github.com/gin-gonic/gin"
	authdb "github.com/panduputragit/gym/backend/app/auth-service/internal/db/gen"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/response"
	"github.com/google/uuid"
)

// ListAdminSessions lists all sessions for the authenticated admin.
func (h *Handler) ListAdminSessions(c *gin.Context) {
	payload, ok := h.requireAdminToken(c)
	if !ok {
		return
	}

	adminID, err := uuid.Parse(payload.UserID)
	if err != nil {
		response.BadRequest(c, "invalid admin id")
		return
	}

	sessions, err := h.query.ListAdminSessions(c.Request.Context(), adminID)
	if err != nil {
		response.InternalError(c, "failed to fetch sessions")
		return
	}

	response.OK(c, sessions)
}

// ForceRevokeAdminSession lets an admin revoke any session by session UUID.
func (h *Handler) ForceRevokeAdminSession(c *gin.Context) {
	payload, ok := h.requireAdminToken(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}

	revokerID, _ := uuid.Parse(payload.UserID)
	if err := h.query.RevokeAdminSessionByID(c.Request.Context(), authdb.RevokeAdminSessionByIDParams{
		ID:        sessionID,
		RevokedBy: uuid.NullUUID{UUID: revokerID, Valid: true},
	}); err != nil {
		response.InternalError(c, "failed to revoke session")
		return
	}

	response.OK(c, gin.H{"message": "session revoked"})
}
