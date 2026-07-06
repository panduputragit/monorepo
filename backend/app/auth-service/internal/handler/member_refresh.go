package handler

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	authdb "github.com/panduputragit/gym/backend/app/auth-service/internal/db/gen"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/response"
)

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handler) MemberRefresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	payload, err := h.token.VerifyToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	if payload.Role != "member_refresh" {
		response.Unauthorized(c, "invalid token type")
		return
	}

	rt, err := h.query.GetMemberRefreshToken(c.Request.Context(), payload.ID)
	if errors.Is(err, sql.ErrNoRows) {
		response.Unauthorized(c, "refresh token not found")
		return
	}
	if err != nil {
		response.InternalError(c, "failed to fetch refresh token")
		return
	}
	if rt.RevokedAt.Valid {
		response.Unauthorized(c, "refresh token revoked")
		return
	}

	// Rotate: revoke old, issue new
	if err := h.query.RevokeMemberRefreshToken(c.Request.Context(), payload.ID); err != nil {
		response.InternalError(c, "failed to rotate token")
		return
	}

	const accessDuration = 15 * time.Minute
	const refreshDuration = 7 * 24 * time.Hour

	accessToken, _, err := h.token.CreateToken(payload.UserID, payload.Email, "member", accessDuration)
	if err != nil {
		response.InternalError(c, "failed to create access token")
		return
	}

	newRefresh, newRefreshPayload, err := h.token.CreateToken(payload.UserID, payload.Email, "member_refresh", refreshDuration)
	if err != nil {
		response.InternalError(c, "failed to create refresh token")
		return
	}

	memberID := rt.MemberID
	if err := h.query.CreateMemberRefreshToken(c.Request.Context(), authdb.CreateMemberRefreshTokenParams{
		MemberID:  memberID,
		TokenID:   newRefreshPayload.ID,
		ExpiresAt: newRefreshPayload.ExpiredAt,
	}); err != nil {
		response.InternalError(c, "failed to save new refresh token")
		return
	}

	response.OK(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefresh,
		"expires_in":    int64(accessDuration.Seconds()),
	})
}
