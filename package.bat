@echo off
chcp 65001 >nul

rem 获取当前日期和时间用于文件名
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set "datetime=%%I"
set "FILENAME=gva-source-%datetime:~0,8%-%datetime:~8,6%.tar.gz"

echo.
echo =======================================================
echo.
echo           Gin-Vue-Admin Source Code Packager (Windows)
echo.
echo =======================================================
echo.

echo [INFO] Preparing to package project into %FILENAME%
echo.

rem 检查 tar 命令是否存在
tar --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] 'tar' command not found.
    echo Please make sure you have Git for Windows installed and that its 'bin' directory is in your system's PATH.
    pause
    exit /b 1
)

echo [INFO] Creating archive, this may take a moment...

rem 使用 tar 命令进行打包
tar ^
  --exclude-vcs ^
  --exclude-from=".dockerignore" ^
  --exclude="*.tar.gz" ^
  --exclude="package.sh" ^
  --exclude="package.bat" ^
  -czvf "%FILENAME%" .

echo.
echo [SUCCESS] Project packaged successfully!
echo [FILE]    %FILENAME%
echo.
echo You can now upload this file to your server and use it in 1panel.
echo.
pause 