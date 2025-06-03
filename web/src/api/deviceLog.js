import service from '@/utils/request'

// @Summary 分页获取设备日志列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.GetDeviceLogsRequest true "分页获取设备日志列表"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "分页获取设备日志列表"
// @Router /deviceLog/getDeviceLogsList [post]
export const getDeviceLogsList = (data) => {
  return service({
    url: '/deviceLog/getDeviceLogsList',
    method: 'post',
    data: data
  })
}

// @Summary 强制设备下线
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.ForceLogoutRequest true "用户ID, 客户端ID, 下线原因"
// @Success 200 {object} response.Response{msg=string} "强制设备下线"
// @Router /deviceLog/forceLogoutDevice [post]
export const forceLogoutDevice = (data) => {
  return service({
    url: '/deviceLog/forceLogoutDevice',
    method: 'post',
    data: data
  })
}

// @Summary 获取设备日志统计
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param userId query string false "用户ID"
// @Success 200 {object} response.Response{data=response.DeviceLogStats,msg=string} "获取设备日志统计"
// @Router /deviceLog/getDeviceLogStats [get]
export const getDeviceLogStats = (userId = '') => {
  return service({
    url: '/deviceLog/getDeviceLogStats',
    method: 'get',
    params: { userId }
  })
}

// @Summary 强制用户下线（所有设备）
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.ForceLogoutUserRequest true "用户ID, 下线原因"
// @Success 200 {object} response.Response{msg=string} "强制用户下线"
// @Router /user/forceLogoutUser [post]
export const forceLogoutUser = (data) => {
  return service({
    url: '/user/forceLogoutUser',
    method: 'post',
    data: data
  })
} 