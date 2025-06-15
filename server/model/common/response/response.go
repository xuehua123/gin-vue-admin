package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

const (
	ERROR   = 7
	SUCCESS = 0
)

func Result(code int, data interface{}, msg string, c *gin.Context) {
	// 开始时间
	c.JSON(http.StatusOK, Response{
		code,
		data,
		msg,
	})
}

func Ok(c *gin.Context) {
	Result(SUCCESS, map[string]interface{}{}, "操作成功", c)
}

func OkWithMessage(message string, c *gin.Context) {
	Result(SUCCESS, map[string]interface{}{}, message, c)
}

func OkWithData(data interface{}, c *gin.Context) {
	Result(SUCCESS, data, "查询成功", c)
}

func OkWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(SUCCESS, data, message, c)
}

func Fail(c *gin.Context) {
	Result(ERROR, map[string]interface{}{}, "操作失败", c)
}

func FailWithMessage(message string, c *gin.Context) {
	Result(ERROR, map[string]interface{}{}, message, c)
}

func NoAuth(message string, c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		Code: 7,
		Data: map[string]interface{}{},
		Msg:  message,
	})
}

func FailWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(ERROR, data, message, c)
}

type Context interface {
	JSON(code int, obj interface{})
}

// MQTTTokenResponse MQTT Token生成响应
type MQTTTokenResponse struct {
	Token     string `json:"token"`      // MQTT JWT Token
	ClientID  string `json:"client_id"`  // MQTT ClientID (username-role-sequence)
	Role      string `json:"role"`       // 角色
	Sequence  int    `json:"sequence"`   // 序号
	ExpiresAt int64  `json:"expires_at"` // 过期时间戳
}

// MQTTTokenInfo MQTT Token信息
type MQTTTokenInfo struct {
	ClientID  string `json:"client_id"`  // ClientID
	Role      string `json:"role"`       // 角色
	Username  string `json:"username"`   // 用户名
	CreatedAt string `json:"created_at"` // 创建时间
	JTI       string `json:"jti"`        // JWT ID
}
