// 权限按钮展示指令
// 移除直接导入
// import { useUserStore } from '@/pinia/modules/user'
export default {
  install: (app) => {
    app.directive('auth', {
      // 当被绑定的元素插入到 DOM 中时……
      mounted: async function (el, binding) {
        // 动态导入useUserStore，避免循环依赖
        const { useUserStore } = await import('@/pinia/modules/user')
        const userStore = useUserStore()
        const userInfo = userStore.userInfo
        let type = ''
        switch (Object.prototype.toString.call(binding.value)) {
          case '[object Array]':
            type = 'Array'
            break
          case '[object String]':
            type = 'String'
            break
          case '[object Number]':
            type = 'Number'
            break
          default:
            type = ''
            break
        }
        if (type === '') {
          el.parentNode.removeChild(el)
          return
        }
        const waitUse = binding.value.toString().split(',')
        let flag = waitUse.some((item) => Number(item) === userInfo.authorityId)
        if (binding.modifiers.not) {
          flag = !flag
        }
        if (!flag) {
          el.parentNode.removeChild(el)
        }
      }
    })
  }
}
