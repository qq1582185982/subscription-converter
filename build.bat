@echo off
echo ??  订阅转换服务器 - Go版本构建脚本
echo ===============================================

REM 检查Go是否安装
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ? Go环境未安装，请先安装Go语言环境
    echo ? 下载地址: https://golang.org/dl/
    pause
    exit /b 1
)

echo ? Go环境检查通过

REM 初始化模块和下载依赖
echo ? 下载依赖包...
go mod tidy

REM 构建Windows版本
echo ? 构建Windows版本...
go build -ldflags "-s -w" -o subscription-converter.exe .
if %errorlevel% neq 0 (
    echo ? Windows版本构建失败
    pause
    exit /b 1
)

REM 构建Linux版本
echo ? 构建Linux版本...
set GOOS=linux
set GOARCH=amd64
go build -ldflags "-s -w" -o subscription-converter-linux .
if %errorlevel% neq 0 (
    echo ? Linux版本构建失败
    pause
    exit /b 1
)

REM 构建MacOS版本
echo ? 构建MacOS版本...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags "-s -w" -o subscription-converter-macos .
if %errorlevel% neq 0 (
    echo ? MacOS版本构建失败
    pause
    exit /b 1
)

echo ? 构建完成！
echo.
echo ? 生成的文件:
echo    - subscription-converter.exe     (Windows版本)
echo    - subscription-converter-linux   (Linux版本)
echo    - subscription-converter-macos   (MacOS版本)
echo.
echo ? 使用说明:
echo    1. 双击 subscription-converter.exe 启动服务器
echo    2. 在浏览器中访问显示的地址
echo    3. 上传或输入Clash配置文件
echo    4. 生成订阅链接
echo.
pause 