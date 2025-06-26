package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type JWT struct {
	SigningKey []byte
}

var (
	TokenValid            = errors.New("未知错误")
	TokenExpired          = errors.New("token已过期")
	TokenNotValidYet      = errors.New("token尚未激活")
	TokenMalformed        = errors.New("这不是一个token")
	TokenSignatureInvalid = errors.New("无效签名")
	TokenInvalid          = errors.New("无法处理此token")
	TokenNotActive        = errors.New("token未激活或已被撤销")
	TokenClaimsInvalid    = errors.New("token声明信息无效")
)

func NewJWT() *JWT {
	return &JWT{
		[]byte(global.GVA_CONFIG.JWT.SigningKey),
	}
}

// CreateClaims 创建JWT声明
// 符合开发手册V2.0规范，确保包含必要的字段
func (j *JWT) CreateClaims(baseClaims request.BaseClaims) request.CustomClaims {
	bf, _ := ParseDuration(global.GVA_CONFIG.JWT.BufferTime)
	ep, _ := ParseDuration(global.GVA_CONFIG.JWT.ExpiresTime)

	// 生成唯一的JTI，符合开发手册要求
	jti := uuid.New().String()

	claims := request.CustomClaims{
		BaseClaims: baseClaims,
		BufferTime: int64(bf / time.Second), // 缓冲时间1天 缓冲时间内会获得新的token刷新令牌
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,                                       // JTI - JWT唯一标识符
			Audience:  jwt.ClaimStrings{"GVA"},                   // 受众
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1000)), // 签名生效时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ep)),    // 过期时间 7天  配置文件
			Issuer:    global.GVA_CONFIG.JWT.Issuer,              // 签名的发行者
		},
	}
	return claims
}

// CreateToken 创建一个token
func (j *JWT) CreateToken(claims request.CustomClaims) (string, error) {
	// 验证Claims的有效性
	if !claims.IsValid() {
		global.GVA_LOG.Error("创建Token失败：Claims无效",
			zap.String("userID", claims.GetUserID()),
			zap.String("clientID", claims.GetClientID()),
			zap.String("jti", claims.GetJTI()))
		return "", TokenClaimsInvalid
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(j.SigningKey)
	if err != nil {
		global.GVA_LOG.Error("Token签名失败", zap.Error(err))
		return "", err
	}

	// 存储到Redis jwt:active机制
	err = j.StoreActiveJWT(&claims)
	if err != nil {
		global.GVA_LOG.Error("存储JWT到Redis失败", zap.Error(err))
		// 虽然Token创建成功，但Redis存储失败，根据安全策略决定是否返回错误
		return "", fmt.Errorf("存储JWT状态失败: %w", err)
	}

	global.GVA_LOG.Info("JWT创建成功",
		zap.String("userID", claims.GetUserID()),
		zap.String("clientID", claims.GetClientID()),
		zap.String("jti", claims.GetJTI()))

	return signedToken, nil
}

// CreateTokenByOldToken 旧token 换新token 使用归并回源避免并发问题
func (j *JWT) CreateTokenByOldToken(oldToken string, claims request.CustomClaims) (string, error) {
	v, err, _ := global.GVA_Concurrency_Control.Do("JWT:"+oldToken, func() (interface{}, error) {
		return j.CreateToken(claims)
	})
	return v.(string), err
}

// ParseToken 解析 token
func (j *JWT) ParseToken(tokenString string) (*request.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &request.CustomClaims{}, func(token *jwt.Token) (i interface{}, e error) {
		return j.SigningKey, nil
	})

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, TokenExpired
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, TokenMalformed
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, TokenSignatureInvalid
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, TokenNotValidYet
		default:
			return nil, TokenInvalid
		}
	}

	if token != nil {
		if claims, ok := token.Claims.(*request.CustomClaims); ok && token.Valid {
			// 验证Claims有效性
			if !claims.IsValid() {
				return nil, TokenClaimsInvalid
			}
			return claims, nil
		}
	}
	return nil, TokenValid
}

// ParseTokenWithoutValidation 解析token但不验证有效性（用于获取过期token信息）
func (j *JWT) ParseTokenWithoutValidation(tokenString string) (*request.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &request.CustomClaims{}, func(token *jwt.Token) (i interface{}, e error) {
		return j.SigningKey, nil
	}, jwt.WithoutClaimsValidation())

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*request.CustomClaims); ok {
		return claims, nil
	}

	return nil, TokenInvalid
}

// StoreActiveJWT 存储活跃的JWT到Redis
// 实现开发手册要求的 jwt:active 机制
func (j *JWT) StoreActiveJWT(claims *request.CustomClaims) error {
	if !claims.IsValid() {
		return TokenClaimsInvalid
	}

	userID := claims.GetUserID()
	jti := claims.GetJTI()
	clientID := claims.GetClientID()

	redisKey := fmt.Sprintf("jwt:active:%s:%s", userID, jti)
	expiration := time.Unix(claims.ExpiresAt.Unix(), 0).Sub(time.Now())

	err := global.GVA_REDIS.Set(context.Background(), redisKey, clientID, expiration).Err()
	if err != nil {
		global.GVA_LOG.Error("存储JWT active状态失败",
			zap.Error(err),
			zap.String("redisKey", redisKey),
			zap.String("clientID", clientID))
		return err
	}

	global.GVA_LOG.Debug("JWT active状态已存储",
		zap.String("redisKey", redisKey),
		zap.String("clientID", clientID),
		zap.Duration("expiration", expiration))

	return nil
}

// IsJWTActive 检查JWT是否处于活跃状态
func (j *JWT) IsJWTActive(claims *request.CustomClaims) (bool, error) {
	if !claims.IsValid() {
		return false, TokenClaimsInvalid
	}

	userID := claims.GetUserID()
	jti := claims.GetJTI()
	clientID := claims.GetClientID()

	redisKey := fmt.Sprintf("jwt:active:%s:%s", userID, jti)
	storedClientID, err := global.GVA_REDIS.Get(context.Background(), redisKey).Result()

	if err != nil {
		if err.Error() == "redis: nil" {
			return false, TokenNotActive
		}
		return false, err
	}

	// 验证ClientID是否匹配
	if storedClientID != clientID {
		global.GVA_LOG.Warn("JWT ClientID不匹配",
			zap.String("expected", clientID),
			zap.String("stored", storedClientID),
			zap.String("redisKey", redisKey))
		return false, TokenNotActive
	}

	return true, nil
}

// RevokeJWT 撤销指定的JWT
func (j *JWT) RevokeJWT(claims *request.CustomClaims) error {
	if !claims.IsValid() {
		return TokenClaimsInvalid
	}

	userID := claims.GetUserID()
	jti := claims.GetJTI()

	redisKey := fmt.Sprintf("jwt:active:%s:%s", userID, jti)

	_, err := global.GVA_REDIS.Del(context.Background(), redisKey).Result()
	if err != nil {
		global.GVA_LOG.Error("撤销JWT失败",
			zap.Error(err),
			zap.String("redisKey", redisKey))
		return err
	}

	global.GVA_LOG.Info("JWT已成功撤销",
		zap.String("userID", userID),
		zap.String("jti", jti),
		zap.String("redisKey", redisKey))

	return nil
}

// RevokeJWTByID 直接根据userID和jti吊销JWT
func (j *JWT) RevokeJWTByID(userID, jti string) error {
	if userID == "" || jti == "" {
		return errors.New("userID和jti不能为空")
	}
	redisKey := fmt.Sprintf("jwt:active:%s:%s", userID, jti)
	_, err := global.GVA_REDIS.Del(context.Background(), redisKey).Result()
	if err != nil {
		global.GVA_LOG.Error("根据ID撤销JWT失败",
			zap.Error(err),
			zap.String("redisKey", redisKey))
		return err
	}

	global.GVA_LOG.Info("JWT已成功撤销",
		zap.String("userID", userID),
		zap.String("jti", jti),
		zap.String("redisKey", redisKey))

	return nil
}

// RevokeAllUserJWTs 撤销用户的所有活跃JWT
func (j *JWT) RevokeAllUserJWTs(userID string) error {
	if userID == "" {
		return errors.New("用户ID不能为空")
	}

	// 查找所有该用户的活跃JWT
	pattern := fmt.Sprintf("jwt:active:%s:*", userID)
	keys, err := global.GVA_REDIS.Keys(context.Background(), pattern).Result()
	if err != nil {
		global.GVA_LOG.Error("查找用户JWT失败",
			zap.Error(err),
			zap.String("userID", userID))
		return err
	}

	if len(keys) == 0 {
		global.GVA_LOG.Info("用户没有活跃的JWT", zap.String("userID", userID))
		return nil
	}

	// 批量删除
	_, err = global.GVA_REDIS.Del(context.Background(), keys...).Result()
	if err != nil {
		global.GVA_LOG.Error("批量撤销用户JWT失败",
			zap.Error(err),
			zap.String("userID", userID),
			zap.Strings("keys", keys))
		return err
	}

	global.GVA_LOG.Info("用户所有JWT已撤销",
		zap.String("userID", userID),
		zap.Int("count", len(keys)))

	return nil
}

// GetUserActiveJWTs 获取用户所有活跃的JWT信息
func (j *JWT) GetUserActiveJWTs(userID string) (map[string]string, error) {
	if userID == "" {
		return nil, errors.New("用户ID不能为空")
	}

	pattern := fmt.Sprintf("jwt:active:%s:*", userID)
	keys, err := global.GVA_REDIS.Keys(context.Background(), pattern).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, key := range keys {
		clientID, err := global.GVA_REDIS.Get(context.Background(), key).Result()
		if err != nil {
			continue // 跳过已过期或无效的key
		}
		result[key] = clientID
	}

	return result, nil
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: SetRedisJWT
//@description: jwt存入redis并设置过期时间 (此函数已废弃，请参考新的 jwt:active 机制)
//@param: jwt string, userName string
//@return: err error

func SetRedisJWT(jwt string, userName string) (err error) {
	// 此处过期时间等于jwt过期时间
	// dr, err := ParseDuration(global.GVA_CONFIG.JWT.ExpiresTime)
	// if err != nil {
	// 	return err
	// }
	// timer := dr
	// err = global.GVA_REDIS.Set(context.Background(), userName, jwt, timer).Err()
	// return err
	global.GVA_LOG.Warn("SetRedisJWT 方法已废弃，不再执行实际操作。请使用 StoreActiveJWT")
	return errors.New("SetRedisJWT 方法已废弃")
}

// GenerateMQTTClientID 生成有意义的MQTT ClientID
// 格式：username-role-sequence
func (j *JWT) GenerateMQTTClientID(username, role string, sequence int) string {
	return fmt.Sprintf("%s-%s-%03d", username, role, sequence)
}

// GetNextMQTTSequence 使用Redis的INCR命令以原子方式获取下一个MQTT序列号
// 这是一个高性能、高并发安全的实现，取代了旧的SCAN...LOOP...MAX模式
func (j *JWT) GetNextMQTTSequence(userID, username, role string) (int, error) {
	ctx := context.Background()
	// 为每个用户和角色组合定义一个独立的计数器
	counterKey := fmt.Sprintf("mqtt:seq:%s:%s", userID, role)

	// 使用INCR原子地增加计数器并获取新值
	seq, err := global.GVA_REDIS.Incr(ctx, counterKey).Result()
	if err != nil {
		global.GVA_LOG.Error("获取MQTT序列号失败 (INCR)",
			zap.Error(err),
			zap.String("counterKey", counterKey),
			zap.String("userID", userID),
			zap.String("role", role),
		)
		return 0, fmt.Errorf("无法生成MQTT序列号: %w", err)
	}

	// 可选：设置一个合理的过期时间，例如7天，防止计数器永久存在
	// 这有助于清理那些不再活跃用户的计数器
	expiration := 7 * 24 * time.Hour
	if err := global.GVA_REDIS.Expire(ctx, counterKey, expiration).Err(); err != nil {
		// 这里只记录警告，因为即使设置过期失败，序列号也已经成功获取
		global.GVA_LOG.Warn("为MQTT序列号计数器设置过期时间失败",
			zap.Error(err),
			zap.String("counterKey", counterKey),
		)
	}

	return int(seq), nil
}

// CreateMQTTClaims 创建MQTT JWT的声明
func (j *JWT) CreateMQTTClaims(userID, username, role string) (request.MQTTClaims, error) {
	// 获取下一个序号
	sequence, err := j.GetNextMQTTSequence(userID, username, role)
	if err != nil {
		return request.MQTTClaims{}, fmt.Errorf("获取MQTT序号失败: %w", err)
	}

	// 生成ClientID
	clientID := j.GenerateMQTTClientID(username, role, sequence)

	// 生成唯一的JTI
	jti := uuid.New().String()

	// 设置过期时间（可以独立配置）
	ep, _ := ParseDuration(global.GVA_CONFIG.JWT.ExpiresTime)

	claims := request.MQTTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		ClientID: clientID,
		Sequence: sequence,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Audience:  jwt.ClaimStrings{"MQTT"},
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1000)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ep)),
			Issuer:    global.GVA_CONFIG.JWT.Issuer,
		},
	}

	return claims, nil
}

// CreateMQTTToken 创建MQTT专用Token
func (j *JWT) CreateMQTTToken(claims request.MQTTClaims) (string, error) {
	if !claims.IsValid() {
		return "", TokenClaimsInvalid
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(j.SigningKey)
	if err != nil {
		global.GVA_LOG.Error("MQTT Token签名失败", zap.Error(err))
		return "", err
	}

	// 关键修复：在Token创建后，立即存储其活跃状态到Redis
	if err := j.StoreMQTTActiveJWT(&claims); err != nil {
		global.GVA_LOG.Error("存储MQTT JWT到Redis失败", zap.Error(err))
		// 返回错误，确保不会发放一个无法被验证的Token
		return "", fmt.Errorf("存储MQTT JWT状态失败: %w", err)
	}

	// 存储 clientID -> role 的映射
	err = j.StoreMQTTRoleByClaims(&claims)
	if err != nil {
		global.GVA_LOG.Error("存储MQTT角色到Redis失败", zap.Error(err))
		return "", fmt.Errorf("存储MQTT角色状态失败: %w", err)
	}

	// 存储 clientID -> [activeKey, roleKey] 的反向引用，用于高效清理
	if err := j.StoreMQTTClientRefByClaims(&claims); err != nil {
		global.GVA_LOG.Error("存储MQTT客户端引用失败", zap.Error(err))
		// 这是一个重要的辅助功能，失败时应该记录错误但不需要中断主流程
		// 但为了健壮性，这里也返回错误
		return "", fmt.Errorf("存储MQTT客户端引用失败: %w", err)
	}

	global.GVA_LOG.Info("MQTT JWT创建成功并已存储",
		zap.String("userID", claims.GetUserID()),
		zap.String("clientID", claims.GetClientID()),
		zap.String("jti", claims.GetJTI()))

	return signedToken, nil
}

// ParseMQTTToken 解析MQTT Token
func (j *JWT) ParseMQTTToken(tokenString string) (*request.MQTTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &request.MQTTClaims{}, func(token *jwt.Token) (i interface{}, e error) {
		return j.SigningKey, nil
	})

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, TokenExpired
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, TokenMalformed
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, TokenSignatureInvalid
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, TokenNotValidYet
		default:
			return nil, TokenInvalid
		}
	}

	if token != nil {
		if claims, ok := token.Claims.(*request.MQTTClaims); ok && token.Valid {
			if !claims.IsValid() {
				return nil, TokenClaimsInvalid
			}
			return claims, nil
		}
	}
	return nil, TokenValid
}

// StoreMQTTActiveJWT 存储活跃的MQTT JWT到Redis
func (j *JWT) StoreMQTTActiveJWT(claims *request.MQTTClaims) error {
	userID := claims.UserID
	jti := claims.ID
	clientID := claims.ClientID

	if userID == "" || jti == "" || clientID == "" {
		return TokenClaimsInvalid
	}

	redisKey := fmt.Sprintf("mqtt:active:%s:%s", userID, jti)
	// 计算正确的过期时间，与JWT的生命周期保持一致
	expiration := time.Unix(claims.ExpiresAt.Unix(), 0).Sub(time.Now())
	// 确保至少有1秒的过期时间
	if expiration < time.Second {
		expiration = time.Second
	}

	err := global.GVA_REDIS.Set(context.Background(), redisKey, clientID, expiration).Err()
	if err != nil {
		global.GVA_LOG.Error("存储MQTT active状态失败",
			zap.Error(err),
			zap.String("redisKey", redisKey),
			zap.String("clientID", clientID))
		return err
	}
	global.GVA_LOG.Debug("MQTT active状态已存储",
		zap.String("redisKey", redisKey),
		zap.String("clientID", clientID),
		zap.Duration("expiration", expiration))
	return nil
}

// StoreMQTTRoleByClaims stores the role of an MQTT client in Redis.
// The key is mqtt:role:<clientID> and the value is the role.
// The expiration of the key is aligned with the JWT's expiration.
func (j *JWT) StoreMQTTRoleByClaims(claims *request.MQTTClaims) error {
	if !claims.IsValid() {
		return TokenClaimsInvalid
	}

	clientID := claims.ClientID
	role := claims.Role
	if clientID == "" || role == "" {
		return errors.New("clientID and role cannot be empty")
	}

	redisKey := common.RedisMqttRoleKeyPrefix + clientID
	expiration := time.Until(claims.ExpiresAt.Time)

	err := global.GVA_REDIS.Set(context.Background(), redisKey, role, expiration).Err()
	if err != nil {
		global.GVA_LOG.Error("Failed to store MQTT client role in Redis",
			zap.Error(err),
			zap.String("redisKey", redisKey),
			zap.String("role", role),
			zap.String("clientID", clientID))
		return err
	}

	global.GVA_LOG.Debug("Successfully stored MQTT client role in Redis",
		zap.String("redisKey", redisKey),
		zap.String("role", role),
		zap.String("clientID", clientID),
		zap.Duration("expiration", expiration))

	return nil
}

// StoreMQTTClientRefByClaims stores a reference from clientID to its associated Redis keys.
// This is crucial for efficient cleanup when a client disconnects.
func (j *JWT) StoreMQTTClientRefByClaims(claims *request.MQTTClaims) error {
	if !claims.IsValid() {
		return TokenClaimsInvalid
	}

	clientID := claims.ClientID
	if clientID == "" {
		return errors.New("clientID cannot be empty")
	}

	// Define the keys that are associated with this clientID
	activeKey := fmt.Sprintf("mqtt:active:%s:%s", claims.UserID, claims.ID)
	roleKey := common.RedisMqttRoleKeyPrefix + clientID

	// Create a struct for the reference data for easy JSON marshalling
	refData := struct {
		ActiveKey string `json:"activeKey"`
		RoleKey   string `json:"roleKey"`
	}{
		ActiveKey: activeKey,
		RoleKey:   roleKey,
	}

	refDataJSON, err := json.Marshal(refData)
	if err != nil {
		global.GVA_LOG.Error("Failed to marshal MQTT client reference data",
			zap.Error(err),
			zap.String("clientID", clientID))
		return err
	}

	// Define the reference key itself
	refKey := "mqtt:client_ref:" + clientID
	expiration := time.Until(claims.ExpiresAt.Time)

	// Ensure at least 1 second expiration
	if expiration < time.Second {
		expiration = time.Second
	}

	err = global.GVA_REDIS.Set(context.Background(), refKey, refDataJSON, expiration).Err()
	if err != nil {
		global.GVA_LOG.Error("Failed to store MQTT client reference in Redis",
			zap.Error(err),
			zap.String("refKey", refKey),
			zap.String("clientID", clientID))
		return err
	}

	global.GVA_LOG.Debug("Successfully stored MQTT client reference in Redis",
		zap.String("refKey", refKey),
		zap.String("clientID", clientID),
		zap.Duration("expiration", expiration))

	return nil
}

// IsClientIDActive 检查指定的ClientID是否有活跃的JWT记录
// 用于验证客户端上报的ClientID的有效性
func (j *JWT) IsClientIDActive(clientID string) bool {
	if clientID == "" {
		return false
	}

	// 通过客户端引用获取相关的Redis键
	refKey := "mqtt:client_ref:" + clientID
	refDataJSON, err := global.GVA_REDIS.Get(context.Background(), refKey).Result()
	if err != nil {
		if err == redis.Nil {
			// 客户端引用不存在，说明没有活跃的JWT
			return false
		}
		global.GVA_LOG.Error("检查ClientID活跃状态时Redis查询失败",
			zap.Error(err),
			zap.String("clientID", clientID))
		return false
	}

	// 解析引用数据
	var refData struct {
		ActiveKey string `json:"activeKey"`
		RoleKey   string `json:"roleKey"`
	}
	if err := json.Unmarshal([]byte(refDataJSON), &refData); err != nil {
		global.GVA_LOG.Error("解析ClientID引用数据失败",
			zap.Error(err),
			zap.String("clientID", clientID))
		return false
	}

	// 检查活跃键是否存在
	exists, err := global.GVA_REDIS.Exists(context.Background(), refData.ActiveKey).Result()
	if err != nil {
		global.GVA_LOG.Error("检查JWT活跃状态时Redis查询失败",
			zap.Error(err),
			zap.String("clientID", clientID),
			zap.String("activeKey", refData.ActiveKey))
		return false
	}

	isActive := exists > 0
	global.GVA_LOG.Debug("ClientID活跃状态检查完成",
		zap.String("clientID", clientID),
		zap.Bool("isActive", isActive))

	return isActive
}

// IsMQTTJWTActive 检查MQTT JWT是否处于活跃状态
func (j *JWT) IsMQTTJWTActive(claims *request.MQTTClaims) (bool, error) {
	redisKey := fmt.Sprintf("mqtt:active:%s:%s", claims.UserID, claims.ID)
	storedClientID, err := global.GVA_REDIS.Get(context.Background(), redisKey).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, TokenNotActive
		}
		return false, err
	}

	if storedClientID != claims.GetClientID() {
		return false, TokenNotActive
	}

	return true, nil
}

// RevokeMQTTJWT 撤销MQTT JWT
func (j *JWT) RevokeMQTTJWT(claims *request.MQTTClaims) error {
	if !claims.IsValid() {
		return TokenClaimsInvalid
	}

	userID := claims.GetUserID()
	jti := claims.GetJTI()

	redisKey := fmt.Sprintf("mqtt:active:%s:%s", userID, jti)

	_, err := global.GVA_REDIS.Del(context.Background(), redisKey).Result()
	if err != nil {
		global.GVA_LOG.Error("撤销MQTT JWT失败",
			zap.Error(err),
			zap.String("redisKey", redisKey))
		return err
	}

	global.GVA_LOG.Info("MQTT JWT已成功撤销",
		zap.String("userID", userID),
		zap.String("jti", jti),
		zap.String("redisKey", redisKey))

	return nil
}

// RevokeMQTTJWTByID 直接根据userID和jti撤销MQTT JWT
func (j *JWT) RevokeMQTTJWTByID(userID, jti string) error {
	if userID == "" || jti == "" {
		return errors.New("userID和jti不能为空")
	}
	redisKey := fmt.Sprintf("mqtt:active:%s:%s", userID, jti)
	_, err := global.GVA_REDIS.Del(context.Background(), redisKey).Result()
	if err != nil {
		global.GVA_LOG.Error("根据ID撤销MQTT JWT失败",
			zap.Error(err),
			zap.String("redisKey", redisKey))
		return err
	}

	global.GVA_LOG.Info("MQTT JWT已成功撤销",
		zap.String("userID", userID),
		zap.String("jti", jti),
		zap.String("redisKey", redisKey))

	return nil
}

// GetClaims 从Gin的Context中获取claims
func GetClaims(c *gin.Context) (*request.CustomClaims, error) {
	token, err := c.Cookie("x-token")
	if err != nil {
		token = c.Request.Header.Get("x-token")
	}
	j := NewJWT()
	claims, err := j.ParseToken(token)
	if err != nil {
		global.GVA_LOG.Error("从Gin的Context中获取claims失败, 请检查token是否正确", zap.Error(err))
	}
	return claims, err
}

// GetUserUuid 从Gin的Context中获取从jwt解析出来的用户UUID
func GetUserUuid(c *gin.Context) uuid.UUID {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return uuid.Nil
		} else {
			return cl.UUID
		}
	} else {
		waitUse := claims.(*request.CustomClaims)
		return waitUse.UUID
	}
}

// GetUserId 从Gin的Context中获取从jwt解析出来的用户ID
func GetUserId(c *gin.Context) uint {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return 0
		} else {
			return cl.BaseClaims.ID
		}
	} else {
		waitUse := claims.(*request.CustomClaims)
		return waitUse.BaseClaims.ID
	}
}

// GetUserAuthorityId 从Gin的Context中获取从jwt解析出来的用户角色id
func GetUserAuthorityId(c *gin.Context) uint {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return 0
		} else {
			return cl.AuthorityId
		}
	} else {
		waitUse := claims.(*request.CustomClaims)
		return waitUse.AuthorityId
	}
}

// GetUserInfo 从Gin的Context中获取从jwt解析出来的用户角色id
func GetUserInfo(c *gin.Context) *request.CustomClaims {
	if claims, exists := c.Get("claims"); !exists {
		if cl, err := GetClaims(c); err != nil {
			return nil
		} else {
			return cl
		}
	} else {
		waitUse := claims.(*request.CustomClaims)
		return waitUse
	}
}

// GetToken 从Gin的Context中获取token
func GetToken(c *gin.Context) string {
	token := c.Request.Header.Get("x-token")
	if token == "" {
		token, _ = c.Cookie("x-token")
	}
	return token
}

// SetToken 设置token到Gin的Context中
func SetToken(c *gin.Context, token string, maxAge int) {
	// 同时设置到Header和Cookie
	c.Header("x-token", token)
	c.SetCookie("x-token", token, maxAge, "/", "", false, true)
}

// ClearToken 清除Gin的Context中的token
func ClearToken(c *gin.Context) {
	// 同时清除Header和Cookie
	c.Header("x-token", "")
	c.SetCookie("x-token", "", -1, "/", "", false, true)
}

// GetUserID 是 GetUserId 的别名，用于向后兼容
func GetUserID(c *gin.Context) uint {
	return GetUserId(c)
}

// GetClientIDFromJWT 从Gin的Context中获取从JWT解析出来的客户端ID
// 优先从MQTT Claims中获取，如果没有则从标准Claims中获取
func GetClientIDFromJWT(c *gin.Context) string {
	// 首先尝试从MQTT JWT中获取ClientID
	if mqttToken := c.Request.Header.Get("x-mqtt-token"); mqttToken != "" {
		j := NewJWT()
		if mqttClaims, err := j.ParseMQTTToken(mqttToken); err == nil && mqttClaims != nil {
			return mqttClaims.GetClientID()
		}
	}

	// 如果没有MQTT token，尝试从标准JWT中获取ClientID
	if claims := GetUserInfo(c); claims != nil {
		return claims.GetClientID()
	}

	return ""
}
