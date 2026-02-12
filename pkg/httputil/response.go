package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData 分页数据
type PageData struct {
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Items interface{} `json:"items"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 成功响应带消息
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// SuccessWithPage 成功响应带分页
func SuccessWithPage(c *gin.Context, total int64, page, size int, items interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PageData{
			Total: total,
			Page:  page,
			Size:  size,
			Items: items,
		},
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 400错误
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    400,
		Message: message,
	})
}

// Unauthorized 401错误
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code:    401,
		Message: message,
	})
}

// Forbidden 403错误
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Code:    403,
		Message: message,
	})
}

// NotFound 404错误
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Code:    404,
		Message: message,
	})
}

// InternalError 500错误
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Code:    500,
		Message: message,
	})
}

// ErrorCode 错误码定义
const (
	ErrCodeSuccess           = 0
	ErrCodeBadRequest        = 400
	ErrCodeUnauthorized      = 401
	ErrCodeForbidden         = 403
	ErrCodeNotFound          = 404
	ErrCodeInternalError     = 500
	ErrCodeInvalidParams     = 1001
	ErrCodeUserNotFound      = 1002
	ErrCodeUserExists        = 1003
	ErrCodeInvalidPassword   = 1004
	ErrCodeWalletNotFound    = 2001
	ErrCodeAddressNotFound   = 2002
	ErrCodeInsufficientFund  = 2003
	ErrCodeTransactionFailed = 3001
	ErrCodeRiskControlFailed = 4001
	ErrCodeWithdrawalFailed  = 5001
)

// ErrorMessages 错误消息映射
var ErrorMessages = map[int]string{
	ErrCodeSuccess:           "success",
	ErrCodeBadRequest:        "bad request",
	ErrCodeUnauthorized:      "unauthorized",
	ErrCodeForbidden:         "forbidden",
	ErrCodeNotFound:          "not found",
	ErrCodeInternalError:     "internal error",
	ErrCodeInvalidParams:     "invalid parameters",
	ErrCodeUserNotFound:      "user not found",
	ErrCodeUserExists:        "user already exists",
	ErrCodeInvalidPassword:   "invalid password",
	ErrCodeWalletNotFound:    "wallet not found",
	ErrCodeAddressNotFound:   "address not found",
	ErrCodeInsufficientFund:  "insufficient fund",
	ErrCodeTransactionFailed: "transaction failed",
	ErrCodeRiskControlFailed: "risk control failed",
	ErrCodeWithdrawalFailed:  "withdrawal failed",
}
