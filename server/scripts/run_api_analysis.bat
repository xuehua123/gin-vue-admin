@echo off
chcp 65001 >nul
echo ========================================
echo API和WebSocket接口分析工具
echo ========================================
echo.

echo 选择分析类型:
echo [1] 快速摘要 (推荐)
echo [2] 详细分析
echo [3] 生成所有报告
echo.

set /p choice="请输入选择 (1-3): "

if "%choice%"=="1" (
    echo.
    echo 正在生成快速摘要...
    python scripts/quick_api_summary.py
    goto end
)

if "%choice%"=="2" (
    echo.
    echo 正在进行详细分析...
    python scripts/analyze_apis_enhanced.py
    goto end
)

if "%choice%"=="3" (
    echo.
    echo 正在生成所有报告...
    echo.
    echo [1/2] 生成快速摘要...
    python scripts/quick_api_summary.py
    echo.
    echo [2/2] 生成详细分析...
    python scripts/analyze_apis_enhanced.py
    goto end
)

echo 无效选择，请重新运行脚本
goto end

:end
echo.
echo 分析完成！
echo.
echo 生成的文件:
echo - API接口文档.md (Markdown格式文档)
echo - 详细API接口和WebSocket分析报告.txt (详细文本报告)
echo.
pause 