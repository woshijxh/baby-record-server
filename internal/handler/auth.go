package handler

import (
	"baby-record-server/internal/apperror"
	"baby-record-server/internal/response"
	"baby-record-server/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *service.Service
}

func NewAuthHandler(svc *service.Service) *AuthHandler {
	return &AuthHandler{service: svc}
}

type wxLoginRequest struct {
	Code      string `json:"code"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

func (h *AuthHandler) WxLogin(c *gin.Context) {
	var req wxLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest("请求参数不正确", err))
		return
	}

	data, err := h.service.Login(c.Request.Context(), service.LoginInput{
		Code:      req.Code,
		Nickname:  req.Nickname,
		AvatarURL: req.AvatarURL,
	})
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, data)
}
