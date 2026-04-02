package main

import (
	"baby-record-server/internal/auth"
	"baby-record-server/internal/handler"
	"baby-record-server/internal/middleware"
	"baby-record-server/internal/service"
	"github.com/gin-gonic/gin"
)

func registerRoutes(router *gin.Engine, svc *service.Service, tokenManager *auth.Manager) {
	authHandler := handler.NewAuthHandler(svc)
	familyHandler := handler.NewFamilyHandler(svc)
	babyHandler := handler.NewBabyHandler(svc)
	dashboardHandler := handler.NewDashboardHandler(svc)
	recordHandler := handler.NewRecordHandler(svc)
	statsHandler := handler.NewStatsHandler(svc)

	api := router.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
		api.POST("/auth/wx-login", authHandler.WxLogin)
	}

	protected := api.Group("")
	protected.Use(middleware.Auth(tokenManager))
	{
		protected.POST("/family", familyHandler.Create)
		protected.POST("/family/join", familyHandler.Join)
		protected.GET("/family/current", familyHandler.Current)
		protected.GET("/family/join-requests", familyHandler.ListJoinRequests)
		protected.POST("/family/join-requests/:id/approve", familyHandler.ApproveJoinRequest)
		protected.POST("/family/join-requests/:id/reject", familyHandler.RejectJoinRequest)

		protected.GET("/babies/current", babyHandler.Current)
		protected.POST("/babies", babyHandler.Create)
		protected.PATCH("/babies/:id", babyHandler.Update)

		protected.GET("/dashboard", dashboardHandler.Get)

		protected.GET("/records", recordHandler.List)
		protected.POST("/records", recordHandler.Create)
		protected.PATCH("/records/:id", recordHandler.Update)
		protected.DELETE("/records/:id", recordHandler.Delete)

		protected.GET("/stats", statsHandler.Get)
	}
}
