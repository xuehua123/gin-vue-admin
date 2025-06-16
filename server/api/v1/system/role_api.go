package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RoleConflictApi struct{}

// GenerateMQTTToken 生成MQTT Token（含挤下线）
// @Tags      RoleConflict
// @Summary   生成用于MQTT连接的JWT Token，并处理角色冲突
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     data  body      request.AssignRoleRequest true "角色分配请求"
// @Success   200   {object}  response.Response{data=request.MQTTTokenResponse,msg=string}  "生成成功"
// @Router    /role/generateMQTTToken [post]
func (a *RoleConflictApi) GenerateMQTTToken(c *gin.Context) {
	var req request.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 1. 从JWT Claims获取用户信息
	claims, err := utils.GetClaims(c)
	if err != nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	// 2. 生成唯一的MQTT ClientID 和 JTI
	// 我们需要一个JWT实例来创建claims
	jwtUtil := utils.NewJWT()
	mqttClaims, err := jwtUtil.CreateMQTTClaims(claims.UUID.String(), claims.Username, req.Role)
	if err != nil {
		global.GVA_LOG.Error("创建MQTT Claims失败!", zap.Error(err))
		response.FailWithMessage("创建MQTT凭证失败", c)
		return
	}

	// 3. 分配角色（包含挤下线逻辑）
	// 将新的JTI传递给服务层，以便存储在client_connections中
	err = roleConflictService.AssignRole(claims.UUID.String(), req.Role, mqttClaims.ClientID, mqttClaims.RegisteredClaims.ID, req.DeviceInfo, req.ForceKickExisting)
	if err != nil {
		global.GVA_LOG.Error("角色分配失败!", zap.Error(err))
		response.FailWithMessage("角色分配失败", c)
		return
	}

	// 4. 生成最终的MQTT JWT Token
	mqttToken, err := jwtUtil.CreateMQTTToken(mqttClaims)
	if err != nil {
		global.GVA_LOG.Error("生成MQTT Token失败!", zap.Error(err))
		response.FailWithMessage("生成Token失败", c)
		return
	}

	// 5. 返回结果
	response.OkWithDetailed(request.MQTTTokenResponse{
		ClientID: mqttClaims.ClientID,
		Token:    mqttToken,
		Role:     req.Role,
	}, "生成成功", c)
}

// CheckRoleConflict 检查角色冲突
// @Tags      RoleConflict
// @Summary   检查用户担当的角色是否存在设备冲突
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     data  body      request.CheckRoleConflictRequest true "角色冲突检测请求"
// @Success   200   {object}  response.Response{data=request.ConflictCheckResult,msg=string}  "检测成功"
// @Router    /role/checkConflict [post]
func (a *RoleConflictApi) CheckRoleConflict(c *gin.Context) {
	var req request.CheckRoleConflictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	claims, err := utils.GetClaims(c)
	if err != nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	result, err := roleConflictService.CheckRoleConflict(claims.UUID.String(), req.Role, req.ClientID)
	if err != nil {
		global.GVA_LOG.Error("检测角色冲突失败!", zap.Error(err))
		response.FailWithMessage("检测失败", c)
		return
	}

	response.OkWithDetailed(result, "检测成功", c)
}
