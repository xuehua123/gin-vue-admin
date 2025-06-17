// Pinia stores 的统一导出
// 注意：为了避免循环依赖，这个文件不再导入createPinia
// pinia实例从 @/pinia/store 导入

export { useAppStore } from '@/pinia/modules/app'
export { useUserStore } from '@/pinia/modules/user'
export { useDictionaryStore } from '@/pinia/modules/dictionary'
export { useRouterStore } from '@/pinia/modules/router'
export { useParamsStore } from '@/pinia/modules/params'
