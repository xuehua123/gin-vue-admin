package request

import (
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CustomClaims JWT自定义声明结构
// 符合开发手册V2.0规范：包含用户信息、客户端会话信息和标准JWT声明
type CustomClaims struct {
	BaseClaims
	BufferTime int64 `json:"buffer_time"` // 缓冲时间（秒）用于token刷新
	jwt.RegisteredClaims
}

// BaseClaims 基础声明信息
// 包含开发手册要求的核心用户和会话标识
type BaseClaims struct {
	UUID        uuid.UUID `json:"user_id"`      // 用户UUID，对应开发手册中的userID
	ID          uint      `json:"id"`           // 用户数据库ID
	Username    string    `json:"username"`     // 用户名
	NickName    string    `json:"nick_name"`    // 用户昵称
	AuthorityId uint      `json:"authority_id"` // 用户权限角色ID
	ClientID    string    `json:"client_id"`    // 客户端会话ID，对应开发手册中的clientID
}

// GetUserID 获取用户UUID字符串
func (c *CustomClaims) GetUserID() string {
	return c.BaseClaims.UUID.String()
}

// GetClientID 获取客户端ID
func (c *CustomClaims) GetClientID() string {
	return c.BaseClaims.ClientID
}

// GetJTI 获取JWT唯一标识符
func (c *CustomClaims) GetJTI() string {
	return c.RegisteredClaims.ID
}

// IsValid 验证Claims是否有效
func (c *CustomClaims) IsValid() bool {
	return c.BaseClaims.UUID != uuid.Nil &&
		c.BaseClaims.ClientID != "" &&
		c.RegisteredClaims.ID != ""
}
