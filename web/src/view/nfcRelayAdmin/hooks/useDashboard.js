/**
 * 仪表盘页面组合式函数
 * 封装仪表盘的数据获取、状态管理和业务逻辑
 */

import { ref, computed, reactive } from 'vue'
import { useRealTimeData } from './useRealTime'
import { getDashboardStatsEnhanced } from '@/api/nfcRelayAdmin'
import { formatDateTime, formatNumber, formatPercentage } from '../utils/formatters'
import { ElMessage } from 'element-plus'

/**
 * 仪表盘数据管理hook
 */
export function useDashboard() {
  // 基础状态
  const loading = ref(false)
  const error = ref(null)
  
  // 仪表盘数据
  const stats = reactive({
    hubStatus: 'online',
    activeConnections: 0,
    activeSessions: 0,
    apduRelayedLastMinute: 0,
    apduErrorsLastHour: 0,
    avgResponseTime: 0,
    systemLoad: 0,
    memoryUsage: 0,
    connectionTrend: [],
    sessionTrend: [],
    lastUpdateTime: null
  })
  
  // 详细数据
  const details = reactive({
    providerCount: 0,
    receiverCount: 0,
    pairedSessions: 0,
    waitingSessions: 0,
    totalApduToday: 0,
    errorRate: 0
  })
  
  // 实时事件
  const recentEvents = ref([])
  
  /**
   * 获取仪表盘数据
   */
  const fetchDashboardData = async () => {
    try {
      loading.value = true
      error.value = null
      
      const response = await getDashboardStatsEnhanced()
      
      if (response.code === 0) {
        const data = response.data
        
        // 更新核心统计
        stats.hubStatus = data.hub_status || 'online'
        stats.activeConnections = data.active_connections || 0
        stats.activeSessions = data.active_sessions || 0
        stats.apduRelayedLastMinute = data.apdu_relayed_last_minute || 0
        stats.apduErrorsLastHour = data.apdu_errors_last_hour || 0
        stats.avgResponseTime = data.avg_response_time || 0
        stats.systemLoad = data.system_load || 0
        stats.memoryUsage = data.memory_usage || 0
        stats.connectionTrend = data.connection_trend || []
        stats.sessionTrend = data.session_trend || []
        stats.lastUpdateTime = new Date()
        
        // 更新详细数据
        details.providerCount = data.provider_count || 0
        details.receiverCount = data.receiver_count || 0
        details.pairedSessions = data.paired_sessions || 0
        details.waitingSessions = data.waiting_sessions || 0
        details.totalApduToday = data.total_apdu_today || 0
        details.errorRate = data.error_rate || 0
        
        // 更新实时事件
        if (data.recent_events) {
          recentEvents.value = data.recent_events.map(event => ({
            id: event.id || Date.now() + Math.random(),
            title: event.title || event.message,
            type: event.type || 'info',
            time: new Date(event.timestamp),
            description: event.description
          }))
        }
        
        return data
      } else {
        throw new Error(response.msg || '获取数据失败')
      }
    } catch (err) {
      error.value = err
      ElMessage.error('获取仪表盘数据失败: ' + err.message)
      throw err
    } finally {
      loading.value = false
    }
  }
  
  // 使用实时数据更新
  const {
    data: realTimeData,
    loading: realTimeLoading,
    error: realTimeError,
    startPolling,
    stopPolling,
    refresh
  } = useRealTimeData(fetchDashboardData, 30000)
  
  // 计算属性
  const isOnline = computed(() => stats.hubStatus === 'online')
  
  const systemHealthScore = computed(() => {
    // 根据各项指标计算系统健康分数 (0-100)
    let score = 100
    
    // 响应时间影响
    if (stats.avgResponseTime > 100) score -= 20
    else if (stats.avgResponseTime > 50) score -= 10
    
    // 系统负载影响
    if (stats.systemLoad > 80) score -= 25
    else if (stats.systemLoad > 60) score -= 15
    
    // 内存使用影响
    if (stats.memoryUsage > 90) score -= 20
    else if (stats.memoryUsage > 75) score -= 10
    
    // 错误率影响
    if (details.errorRate > 5) score -= 15
    else if (details.errorRate > 2) score -= 8
    
    return Math.max(0, score)
  })
  
  const performanceLevel = computed(() => {
    const score = systemHealthScore.value
    if (score >= 90) return { text: '优秀', color: '#67C23A', type: 'success' }
    if (score >= 75) return { text: '良好', color: '#E6A23C', type: 'warning' }
    if (score >= 60) return { text: '一般', color: '#F56C6C', type: 'danger' }
    return { text: '较差', color: '#F56C6C', type: 'danger' }
  })
  
  const statusIndicator = computed(() => ({
    text: isOnline.value ? '运行正常' : '服务离线',
    color: isOnline.value ? '#67C23A' : '#F56C6C',
    type: isOnline.value ? 'success' : 'danger',
    icon: isOnline.value ? 'CircleCheck' : 'CircleClose'
  }))
  
  // 格式化方法
  const formatters = {
    /**
     * 格式化连接数显示
     */
    connectionText: computed(() => {
      return `${stats.activeConnections} 个连接`
    }),
    
    /**
     * 格式化会话数显示
     */
    sessionText: computed(() => {
      return `${stats.activeSessions} 个会话`
    }),
    
    /**
     * 格式化APDU转发速率
     */
    apduRateText: computed(() => {
      const rate = stats.apduRelayedLastMinute
      if (rate === 0) return '0/分钟'
      if (rate < 60) return `${rate}/分钟`
      return `${Math.round(rate / 60)}/秒`
    }),
    
    /**
     * 格式化错误率
     */
    errorRateText: computed(() => {
      return formatPercentage(details.errorRate / 100, true, 2)
    }),
    
    /**
     * 格式化响应时间
     */
    responseTimeText: computed(() => {
      return `${stats.avgResponseTime}ms`
    }),
    
    /**
     * 格式化系统负载
     */
    systemLoadText: computed(() => {
      return `${stats.systemLoad}%`
    }),
    
    /**
     * 格式化内存使用
     */
    memoryUsageText: computed(() => {
      return `${stats.memoryUsage}%`
    }),
    
    /**
     * 格式化最后更新时间
     */
    lastUpdateText: computed(() => {
      return stats.lastUpdateTime ? formatDateTime(stats.lastUpdateTime, 'HH:mm:ss') : '-'
    })
  }
  
  // 趋势数据处理
  const trendData = computed(() => ({
    connection: stats.connectionTrend.map(item => ({
      time: formatDateTime(item.time, 'HH:mm'),
      count: item.count || 0
    })),
    session: stats.sessionTrend.map(item => ({
      time: formatDateTime(item.time, 'HH:mm'),
      count: item.count || 0
    }))
  }))
  
  // 统计卡片数据
  const statCards = computed(() => [
    {
      title: '运行状态',
      value: statusIndicator.value.text,
      icon: statusIndicator.value.icon,
      iconColor: statusIndicator.value.color,
      type: statusIndicator.value.type,
      subtitle: `健康分数: ${systemHealthScore.value}`,
      trend: null
    },
    {
      title: '活动连接',
      value: stats.activeConnections,
      icon: 'Connection',
      iconColor: '#67C23A',
      type: 'primary',
      subtitle: `Provider: ${details.providerCount} | Receiver: ${details.receiverCount}`,
      trend: {
        type: 'neutral',
        value: formatters.connectionText.value,
        label: '当前连接'
      }
    },
    {
      title: '活动会话',
      value: stats.activeSessions,
      icon: 'ChatDotRound',
      iconColor: '#E6A23C',
      type: 'warning',
      subtitle: `已配对: ${details.pairedSessions} | 等待: ${details.waitingSessions}`,
      trend: {
        type: 'neutral',
        value: formatters.sessionText.value,
        label: '当前会话'
      }
    },
    {
      title: 'APDU转发',
      value: stats.apduRelayedLastMinute,
      icon: 'DataLine',
      iconColor: '#409EFF',
      type: 'info',
      subtitle: `错误数: ${stats.apduErrorsLastHour}/小时`,
      trend: {
        type: stats.apduErrorsLastHour === 0 ? 'up' : 'down',
        value: formatters.apduRateText.value,
        label: '转发速率'
      }
    }
  ])
  
  /**
   * 手动刷新数据
   */
  const refreshData = async () => {
    try {
      await refresh()
      ElMessage.success('数据已刷新')
    } catch (err) {
      // 错误已在fetchDashboardData中处理
    }
  }
  
  /**
   * 添加实时事件
   */
  const addEvent = (event) => {
    const newEvent = {
      id: Date.now() + Math.random(),
      title: event.title,
      type: event.type || 'info',
      time: new Date(),
      description: event.description
    }
    
    recentEvents.value.unshift(newEvent)
    
    // 保持最多20个事件
    if (recentEvents.value.length > 20) {
      recentEvents.value = recentEvents.value.slice(0, 20)
    }
  }
  
  /**
   * 清空事件列表
   */
  const clearEvents = () => {
    recentEvents.value = []
  }
  
  return {
    // 状态
    loading: computed(() => loading.value || realTimeLoading.value),
    error: computed(() => error.value || realTimeError.value),
    
    // 数据
    stats,
    details,
    recentEvents,
    
    // 计算属性
    isOnline,
    systemHealthScore,
    performanceLevel,
    statusIndicator,
    statCards,
    trendData,
    
    // 格式化方法
    formatters,
    
    // 操作方法
    fetchDashboardData,
    refreshData,
    startPolling,
    stopPolling,
    addEvent,
    clearEvents
  }
} 