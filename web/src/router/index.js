import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/init',
    name: 'Init',
    component: () => import('@/view/init/index.vue')
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/view/login/index.vue')
  },
  {
    path: '/scanUpload',
    name: 'ScanUpload',
    meta: {
      title: '扫码上传',
      client: true
    },
    component: () => import('@/view/example/upload/scanUpload.vue')
  },
  {
    path: '/test-device-log',
    name: 'DeviceLogTest',
    meta: {
      title: '设备日志测试'
    },
    component: () => import('@/view/test/deviceLogTest.vue')
  },
  {
    path: '/user-enhanced',
    name: 'UserEnhanced',
    meta: {
      title: '用户管理增强版'
    },
    component: () => import('@/view/superAdmin/user/userEnhanced.vue')
  },
  {
    path: '/device-log',
    name: 'DeviceLogManagement',
    meta: {
      title: '设备日志管理'
    },
    component: () => import('@/view/superAdmin/deviceLog/deviceLogManagement.vue')
  },
  {
    path: '/nfc-relay/monitor',
    name: 'NFCRelayMonitor',
    meta: {
      title: 'NFC中继监控'
    },
    component: () => import('@/view/nfcRelay/transaction/monitor.vue')
  },
  {
    path: '/:catchAll(.*)',
    meta: {
      closeTab: true
    },
    component: () => import('@/view/error/index.vue')
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
