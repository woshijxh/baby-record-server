package handler

import (
	"baby-record-server/internal/apperror"
	"baby-record-server/internal/response"
	"baby-record-server/internal/service"
	"strconv"
	"github.com/gin-gonic/gin"
)

type FamilyHandler struct {
	service *service.Service
}

func NewFamilyHandler(svc *service.Service) *FamilyHandler {
	return &FamilyHandler{service: svc}
}

type createFamilyRequest struct {
	Name string `json:"name"`
}

type joinFamilyRequest struct {
	Code string `json:"code"`
}

func (h *FamilyHandler) Create(c *gin.Context) {
	var req createFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.CreateFamily(c.Request.Context(), currentUserID(c), service.CreateFamilyInput{Name: req.Name})
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *FamilyHandler) Join(c *gin.Context) {
	var req joinFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.JoinFamily(c.Request.Context(), currentUserID(c), service.JoinFamilyInput{Code: req.Code})
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *FamilyHandler) Current(c *gin.Context) {
	data, err := h.service.GetCurrentFamily(c.Request.Context(), currentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *FamilyHandler) ListJoinRequests(c *gin.Context) {
	data, err := h.service.ListJoinRequests(c.Request.Context(), currentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *FamilyHandler) ApproveJoinRequest(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("申请编号不正确", err))
		return
	}
	data, err := h.service.ReviewJoinRequest(c.Request.Context(), currentUserID(c), service.ReviewJoinRequestInput{
		RequestID: requestID,
		Approve:   true,
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *FamilyHandler) RejectJoinRequest(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("申请编号不正确", err))
		return
	}
	data, err := h.service.ReviewJoinRequest(c.Request.Context(), currentUserID(c), service.ReviewJoinRequestInput{
		RequestID: requestID,
		Approve:   false,
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}
