@echo off
setlocal enabledelayedexpansion

echo 🚀 订阅转换工具 - 一键安装构建脚本
echo =============================================

REM 检查Go是否已安装
go version >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Go环境已安装
    goto BUILD
)

echo ❌ Go环境未安装，正在自动安装...

REM 检查是否为64位系统
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set GOARCH=amd64
) else (
    set GOARCH=386
)

REM 设置Go版本和下载URL
set GO_VERSION=1.21.5
set GO_URL=https://golang.org/dl/go%GO_VERSION%.windows-%GOARCH%.msi

echo 📦 下载Go安装程序...
echo 下载地址: %GO_URL%

REM 创建临时目录
if not exist temp mkdir temp

REM 下载Go安装程序
echo 正在下载，请稍候...
curl -L -o temp\go-installer.msi %GO_URL%
if %errorlevel% neq 0 (
    echo ❌ 下载失败，请检查网络连接
    echo 💡 手动下载地址: https://golang.org/dl/
    pause
    exit /b 1
)

echo ✅ 下载完成

REM 安装Go
echo 📦 正在安装Go...
echo 请在弹出的安装程序中按照提示安装Go
msiexec /i temp\go-installer.msi
if %errorlevel% neq 0 (
    echo ❌ 安装失败
    pause
    exit /b 1
)

REM 刷新环境变量
echo 🔄 刷新环境变量...
call refreshenv.cmd 2>nul || (
    echo 请重新打开命令提示符窗口，然后再次运行此脚本
    pause
    exit /b 1
)

:BUILD
echo 🔨 开始构建项目...

REM 初始化Go模块
go mod tidy
if %errorlevel% neq 0 (
    echo ❌ 下载依赖失败
    pause
    exit /b 1
)

REM 构建Windows版本
echo 📦 构建Windows版本...
go build -ldflags "-s -w" -o subscription-converter.exe .
if %errorlevel% neq 0 (
    echo ❌ 构建失败
    pause
    exit /b 1
)

echo ✅ 构建成功！

REM 清理临时文件
if exist temp rmdir /s /q temp

echo.
echo 🎉 安装和构建完成！
echo 📁 生成的文件: subscription-converter.exe
echo.
echo 🚀 使用方法:
echo    1. 双击 subscription-converter.exe 启动服务器
echo    2. 在浏览器中访问: http://localhost:8080
echo    3. 输入Clash配置文件并生成订阅链接
echo.

REM 询问是否立即启动
set /p START_NOW="是否立即启动服务器？(y/n): "
if /i "%START_NOW%"=="y" (
    echo 🚀 启动服务器...
    subscription-converter.exe
)

pause 