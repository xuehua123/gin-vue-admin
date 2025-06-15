import service from '@/utils/request'

// ====== 交易管理API ======

// @Summary 创建交易
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.CreateTransactionRequest true "创建交易"
// @Success 200 {object} response.Response{data=response.CreateTransactionResponse,msg=string} "创建交易成功"
// @Router /nfc-relay/transactions [post]
export const createTransaction = (data) => {
  return service({
    url: '/nfc-relay/transactions',
    method: 'post',
    data: data
  })
}

// @Summary 获取交易列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.GetTransactionListRequest true "分页获取交易列表"
// @Success 200 {object} response.Response{data=response.TransactionListResponse,msg=string} "获取交易列表成功"
// @Router /nfc-relay/transactions [get]
export const getTransactionList = (params) => {
  return service({
    url: '/nfc-relay/transactions',
    method: 'get',
    params: params
  })
}

// @Summary 获取交易详情
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param transaction_id path string true "交易ID"
// @Success 200 {object} response.Response{data=response.TransactionDetailResponse,msg=string} "获取交易详情成功"
// @Router /nfc-relay/transactions/{transaction_id} [get]
export const getTransaction = (transactionId, includeAPDU = false) => {
  return service({
    url: `/nfc-relay/transactions/${transactionId}`,
    method: 'get',
    params: { include_apdu: includeAPDU }
  })
}

// @Summary 更新交易状态
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.UpdateTransactionStatusRequest true "更新交易状态"
// @Success 200 {object} response.Response{data=response.UpdateTransactionStatusResponse,msg=string} "更新交易状态成功"
// @Router /nfc-relay/transactions/status [put]
export const updateTransactionStatus = (data) => {
  return service({
    url: '/nfc-relay/transactions/status',
    method: 'put',
    data: data
  })
}

// @Summary 删除交易
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param transaction_id path string true "交易ID"
// @Success 200 {object} response.Response{msg=string} "删除交易成功"
// @Router /nfc-relay/transactions/{transaction_id} [delete]
export const deleteTransaction = (transactionId) => {
  return service({
    url: `/nfc-relay/transactions/${transactionId}`,
    method: 'delete'
  })
}

// @Summary 批量更新交易状态
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.BatchUpdateTransactionRequest true "批量更新交易状态"
// @Success 200 {object} response.Response{data=response.BatchOperationResponse,msg=string} "批量更新成功"
// @Router /nfc-relay/transactions/batch-update [put]
export const batchUpdateStatus = (data) => {
  return service({
    url: '/nfc-relay/transactions/batch-update',
    method: 'put',
    data: data
  })
}

// ====== APDU消息API ======

// @Summary 发送APDU消息
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.SendAPDURequest true "发送APDU消息"
// @Success 200 {object} response.Response{msg=string} "发送APDU消息成功"
// @Router /nfc-relay/transactions/apdu [post]
export const sendAPDU = (data) => {
  return service({
    url: '/nfc-relay/transactions/apdu',
    method: 'post',
    data: data
  })
}

// @Summary 获取APDU消息列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param transaction_id query string true "交易ID"
// @Param direction query string false "消息方向"
// @Success 200 {object} response.Response{data=[]nfc_relay.NFCAPDUMessage,msg=string} "获取APDU消息列表成功"
// @Router /nfc-relay/transactions/apdu [get]
export const getAPDUList = (params) => {
  return service({
    url: '/nfc-relay/transactions/apdu',
    method: 'get',
    params: params
  })
}

// ====== 统计和监控API ======

// @Summary 获取统计信息
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param period query string false "统计周期"
// @Param status query string false "状态筛选"
// @Success 200 {object} response.Response{data=response.TransactionStatisticsResponse,msg=string} "获取统计信息成功"
// @Router /nfc-relay/transactions/statistics [get]
export const getStatistics = (params) => {
  return service({
    url: '/nfc-relay/transactions/statistics',
    method: 'get',
    params: params
  })
}

// @Summary 导出交易数据
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param format query string false "导出格式"
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} response.Response{msg=string} "导出成功"
// @Router /nfc-relay/transactions/export [get]
export const exportTransactions = (params) => {
  return service({
    url: '/nfc-relay/transactions/export',
    method: 'get',
    params: params,
    responseType: 'blob'
  })
}

// ====== 系统状态API ======

// @Summary 健康检查
// @accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{msg=string} "系统正常"
// @Router /nfc-relay/health [get]
export const healthCheck = () => {
  return service({
    url: '/nfc-relay/health',
    method: 'get',
    donNotShowLoading: true
  })
}

// @Summary MQTT服务状态
// @accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{msg=string} "MQTT状态"
// @Router /nfc-relay/mqtt/status [get]
export const getMQTTStatus = () => {
  return service({
    url: '/nfc-relay/mqtt/status',
    method: 'get',
    donNotShowLoading: true
  })
} 