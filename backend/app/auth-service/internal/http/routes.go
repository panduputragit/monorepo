package http

import (
	"github.com/gin-gonic/gin"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/handler"
)

func RegisterRoutes(router *gin.Engine, h *handler.Handler) {
	admin := router.Group("/admin")
	admin.POST("/login", h.AdminLogin)
	admin.POST("/logout", h.AdminLogout)
	admin.GET("/sessions", h.ListAdminSessions)
	admin.DELETE("/sessions/:id", h.ForceRevokeAdminSession)
	admin.GET("/my-profile", h.GetAdminProfile)

	member := router.Group("/member")
	member.POST("/login", h.MemberLogin)
	member.POST("/logout", h.MemberLogout)
	member.POST("/refresh", h.MemberRefresh)
}
