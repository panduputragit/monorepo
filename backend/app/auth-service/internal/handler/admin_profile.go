package handler

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/response"
)

func (h *Handler) GetAdminProfile(c *gin.Context) {
	payload, ok := h.requireAdminToken(c)
	if !ok {
		return
	}

	adminID, err := uuid.Parse(payload.UserID)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	admin, err := h.query.GetAdminProfile(c.Request.Context(), adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.NotFound(c, "admin not found")
			return
		}

		response.InternalError(c, "failed to fetch profile")
		return
	}

	response.OK(c, gin.H{
		"id":         admin.ID,
		"email":      admin.Email,
		"created_at": admin.CreatedAt,
		"role":       "admin",
	})
}
