package system

import (
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"go.uber.org/zap"
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
	// 旧的内存黑名单缓存机制也不再使用：
	// if err != nil {
	// 	return
	// }
	// global.BlackCache.SetDefault(jwtList.Jwt, struct{}{}) // 不再使用内存黑名单缓存
	global.GVA_LOG.Warn("JsonInBlacklist 方法已废弃，请使用 RevokeActiveJWT")
	return fmt.Errorf("JsonInBlacklist 方法已废弃")
}

// RevokeActiveJWT 撤销活跃的JWT，在用户登出时调用
// 符合开发手册V2.0的jwt:active机制
// @param: claims *request.CustomClaims 来自当前用户的JWT声明
// @return: error
func (jwtService *JwtService) RevokeActiveJWT(claims *request.CustomClaims) error {
	if !claims.IsValid() {
		global.GVA_LOG.Error("无法撤销JWT：claims为空或必要字段缺失")
		return fmt.Errorf("无效的claims用于撤销JWT")
	}

	j := utils.NewJWT()
	err := j.RevokeJWT(claims)
	if err != nil {
		global.GVA_LOG.Error("撤销JWT失败",
			zap.Error(err),
			zap.String("userID", claims.GetUserID()),
			zap.String("jti", claims.GetJTI()))
		return fmt.Errorf("撤销JWT失败: %w", err)
	}

	global.GVA_LOG.Info("JWT已成功撤销",
		zap.String("userID", claims.GetUserID()),
		zap.String("jti", claims.GetJTI()))
	return nil
}

// RevokeAllUserJWTs 撤销用户的所有活跃JWT（强制下线）
// @param: userID string 用户UUID字符串
// @return: error
func (jwtService *JwtService) RevokeAllUserJWTs(userID string) error {
	j := utils.NewJWT()
	err := j.RevokeAllUserJWTs(userID)
	if err != nil {
		global.GVA_LOG.Error("撤销用户所有JWT失败",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("撤销用户所有JWT失败: %w", err)
	}

	global.GVA_LOG.Info("用户所有JWT已撤销", zap.String("userID", userID))
	return nil
}

// GetUserActiveJWTs 获取用户所有活跃的JWT信息
// @param: userID string 用户UUID字符串
// @return: map[string]string JWT键值对映射 (redisKey -> clientID)
// @return: error
func (jwtService *JwtService) GetUserActiveJWTs(userID string) (map[string]string, error) {
	j := utils.NewJWT()
	activeJWTs, err := j.GetUserActiveJWTs(userID)
	if err != nil {
		global.GVA_LOG.Error("获取用户活跃JWT失败",
			zap.Error(err),
			zap.String("userID", userID))
		return nil, fmt.Errorf("获取用户活跃JWT失败: %w", err)
	}

	global.GVA_LOG.Debug("获取用户活跃JWT成功",
		zap.String("userID", userID),
		zap.Int("count", len(activeJWTs)))

	return activeJWTs, nil
}

// IsJWTActive 检查JWT是否处于活跃状态
// @param: claims *request.CustomClaims JWT声明
// @return: bool 是否活跃
// @return: error
func (jwtService *JwtService) IsJWTActive(claims *request.CustomClaims) (bool, error) {
	j := utils.NewJWT()
	isActive, err := j.IsJWTActive(claims)
	if err != nil {
		global.GVA_LOG.Error("检查JWT活跃状态失败",
			zap.Error(err),
			zap.String("userID", claims.GetUserID()),
			zap.String("jti", claims.GetJTI()))
		return false, fmt.Errorf("检查JWT活跃状态失败: %w", err)
	}

	return isActive, nil
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
	global.GVA_LOG.Warn("GetRedisJWT 方法已废弃，请使用 GetUserActiveJWTs")
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
