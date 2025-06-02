package middleware

import (
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/gin-gonic/gin"
)

// JWTAuth JWT认证中间件
// 实现开发手册V2.0要求的完整JWT认证流程
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取token
		token := utils.GetToken(c)
		if token == "" {
			response.NoAuth("未登录或非法访问", c)
			c.Abort()
			return
		}

		j := utils.NewJWT()
		// 解析token获取claims
		claims, err := j.ParseToken(token)
		if err != nil {
			handleTokenError(c, err)
			return
		}

		// 验证JWT是否处于活跃状态（实现开发手册的jwt:active机制）
		isActive, err := j.IsJWTActive(claims)
		if err != nil {
			global.GVA_LOG.Error("验证JWT活跃状态失败",
				zap.Error(err),
				zap.String("userID", claims.GetUserID()),
				zap.String("jti", claims.GetJTI()))
			response.NoAuth("验证令牌状态失败，请稍后重试", c)
			utils.ClearToken(c)
			c.Abort()
			return
		}

		if !isActive {
			response.NoAuth("令牌已失效或已被撤销", c)
			utils.ClearToken(c)
			c.Abort()
			return
		}

		// 可选：检查用户状态（被禁用的用户）
		// if user, err := userService.FindUserByUuid(claims.UUID.String()); err != nil || user.Enable == 2 {
		//	   _ = j.RevokeJWT(claims)
		//	   response.NoAuth("用户已被禁用", c)
		//	   c.Abort()
		//	   return
		// }

		// 设置claims到上下文
		c.Set("claims", claims)

		// JWT刷新逻辑
		if shouldRefreshToken(claims) {
			newToken, err := refreshJWTToken(j, claims)
			if err != nil {
				global.GVA_LOG.Error("刷新token失败!", zap.Error(err))
			} else {
				// 设置新token到响应头
				c.Header("new-token", newToken)
				c.Header("new-expires-at", strconv.FormatInt(claims.ExpiresAt.Unix(), 10))

				// 设置新token到cookie
				dr, _ := utils.ParseDuration(global.GVA_CONFIG.JWT.ExpiresTime)
				utils.SetToken(c, newToken, int(dr.Seconds()))
			}
		}

		c.Next()

		// 处理响应后的新token设置（保持兼容性）
		if newToken, exists := c.Get("new-token"); exists {
			c.Header("new-token", newToken.(string))
		}
		if newExpiresAt, exists := c.Get("new-expires-at"); exists {
			c.Header("new-expires-at", newExpiresAt.(string))
		}
	}
}

// handleTokenError 处理token解析错误
func handleTokenError(c *gin.Context, err error) {
	var message string
	switch {
	case errors.Is(err, utils.TokenExpired):
		message = "授权已过期"
	case errors.Is(err, utils.TokenMalformed):
		message = "令牌格式错误"
	case errors.Is(err, utils.TokenSignatureInvalid):
		message = "令牌签名无效"
	case errors.Is(err, utils.TokenNotValidYet):
		message = "令牌尚未生效"
	case errors.Is(err, utils.TokenClaimsInvalid):
		message = "令牌声明信息无效"
	case errors.Is(err, utils.TokenNotActive):
		message = "令牌未激活或已被撤销"
	default:
		message = "令牌验证失败"
	}

	global.GVA_LOG.Warn("JWT验证失败",
		zap.Error(err),
		zap.String("message", message))

	response.NoAuth(message, c)
	utils.ClearToken(c)
	c.Abort()
}

// shouldRefreshToken 判断是否需要刷新token
func shouldRefreshToken(claims *request.CustomClaims) bool {
	return claims.ExpiresAt.Unix()-time.Now().Unix() < claims.BufferTime
}

// refreshJWTToken 刷新JWT token
func refreshJWTToken(j *utils.JWT, oldClaims *request.CustomClaims) (string, error) {
	// 计算新的过期时间
	dr, _ := utils.ParseDuration(global.GVA_CONFIG.JWT.ExpiresTime)
	newExpiresAt := jwt.NewNumericDate(time.Now().Add(dr))

	// 生成新的JTI和ClientID
	newJTI := uuid.New().String()
	newClientID := uuid.New().String()

	// 创建新的Claims
	newBaseClaims := oldClaims.BaseClaims
	newBaseClaims.ClientID = newClientID

	refreshedClaims := j.CreateClaims(newBaseClaims)
	refreshedClaims.ExpiresAt = newExpiresAt
	refreshedClaims.RegisteredClaims.ID = newJTI

	// 创建新token（会自动存储到Redis）
	newToken, err := j.CreateToken(refreshedClaims)
	if err != nil {
		return "", err
	}

	// 可选：立即移除旧的JWT（还是让其自然过期？）
	// 方案1: 立即移除旧JWT
	// _ = j.RevokeJWT(oldClaims)
	// 方案2: 让旧JWT自然过期（默认方案）
	// 这里选择方案2，保持现有逻辑

	global.GVA_LOG.Info("JWT刷新成功",
		zap.String("userID", oldClaims.GetUserID()),
		zap.String("oldJTI", oldClaims.GetJTI()),
		zap.String("newJTI", newJTI),
		zap.String("oldClientID", oldClaims.GetClientID()),
		zap.String("newClientID", newClientID))

	return newToken, nil
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: IsBlacklist
//@description: 判断JWT是否在黑名单内部
//@param: jwt string
//@return: bool

// func isBlacklist(jwt string) bool {
// 	 _, ok := global.BlackCache.Get(jwt)
// 	 return ok
// }
