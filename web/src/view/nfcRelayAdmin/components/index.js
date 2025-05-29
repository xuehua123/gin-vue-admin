// NFC中继管理组件导出
import { markRaw } from 'vue'

// 基础组件
import StatCardComponent from './StatCard.vue'
import TrendChartComponent from './TrendChart.vue'
import SearchFormComponent from './SearchForm.vue'
import DataTableComponent from './DataTable.vue'
import ConfirmDialogComponent from './ConfirmDialog.vue'

// 高级组件
import RealtimeStatCardComponent from './RealtimeStatCard.vue'
import AdvancedSearchFormComponent from './AdvancedSearchForm.vue'
import AdvancedFiltersComponent from './AdvancedFilters.vue'
import DetailModalComponent from './DetailModal.vue'
import FullscreenLayoutComponent from './FullscreenLayout.vue'

// 专业组件
import AlertCenterComponent from './AlertCenter.vue'
import ApduMonitorComponent from './ApduMonitor.vue'
import RealtimeLogStreamComponent from './RealtimeLogStream.vue'
import SessionTimelineComponent from './SessionTimeline.vue'
import SystemConfigManagerComponent from './SystemConfigManager.vue'

// 客户端管理相关组件
import ClientDetailDialogComponent from '../clientManagement/components/ClientDetailDialog.vue'

// 会话管理相关组件
import SessionDetailDialogComponent from '../sessionManagement/components/SessionDetailDialog.vue'

// 审计日志相关组件
import LogDetailDialogComponent from '../auditLogs/components/LogDetailDialog.vue'

// 系统配置相关组件
import ConfigDetailDialogComponent from '../configuration/components/ConfigDetailDialog.vue'

// 使用 markRaw 导出所有组件，防止响应式警告
export const StatCard = markRaw(StatCardComponent)
export const TrendChart = markRaw(TrendChartComponent)
export const SearchForm = markRaw(SearchFormComponent)
export const DataTable = markRaw(DataTableComponent)
export const ConfirmDialog = markRaw(ConfirmDialogComponent)

export const RealtimeStatCard = markRaw(RealtimeStatCardComponent)
export const AdvancedSearchForm = markRaw(AdvancedSearchFormComponent)
export const AdvancedFilters = markRaw(AdvancedFiltersComponent)
export const DetailModal = markRaw(DetailModalComponent)
export const FullscreenLayout = markRaw(FullscreenLayoutComponent)

export const AlertCenter = markRaw(AlertCenterComponent)
export const ApduMonitor = markRaw(ApduMonitorComponent)
export const RealtimeLogStream = markRaw(RealtimeLogStreamComponent)
export const SessionTimeline = markRaw(SessionTimelineComponent)
export const SystemConfigManager = markRaw(SystemConfigManagerComponent)

export const ClientDetailDialog = markRaw(ClientDetailDialogComponent)
export const SessionDetailDialog = markRaw(SessionDetailDialogComponent)
export const LogDetailDialog = markRaw(LogDetailDialogComponent)
export const ConfigDetailDialog = markRaw(ConfigDetailDialogComponent) 