/**
 * 模拟数据生成hook
 * 用于在后端API尚未完全实现时提供测试数据
 */

import { ref } from 'vue'

/**
 * 生成模拟客户端列表数据
 */
export function useMockClientData() {
  const generateMockClients = (pageSize = 20) => {
    const roles = ['provider', 'receiver', 'none']
    const deviceNames = [
      'iPhone 13 Pro', 
      'Samsung Galaxy S23', 
      'POS Terminal A1', 
      'Card Reader X200', 
      'Mobile App v2.1',
      'Android Device',
      'Payment Terminal',
      'NFC Reader Pro'
    ]
    
    const userNames = [
      'alice.chen', 'bob.wang', 'carol.liu', 'david.zhang', 
      'emma.li', 'frank.wu', 'grace.xu', 'henry.zhao'
    ]
    
    const mockData = []
    
    for (let i = 1; i <= pageSize; i++) {
      const role = roles[Math.floor(Math.random() * roles.length)]
      const isOnline = Math.random() > 0.2 // 80%在线率
      const hasSession = isOnline && Math.random() > 0.7 // 30%的在线客户端有会话
      const connectedTime = new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000) // 最近7天内连接
      
      mockData.push({
        client_id: `client-${String(i).padStart(4, '0')}-${Math.random().toString(36).substr(2, 8)}`,
        user_id: userNames[Math.floor(Math.random() * userNames.length)],
        display_name: deviceNames[Math.floor(Math.random() * deviceNames.length)],
        role: role,
        ip_address: `192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 254) + 1}`,
        connected_at: connectedTime.toISOString(),
        is_online: isOnline,
        session_id: hasSession ? `session-${Date.now()}-${i}` : null,
        user_agent: generateUserAgent()
      })
    }
    
    return mockData
  }
  
  const generateUserAgent = () => {
    const agents = [
      'Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15',
      'Mozilla/5.0 (Linux; Android 12; SM-G975F) AppleWebKit/537.36',
      'NFCRelayApp/2.1.0 (Android 12; SDK 31)',
      'POS-Terminal/1.0 (Linux; ARM64)',
      'CardReader/3.2.1 (Embedded Linux)'
    ]
    return agents[Math.floor(Math.random() * agents.length)]
  }
  
  return {
    generateMockClients
  }
}

/**
 * 生成模拟会话列表数据
 */
export function useMockSessionData() {
  const generateMockSessions = (pageSize = 20) => {
    const statuses = ['paired', 'waiting', 'terminated']
    const sessionTypes = ['card_to_pos', 'pos_to_card', 'peer_to_peer']
    const deviceNames = [
      'iPhone 13 Pro', 
      'Samsung Galaxy S23', 
      'POS Terminal A1', 
      'Card Reader X200', 
      'Mobile Device',
      'Payment Terminal',
      'NFC Reader Pro'
    ]
    
    const userNames = [
      'alice.chen', 'bob.wang', 'carol.liu', 'david.zhang', 
      'emma.li', 'frank.wu', 'grace.xu', 'henry.zhao'
    ]
    
    const mockData = []
    
    for (let i = 1; i <= pageSize; i++) {
      const status = statuses[Math.floor(Math.random() * statuses.length)]
      const sessionType = sessionTypes[Math.floor(Math.random() * sessionTypes.length)]
      const createdTime = new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000) // 最近24小时内创建
      const lastActivity = new Date(createdTime.getTime() + Math.random() * 60 * 60 * 1000) // 创建后的随机时间
      
      mockData.push({
        session_id: `session-${String(i).padStart(4, '0')}-${Math.random().toString(36).substr(2, 8)}`,
        provider_client_id: `client-provider-${String(i).padStart(3, '0')}`,
        provider_user_id: userNames[Math.floor(Math.random() * userNames.length)],
        provider_display_name: deviceNames[Math.floor(Math.random() * deviceNames.length)],
        receiver_client_id: `client-receiver-${String(i).padStart(3, '0')}`,
        receiver_user_id: userNames[Math.floor(Math.random() * userNames.length)],
        receiver_display_name: deviceNames[Math.floor(Math.random() * deviceNames.length)],
        status: status,
        session_type: sessionType,
        created_at: createdTime.toISOString(),
        last_activity_at: lastActivity.toISOString(),
        apdu_count: Math.floor(Math.random() * 1000),
        data_transferred: Math.floor(Math.random() * 1024 * 1024), // bytes
        avg_latency: Math.floor(Math.random() * 100) + 20, // 20-120ms
        error_count: Math.floor(Math.random() * 5) // 0-4 errors
      })
    }
    
    return mockData
  }
  
  return {
    generateMockSessions
  }
}

/**
 * 生成模拟审计日志数据
 */
export function useMockAuditLogData() {
  const generateMockAuditLogs = (pageSize = 20) => {
    const eventTypes = [
      'session_established',
      'session_terminated', 
      'apdu_relayed_success',
      'apdu_relayed_failure',
      'client_connected',
      'client_disconnected',
      'auth_failure',
      'permission_denied'
    ]
    
    const mockData = []
    
    for (let i = 1; i <= pageSize; i++) {
      const eventType = eventTypes[Math.floor(Math.random() * eventTypes.length)]
      const timestamp = new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000) // 最近7天
      
      mockData.push({
        id: `audit-${i}`,
        timestamp: timestamp.toISOString(),
        event_type: eventType,
        session_id: Math.random() > 0.5 ? `session-${i}` : null,
        client_id_initiator: `client-${i}`,
        client_id_responder: Math.random() > 0.5 ? `client-${i + 1}` : null,
        user_id: `user-${i}`,
        source_ip: `192.168.1.${Math.floor(Math.random() * 254) + 1}`,
        details: generateEventDetails(eventType),
        severity: Math.random() > 0.8 ? 'high' : Math.random() > 0.5 ? 'medium' : 'low'
      })
    }
    
    return mockData.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
  }
  
  const generateEventDetails = (eventType) => {
    const detailsMap = {
      session_established: { message: 'NFC会话建立成功', duration: '2.3s' },
      session_terminated: { message: '会话正常终止', reason: 'client_request' },
      apdu_relayed_success: { message: 'APDU转发成功', data_length: Math.floor(Math.random() * 512) },
      apdu_relayed_failure: { message: 'APDU转发失败', error: 'timeout' },
      client_connected: { message: '客户端连接成功', protocol: 'websocket' },
      client_disconnected: { message: '客户端断开连接', reason: 'normal' },
      auth_failure: { message: '认证失败', attempts: Math.floor(Math.random() * 5) + 1 },
      permission_denied: { message: '权限不足', resource: 'session_create' }
    }
    
    return detailsMap[eventType] || { message: '未知事件' }
  }
  
  return {
    generateMockAuditLogs
  }
}

/**
 * 生成模拟仪表盘数据
 */
export function useMockDashboardData() {
  const generateMockDashboardStats = () => {
    const now = new Date()
    const connectionTrend = []
    const sessionTrend = []
    
    // 生成最近12个时间点的趋势数据
    for (let i = 11; i >= 0; i--) {
      const time = new Date(now.getTime() - i * 5 * 60 * 1000) // 每5分钟一个点
      
      connectionTrend.push({
        time: time.toISOString(),
        count: Math.floor(Math.random() * 50) + 20 // 20-70个连接
      })
      
      sessionTrend.push({
        time: time.toISOString(),
        count: Math.floor(Math.random() * 20) + 5 // 5-25个会话
      })
    }
    
    const activeConnections = Math.floor(Math.random() * 50) + 30
    const activeSessions = Math.floor(Math.random() * 20) + 10
    
    return {
      hub_status: 'online',
      active_connections: activeConnections,
      active_sessions: activeSessions,
      apdu_relayed_last_minute: Math.floor(Math.random() * 200) + 50,
      apdu_errors_last_hour: Math.floor(Math.random() * 10),
      avg_response_time: Math.floor(Math.random() * 100) + 20,
      system_load: Math.floor(Math.random() * 60) + 20,
      memory_usage: Math.floor(Math.random() * 70) + 30,
      connection_trend: connectionTrend,
      session_trend: sessionTrend,
      provider_count: Math.floor(activeConnections * 0.6),
      receiver_count: Math.floor(activeConnections * 0.4),
      paired_sessions: Math.floor(activeSessions * 0.8),
      waiting_sessions: Math.floor(activeSessions * 0.2),
      total_apdu_today: Math.floor(Math.random() * 10000) + 5000,
      error_rate: Math.random() * 5,
      recent_events: generateMockRecentEvents()
    }
  }
  
  const generateMockRecentEvents = () => {
    const eventTypes = ['connect', 'disconnect', 'session', 'error', 'info']
    const events = []
    
    for (let i = 0; i < 10; i++) {
      const type = eventTypes[Math.floor(Math.random() * eventTypes.length)]
      const time = new Date(Date.now() - Math.random() * 60 * 60 * 1000) // 最近1小时
      
      events.push({
        id: `event-${i}`,
        type: type,
        title: generateEventTitle(type),
        time: time.toISOString(),
        description: `Event details for ${type} event`
      })
    }
    
    return events.sort((a, b) => new Date(b.time) - new Date(a.time))
  }
  
  const generateEventTitle = (type) => {
    const titleMap = {
      connect: '新客户端连接',
      disconnect: '客户端断开连接', 
      session: '新会话建立',
      error: 'APDU转发错误',
      info: '系统状态更新'
    }
    return titleMap[type] || '未知事件'
  }
  
  return {
    generateMockDashboardStats
  }
}

/**
 * 生成模拟配置数据
 */
export function useMockConfigData() {
  const generateMockConfigData = () => {
    const configCategories = ['server', 'session', 'security', 'logging', 'monitoring', 'network']
    const configTypes = ['string', 'number', 'boolean', 'object', 'array']
    const configStatuses = ['default', 'modified', 'restart_required', 'error']
    
    const configTemplates = [
      // 服务器配置
      {
        category: 'server',
        name: '监听地址',
        path: 'server.listen_address',
        type: 'string',
        value: '0.0.0.0',
        defaultValue: '0.0.0.0',
        description: 'NFC中继服务器监听的IP地址',
        constraints: {
          pattern: '^(?:[0-9]{1,3}\\.){3}[0-9]{1,3}$',
          examples: ['0.0.0.0', '127.0.0.1', '192.168.1.100']
        }
      },
      {
        category: 'server',
        name: '监听端口',
        path: 'server.listen_port',
        type: 'number',
        value: 8080,
        defaultValue: 8080,
        description: 'NFC中继服务器监听的端口号',
        constraints: {
          minValue: 1024,
          maxValue: 65535,
          examples: [8080, 9000, 3000]
        }
      },
      {
        category: 'server',
        name: '最大连接数',
        path: 'server.max_connections',
        type: 'number',
        value: 1000,
        defaultValue: 500,
        description: '服务器允许的最大并发连接数',
        constraints: {
          minValue: 10,
          maxValue: 10000
        }
      },
      {
        category: 'server',
        name: '连接超时',
        path: 'server.connection_timeout',
        type: 'string',
        value: '30s',
        defaultValue: '30s',
        description: '客户端连接超时时间',
        constraints: {
          pattern: '^\\d+[smh]$',
          examples: ['30s', '5m', '1h']
        }
      },
      
      // 会话配置
      {
        category: 'session',
        name: '会话超时',
        path: 'session.session_timeout',
        type: 'string',
        value: '300s',
        defaultValue: '300s',
        description: 'NFC中继会话的超时时间',
        constraints: {
          pattern: '^\\d+[smh]$',
          examples: ['300s', '10m', '1h']
        }
      },
      {
        category: 'session',
        name: '最大会话数',
        path: 'session.max_sessions',
        type: 'number',
        value: 100,
        defaultValue: 50,
        description: '系统允许的最大并发会话数',
        constraints: {
          minValue: 1,
          maxValue: 1000
        }
      },
      {
        category: 'session',
        name: '配对超时',
        path: 'session.pairing_timeout',
        type: 'string',
        value: '60s',
        defaultValue: '60s',
        description: '会话配对的超时时间'
      },
      {
        category: 'session',
        name: '心跳间隔',
        path: 'session.heartbeat_interval',
        type: 'string',
        value: '30s',
        defaultValue: '30s',
        description: '会话心跳检测间隔'
      },
      
      // 安全配置
      {
        category: 'security',
        name: '启用认证',
        path: 'security.enable_auth',
        type: 'boolean',
        value: true,
        defaultValue: false,
        description: '是否启用客户端认证'
      },
      {
        category: 'security',
        name: '启用TLS',
        path: 'security.enable_tls',
        type: 'boolean',
        value: true,
        defaultValue: false,
        description: '是否启用TLS加密传输'
      },
      {
        category: 'security',
        name: '认证密钥',
        path: 'security.auth_key',
        type: 'password',
        value: 'secret-key-12345',
        defaultValue: '',
        description: '客户端认证使用的密钥',
        readonly: false
      },
      {
        category: 'security',
        name: '最大失败尝试',
        path: 'security.max_auth_failures',
        type: 'number',
        value: 5,
        defaultValue: 3,
        description: '允许的最大认证失败次数',
        constraints: {
          minValue: 1,
          maxValue: 10
        }
      },
      {
        category: 'security',
        name: '允许的客户端',
        path: 'security.allowed_clients',
        type: 'array',
        value: ['192.168.1.*', '10.0.0.*'],
        defaultValue: ['*'],
        description: '允许连接的客户端IP地址列表'
      },
      
      // 日志配置
      {
        category: 'logging',
        name: '日志级别',
        path: 'logging.level',
        type: 'string',
        value: 'info',
        defaultValue: 'info',
        description: '系统日志记录级别',
        constraints: {
          options: ['debug', 'info', 'warn', 'error', 'fatal']
        }
      },
      {
        category: 'logging',
        name: '日志文件',
        path: 'logging.log_file',
        type: 'string',
        value: '/var/log/nfc-relay/app.log',
        defaultValue: '/var/log/nfc-relay/app.log',
        description: '日志文件的存储路径'
      },
      {
        category: 'logging',
        name: '启用日志轮转',
        path: 'logging.enable_rotation',
        type: 'boolean',
        value: true,
        defaultValue: true,
        description: '是否启用日志文件轮转'
      },
      {
        category: 'logging',
        name: '最大文件大小',
        path: 'logging.max_file_size',
        type: 'string',
        value: '100MB',
        defaultValue: '100MB',
        description: '单个日志文件的最大大小',
        constraints: {
          pattern: '^\\d+[KMGT]B$',
          examples: ['10MB', '100MB', '1GB']
        }
      },
      
      // 监控配置
      {
        category: 'monitoring',
        name: '启用监控',
        path: 'monitoring.enabled',
        type: 'boolean',
        value: true,
        defaultValue: false,
        description: '是否启用系统监控功能'
      },
      {
        category: 'monitoring',
        name: '监控端口',
        path: 'monitoring.port',
        type: 'number',
        value: 9090,
        defaultValue: 9090,
        description: 'Prometheus监控指标端口',
        constraints: {
          minValue: 1024,
          maxValue: 65535
        }
      },
      {
        category: 'monitoring',
        name: '指标路径',
        path: 'monitoring.metrics_path',
        type: 'string',
        value: '/metrics',
        defaultValue: '/metrics',
        description: 'Prometheus指标的HTTP路径'
      },
      {
        category: 'monitoring',
        name: '健康检查间隔',
        path: 'monitoring.health_check_interval',
        type: 'string',
        value: '30s',
        defaultValue: '30s',
        description: '系统健康检查的间隔时间'
      },
      
      // 网络配置
      {
        category: 'network',
        name: 'WebSocket写入等待',
        path: 'network.websocket_write_wait',
        type: 'string',
        value: '10s',
        defaultValue: '10s',
        description: 'WebSocket写入操作的等待时间'
      },
      {
        category: 'network',
        name: 'WebSocket Pong等待',
        path: 'network.websocket_pong_wait',
        type: 'string',
        value: '60s',
        defaultValue: '60s',
        description: 'WebSocket Pong消息的等待时间'
      },
      {
        category: 'network',
        name: '最大消息大小',
        path: 'network.websocket_max_message_bytes',
        type: 'number',
        value: 2048,
        defaultValue: 1024,
        description: 'WebSocket消息的最大字节数',
        constraints: {
          minValue: 512,
          maxValue: 65536
        }
      },
      {
        category: 'network',
        name: 'TCP KeepAlive',
        path: 'network.tcp_keepalive',
        type: 'boolean',
        value: true,
        defaultValue: true,
        description: '是否启用TCP KeepAlive'
      }
    ]
    
    // 为每个配置项添加随机状态和修改时间
    const mockData = configTemplates.map((template, index) => {
      const status = configStatuses[Math.floor(Math.random() * configStatuses.length)]
      const isModified = status === 'modified' || status === 'restart_required'
      
      return {
        ...template,
        key: `config-${index + 1}`,
        status: status,
        lastModified: isModified ? 
          new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString() : 
          null,
        modifiedBy: isModified ? 'admin' : null,
        readonly: template.readonly !== undefined ? template.readonly : Math.random() > 0.8 // 20%只读
      }
    })
    
    return mockData
  }
  
  return {
    generateMockConfigData
  }
} 