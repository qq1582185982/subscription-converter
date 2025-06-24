#!/bin/bash

echo "🏗️  订阅转换服务器 - Go版本构建脚本"
echo "==============================================="

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Go环境未安装，请先安装Go语言环境"
    echo "💡 下载地址: https://golang.org/dl/"
    exit 1
fi

echo "✅ Go环境检查通过"

# 初始化模块和下载依赖
echo "📦 下载依赖包..."
go mod tidy

# 构建当前平台版本
echo "🔨 构建当前平台版本..."
go build -ldflags "-s -w" -o subscription-converter .
if [ $? -ne 0 ]; then
    echo "❌ 构建失败"
    exit 1
fi

# 构建Windows版本
echo "🔨 构建Windows版本..."
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o subscription-converter.exe .
if [ $? -ne 0 ]; then
    echo "❌ Windows版本构建失败"
    exit 1
fi

# 构建Linux版本
echo "🔨 构建Linux版本..."
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o subscription-converter-linux .
if [ $? -ne 0 ]; then
    echo "❌ Linux版本构建失败"
    exit 1
fi

# 构建MacOS版本
echo "🔨 构建MacOS版本..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o subscription-converter-macos .
if [ $? -ne 0 ]; then
    echo "❌ MacOS版本构建失败"
    exit 1
fi

echo "✅ 构建完成！"
echo ""
echo "📁 生成的文件:"
echo "   - subscription-converter         (当前平台版本)"
echo "   - subscription-converter.exe     (Windows版本)"
echo "   - subscription-converter-linux   (Linux版本)"
echo "   - subscription-converter-macos   (MacOS版本)"
echo ""
echo "🚀 使用说明:"
echo "   1. ./subscription-converter 启动服务器"
echo "   2. 在浏览器中访问显示的地址"
echo "   3. 上传或输入Clash配置文件"
echo "   4. 生成订阅链接"
echo ""

# 设置执行权限
chmod +x subscription-converter
chmod +x subscription-converter-linux
chmod +x subscription-converter-macos

echo "✅ 已设置执行权限" 