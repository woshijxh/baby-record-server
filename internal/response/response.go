package response

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, APIResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Failure(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, APIResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
