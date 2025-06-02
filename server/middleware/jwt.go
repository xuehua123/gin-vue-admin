package middleware

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 我们这里jwt鉴权取头部信息 x-token 登录时回返回token信息 这里前端需要把token存储到cookie或者本地localStorage中 不过需要跟后端协商过期时间 可以约定刷新令牌或者重新登录
		token := utils.GetToken(c)
		if token == "" {
			response.NoAuth("未登录或非法访问", c)
			c.Abort()
			return
		}

		// 旧的黑名单检查逻辑，将被移除或注释
		// if isBlacklist(token) {
		// 	response.NoAuth("您的帐户异地登陆或令牌失效", c)
		// 	utils.ClearToken(c)
		// 	c.Abort()
		// 	return
		// }

		j := utils.NewJWT()
		// parseToken 解析token包含的信息
		claims, err := j.ParseToken(token)
		if err != nil {
			if errors.Is(err, utils.TokenExpired) {
				response.NoAuth("授权已过期", c)
				utils.ClearToken(c)
				c.Abort()
				return
			}
			response.NoAuth(err.Error(), c)
			utils.ClearToken(c)
			c.Abort()
			return
		}

		// 新增：基于 jwt:active 机制的校验
		if claims.BaseClaims.UUID.String() == "" || claims.RegisteredClaims.ID == "" || claims.BaseClaims.ClientID == "" {
			response.NoAuth("令牌信息不完整或无效", c)
			c.Abort()
			return
		}
		userID := claims.BaseClaims.UUID.String()
		jti := claims.RegisteredClaims.ID
		clientID := claims.BaseClaims.ClientID

		redisKey := fmt.Sprintf("jwt:active:%s:%s", userID, jti)
		storedClientID, redisErr := global.GVA_REDIS.Get(context.Background(), redisKey).Result()

		if redisErr == redis.Nil {
			response.NoAuth("令牌已失效或不存在 (Not Active)", c)
			utils.ClearToken(c)
			c.Abort()
			return
		} else if redisErr != nil {
			global.GVA_LOG.Error("Redis查询JWT Active状态失败", zap.Error(redisErr), zap.String("redisKey", redisKey))
			response.NoAuth("验证令牌状态失败，请稍后重试", c)
			c.Abort()
			return
		}

		if storedClientID != clientID {
			response.NoAuth("令牌与会话不匹配 (ClientID Mismatch)", c)
			utils.ClearToken(c)
			c.Abort()
			return
		}

		// 已登录用户被管理员禁用 需要使该用户的jwt失效 此处比较消耗性能 如果需要 请自行打开
		// 用户被删除的逻辑 需要优化 此处比较消耗性能 如果需要 请自行打开

		//if user, err := userService.FindUserByUuid(claims.UUID.String()); err != nil || user.Enable == 2 {
		//	_ = jwtService.JsonInBlacklist(system.JwtBlacklist{Jwt: token})
		//	response.FailWithDetailed(gin.H{"reload": true}, err.Error(), c)
		//	c.Abort()
		//}
		c.Set("claims", claims)
		if claims.ExpiresAt.Unix()-time.Now().Unix() < claims.BufferTime {
			dr, _ := utils.ParseDuration(global.GVA_CONFIG.JWT.ExpiresTime)
			newClaimsExpiresAt := jwt.NewNumericDate(time.Now().Add(dr))

			// 创建新的 JTI 和 ClientID 用于刷新后的 token
			newJTI := uuid.New().String()      // 为刷新后的 token 生成新的 JTI
			newClientID := uuid.New().String() // 为刷新后的 token 生成新的 ClientID

			newBaseClaims := claims.BaseClaims
			newBaseClaims.ClientID = newClientID // 更新 ClientID

			refreshedClaims := j.CreateClaims(newBaseClaims) // 使用更新后的 BaseClaims
			refreshedClaims.ExpiresAt = newClaimsExpiresAt
			refreshedClaims.RegisteredClaims.ID = newJTI // 明确指定是 RegisteredClaims.ID

			newToken, errToken := j.CreateToken(refreshedClaims)
			if errToken != nil {
				global.GVA_LOG.Error("刷新token失败!", zap.Error(errToken))
			} else {
				// 将新的 active JWT 存入 Redis
				newRedisKey := fmt.Sprintf("jwt:active:%s:%s", userID, newJTI)
				newExpiration := time.Unix(refreshedClaims.ExpiresAt.Unix(), 0).Sub(time.Now())
				if redisSetErr := global.GVA_REDIS.Set(context.Background(), newRedisKey, newClientID, newExpiration).Err(); redisSetErr != nil {
					global.GVA_LOG.Error("存储刷新后的JWT到Redis失败!", zap.Error(redisSetErr), zap.String("redisKey", newRedisKey))
				}

				// (可选) 旧的 active JWT 如何处理？
				// 方案1: 立即移除旧的 active JWT。这会导致原 token 立即失效。
				// global.GVA_REDIS.Del(context.Background(), redisKey)
				// 方案2: 让旧的 active JWT 自然过期。原 token 在其原始有效期内仍然有效，但客户端应使用新 token。
				// 目前选择方案2，不立即删除旧 Redis Key。

				c.Header("new-token", newToken)
				c.Header("new-expires-at", strconv.FormatInt(refreshedClaims.ExpiresAt.Unix(), 10))
				utils.SetToken(c, newToken, int(dr.Seconds()))
			}
		}
		c.Next()

		// 这部分是为了确保即使在 c.Next() 之后，如果 new-token 被设置了，也能正确返回给客户端
		// 原代码中这部分逻辑似乎有点多余或可以简化，但为保持兼容性暂时保留
		if newToken, exists := c.Get("new-token"); exists {
			c.Header("new-token", newToken.(string))
		}
		if newExpiresAt, exists := c.Get("new-expires-at"); exists {
			c.Header("new-expires-at", newExpiresAt.(string))
		}
	}
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
