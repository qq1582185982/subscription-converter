#!/bin/bash

# CentOS 订阅转换服务器部署脚本
# 适用于 CentOS 7/8/9

echo "🚀 CentOS 订阅转换服务器部署脚本"
echo "===================================="

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then
    echo "❌ 请使用root用户运行此脚本"
    echo "   sudo bash deploy-centos.sh"
    exit 1
fi

# 获取系统信息
OS_VERSION=$(cat /etc/centos-release 2>/dev/null || echo "Unknown")
echo "📋 系统信息: $OS_VERSION"

# 创建服务用户
echo "👤 创建服务用户..."
useradd -r -s /bin/false subscription || echo "用户已存在"

# 获取当前脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "📂 脚本目录: $SCRIPT_DIR"

# 检查可执行文件
if [ ! -f "$SCRIPT_DIR/subscription-converter-linux" ]; then
    echo "❌ 找不到 subscription-converter-linux 文件"
    echo "   请确保已将编译好的文件上传到脚本同一目录"
    exit 1
fi

# 创建服务目录
SERVICE_DIR="/opt/subscription-converter"
echo "📁 创建服务目录: $SERVICE_DIR"
mkdir -p $SERVICE_DIR

# 复制文件到服务目录
echo "📋 复制文件到服务目录..."
cp "$SCRIPT_DIR/subscription-converter-linux" "$SERVICE_DIR/"

# 设置文件权限
echo "🔒 设置文件权限..."
chmod +x "$SERVICE_DIR/subscription-converter-linux"
chown -R subscription:subscription $SERVICE_DIR

# 打开防火墙端口
echo "🔥 配置防火墙..."
if command -v firewall-cmd >/dev/null 2>&1; then
    firewall-cmd --permanent --add-port=8856/tcp
    firewall-cmd --reload
    echo "✅ 防火墙端口 8856 已开放"
else
    echo "⚠️  未检测到 firewall-cmd，请手动开放端口 8856"
fi

# 创建systemd服务文件
echo "⚙️  创建系统服务..."
cat > /etc/systemd/system/subscription-converter.service << EOF
[Unit]
Description=Subscription Converter Server
After=network.target

[Service]
Type=simple
User=subscription
Group=subscription
WorkingDirectory=$SERVICE_DIR
ExecStart=$SERVICE_DIR/subscription-converter-linux
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

# 安全设置
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=$SERVICE_DIR

[Install]
WantedBy=multi-user.target
EOF

# 重新加载systemd配置
systemctl daemon-reload

# 启用并启动服务
echo "🎯 启动服务..."
systemctl enable subscription-converter
systemctl start subscription-converter

# 检查服务状态
if systemctl is-active --quiet subscription-converter; then
    echo ""
    echo "✅ 部署成功！"
    echo "===================================="
    echo "📡 服务状态: $(systemctl is-active subscription-converter)"
    echo "🌐 访问地址: http://$(hostname -I | awk '{print $1}'):8856"
echo "🌐 本地访问: http://localhost:8856"
    echo ""
    echo "📋 常用命令:"
    echo "   查看状态: systemctl status subscription-converter"
    echo "   查看日志: journalctl -fu subscription-converter"
    echo "   重启服务: systemctl restart subscription-converter"
    echo "   停止服务: systemctl stop subscription-converter"
    echo ""
    echo "⚠️  首次访问会要求设置管理员账号密码"
else
    echo ""
    echo "❌ 服务启动失败"
    echo "查看错误日志: journalctl -fu subscription-converter"
    exit 1
fi 