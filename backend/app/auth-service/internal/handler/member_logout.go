package handler

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/response"
)

func (h *Handler) MemberLogout(c *gin.Context) {
	tokenStr := bearerToken(c.GetHeader("Authorization"))
	if tokenStr == "" {
		response.Unauthorized(c, "missing token")
		return
	}

	// Logout uses the refresh token (access tokens are stateless, can't revoke)
	payload, err := h.token.VerifyToken(tokenStr)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	if payload.Role != "member_refresh" {
		response.BadRequest(c, "provide refresh token to logout")
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
		response.Unauthorized(c, "already logged out")
		return
	}

	if err := h.query.RevokeMemberRefreshToken(c.Request.Context(), payload.ID); err != nil {
		response.InternalError(c, "failed to revoke token")
		return
	}

	response.OK(c, gin.H{"message": "logged out successfully"})
}
