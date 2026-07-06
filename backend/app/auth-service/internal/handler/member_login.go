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

func (h *Handler) MemberLogin(c *gin.Context) {
	var req validation.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	member, err := h.query.GetMemberByEmail(c.Request.Context(), req.Email)
	if errors.Is(err, sql.ErrNoRows) {
		response.Unauthorized(c, "invalid credentials")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to fetch member")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(member.PasswordHash), []byte(req.Password)); err != nil {
		response.Unauthorized(c, "invalid credentials")
		return
	}

	const accessDuration = 15 * time.Minute
	const refreshDuration = 7 * 24 * time.Hour

	// Stateless short-lived access token
	accessToken, _, err := h.token.CreateToken(member.ID.String(), member.Email, "member", accessDuration)
	if err != nil {
		response.InternalError(c, "failed to create access token")
		return
	}

	// Refresh token saved to DB
	refreshToken, refreshPayload, err := h.token.CreateToken(member.ID.String(), member.Email, "member_refresh", refreshDuration)
	if err != nil {
		response.InternalError(c, "failed to create refresh token")
		return
	}

	if err := h.query.CreateMemberRefreshToken(c.Request.Context(), authdb.CreateMemberRefreshTokenParams{
		MemberID:  member.ID,
		TokenID:   refreshPayload.ID,
		ExpiresAt: refreshPayload.ExpiredAt,
	}); err != nil {
		response.InternalError(c, "failed to save refresh token")
		return
	}

	response.OK(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int64(accessDuration.Seconds()),
		"role":          "member",
	})
}
