package handler

import (
	"baby-record-server/internal/apperror"
	"baby-record-server/internal/response"
	"baby-record-server/internal/service"
	"github.com/gin-gonic/gin"
)

type BabyHandler struct {
	service *service.Service
}

func NewBabyHandler(svc *service.Service) *BabyHandler {
	return &BabyHandler{service: svc}
}

type babyRequest struct {
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	Birthday    string `json:"birthday"`
	AvatarURL   string `json:"avatarUrl"`
	FeedingMode string `json:"feedingMode"`
	AllergyNote string `json:"allergyNote"`
}

func (h *BabyHandler) Current(c *gin.Context) {
	data, err := h.service.GetCurrentBaby(c.Request.Context(), currentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *BabyHandler) Create(c *gin.Context) {
	var req babyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.CreateBaby(c.Request.Context(), currentUserID(c), service.UpsertBabyInput(req))
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}

func (h *BabyHandler) Update(c *gin.Context) {
	babyID, err := parseUintParam(c, "id")
	if err != nil {
		c.Error(err)
		return
	}

	var req babyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.UpdateBaby(c.Request.Context(), currentUserID(c), babyID, service.UpsertBabyInput(req))
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}
