import legacyPlugin from '@vitejs/plugin-legacy'
import { viteLogo } from './src/core/config'
import Banner from 'vite-plugin-banner'
import * as path from 'path'
import * as dotenv from 'dotenv'
import * as fs from 'fs'
import vuePlugin from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import VueFilePathPlugin from './vitePlugin/componentName/index.js'
import { svgBuilder } from 'vite-auto-import-svg'
import { AddSecret } from './vitePlugin/secret'
// @see https://cn.vitejs.dev/config/
export default ({ mode }) => {
  AddSecret('')
  const NODE_ENV = mode || 'development'
  const envFiles = [`.env.${NODE_ENV}`]
  for (const file of envFiles) {
    const envConfig = dotenv.parse(fs.readFileSync(file))
    for (const k in envConfig) {
      process.env[k] = envConfig[k]
    }
  }

  viteLogo(process.env)

  const timestamp = Date.parse(new Date())

  const optimizeDeps = {}

  const alias = {
    '@': path.resolve(__dirname, './src'),
    vue$: 'vue/dist/vue.runtime.esm-bundler.js'
  }

  const esbuild = {}

  const rollupOptions = {
    output: {
      entryFileNames: 'assets/087AC4D233B64EB0[name].[hash].js',
      chunkFileNames: 'assets/087AC4D233B64EB0[name].[hash].js',
      assetFileNames: 'assets/087AC4D233B64EB0[name].[hash].[ext]'
    }
  }

  const base = "/"
  const root = "./"
  const outDir = "dist"

  // 生产环境优化配置
  const isProduction = NODE_ENV === 'production'

  const config = {
    base: base, // 编译后js导入的资源路径
    root: root, // index.html文件所在位置
    publicDir: 'public', // 静态资源文件夹
    resolve: {
      alias
    },
    define: {
      'process.env': {}
    },
    css: {
      preprocessorOptions: {
        scss: {
          api: 'modern-compiler' // or "modern"
        }
      }
    },
    server: {
      // 如果使用docker-compose开发模式，设置为false
      open: true,
      port: process.env.VITE_CLI_PORT,
      proxy: {
        // 把key的路径代理到target位置
        // detail: https://cli.vuejs.org/config/#devserver-proxy
        [process.env.VITE_BASE_API]: {
          // 需要代理的路径   例如 '/api'
          target: `${process.env.VITE_BASE_PATH}:${process.env.VITE_SERVER_PORT}/`, // 代理到 目标路径
          changeOrigin: true,
          ws: true,
          rewrite: (path) =>
            path.replace(new RegExp('^' + process.env.VITE_BASE_API), '')
        }
      }
    },
    build: {
      minify: isProduction ? 'esbuild' : false, // esbuild更快，terser更慢但压缩更好
      manifest: false, // 是否产出manifest.json
      sourcemap: false, // 生产环境关闭sourcemap加速构建
      outDir: outDir, // 产出目录
      // 简化terser配置（如果使用terser）
      terserOptions: isProduction ? {
        compress: {
          drop_console: true,
          drop_debugger: true
        }
      } : {},
      rollupOptions,
      // 优化构建性能
      target: 'es2018', // 现代浏览器目标，减少编译工作
      cssCodeSplit: true, // CSS代码分割
      chunkSizeWarningLimit: 1000, // 提高chunk大小警告阈值
    },
    esbuild: isProduction ? {
      // 生产环境使用esbuild优化
      drop: ['console', 'debugger'],
    } : {},
    optimizeDeps,
    plugins: [
      // 开发环境才启用开发工具
      !isProduction && process.env.VITE_POSITION === 'open' &&
        vueDevTools({ launchEditor: process.env.VITE_EDITOR }),
      // 生产环境移除legacy插件以提高构建速度
      // legacyPlugin({
      //   targets: [
      //     'Android > 39',
      //     'Chrome >= 60',
      //     'Safari >= 10.1',
      //     'iOS >= 10.3',
      //     'Firefox >= 54',
      //     'Edge >= 15'
      //   ]
      // }),
      vuePlugin(),
      svgBuilder(['./src/plugin/','./src/assets/icons/'],base, outDir,'assets', NODE_ENV),
      [Banner(`\n Build based on gin-vue-admin \n Time : ${timestamp}`)],
      VueFilePathPlugin('./src/pathInfo.json')
    ].filter(Boolean) // 过滤掉false值
  }
  return config
}
