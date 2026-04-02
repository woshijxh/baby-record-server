package handler

import (
	"baby-record-server/internal/response"
	"baby-record-server/internal/service"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	service *service.Service
}

func NewDashboardHandler(svc *service.Service) *DashboardHandler {
	return &DashboardHandler{service: svc}
}

type dashboardQuery struct {
	Date string `form:"date"`
}

func (h *DashboardHandler) Get(c *gin.Context) {
	var query dashboardQuery
	_ = c.ShouldBindQuery(&query)

	data, err := h.service.GetDashboard(c.Request.Context(), currentUserID(c), query.Date)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}
