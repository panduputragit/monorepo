package handler

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	authdb "github.com/panduputragit/gym/backend/app/auth-service/internal/db/gen"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/response"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/validation"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) AdminLogin(c *gin.Context) {
	var req validation.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	admin, err := h.query.GetAdminByEmail(c.Request.Context(), req.Email)
	if errors.Is(err, sql.ErrNoRows) {
		response.Unauthorized(c, "invalid credentials")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to fetch admin")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		response.Unauthorized(c, "invalid credentials")
		return
	}

	const duration = 8 * time.Hour
	tokenStr, payload, err := h.token.CreateToken(admin.ID.String(), admin.Email, "admin", duration)
	if err != nil {
		response.InternalError(c, "failed to create token")
		return
	}

	if err := h.query.CreateAdminSession(c.Request.Context(), authdb.CreateAdminSessionParams{
		AdminID:   admin.ID,
		TokenID:   payload.ID,
		ExpiresAt: payload.ExpiredAt,
	}); err != nil {
		response.InternalError(c, "failed to save session")
		return
	}

	response.OK(c, gin.H{
		"access_token": tokenStr,
		"expires_in":   int64(duration.Seconds()),
		"role":         "admin",
	})
}
