import { reactive } from 'vue'

export const useBtnAuth = async () => {
  // 动态导入useRoute，避免循环依赖
  const { useRoute } = await import('vue-router')
  const route = useRoute()
  return route.meta.btns || reactive({})
}
