// NFC中继管理模块测试用模拟数据

/**
 * 生成模拟的仪表盘统计数据
 */
export const mockDashboardStats = () => ({
  hub_status: 'online',
  active_connections: Math.floor(Math.random() * 200) + 50,
  active_sessions: Math.floor(Math.random() * 50) + 10,
  apdu_relayed_last_minute: Math.floor(Math.random() * 300) + 50,
  apdu_errors_last_hour: Math.floor(Math.random() * 10),
  connection_trend: generateTrendData('connections'),
  session_trend: generateTrendData('sessions')
})

/**
 * 生成模拟的客户端列表数据
 */
export const mockClientList = (count = 20) => {
  const clients = []
  const roles = ['provider', 'receiver', 'none']
  const deviceNames = [
    'iPhone 13 Pro', 'Samsung Galaxy S22', 'Pixel 6 Pro', 
    'POS Terminal A', 'Card Reader B', 'Payment Gateway C'
  ]
  
  for (let i = 1; i <= count; i++) {
    const role = roles[Math.floor(Math.random() * roles.length)]
    const isOnline = Math.random() > 0.2
    const hasSession = isOnline && Math.random() > 0.7
    
    clients.push({
      client_id: `client-${Date.now()}-${i}`,
      user_id: `user-${1000 + i}`,
      display_name: deviceNames[Math.floor(Math.random() * deviceNames.length)],
      role: role,
      ip_address: `192.168.1.${Math.floor(Math.random() * 254) + 1}`,
      connected_at: new Date(Date.now() - Math.random() * 86400000).toISOString(),
      is_online: isOnline,
      session_id: hasSession ? `session-${Date.now()}-${i}` : null
    })
  }
  
  return clients
}

/**
 * 生成模拟的会话列表数据
 */
export const mockSessionList = (count = 15) => {
  const sessions = []
  const statuses = ['paired', 'waiting_for_pairing']
  const providerNames = ['iPhone 13', 'Samsung S22', 'Pixel 6']
  const receiverNames = ['POS Terminal A', 'Card Reader B', 'Payment Gateway']
  
  for (let i = 1; i <= count; i++) {
    const status = statuses[Math.floor(Math.random() * statuses.length)]
    const createdTime = new Date(Date.now() - Math.random() * 86400000)
    const lastActivityTime = new Date(createdTime.getTime() + Math.random() * 3600000)
    
    sessions.push({
      session_id: `session-${Date.now()}-${i}`,
      provider_client_id: `provider-client-${i}`,
      provider_user_id: `provider-user-${1000 + i}`,
      provider_display_name: providerNames[Math.floor(Math.random() * providerNames.length)],
      receiver_client_id: `receiver-client-${i}`,
      receiver_user_id: `receiver-user-${2000 + i}`,
      receiver_display_name: receiverNames[Math.floor(Math.random() * receiverNames.length)],
      status: status,
      created_at: createdTime.toISOString(),
      last_activity_at: lastActivityTime.toISOString()
    })
  }
  
  return sessions
}

/**
 * 生成模拟的审计日志数据
 */
export const mockAuditLogs = (count = 50) => {
  const logs = []
  const eventTypes = [
    'session_established', 
    'apdu_relayed_success', 
    'apdu_relayed_failure', 
    'auth_failure', 
    'client_disconnected', 
    'session_terminated'
  ]
  
  for (let i = 1; i <= count; i++) {
    const eventType = eventTypes[Math.floor(Math.random() * eventTypes.length)]
    const hasSession = ['session_established', 'apdu_relayed_success', 'session_terminated'].includes(eventType)
    
    logs.push({
      timestamp: new Date(Date.now() - Math.random() * 86400000 * 7).toISOString(),
      event_type: eventType,
      session_id: hasSession ? `session-${Date.now()}-${i}` : null,
      client_id_initiator: `client-${i}`,
      client_id_responder: Math.random() > 0.5 ? `client-${i + 100}` : null,
      user_id: `user-${1000 + i}`,
      source_ip: `192.168.1.${Math.floor(Math.random() * 254) + 1}`,
      details: generateLogDetails(eventType)
    })
  }
  
  return logs.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
}

/**
 * 生成模拟的客户端详情数据
 */
export const mockClientDetails = (clientId) => ({
  client_id: clientId,
  user_id: 'user-' + Math.floor(Math.random() * 1000),
  display_name: 'iPhone 13 Pro',
  role: 'provider',
  ip_address: '192.168.1.100',
  user_agent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15',
  connected_at: new Date(Date.now() - 3600000).toISOString(),
  last_message_at: new Date(Date.now() - 300000).toISOString(),
  is_online: true,
  session_id: 'session-' + Date.now(),
  sent_message_count: Math.floor(Math.random() * 100) + 20,
  received_message_count: Math.floor(Math.random() * 100) + 15,
  connection_events: [
    {
      timestamp: new Date(Date.now() - 3600000).toISOString(),
      event: 'Connected'
    },
    {
      timestamp: new Date(Date.now() - 3590000).toISOString(),
      event: 'Authenticated'
    },
    {
      timestamp: new Date(Date.now() - 3580000).toISOString(),
      event: 'RoleDeclared'
    }
  ],
  related_audit_logs_summary: [
    {
      timestamp: new Date(Date.now() - 600000).toISOString(),
      event_type: 'apdu_relayed_success',
      details_summary: 'APDU转发成功，长度32字节'
    }
  ]
})

/**
 * 生成模拟的会话详情数据
 */
export const mockSessionDetails = (sessionId) => ({
  session_id: sessionId,
  status: 'paired',
  created_at: new Date(Date.now() - 7200000).toISOString(),
  last_activity_at: new Date(Date.now() - 300000).toISOString(),
  terminated_at: null,
  termination_reason: null,
  provider_info: {
    client_id: 'provider-client-123',
    user_id: 'provider-user-456',
    display_name: 'iPhone 13 Pro',
    ip_address: '192.168.1.100'
  },
  receiver_info: {
    client_id: 'receiver-client-789',
    user_id: 'receiver-user-012',
    display_name: 'POS Terminal A',
    ip_address: '192.168.1.200'
  },
  apdu_exchange_count: {
    upstream: Math.floor(Math.random() * 50) + 10,
    downstream: Math.floor(Math.random() * 50) + 8
  },
  session_events_history: [
    {
      timestamp: new Date(Date.now() - 7200000).toISOString(),
      event: 'SessionCreated'
    },
    {
      timestamp: new Date(Date.now() - 7180000).toISOString(),
      event: 'ProviderJoined',
      client_id: 'provider-client-123'
    },
    {
      timestamp: new Date(Date.now() - 7160000).toISOString(),
      event: 'ReceiverJoined',
      client_id: 'receiver-client-789'
    },
    {
      timestamp: new Date(Date.now() - 7140000).toISOString(),
      event: 'SessionPaired'
    }
  ]
})

/**
 * 生成模拟的系统配置数据
 */
export const mockSystemConfig = () => ({
  server: {
    listen_address: '0.0.0.0',
    listen_port: 8080,
    max_connections: 1000,
    connection_timeout: '30s'
  },
  session: {
    session_timeout: '30m',
    max_sessions: 100,
    pairing_timeout: '5m',
    heartbeat_interval: '30s'
  },
  security: {
    enable_auth: true,
    enable_tls: false,
    auth_key: 'your-secret-auth-key-here-12345',
    max_auth_failures: 5
  },
  logging: {
    level: 'INFO',
    log_file: '/var/log/nfc-relay/nfc-relay.log',
    enable_rotation: true,
    max_file_size: '100MB'
  },
  runtime: {
    uptime: Math.floor(Math.random() * 500000) + 100000,
    version: 'v1.0.0',
    go_version: 'go1.21.0',
    build_time: '2024-01-15T10:30:00Z'
  }
})

// 私有工具函数

/**
 * 生成趋势数据
 */
function generateTrendData(type) {
  const data = []
  const now = new Date()
  const baseValue = type === 'connections' ? 120 : 15
  
  for (let i = 11; i >= 0; i--) {
    const time = new Date(now.getTime() - i * 5 * 60 * 1000)
    const timeStr = time.getHours().toString().padStart(2, '0') + ':' + 
                   time.getMinutes().toString().padStart(2, '0')
    const count = baseValue + Math.floor(Math.random() * 20) - 10
    data.push({ time: timeStr, count: Math.max(0, count) })
  }
  
  return data
}

/**
 * 生成日志详情
 */
function generateLogDetails(eventType) {
  const details = {}
  
  switch (eventType) {
    case 'apdu_relayed_success':
    case 'apdu_relayed_failure':
      details.apdu_length = Math.floor(Math.random() * 64) + 16
      if (eventType === 'apdu_relayed_failure') {
        details.error_message = '连接超时'
      }
      break
      
    case 'session_terminated':
      details.reason = '客户端主动断开'
      details.session_duration = '00:' + 
        Math.floor(Math.random() * 60).toString().padStart(2, '0') + ':' +
        Math.floor(Math.random() * 60).toString().padStart(2, '0')
      break
      
    case 'auth_failure':
      details.error_message = '认证令牌无效'
      details.retry_count = Math.floor(Math.random() * 3) + 1
      break
      
    default:
      break
  }
  
  return details
} 