<!--
  趋势图表组件
  基于ECharts的数据趋势展示
-->
<template>
  <div class="trend-chart">
    <v-chart 
      ref="chartRef"
      :option="chartOption" 
      :style="{ height, width: '100%' }"
      autoresize
    />
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent
} from 'echarts/components'
import VChart from 'vue-echarts'

use([
  CanvasRenderer,
  LineChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent
])

const props = defineProps({
  data: {
    type: Array,
    default: () => []
    // data: [{ time: string, count: number }]
  },
  title: {
    type: String,
    default: ''
  },
  height: {
    type: String,
    default: '200px'
  },
  color: {
    type: String,
    default: '#409eff'
  },
  smooth: {
    type: Boolean,
    default: true
  },
  showSymbol: {
    type: Boolean,
    default: false
  }
})

const chartRef = ref()

const chartOption = computed(() => {
  const xAxisData = props.data.map(item => item.time)
  const seriesData = props.data.map(item => item.count)

  return {
    title: {
      text: props.title,
      left: 'center',
      textStyle: {
        fontSize: 14,
        fontWeight: 'normal',
        color: '#606266'
      }
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'line',
        lineStyle: {
          color: props.color,
          width: 1,
          type: 'solid'
        }
      },
      backgroundColor: 'rgba(50, 50, 50, 0.9)',
      borderColor: props.color,
      borderWidth: 1,
      textStyle: {
        color: '#fff',
        fontSize: 12
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: xAxisData,
      axisLine: {
        show: false
      },
      axisTick: {
        show: false
      },
      axisLabel: {
        color: '#909399',
        fontSize: 11
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        show: false
      },
      axisTick: {
        show: false
      },
      axisLabel: {
        color: '#909399',
        fontSize: 11
      },
      splitLine: {
        lineStyle: {
          color: '#f0f0f0',
          type: 'dashed'
        }
      }
    },
    series: [
      {
        data: seriesData,
        type: 'line',
        smooth: props.smooth,
        showSymbol: props.showSymbol,
        symbolSize: 4,
        lineStyle: {
          color: props.color,
          width: 2
        },
        itemStyle: {
          color: props.color,
          borderColor: '#fff',
          borderWidth: 2
        },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              {
                offset: 0,
                color: props.color + '40'
              },
              {
                offset: 1,
                color: props.color + '10'
              }
            ]
          }
        }
      }
    ]
  }
})

// 监听数据变化，刷新图表
watch(() => props.data, () => {
  if (chartRef.value) {
    chartRef.value.resize()
  }
}, { deep: true })

const initChart = () => {
  if (!chartRef.value) return
  
  chart = echarts.init(chartRef.value)
  
  // 配置ECharts使用被动事件监听器
  const option = {
    // ... existing code ...
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.data.map(item => item.time),
      axisLine: {
        lineStyle: {
          color: '#e4e7ed'
        }
      },
      axisLabel: {
        color: '#606266'
      }
    },
    yAxis: {
      type: 'value',
      axisLine: {
        lineStyle: {
          color: '#e4e7ed'
        }
      },
      axisLabel: {
        color: '#606266'
      },
      splitLine: {
        lineStyle: {
          color: '#f0f2f5'
        }
      }
    },
    series: [{
      data: props.data.map(item => item.count),
      type: 'line',
      smooth: props.smooth,
      symbol: 'circle',
      symbolSize: 6,
      lineStyle: {
        color: props.color,
        width: 3
      },
      itemStyle: {
        color: props.color
      },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0,
          y: 0,
          x2: 0,
          y2: 1,
          colorStops: [{
            offset: 0,
            color: props.color + '40'
          }, {
            offset: 1,
            color: props.color + '10'
          }]
        }
      }
    }]
  }
  
  chart.setOption(option)
  
  // 添加被动事件监听器配置
  const chartDom = chartRef.value
  if (chartDom) {
    // 移除默认的滚轮事件监听器
    chartDom.addEventListener('wheel', preventDefaultWheel, { passive: false })
    chartDom.addEventListener('mousewheel', preventDefaultWheel, { passive: false })
  }
  
  // 监听窗口大小变化
  window.addEventListener('resize', handleResize, { passive: true })
}

const preventDefaultWheel = (e) => {
  // 只在图表区域内才阻止默认行为
  if (e.target.closest('.trend-chart-container')) {
    e.preventDefault()
  }
}

const handleResize = () => {
  if (chart) {
    chart.resize()
  }
}

defineExpose({
  chartRef
})
</script>

<style scoped lang="scss">
.trend-chart {
  width: 100%;
}
</style> 