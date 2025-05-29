<!--
  数据表格组件
  功能完整的表格，支持分页、排序、选择、操作等
-->
<template>
  <div class="data-table">
    <el-table
      ref="tableRef"
      :data="data"
      :loading="loading"
      :height="height"
      :max-height="maxHeight"
      v-bind="tableProps"
      @selection-change="handleSelectionChange"
      @sort-change="handleSortChange"
      @row-click="handleRowClick"
      stripe
      border
    >
      <!-- 选择列 -->
      <el-table-column
        v-if="selectable"
        type="selection"
        width="55"
        align="center"
        fixed="left"
      />
      
      <!-- 序号列 -->
      <el-table-column
        v-if="showIndex"
        type="index"
        label="序号"
        width="80"
        align="center"
        fixed="left"
        :index="indexMethod"
      />
      
      <!-- 数据列 -->
      <template v-for="column in columns" :key="column.prop">
        <el-table-column
          :prop="column.prop"
          :label="column.label"
          :width="column.width"
          :min-width="column.minWidth"
          :fixed="column.fixed"
          :align="column.align || 'left'"
          :sortable="column.sortable"
          :show-overflow-tooltip="column.showOverflowTooltip !== false"
        >
          <template #default="{ row, column: col, $index }">
            <!-- 自定义插槽 -->
            <slot 
              v-if="column.slot" 
              :name="column.slot" 
              :row="row" 
              :column="col" 
              :index="$index"
            />
            
            <!-- 状态标签 -->
            <el-tag
              v-else-if="column.type === 'tag'"
              :type="getTagType(row[column.prop], column.tagMap)"
              size="small"
            >
              {{ getTagText(row[column.prop], column.tagMap) }}
            </el-tag>
            
            <!-- 时间格式化 -->
            <span v-else-if="column.type === 'datetime'">
              {{ formatDateTime(row[column.prop]) }}
            </span>
            
            <!-- 数字格式化 -->
            <span v-else-if="column.type === 'number'">
              {{ formatNumber(row[column.prop], column.format) }}
            </span>
            
            <!-- 默认文本 -->
            <span v-else>
              {{ row[column.prop] }}
            </span>
          </template>
        </el-table-column>
      </template>
      
      <!-- 操作列 -->
      <el-table-column
        v-if="actions && actions.length > 0"
        label="操作"
        :width="actionWidth"
        fixed="right"
        align="center"
      >
        <template #default="{ row, $index }">
          <template v-for="action in actions" :key="action.key">
            <el-button
              v-if="!action.hidden || !action.hidden(row)"
              :type="action.type || 'primary'"
              :size="action.size || 'small'"
              :disabled="action.disabled && action.disabled(row)"
              :loading="action.loading && action.loading(row)"
              link
              @click="handleAction(action, row, $index)"
            >
              <el-icon v-if="action.icon">
                <component :is="action.icon" />
              </el-icon>
              {{ action.label }}
            </el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>
    
    <!-- 分页 -->
    <div v-if="pagination" class="table-pagination">
      <el-pagination
        :current-page="currentPage"
        :page-size="pageSize"
        :total="total"
        :page-sizes="pageSizes"
        :layout="paginationLayout"
        :background="true"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { formatDateTime, formatNumber } from '../utils/formatters'

const props = defineProps({
  // 表格数据
  data: {
    type: Array,
    default: () => []
  },
  
  // 列配置
  columns: {
    type: Array,
    required: true
    // columns: [{ prop, label, width?, minWidth?, fixed?, align?, sortable?, type?, tagMap?, format?, slot?, showOverflowTooltip? }]
  },
  
  // 操作按钮配置
  actions: {
    type: Array,
    default: () => []
    // actions: [{ key, label, type?, size?, icon?, disabled?, hidden?, loading? }]
  },
  
  // 表格配置
  loading: {
    type: Boolean,
    default: false
  },
  
  height: {
    type: [String, Number],
    default: undefined
  },
  
  maxHeight: {
    type: [String, Number],
    default: undefined
  },
  
  selectable: {
    type: Boolean,
    default: false
  },
  
  showIndex: {
    type: Boolean,
    default: false
  },
  
  actionWidth: {
    type: [String, Number],
    default: 200
  },
  
  // 分页配置
  pagination: {
    type: Boolean,
    default: true
  },
  
  total: {
    type: Number,
    default: 0
  },
  
  currentPage: {
    type: Number,
    default: 1
  },
  
  pageSize: {
    type: Number,
    default: 20
  },
  
  pageSizes: {
    type: Array,
    default: () => [10, 20, 50, 100]
  },
  
  paginationLayout: {
    type: String,
    default: 'total, sizes, prev, pager, next, jumper'
  },
  
  // 其他表格属性
  tableProps: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits([
  'update:currentPage',
  'update:pageSize',
  'selection-change',
  'sort-change',
  'row-click',
  'action',
  'page-change',
  'size-change'
])

const tableRef = ref()

// 序号计算方法
const indexMethod = (index) => {
  return (props.currentPage - 1) * props.pageSize + index + 1
}

// 获取标签类型
const getTagType = (value, tagMap) => {
  if (!tagMap || !tagMap[value]) return 'info'
  return tagMap[value].type || 'info'
}

// 获取标签文本
const getTagText = (value, tagMap) => {
  if (!tagMap || !tagMap[value]) return value
  return tagMap[value].text || value
}

// 事件处理
const handleSelectionChange = (selection) => {
  emit('selection-change', selection)
}

const handleSortChange = (sortInfo) => {
  emit('sort-change', sortInfo)
}

const handleRowClick = (row, column, event) => {
  emit('row-click', row, column, event)
}

const handleAction = (action, row, index) => {
  emit('action', {
    action: action.key,
    row,
    index
  })
}

const handleCurrentChange = (page) => {
  emit('update:currentPage', page)
  emit('page-change', {
    page,
    pageSize: props.pageSize
  })
}

const handleSizeChange = (size) => {
  emit('update:pageSize', size)
  emit('size-change', {
    page: props.currentPage,
    pageSize: size
  })
}

// 暴露方法
const clearSelection = () => {
  tableRef.value?.clearSelection()
}

const toggleRowSelection = (row, selected) => {
  tableRef.value?.toggleRowSelection(row, selected)
}

const toggleAllSelection = () => {
  tableRef.value?.toggleAllSelection()
}

defineExpose({
  clearSelection,
  toggleRowSelection,
  toggleAllSelection,
  tableRef
})
</script>

<style scoped lang="scss">
.data-table {
  .table-pagination {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    padding: 16px 0;
    margin-top: 16px;
    border-top: 1px solid #f0f0f0;
  }
  
  :deep(.el-table) {
    .el-table__header-wrapper {
      .el-table__header {
        th {
          background-color: #fafafa;
          color: #606266;
          font-weight: 500;
        }
      }
    }
    
    .el-table__body-wrapper {
      .el-table__body {
        tr:hover {
          background-color: #f8f9fa;
        }
      }
    }
  }
  
  :deep(.el-button + .el-button) {
    margin-left: 8px;
  }
}
</style> 