<!--
  实时数据卡片组件
  支持数据变化动画和详情查看
-->
<template>
  <div 
    class="realtime-stat-card" 
    :class="{ 
      'card-clickable': clickable,
      'card-pulse': isAnimating,
      'card-increase': animationType === 'increase',
      'card-decrease': animationType === 'decrease'
    }"
    @click="handleClick"
  >
    <!-- 连接状态指示器 -->
    <div class="connection-indicator" :class="connectionStatusClass">
      <div class="status-dot"></div>
    </div>
    
    <!-- 卡片内容 -->
    <div class="card-content">
      <!-- 标题和图标 -->
      <div class="card-header">
        <div class="icon-wrapper" :style="{ color: iconColor }">
          <el-icon :size="28">
            <component :is="icon" />
          </el-icon>
        </div>
        <div class="title-section">
          <h3 class="card-title">{{ title }}</h3>
          <p class="card-subtitle" v-if="subtitle">{{ subtitle }}</p>
        </div>
      </div>
      
      <!-- 主要数值 -->
      <div class="value-section">
        <div class="main-value" :class="{ 'value-animating': isAnimating }">
          <span class="value-number">{{ formattedValue }}</span>
          <span class="value-unit" v-if="unit">{{ unit }}</span>
        </div>
        
        <!-- 变化提示 -->
        <div class="change-indicator" v-if="changeInfo">
          <el-icon class="change-icon" :class="changeInfo.type">
            <component :is="changeInfo.type === 'increase' ? TrendCharts : TrendCharts" />
          </el-icon>
          <span class="change-text">
            {{ changeInfo.type === 'increase' ? '+' : '-' }}{{ changeInfo.diff }}
          </span>
        </div>
      </div>
      
      <!-- 趋势信息 -->
      <div class="trend-section" v-if="trend">
        <el-icon class="trend-icon" :class="trend.type">
          <component :is="getTrendIcon(trend.type)" />
        </el-icon>
        <span class="trend-text">{{ trend.value }} {{ trend.label }}</span>
      </div>
      
      <!-- 额外信息 -->
      <div class="extra-info" v-if="extraInfo">
        <div class="info-item" v-for="info in extraInfo" :key="info.label">
          <span class="info-label">{{ info.label }}:</span>
          <span class="info-value" :style="{ color: info.color }">{{ info.value }}</span>
        </div>
      </div>
      
      <!-- 最后更新时间 -->
      <div class="update-time" v-if="showUpdateTime">
        <el-icon class="time-icon">
          <Clock />
        </el-icon>
        <span class="time-text">{{ updateTimeText }}</span>
      </div>
    </div>
    
    <!-- 点击波纹效果 -->
    <div class="ripple-effect" ref="rippleRef" v-if="clickable"></div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { 
  TrendCharts,
  ArrowUp, 
  ArrowDown, 
  Minus,
  Clock
} from '@element-plus/icons-vue'
import { formatTime } from '@/utils/format'

const props = defineProps({
  title: {
    type: String,
    required: true
  },
  value: {
    type: [Number, String],
    required: true
  },
  unit: {
    type: String,
    default: ''
  },
  subtitle: {
    type: String,
    default: ''
  },
  icon: {
    type: [String, Object],
    required: true
  },
  iconColor: {
    type: String,
    default: '#409eff'
  },
  trend: {
    type: Object,
    default: null
    // { type: 'up'|'down'|'neutral', value: string, label: string }
  },
  changeInfo: {
    type: Object,
    default: null
    // { type: 'increase'|'decrease', diff: number }
  },
  extraInfo: {
    type: Array,
    default: () => []
    // [{ label: string, value: string, color?: string }]
  },
  clickable: {
    type: Boolean,
    default: false
  },
  showUpdateTime: {
    type: Boolean,
    default: true
  },
  lastUpdateTime: {
    type: Date,
    default: null
  },
  connectionStatus: {
    type: String,
    default: 'connected'
    // 'connected', 'disconnected', 'connecting', 'error'
  }
})

const emit = defineEmits(['click'])

const rippleRef = ref()
const isAnimating = ref(false)
const animationType = ref('')

// 格式化数值
const formattedValue = computed(() => {
  if (typeof props.value === 'number') {
    if (props.value >= 1000000) {
      return (props.value / 1000000).toFixed(1) + 'M'
    } else if (props.value >= 1000) {
      return (props.value / 1000).toFixed(1) + 'K'
    }
    return props.value.toLocaleString()
  }
  return props.value
})

// 连接状态样式
const connectionStatusClass = computed(() => {
  return `status-${props.connectionStatus}`
})

// 更新时间文本
const updateTimeText = computed(() => {
  if (!props.lastUpdateTime) return '未更新'
  
  const now = new Date()
  const diff = now - props.lastUpdateTime
  
  if (diff < 60000) { // 小于1分钟
    return '刚刚更新'
  } else if (diff < 3600000) { // 小于1小时
    const minutes = Math.floor(diff / 60000)
    return `${minutes}分钟前`
  } else {
    return formatTime(props.lastUpdateTime)
  }
})

// 获取趋势图标
const getTrendIcon = (type) => {
  switch (type) {
    case 'up': return ArrowUp
    case 'down': return ArrowDown
    default: return Minus
  }
}

// 监听变化信息，触发动画
watch(() => props.changeInfo, (newChange) => {
  if (newChange) {
    triggerAnimation(newChange.type)
  }
}, { deep: true })

// 触发动画效果
const triggerAnimation = (type) => {
  isAnimating.value = true
  animationType.value = type
  
  setTimeout(() => {
    isAnimating.value = false
    animationType.value = ''
  }, 1000)
}

// 处理点击事件
const handleClick = (event) => {
  if (!props.clickable) return
  
  // 创建波纹效果
  createRippleEffect(event)
  
  emit('click')
}

// 创建波纹效果
const createRippleEffect = (event) => {
  if (!rippleRef.value) return
  
  const rect = event.currentTarget.getBoundingClientRect()
  const x = event.clientX - rect.left
  const y = event.clientY - rect.top
  
  const ripple = document.createElement('span')
  ripple.className = 'ripple'
  ripple.style.left = x + 'px'
  ripple.style.top = y + 'px'
  
  rippleRef.value.appendChild(ripple)
  
  setTimeout(() => {
    ripple.remove()
  }, 600)
}
</script>

<style scoped lang="scss">
.realtime-stat-card {
  position: relative;
  padding: 20px;
  background: linear-gradient(135deg, #ffffff 0%, #f8f9fa 100%);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  border: 1px solid #e4e7ed;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  
  &.card-clickable {
    cursor: pointer;
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 8px 30px rgba(0, 0, 0, 0.12);
      border-color: #409eff;
    }
  }
  
  &.card-pulse {
    animation: cardPulse 1s ease-in-out;
  }
  
  &.card-increase {
    border-color: #67c23a;
    background: linear-gradient(135deg, #f0f9ff 0%, #e7f9e7 100%);
  }
  
  &.card-decrease {
    border-color: #f56c6c;
    background: linear-gradient(135deg, #fff9f9 0%, #fef0f0 100%);
  }
  
  .connection-indicator {
    position: absolute;
    top: 12px;
    right: 12px;
    
    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      animation: statusBlink 2s infinite;
    }
    
    &.status-connected .status-dot {
      background-color: #67c23a;
    }
    
    &.status-disconnected .status-dot {
      background-color: #c0c4cc;
    }
    
    &.status-connecting .status-dot {
      background-color: #e6a23c;
    }
    
    &.status-error .status-dot {
      background-color: #f56c6c;
    }
  }
  
  .card-content {
    position: relative;
    z-index: 1;
  }
  
  .card-header {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px;
    
    .icon-wrapper {
      flex-shrink: 0;
      padding: 8px;
      background: rgba(64, 158, 255, 0.1);
      border-radius: 8px;
    }
    
    .title-section {
      flex: 1;
      
      .card-title {
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #303133;
        line-height: 1.2;
      }
      
      .card-subtitle {
        margin: 4px 0 0;
        font-size: 12px;
        color: #909399;
        line-height: 1.2;
      }
    }
  }
  
  .value-section {
    margin-bottom: 12px;
    display: flex;
    align-items: baseline;
    gap: 8px;
    
    .main-value {
      display: flex;
      align-items: baseline;
      gap: 4px;
      
      &.value-animating {
        animation: valueFlash 0.8s ease-in-out;
      }
      
      .value-number {
        font-size: 32px;
        font-weight: 700;
        color: #303133;
        line-height: 1;
        font-family: 'Roboto Mono', monospace;
      }
      
      .value-unit {
        font-size: 14px;
        color: #606266;
        font-weight: 500;
      }
    }
    
    .change-indicator {
      display: flex;
      align-items: center;
      gap: 2px;
      padding: 2px 6px;
      border-radius: 4px;
      font-size: 12px;
      font-weight: 500;
      
      &.increase {
        background-color: rgba(103, 194, 58, 0.1);
        color: #67c23a;
      }
      
      &.decrease {
        background-color: rgba(245, 108, 108, 0.1);
        color: #f56c6c;
      }
      
      .change-icon {
        font-size: 12px;
        
        &.increase {
          transform: rotate(0deg);
        }
        
        &.decrease {
          transform: rotate(180deg);
        }
      }
    }
  }
  
  .trend-section {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 8px;
    font-size: 12px;
    
    .trend-icon {
      font-size: 12px;
      
      &.up {
        color: #67c23a;
      }
      
      &.down {
        color: #f56c6c;
      }
      
      &.neutral {
        color: #909399;
      }
    }
    
    .trend-text {
      color: #606266;
    }
  }
  
  .extra-info {
    margin-bottom: 8px;
    
    .info-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 4px;
      font-size: 12px;
      
      &:last-child {
        margin-bottom: 0;
      }
      
      .info-label {
        color: #909399;
      }
      
      .info-value {
        font-weight: 500;
        color: #606266;
      }
    }
  }
  
  .update-time {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    color: #c0c4cc;
    
    .time-icon {
      font-size: 11px;
    }
  }
  
  .ripple-effect {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    pointer-events: none;
    
    :global(.ripple) {
      position: absolute;
      width: 40px;
      height: 40px;
      background: rgba(64, 158, 255, 0.3);
      border-radius: 50%;
      transform: translate(-50%, -50%) scale(0);
      animation: rippleEffect 0.6s ease-out;
    }
  }
}

@keyframes cardPulse {
  0% { transform: scale(1); }
  50% { transform: scale(1.02); }
  100% { transform: scale(1); }
}

@keyframes valueFlash {
  0%, 100% { color: #303133; }
  50% { color: #409eff; }
}

@keyframes statusBlink {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0.3; }
}

@keyframes rippleEffect {
  to {
    transform: translate(-50%, -50%) scale(4);
    opacity: 0;
  }
}

// 深色主题支持
.dark {
  .realtime-stat-card {
    background: linear-gradient(135deg, #1d1e1f 0%, #252526 100%);
    border-color: #414243;
    
    &.card-increase {
      background: linear-gradient(135deg, #1f2f1f 0%, #2a4a2a 100%);
    }
    
    &.card-decrease {
      background: linear-gradient(135deg, #2f1f1f 0%, #4a2a2a 100%);
    }
    
    .card-title {
      color: #ffffff;
    }
    
    .card-subtitle {
      color: #cccccc;
    }
    
    .value-number {
      color: #ffffff;
    }
  }
}
</style> 