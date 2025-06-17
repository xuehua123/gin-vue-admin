#!/usr/bin/env node

/**
 * 循环依赖检测脚本
 * 使用 madge 工具检测项目中的循环依赖
 */

const { execSync } = require('child_process');
const path = require('path');

const projectRoot = path.resolve(__dirname, '..');

console.log('🔍 开始检测循环依赖...\n');

try {
  // 使用npm script检测循环依赖
  execSync('npm run check-deps', { 
    cwd: projectRoot,
    stdio: 'inherit' // 直接输出到控制台
  });
  
  console.log('\n✅ 没有发现循环依赖问题！');
  
} catch (error) {
  console.log('\n❌ 发现循环依赖问题！');
  console.log('请检查上面的输出信息。');
  process.exit(1);
}

console.log('\n🔍 生成依赖分析报告...');

try {
  // 生成依赖统计
  const stats = execSync(
    `npx madge --extensions js,vue,ts --json src/`,
    { 
      encoding: 'utf8',
      cwd: projectRoot 
    }
  );

  const dependencies = JSON.parse(stats);
  const totalFiles = Object.keys(dependencies).length;
  
  // 计算依赖统计
  let totalDeps = 0;
  let maxDeps = 0;
  let maxDepsFile = '';
  
  for (const [file, deps] of Object.entries(dependencies)) {
    totalDeps += deps.length;
    if (deps.length > maxDeps) {
      maxDeps = deps.length;
      maxDepsFile = file;
    }
  }
  
  console.log(`📊 依赖统计：`);
  console.log(`   总文件数: ${totalFiles}`);
  console.log(`   总依赖数: ${totalDeps}`);
  console.log(`   平均每文件依赖数: ${(totalDeps / totalFiles).toFixed(2)}`);
  console.log(`   最多依赖的文件: ${maxDepsFile} (${maxDeps} 个依赖)`);
  
} catch (error) {
  console.warn('⚠️  无法生成依赖统计:', error.message);
}

console.log('\n✨ 循环依赖检测完成！'); 