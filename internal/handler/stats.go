package handler

import (
	"baby-record-server/internal/apperror"
	"baby-record-server/internal/response"
	"baby-record-server/internal/service"
	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	service *service.Service
}

func NewStatsHandler(svc *service.Service) *StatsHandler {
	return &StatsHandler{service: svc}
}

type statsQuery struct {
	Range     string `form:"range"`
	StartDate string `form:"startDate"`
	EndDate   string `form:"endDate"`
}

func (h *StatsHandler) Get(c *gin.Context) {
	var query statsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.GetStats(c.Request.Context(), currentUserID(c), service.StatsQueryInput{
		Range:     query.Range,
		StartDate: query.StartDate,
		EndDate:   query.EndDate,
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}
