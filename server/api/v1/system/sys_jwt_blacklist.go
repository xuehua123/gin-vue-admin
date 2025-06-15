package system

import (
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type JwtApi struct{}

// JsonInBlacklist
// @Tags      Jwt
// @Summary   jwt加入黑名单(登出，废弃)
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Response{msg=string}  "拉黑成功"
// @Router    /jwt/jsonInBlacklist [post]
func (jwtApi *JwtApi) JsonInBlacklist(c *gin.Context) {
	token := utils.GetToken(c)
	jwt := utils.NewJWT()
	claims, err := jwt.ParseToken(token)
	if err != nil {
		global.GVA_LOG.Error("解析token失败!", zap.Error(err))
		response.FailWithMessage("解析token失败", c)
		return
	}

	// 使用新的撤销机制
	err = jwtService.RevokeActiveJWT(claims)
	if err != nil {
		global.GVA_LOG.Error("JWT拉黑失败!", zap.Error(err))
		response.FailWithMessage("JWT拉黑失败", c)
	} else {
		utils.ClearToken(c)
		response.OkWithMessage("JWT拉黑成功", c)
	}
}

// GenerateMQTTToken 生成MQTT专用JWT
// @Tags      Jwt
// @Summary   生成MQTT专用JWT Token
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      request.MQTTTokenRequest  true  "角色和其他信息"
// @Success   200   {object}  response.Response{data=response.MQTTTokenResponse}  "生成成功"
// @Router    /jwt/generateMQTTToken [post]
func (jwtApi *JwtApi) GenerateMQTTToken(c *gin.Context) {
	var req request.MQTTTokenRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	// 从当前JWT中获取用户信息
	claims := utils.GetUserInfo(c)
	if claims == nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	// 验证角色参数
	if req.Role != "transmitter" && req.Role != "receiver" && req.Role != "admin" {
		response.FailWithMessage("无效的角色类型", c)
		return
	}

	jwt := utils.NewJWT()

	// 创建MQTT Claims
	mqttClaims, err := jwt.CreateMQTTClaims(
		claims.GetUserID(),
		claims.Username,
		req.Role,
	)
	if err != nil {
		global.GVA_LOG.Error("创建MQTT Claims失败", zap.Error(err))
		response.FailWithMessage("创建MQTT Claims失败", c)
		return
	}

	// 生成MQTT Token
	mqttToken, err := jwt.CreateMQTTToken(mqttClaims)
	if err != nil {
		global.GVA_LOG.Error("生成MQTT Token失败", zap.Error(err))
		response.FailWithMessage("生成MQTT Token失败", c)
		return
	}

	resp := response.MQTTTokenResponse{
		Token:     mqttToken,
		ClientID:  mqttClaims.ClientID,
		Role:      mqttClaims.Role,
		Sequence:  mqttClaims.Sequence,
		ExpiresAt: mqttClaims.ExpiresAt.Unix(),
	}

	global.GVA_LOG.Info("MQTT Token生成成功",
		zap.String("userID", claims.GetUserID()),
		zap.String("username", claims.Username),
		zap.String("role", req.Role),
		zap.String("clientID", mqttClaims.ClientID))

	response.OkWithDetailed(resp, "MQTT Token生成成功", c)
}

// RevokeMQTTToken 撤销MQTT JWT
// @Tags      Jwt
// @Summary   撤销MQTT JWT Token
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      request.RevokeMQTTTokenRequest  true  "要撤销的ClientID"
// @Success   200   {object}  response.Response{msg=string}  "撤销成功"
// @Router    /jwt/revokeMQTTToken [post]
func (jwtApi *JwtApi) RevokeMQTTToken(c *gin.Context) {
	var req request.RevokeMQTTTokenRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	// 从当前JWT中获取用户信息
	claims := utils.GetUserInfo(c)
	if claims == nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	jwt := utils.NewJWT()

	// 构造MQTT Claims用于撤销（只需要关键字段）
	mqttClaims := request.MQTTClaims{
		UserID:   claims.GetUserID(),
		ClientID: req.ClientID,
	}

	err = jwt.RevokeMQTTJWT(&mqttClaims)
	if err != nil {
		global.GVA_LOG.Error("撤销MQTT Token失败", zap.Error(err))
		response.FailWithMessage("撤销MQTT Token失败", c)
		return
	}

	global.GVA_LOG.Info("MQTT Token撤销成功",
		zap.String("userID", claims.GetUserID()),
		zap.String("clientID", req.ClientID))

	response.OkWithMessage("MQTT Token撤销成功", c)
}

// GetUserMQTTTokens 获取用户的所有活跃MQTT Token
// @Tags      Jwt
// @Summary   获取用户的所有活跃MQTT Token
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Response{data=[]response.MQTTTokenInfo}  "获取成功"
// @Router    /jwt/getUserMQTTTokens [get]
func (jwtApi *JwtApi) GetUserMQTTTokens(c *gin.Context) {
	// 从当前JWT中获取用户信息
	claims := utils.GetUserInfo(c)
	if claims == nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	userID := claims.GetUserID()
	pattern := "mqtt:active:" + userID + ":*"

	keys, err := global.GVA_REDIS.Keys(c, pattern).Result()
	if err != nil {
		global.GVA_LOG.Error("获取用户MQTT Token失败", zap.Error(err))
		response.FailWithMessage("获取用户MQTT Token失败", c)
		return
	}

	var tokenInfos []response.MQTTTokenInfo
	for _, key := range keys {
		info, err := global.GVA_REDIS.HGetAll(c, key).Result()
		if err != nil {
			continue
		}

		// 解析ClientID
		parts := strings.Split(key, ":")
		if len(parts) >= 3 {
			clientID := parts[len(parts)-1]
			tokenInfo := response.MQTTTokenInfo{
				ClientID:  clientID,
				Role:      info["role"],
				Username:  info["username"],
				CreatedAt: info["created_at"],
				JTI:       info["jti"],
			}
			tokenInfos = append(tokenInfos, tokenInfo)
		}
	}

	response.OkWithDetailed(tokenInfos, "获取成功", c)
}
