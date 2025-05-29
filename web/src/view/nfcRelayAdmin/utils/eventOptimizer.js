// 事件监听器优化工具
// 用于减少passive event listener警告

/**
 * 优化第三方库的事件监听器
 * 主要针对ECharts、Element Plus等组件的滚轮事件
 */
export const optimizeEventListeners = () => {
  // 标记已经优化过，避免重复执行
  if (window.__eventListenersOptimized) {
    return
  }

  // 重写addEventListener方法，为滚轮事件自动添加passive选项
  const originalAddEventListener = EventTarget.prototype.addEventListener
  
  EventTarget.prototype.addEventListener = function(type, listener, options) {
    // 对于滚轮相关事件，默认使用passive: true
    if (type === 'wheel' || type === 'mousewheel' || type === 'touchmove') {
      // 如果没有指定options或者options不是对象，转换为对象格式
      if (typeof options === 'boolean') {
        options = { capture: options, passive: true }
      } else if (!options || typeof options !== 'object') {
        options = { passive: true }
      } else if (options.passive === undefined) {
        // 只有在没有明确指定passive时才设为true
        options = { ...options, passive: true }
      }
    }
    
    return originalAddEventListener.call(this, type, listener, options)
  }

  // 标记已优化
  window.__eventListenersOptimized = true
  console.log('✅ Event listeners optimized for better performance')
}

/**
 * 在需要阻止默认行为的区域禁用passive
 * @param {HTMLElement} element 需要禁用passive的元素
 * @param {Array} events 事件类型数组
 */
export const disablePassiveForElement = (element, events = ['wheel', 'mousewheel']) => {
  if (!element) return

  events.forEach(eventType => {
    element.addEventListener(eventType, (e) => {
      // 这里可以根据需要决定是否阻止默认行为
      // e.preventDefault()
    }, { passive: false })
  })
}

/**
 * 为ECharts图表容器优化事件监听
 * @param {HTMLElement} chartContainer 图表容器元素
 */
export const optimizeChartEvents = (chartContainer) => {
  if (!chartContainer) return

  // 为图表容器添加CSS优化
  chartContainer.style.touchAction = 'pan-y'
  
  // 添加自定义的滚轮事件处理
  chartContainer.addEventListener('wheel', (e) => {
    // 允许图表内的缩放功能，但不阻塞页面滚动
    if (e.ctrlKey) {
      e.preventDefault() // 只在Ctrl+滚轮时阻止默认行为（用于缩放）
    }
  }, { passive: false })
}

/**
 * 控制台警告过滤器（可选）
 * 过滤掉已知的passive event listener警告
 */
export const filterConsoleWarnings = () => {
  const originalWarn = console.warn
  const originalError = console.error
  
  const warningPatterns = [
    /Added non-passive event listener/,
    /Consider marking event handler as 'passive'/,
    /scroll-blocking.*event/i
  ]
  
  console.warn = function(...args) {
    const message = args.join(' ')
    if (!warningPatterns.some(pattern => pattern.test(message))) {
      originalWarn.apply(console, args)
    }
  }
  
  console.error = function(...args) {
    const message = args.join(' ')
    if (!warningPatterns.some(pattern => pattern.test(message))) {
      originalError.apply(console, args)
    }
  }
}

/**
 * 强制优化模式（更激进的优化）
 * 如果基础优化不够，可以使用此方法
 */
export const enableAggressiveOptimization = () => {
  // 拦截所有可能导致警告的事件类型
  const problematicEvents = ['wheel', 'mousewheel', 'touchmove', 'touchstart', 'touchend']
  
  const originalAddEventListener = EventTarget.prototype.addEventListener
  
  EventTarget.prototype.addEventListener = function(type, listener, options) {
    if (problematicEvents.includes(type)) {
      // 强制所有这些事件都使用passive模式
      if (typeof options === 'boolean') {
        options = { capture: options, passive: true }
      } else {
        options = { ...(options || {}), passive: true }
      }
    }
    
    return originalAddEventListener.call(this, type, listener, options)
  }
  
  console.log('🚀 Aggressive event optimization enabled')
}

// 自动执行优化（如果在浏览器环境中）
if (typeof window !== 'undefined') {
  // 在DOM加载完成后执行优化
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', optimizeEventListeners, { passive: true })
  } else {
    optimizeEventListeners()
  }
  
  // 如果是开发环境，可以启用警告过滤
  if (process.env.NODE_ENV === 'development') {
    // 默认不过滤警告，让开发者看到优化效果
    // 如需过滤，取消注释下面这行
    // filterConsoleWarnings()
  }
} 