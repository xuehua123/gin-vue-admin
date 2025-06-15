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

// MQTTClaims MQTT专用JWT声明结构
// 用于MQTT连接认证，包含角色信息
type MQTTClaims struct {
	UserID   string `json:"user_id"`   // 用户UUID字符串
	Username string `json:"username"`  // 用户名
	Role     string `json:"role"`      // 角色：transmitter, receiver, admin
	ClientID string `json:"client_id"` // MQTT ClientID: username-role-sequence
	Sequence int    `json:"sequence"`  // 序号，用于区分同用户同角色的多个连接
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

// GetUserID 获取用户ID
func (m *MQTTClaims) GetUserID() string {
	return m.UserID
}

// GetClientID 获取MQTT ClientID
func (m *MQTTClaims) GetClientID() string {
	return m.ClientID
}

// GetJTI 获取JWT唯一标识符
func (m *MQTTClaims) GetJTI() string {
	return m.RegisteredClaims.ID
}

// IsValid 验证MQTT Claims是否有效
func (m *MQTTClaims) IsValid() bool {
	return m.UserID != "" &&
		m.Username != "" &&
		m.ClientID != "" &&
		m.Role != "" &&
		m.RegisteredClaims.ID != ""
}

// MQTTTokenRequest MQTT Token生成请求
type MQTTTokenRequest struct {
	Role string `json:"role" binding:"required"` // 角色：transmitter, receiver, admin
}

// RevokeMQTTTokenRequest 撤销MQTT Token请求
type RevokeMQTTTokenRequest struct {
	ClientID string `json:"client_id" binding:"required"` // 要撤销的ClientID
}
