package system

import (
	"context"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"go.uber.org/zap"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
)

type JwtService struct{}

var JwtServiceApp = new(JwtService)

// JsonInBlacklist 旧的拉黑jwt逻辑，将不再使用或被 RevokeActiveJWT 替代
// @author: [piexlmax](https://github.com/piexlmax)
// @function: JsonInBlacklist
// @description: 拉黑jwt
// @param: jwtList model.JwtBlacklist
// @return: err error
func (jwtService *JwtService) JsonInBlacklist(jwtList system.JwtBlacklist) (err error) {
	// err = global.GVA_DB.Create(&jwtList).Error // 不再写入数据库黑名单
	// if err != nil {
	// 	return
	// }
	// global.BlackCache.SetDefault(jwtList.Jwt, struct{}{}) // 不再使用内存黑名单缓存
	global.GVA_LOG.Warn("JsonInBlacklist 方法已废弃，请使用 RevokeActiveJWT")
	return nil
}

// RevokeActiveJWT 撤销活跃的JWT，在用户登出时调用
// @param: claims *request.CustomClaims 来自当前用户的JWT声明
// @return: err error
func (jwtService *JwtService) RevokeActiveJWT(claims *request.CustomClaims) error {
	if claims == nil || claims.BaseClaims.UUID.String() == "" || claims.RegisteredClaims.ID == "" {
		global.GVA_LOG.Error("无法撤销JWT：claims为空或必要字段缺失")
		return fmt.Errorf("无效的claims用于撤销JWT")
	}

	userID := claims.BaseClaims.UUID.String()
	jti := claims.RegisteredClaims.ID
	redisKey := fmt.Sprintf("jwt:active:%s:%s", userID, jti)

	_, err := global.GVA_REDIS.Del(context.Background(), redisKey).Result()
	if err != nil {
		global.GVA_LOG.Error("从Redis删除jwt:active失败", zap.Error(err), zap.String("key", redisKey))
		return fmt.Errorf("redis删除jwt:active失败: %w", err)
	}
	global.GVA_LOG.Info("JWT已成功从active列表移除(撤销)", zap.String("key", redisKey))
	return nil
}

// GetRedisJWT 旧的从redis取jwt的方法，已被新的 jwt:active 机制取代，应废弃
// @author: [piexlmax](https://github.com/piexlmax)
// @function: GetRedisJWT
// @description: 从redis取jwt
// @param: userName string
// @return: redisJWT string, err error
func (jwtService *JwtService) GetRedisJWT(userName string) (redisJWT string, err error) {
	// redisJWT, err = global.GVA_REDIS.Get(context.Background(), userName).Result()
	// return redisJWT, err
	global.GVA_LOG.Warn("GetRedisJWT 方法已废弃")
	return "", fmt.Errorf("GetRedisJWT 方法已废弃")
}

// LoadAll 旧的加载黑名单逻辑，不再需要
func LoadAll() {
	// var data []string
	// err := global.GVA_DB.Model(&system.JwtBlacklist{}).Select("jwt").Find(&data).Error
	// if err != nil {
	// 	global.GVA_LOG.Error("加载数据库jwt黑名单失败!", zap.Error(err))
	// 	return
	// }
	// for i := 0; i < len(data); i++ {
	// 	global.BlackCache.SetDefault(data[i], struct{}{})
	// } // jwt黑名单 加入 BlackCache 中
	global.GVA_LOG.Info("旧的JWT黑名单加载逻辑(LoadAll)已不再执行。")
}
