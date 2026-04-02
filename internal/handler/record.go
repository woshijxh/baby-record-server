package handler

import (
	"baby-record-server/internal/apperror"
	"baby-record-server/internal/response"
	"baby-record-server/internal/service"
	"github.com/gin-gonic/gin"
)

type RecordHandler struct {
	service *service.Service
}

func NewRecordHandler(svc *service.Service) *RecordHandler {
	return &RecordHandler{service: svc}
}

type recordQuery struct {
	Date string `form:"date"`
	Type string `form:"type"`
}

func (h *RecordHandler) List(c *gin.Context) {
	var query recordQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.ListRecords(c.Request.Context(), currentUserID(c), query.Date, query.Type)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *RecordHandler) Create(c *gin.Context) {
	var req service.UpsertRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.CreateRecord(c.Request.Context(), currentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *RecordHandler) Update(c *gin.Context) {
	recordID, err := parseUintParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}

	var req service.UpsertRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.UpdateRecord(c.Request.Context(), currentUserID(c), recordID, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *RecordHandler) Delete(c *gin.Context) {
	recordID, err := parseUintParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}

	if err := h.service.DeleteRecord(c.Request.Context(), currentUserID(c), recordID); err != nil {
		c.Error(err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
