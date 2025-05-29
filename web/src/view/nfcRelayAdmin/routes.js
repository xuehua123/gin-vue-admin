// NFC中继管理模块路由配置 - 用于菜单注册
// 这个文件符合gin-vue-admin的路由结构要求

const nfcRelayAdminRoute = {
  path: '/nfc-relay-admin',
  name: 'nfc-relay-admin',
  redirect: '/nfc-relay-admin/dashboard',
  component: 'view/nfcRelayAdmin/index.vue',
  meta: {
    title: 'NFC中继管理',
    icon: 'Connection',
    keepAlive: true
  },
  children: [
    {
      path: 'dashboard',
      name: 'nfc-relay-dashboard',
      component: 'view/nfcRelayAdmin/dashboard/index.vue',
      meta: {
        title: '概览仪表盘',
        icon: 'Odometer',
        keepAlive: true
      }
    },
    {
      path: 'clients',
      name: 'nfc-relay-clients',
      component: 'view/nfcRelayAdmin/clientManagement/index.vue',
      meta: {
        title: '连接管理',
        icon: 'User',
        keepAlive: true
      }
    },
    {
      path: 'sessions',
      name: 'nfc-relay-sessions',
      component: 'view/nfcRelayAdmin/sessionManagement/index.vue',
      meta: {
        title: '会话管理',
        icon: 'ChatDotRound',
        keepAlive: true
      }
    },
    {
      path: 'audit-logs',
      name: 'nfc-relay-audit-logs',
      component: 'view/nfcRelayAdmin/auditLogs/index.vue',
      meta: {
        title: '审计日志',
        icon: 'Document',
        keepAlive: true
      }
    },
    {
      path: 'security',
      name: 'nfc-relay-security',
      component: 'view/nfcRelayAdmin/security/index.vue',
      meta: {
        title: '安全管理',
        icon: 'Lock',
        keepAlive: true
      }
    },
    {
      path: 'configuration',
      name: 'nfc-relay-configuration',
      component: 'view/nfcRelayAdmin/configuration/index.vue',
      meta: {
        title: '系统配置',
        icon: 'Setting',
        keepAlive: true
      }
    }
  ]
}

export default nfcRelayAdminRoute 