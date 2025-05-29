<!--
  全屏监控大屏布局组件
  支持全屏切换、深色主题、自动隐藏导航
-->
<template>
  <div 
    class="fullscreen-layout" 
    :class="{ 
      'layout-fullscreen': isFullscreen,
      'layout-dark': isDarkMode 
    }"
  >
    <!-- 顶部控制栏 -->
    <div class="control-bar" :class="{ 'bar-hidden': isFullscreen && autoHideControls }">
      <div class="control-left">
        <div class="title-section">
          <h1 class="main-title">{{ title }}</h1>
          <p class="subtitle" v-if="subtitle">{{ subtitle }}</p>
        </div>
        
        <!-- 连接状态指示 -->
        <div class="connection-status" :class="connectionStatusClass">
          <el-icon class="status-icon">
            <component :is="getStatusIcon()" />
          </el-icon>
          <span class="status-text">{{ connectionStatusText }}</span>
          <div class="status-indicator" :class="connectionStatus"></div>
        </div>
      </div>
      
      <div class="control-right">
        <!-- 最后更新时间 -->
        <div class="update-info" v-if="lastUpdateTime">
          <el-icon class="update-icon">
            <Clock />
          </el-icon>
          <span class="update-text">{{ updateTimeText }}</span>
        </div>
        
        <!-- 控制按钮 -->
        <div class="control-buttons">
          <!-- 主题切换 -->
          <el-tooltip :content="isDarkMode ? '切换到亮色主题' : '切换到深色主题'" placement="bottom">
            <el-button 
              circle 
              size="small"
              :type="isDarkMode ? 'warning' : 'primary'"
              @click="toggleDarkMode"
            >
              <el-icon>
                <component :is="isDarkMode ? Sunny : Moon" />
              </el-icon>
            </el-button>
          </el-tooltip>
          
          <!-- 自动隐藏切换 -->
          <el-tooltip :content="autoHideControls ? '显示控制栏' : '自动隐藏控制栏'" placement="bottom">
            <el-button 
              circle 
              size="small"
              :type="autoHideControls ? 'success' : 'info'"
              @click="toggleAutoHide"
            >
              <el-icon>
                <component :is="autoHideControls ? View : Hide" />
              </el-icon>
            </el-button>
          </el-tooltip>
          
          <!-- 全屏切换 -->
          <el-tooltip :content="isFullscreen ? '退出全屏' : '进入全屏'" placement="bottom">
            <el-button 
              circle 
              size="small"
              type="primary"
              @click="toggleFullscreen"
            >
              <el-icon>
                <component :is="isFullscreen ? Rank : FullScreen" />
              </el-icon>
            </el-button>
          </el-tooltip>
          
          <!-- 设置菜单 -->
          <el-dropdown @command="handleSettingsCommand">
            <el-button circle size="small">
              <el-icon>
                <Setting />
              </el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="refresh">
                  <el-icon><Refresh /></el-icon>
                  刷新数据
                </el-dropdown-item>
                <el-dropdown-item command="export">
                  <el-icon><Download /></el-icon>
                  导出数据
                </el-dropdown-item>
                <el-dropdown-item command="screenshot" divided>
                  <el-icon><Camera /></el-icon>
                  截图保存
                </el-dropdown-item>
                <el-dropdown-item command="config">
                  <el-icon><Tools /></el-icon>
                  显示设置
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      
      <!-- 鼠标悬停显示控制栏 -->
      <div 
        class="hover-trigger" 
        v-if="isFullscreen && autoHideControls"
        @mouseenter="showControls"
        @mouseleave="hideControls"
      ></div>
    </div>
    
    <!-- 主要内容区域 -->
    <div class="content-area" :class="{ 'content-fullscreen': isFullscreen }">
      <slot></slot>
    </div>
    
    <!-- 底部状态栏 -->
    <div class="status-bar" :class="{ 'bar-hidden': isFullscreen && autoHideControls }">
      <div class="status-left">
        <span class="status-item">
          <el-icon><Monitor /></el-icon>
          {{ screenInfo }}
        </span>
        <span class="status-item">
          <el-icon><Timer /></el-icon>
          运行时间: {{ runtimeText }}
        </span>
      </div>
      
      <div class="status-right">
        <span class="status-item">
          <el-icon><DataAnalysis /></el-icon>
          刷新频率: {{ refreshRate }}s
        </span>
        <span class="status-item">
          <el-icon><Connection /></el-icon>
          {{ connectionStatusText }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { 
  FullScreen, 
  Rank, 
  Setting, 
  Moon, 
  Sunny,
  Clock,
  Monitor,
  Timer,
  DataAnalysis,
  Connection,
  Refresh,
  Download,
  Camera,
  Tools,
  View,
  Hide,
  CircleCheck,
  CircleClose,
  Loading
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { formatTime } from '@/utils/format'

const props = defineProps({
  title: {
    type: String,
    default: 'NFC中继监控大屏'
  },
  subtitle: {
    type: String,
    default: ''
  },
  connectionStatus: {
    type: String,
    default: 'connected'
    // 'connected', 'disconnected', 'connecting', 'error'
  },
  lastUpdateTime: {
    type: Date,
    default: null
  },
  refreshRate: {
    type: Number,
    default: 30
  }
})

const emit = defineEmits(['fullscreen-change', 'theme-change', 'refresh', 'export', 'screenshot', 'config'])

// 响应式状态
const isFullscreen = ref(false)
const isDarkMode = ref(false)
const autoHideControls = ref(false)
const controlsVisible = ref(true)
const startTime = ref(new Date())

// 计算属性
const connectionStatusClass = computed(() => `status-${props.connectionStatus}`)

const connectionStatusText = computed(() => {
  switch (props.connectionStatus) {
    case 'connected': return '已连接'
    case 'disconnected': return '已断开'
    case 'connecting': return '连接中'
    case 'error': return '连接错误'
    default: return '未知状态'
  }
})

const updateTimeText = computed(() => {
  if (!props.lastUpdateTime) return '暂无更新'
  
  const now = new Date()
  const diff = now - props.lastUpdateTime
  
  if (diff < 60000) {
    return '刚刚更新'
  } else if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    return `${minutes}分钟前更新`
  } else {
    return `${formatTime(props.lastUpdateTime)} 更新`
  }
})

const screenInfo = computed(() => {
  const { width, height } = window.screen
  return `${width} × ${height}`
})

const runtimeText = computed(() => {
  const diff = Date.now() - startTime.value.getTime()
  const hours = Math.floor(diff / 3600000)
  const minutes = Math.floor((diff % 3600000) / 60000)
  return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`
})

// 方法
const getStatusIcon = () => {
  switch (props.connectionStatus) {
    case 'connected': return CircleCheck
    case 'disconnected': return CircleClose
    case 'connecting': return Loading
    case 'error': return CircleClose
    default: return CircleClose
  }
}

const toggleFullscreen = async () => {
  try {
    if (!isFullscreen.value) {
      await document.documentElement.requestFullscreen()
    } else {
      await document.exitFullscreen()
    }
  } catch (error) {
    ElMessage.error('全屏切换失败: ' + error.message)
  }
}

const toggleDarkMode = () => {
  isDarkMode.value = !isDarkMode.value
  
  // 切换body的dark类
  if (isDarkMode.value) {
    document.body.classList.add('dark')
  } else {
    document.body.classList.remove('dark')
  }
  
  emit('theme-change', isDarkMode.value)
}

const toggleAutoHide = () => {
  autoHideControls.value = !autoHideControls.value
  
  if (!autoHideControls.value) {
    showControls()
  }
}

const showControls = () => {
  controlsVisible.value = true
}

const hideControls = () => {
  if (autoHideControls.value && isFullscreen.value) {
    controlsVisible.value = false
  }
}

const handleSettingsCommand = (command) => {
  switch (command) {
    case 'refresh':
      emit('refresh')
      break
    case 'export':
      emit('export')
      break
    case 'screenshot':
      emit('screenshot')
      break
    case 'config':
      emit('config')
      break
  }
}

const handleFullscreenChange = () => {
  isFullscreen.value = !!document.fullscreenElement
  emit('fullscreen-change', isFullscreen.value)
}

// 生命周期
onMounted(() => {
  document.addEventListener('fullscreenchange', handleFullscreenChange)
  
  // 检查系统主题偏好
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  if (prefersDark) {
    toggleDarkMode()
  }
})

onUnmounted(() => {
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
})
</script>

<style scoped lang="scss">
.fullscreen-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
  transition: all 0.3s ease;
  
  &.layout-fullscreen {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 9999;
  }
  
  &.layout-dark {
    background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
    color: #ffffff;
  }
  
  .control-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 20px;
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    border-bottom: 1px solid rgba(0, 0, 0, 0.1);
    transition: all 0.3s ease;
    position: relative;
    z-index: 1000;
    
    &.bar-hidden {
      transform: translateY(-100%);
      opacity: 0;
      pointer-events: none;
    }
    
    .control-left {
      display: flex;
      align-items: center;
      gap: 20px;
      
      .title-section {
        .main-title {
          margin: 0;
          font-size: 20px;
          font-weight: 700;
          color: #303133;
          line-height: 1.2;
        }
        
        .subtitle {
          margin: 2px 0 0;
          font-size: 12px;
          color: #909399;
          line-height: 1.2;
        }
      }
      
      .connection-status {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        border-radius: 16px;
        font-size: 12px;
        font-weight: 500;
        
        &.status-connected {
          background: rgba(103, 194, 58, 0.1);
          color: #67c23a;
        }
        
        &.status-disconnected {
          background: rgba(192, 196, 204, 0.1);
          color: #c0c4cc;
        }
        
        &.status-connecting {
          background: rgba(230, 162, 60, 0.1);
          color: #e6a23c;
        }
        
        &.status-error {
          background: rgba(245, 108, 108, 0.1);
          color: #f56c6c;
        }
        
        .status-icon {
          font-size: 12px;
        }
        
        .status-indicator {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          animation: statusPulse 2s infinite;
          
          &.connected {
            background-color: #67c23a;
          }
          
          &.disconnected {
            background-color: #c0c4cc;
          }
          
          &.connecting {
            background-color: #e6a23c;
          }
          
          &.error {
            background-color: #f56c6c;
          }
        }
      }
    }
    
    .control-right {
      display: flex;
      align-items: center;
      gap: 16px;
      
      .update-info {
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: 12px;
        color: #606266;
        
        .update-icon {
          font-size: 12px;
        }
      }
      
      .control-buttons {
        display: flex;
        align-items: center;
        gap: 8px;
      }
    }
    
    .hover-trigger {
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 20px;
      z-index: 1001;
    }
  }
  
  .content-area {
    flex: 1;
    padding: 20px;
    overflow: auto;
    
    &.content-fullscreen {
      padding: 10px;
    }
  }
  
  .status-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 20px;
    background: rgba(255, 255, 255, 0.9);
    backdrop-filter: blur(10px);
    border-top: 1px solid rgba(0, 0, 0, 0.1);
    font-size: 11px;
    color: #909399;
    transition: all 0.3s ease;
    
    &.bar-hidden {
      transform: translateY(100%);
      opacity: 0;
      pointer-events: none;
    }
    
    .status-left,
    .status-right {
      display: flex;
      align-items: center;
      gap: 16px;
    }
    
    .status-item {
      display: flex;
      align-items: center;
      gap: 4px;
      
      .el-icon {
        font-size: 11px;
      }
    }
  }
}

// 深色主题样式
.layout-dark {
  .control-bar {
    background: rgba(30, 30, 30, 0.95);
    border-bottom-color: rgba(255, 255, 255, 0.1);
    
    .title-section .main-title {
      color: #ffffff;
    }
    
    .title-section .subtitle {
      color: #cccccc;
    }
    
    .update-info {
      color: #cccccc;
    }
  }
  
  .status-bar {
    background: rgba(30, 30, 30, 0.9);
    border-top-color: rgba(255, 255, 255, 0.1);
    color: #cccccc;
  }
}

@keyframes statusPulse {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0.3; }
}

// 全屏模式下的特殊样式
:global(.fullscreen-layout.layout-fullscreen) {
  .control-bar.bar-hidden:hover,
  .status-bar.bar-hidden:hover {
    transform: translateY(0);
    opacity: 1;
    pointer-events: auto;
  }
}
</style> 