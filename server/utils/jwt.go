package utils

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

// GetNextMQTTSequence 获取用户指定角色的下一个序号
func (j *JWT) GetNextMQTTSequence(userID, username, role string) (int, error) {
	ctx := context.Background()

	// 查找当前用户的所有活跃MQTT连接
	pattern := fmt.Sprintf("mqtt:active:%s:*", userID)
	keys, err := global.GVA_REDIS.Keys(ctx, pattern).Result()
	if err != nil {
		global.GVA_LOG.Error("查询用户活跃MQTT连接失败", zap.Error(err), zap.String("pattern", pattern))
		return 1, err // 如果查询失败，返回序号1
	}

	global.GVA_LOG.Debug("查找用户活跃MQTT连接",
		zap.String("userID", userID),
		zap.String("username", username),
		zap.String("role", role),
		zap.String("pattern", pattern),
		zap.Strings("keys", keys))

	// 找到当前角色的最大序号
	maxSequence := 0
	expectedPrefix := fmt.Sprintf("%s-%s-", username, role)

	for _, key := range keys {
		// 获取clientID（存储在Redis value中）
		clientID, err := global.GVA_REDIS.Get(ctx, key).Result()
		if err != nil {
			global.GVA_LOG.Debug("获取clientID失败", zap.String("key", key), zap.Error(err))
			continue // 跳过无效的key
		}

		global.GVA_LOG.Debug("检查clientID",
			zap.String("key", key),
			zap.String("clientID", clientID),
			zap.String("expectedPrefix", expectedPrefix))

		// 解析clientID格式：username-role-sequence
		// 只处理匹配当前角色的clientID
		if strings.HasPrefix(clientID, expectedPrefix) {
			// 提取序号部分
			seqStr := strings.TrimPrefix(clientID, expectedPrefix)
			if seq, err := strconv.Atoi(seqStr); err == nil {
				global.GVA_LOG.Debug("找到匹配的序号",
					zap.String("clientID", clientID),
					zap.String("seqStr", seqStr),
					zap.Int("seq", seq),
					zap.Int("currentMax", maxSequence))
				if seq > maxSequence {
					maxSequence = seq
				}
			} else {
				global.GVA_LOG.Debug("序号解析失败", zap.String("seqStr", seqStr), zap.Error(err))
			}
		}
	}

	nextSequence := maxSequence + 1
	global.GVA_LOG.Info("MQTT序号生成完成",
		zap.String("userID", userID),
		zap.String("username", username),
		zap.String("role", role),
		zap.Int("maxSequence", maxSequence),
		zap.Int("nextSequence", nextSequence))

	return nextSequence, nil
}

// CreateMQTTClaims 创建MQTT专用JWT声明
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
	if !claims.IsValid() {
		return TokenClaimsInvalid
	}
	ctx := context.Background()
	redisKey := fmt.Sprintf("mqtt:active:%s:%s", claims.GetUserID(), claims.GetJTI())
	expiration := time.Unix(claims.ExpiresAt.Unix(), 0).Sub(time.Now())

	err := global.GVA_REDIS.Set(ctx, redisKey, claims.GetClientID(), expiration).Err()
	if err != nil {
		global.GVA_LOG.Error("存储MQTT active状态失败", zap.Error(err), zap.String("key", redisKey))
		return err
	}
	return nil
}

// IsMQTTJWTActive 检查MQTT JWT是否处于活跃状态
func (j *JWT) IsMQTTJWTActive(claims *request.MQTTClaims) (bool, error) {
	if !claims.IsValid() {
		return false, TokenClaimsInvalid
	}
	ctx := context.Background()
	redisKey := fmt.Sprintf("mqtt:active:%s:%s", claims.GetUserID(), claims.GetJTI())
	storedClientID, err := global.GVA_REDIS.Get(ctx, redisKey).Result()
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
