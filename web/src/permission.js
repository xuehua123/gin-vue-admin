import { useUserStore } from '@/pinia/modules/user'
import { useRouterStore } from '@/pinia/modules/router'
import getPageTitle from '@/utils/page'
import router from '@/router'
import Nprogress from 'nprogress'
import 'nprogress/nprogress.css'

// 导出设置函数，而不是立即执行
export function setupPermissions() {
  // 配置 NProgress
  Nprogress.configure({
    showSpinner: false,
    ease: 'ease',
    speed: 500
  })

  // 白名单路由
  const WHITE_LIST = ['Login', 'Init']

  // 处理路由加载
  const setupRouter = async () => {
    const routerStore = useRouterStore()
    const userStore = useUserStore()
    await Promise.all([routerStore.SetAsyncRouter(), userStore.GetUserInfo()])
    routerStore.asyncRouters.forEach((route) => router.addRoute(route))
    return true
  }

  // 移除加载动画
  const removeLoading = () => {
    const element = document.getElementById('gva-loading-box')
    element?.remove()
  }

  // 处理组件缓存
  const handleKeepAlive = async (to) => {
    if (to.matched && to.matched.length > 2) {
      for (let i = 1; i < to.matched.length; i++) {
        const element = to.matched[i - 1]
        if (element.name === 'layout' || to.matched[i - 1].redirect) {
          to.matched.splice(i, 1)
          await handleKeepAlive(to)
        }
      }
    }
  }

  // 处理路由重定向
  const handleRedirect = (to, userStore) => {
    if (router.hasRoute(userStore.userInfo.authority.defaultRouter)) {
      return { ...to, replace: true }
    }
    return { path: '/layout/404' }
  }

  // 路由守卫
  router.beforeEach(async (to, from, next) => {
    const userStore = useUserStore()
    const routerStore = useRouterStore()
    to.meta.matched = [...to.matched]
    handleKeepAlive(to)
    const token = userStore.token
    const name = to.meta.title || to.name
    document.title = getPageTitle(name, to)
    Nprogress.start()
    if (to.meta.client) {
      next()
      Nprogress.done()
      return
    }
    if (token) {
      if (WHITE_LIST.indexOf(to.name) > -1) {
        if (!routerStore.asyncRouterFlag && router.hasRoute(userStore.userInfo.authority.defaultRouter)) {
          next({ name: userStore.userInfo.authority.defaultRouter })
        } else {
          next()
        }
      } else {
        if (routerStore.asyncRouterFlag) {
          next()
        } else {
          await setupRouter()
          if (router.hasRoute(to.name)) {
            next({ ...to, replace: true })
          } else {
            next({ path: '/layout/404' })
          }
        }
      }
    } else {
      if (WHITE_LIST.indexOf(to.name) > -1) {
        next()
      } else {
        next({
          name: 'Login',
          query: {
            redirect: document.location.hash
          }
        })
      }
    }
  })

  // 路由加载完成
  router.afterEach(() => {
    document.querySelector('.main-cont.main-right')?.scrollTo(0, 0)
    Nprogress.done()
  })

  // 路由错误处理
  router.onError(() => {
    Nprogress.remove()
  })

  // 移除初始加载动画
  removeLoading()
}
