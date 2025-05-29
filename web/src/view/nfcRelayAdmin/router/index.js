// NFC中继管理模块路由配置
export const nfcRelayAdminRoutes = {
  path: '/nfc-relay-admin',
  name: 'NfcRelayAdmin',
  redirect: '/nfc-relay-admin/dashboard',
  component: () => import('@/view/nfcRelayAdmin/index.vue'),
  meta: {
    title: 'NFC中继管理',
    icon: 'Connection',
    keepAlive: true
  },
  children: [
    {
      path: 'dashboard',
      name: 'NfcRelayDashboard',
      component: () => import('@/view/nfcRelayAdmin/dashboard/index.vue'),
      meta: {
        title: '概览仪表盘',
        icon: 'Odometer',
        keepAlive: true
      }
    },
    {
      path: 'clients',
      name: 'NfcRelayClientManagement',
      component: () => import('@/view/nfcRelayAdmin/clientManagement/index.vue'),
      meta: {
        title: '连接管理',
        icon: 'User',
        keepAlive: true
      }
    },
    {
      path: 'sessions',
      name: 'NfcRelaySessionManagement',
      component: () => import('@/view/nfcRelayAdmin/sessionManagement/index.vue'),
      meta: {
        title: '会话管理',
        icon: 'ChatDotRound',
        keepAlive: true
      }
    },
    {
      path: 'audit-logs',
      name: 'NfcRelayAuditLogs',
      component: () => import('@/view/nfcRelayAdmin/auditLogs/index.vue'),
      meta: {
        title: '审计日志',
        icon: 'Document',
        keepAlive: true
      }
    },
    {
      path: 'security',
      name: 'NfcRelaySecurity',
      component: () => import('@/view/nfcRelayAdmin/security/index.vue'),
      meta: {
        title: '安全管理',
        icon: 'Lock',
        keepAlive: true
      }
    },
    {
      path: 'configuration',
      name: 'NfcRelayConfiguration',
      component: () => import('@/view/nfcRelayAdmin/configuration/index.vue'),
      meta: {
        title: '系统配置',
        icon: 'Setting',
        keepAlive: true
      }
    }
  ]
} 