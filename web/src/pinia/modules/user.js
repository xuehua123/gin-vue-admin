import { login, getUserInfo } from '@/api/user'
import { jsonInBlacklist } from '@/api/jwt'
import { ElLoading, ElMessage } from 'element-plus'
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useRouterStore } from './router'
import { useCookies } from '@vueuse/integrations/useCookies'
import { useStorage } from '@vueuse/core'
import { emitter } from '@/utils/bus.js'

export const useUserStore = defineStore('user', () => {
  const loadingInstance = ref(null)

  const userInfo = ref({
    uuid: '',
    nickName: '',
    headerImg: '',
    authority: {}
  })
  const token = useStorage('token', '')
  const xToken = useCookies('x-token')
  const currentToken = computed(() => token.value || xToken.value || '')

  const setUserInfo = (val) => {
    userInfo.value = val
    // 通过事件总线通知app store更新配置，避免循环依赖
    if (val.originSetting) {
      emitter.emit('updateUserSettings', val.originSetting)
    }
  }

  const setToken = (val) => {
    token.value = val
    xToken.value = val
  }

  const NeedInit = async () => {
    await ClearStorage()
    return true
  }

  const ResetUserInfo = (value = {}) => {
    userInfo.value = {
      ...userInfo.value,
      ...value
    }
  }
  /* 获取用户信息*/
  const GetUserInfo = async () => {
    const res = await getUserInfo()
    if (res.code === 0) {
      setUserInfo(res.data.userInfo)
    }
    return res
  }
  /* 登录*/
  const LoginIn = async (loginInfo) => {
    try {
      loadingInstance.value = ElLoading.service({
        fullscreen: true,
        text: '登录中，请稍候...'
      })

      const res = await login(loginInfo)

      if (res.code !== 0) {
        return false
      }
      // 登陆成功，设置用户信息和权限相关信息
      setUserInfo(res.data.user)
      setToken(res.data.token)

      const routerStore = useRouterStore()
      await routerStore.SetAsyncRouter()

      // 移除所有路由操作，只返回成功状态和目标路由
      const defaultRouter = userInfo.value.authority.defaultRouter
      return { success: true, defaultRouter }
    } catch (error) {
      console.error('LoginIn error:', error)
      return { success: false }
    } finally {
      loadingInstance.value?.close()
    }
  }
  /* 登出*/
  const LoginOut = async () => {
    const res = await jsonInBlacklist()
    if (res.code !== 0) {
      // 即使登出失败，也应该清理前端状态
      console.error('jsonInBlacklist fail', res)
    }
    await ClearStorage()
  }
  /* 清理数据 */
  const ClearStorage = async () => {
    token.value = ''
    // 使用remove方法正确删除cookie
    xToken.remove()
    sessionStorage.clear()
    // 清理所有相关的localStorage项
    localStorage.removeItem('originSetting')
    localStorage.removeItem('token')
  }

  return {
    userInfo,
    token: currentToken,
    NeedInit,
    ResetUserInfo,
    GetUserInfo,
    LoginIn,
    LoginOut,
    setToken,
    loadingInstance,
    ClearStorage
  }
})
