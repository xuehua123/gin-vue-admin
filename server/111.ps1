# ==============================================================================
# Script: convert_utf8_to_crlf_final.ps1 (最终版)
#
# Description:
#   专门用于处理【本身已经是 UTF-8 编码】的文件。
#   本脚本会：
#   1. 以 UTF-8 格式读取文件。
#   2. 将行尾符从 LF (Unix/Linux) 转换为 CRLF (Windows)。
#   3. 以 UTF-8 格式写回文件，且【不添加BOM】，确保与Go等开发工具兼容。
#
# 前提:
#   运行此脚本前，请确保你的文件确实是 UTF-8 编码。
# ==============================================================================

# 定义要搜索的根目录路径，"." 代表当前脚本所在的目录
$targetDirectory = "."

# 定义要转换的文件类型
$fileFilter = "*.go"

# 初始化计数器
$processedFilesCount = 0

Write-Host "UTF-8 文件专用转换脚本已启动。" -ForegroundColor Green
Write-Host "目标: LF -> CRLF | 保持 UTF-8 无 BOM 编码" -ForegroundColor Cyan
Write-Host "--------------------------------------------------------"

try {
    # 递归查找所有 .go 文件
    $goFiles = Get-ChildItem -Path $targetDirectory -Filter $fileFilter -Recurse -File

    # 创建一个不带BOM的UTF8编码器对象，这是最可靠的无BOM写入方法
    $utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)

    # 遍历所有找到的文件
    foreach ($file in $goFiles) {
        $processedFilesCount++
        Write-Host "[$processedFilesCount] 正在处理: $($file.FullName)"

        # 步骤 1: 强制以 UTF-8 编码读取文件内容到单个字符串
        $content = Get-Content -Path $file.FullName -Raw -Encoding UTF8

        # 步骤 2: 使用正则表达式替换行尾符，此方法能正确处理已混合LF/CRLF的文件
        $newContent = $content -replace '(?<!\r)\n', "`r`n"

        # 步骤 3: 使用.NET类库将新内容以【无BOM的UTF-8】格式精确地写回文件
        # 这是比 Set-Content 更底层、更可靠的方法，能跨PowerShell版本保证行为一致
        [System.IO.File]::WriteAllText($file.FullName, $newContent, $utf8WithoutBom)
    }

    Write-Host "--------------------------------------------------------"
    Write-Host "转换完成！" -ForegroundColor Green
    Write-Host "总共处理了 $processedFilesCount 个 .go 文件。"

} catch {
    # 如果在过程中发生任何错误，打印错误信息
    Write-Error "在执行过程中发生错误: $_"
}