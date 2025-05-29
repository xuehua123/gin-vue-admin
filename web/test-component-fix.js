// 测试组件修复的脚本
import { isReactive } from 'vue'
import { ConfirmDialog, StatCard, SearchForm, DataTable } from './src/view/nfcRelayAdmin/components/index.js'

console.log('🧪 测试组件响应式状态:')
console.log('ConfirmDialog is reactive:', isReactive(ConfirmDialog))
console.log('StatCard is reactive:', isReactive(StatCard))
console.log('SearchForm is reactive:', isReactive(SearchForm))
console.log('DataTable is reactive:', isReactive(DataTable))

if (!isReactive(ConfirmDialog) && !isReactive(StatCard) && !isReactive(SearchForm) && !isReactive(DataTable)) {
  console.log('✅ 所有组件都已正确标记为非响应式!')
} else {
  console.log('❌ 仍有组件被设为响应式')
} 