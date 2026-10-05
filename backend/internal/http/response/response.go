package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 是 DevHub 所有 HTTP API 的统一响应结构。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Success 返回成功响应。
func Success(
	c *gin.Context,
	data any,
) {
	c.JSON(
		http.StatusOK,
		Response{
			Code:    CodeSuccess,
			Message: "success",
			Data:    data,
		},
	)
}

// SuccessMessage 返回带指定 message 的成功响应。
func SuccessMessage(
	c *gin.Context,
	message string,
	data any,
) {
	c.JSON(
		http.StatusOK,
		Response{
			Code:    CodeSuccess,
			Message: message,
			Data:    data,
		},
	)
}

// Fail 返回业务失败。
//
// 注意：
// DevHub 约定业务错误 HTTP Status 仍然返回 200，
// 客户端通过 code 判断具体业务状态。
func Fail(
	c *gin.Context,
	code int,
	message string,
) {
	c.JSON(
		http.StatusOK,
		Response{
			Code:    code,
			Message: message,
			Data:    nil,
		},
	)
}

// Error 返回真正的服务器异常。
func Error(c *gin.Context) {
	c.JSON(
		http.StatusInternalServerError,
		Response{
			Code:    CodeInternalError,
			Message: "internal server error",
			Data:    nil,
		},
	)
}
