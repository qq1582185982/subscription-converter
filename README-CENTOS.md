# 🐧 CentOS 部署指南

## 📋 系统要求

- **操作系统**: CentOS 7/8/9 或 RHEL 7/8/9
- **架构**: x86_64 (AMD64)
- **内存**: 最低 512MB
- **磁盘**: 最低 100MB
- **网络**: 开放 8856 端口

## 🚀 快速部署

### 方法一：一键部署脚本

1. **上传文件到服务器**
```bash
# 将以下文件上传到服务器同一目录：
# - subscription-converter-linux    (可执行文件)
# - deploy-centos.sh                (部署脚本)
```

2. **运行部署脚本**
```bash
# 赋予脚本执行权限
chmod +x deploy-centos.sh

# 以root用户运行部署脚本
sudo bash deploy-centos.sh
```

3. **完成！**
   - 脚本会自动配置系统服务
   - 开放防火墙端口
   - 启动订阅转换服务

### 方法二：手动部署

1. **创建服务目录**
```bash
sudo mkdir -p /opt/subscription-converter
cd /opt/subscription-converter
```

2. **上传可执行文件**
```bash
# 将 subscription-converter-linux 上传到此目录
sudo chmod +x subscription-converter-linux
```

3. **创建服务用户**
```bash
sudo useradd -r -s /bin/false subscription
sudo chown -R subscription:subscription /opt/subscription-converter
```

4. **配置防火墙**
```bash
# CentOS 7/8/9 使用 firewalld
sudo firewall-cmd --permanent --add-port=8856/tcp
sudo firewall-cmd --reload

# 或者如果使用 iptables
sudo iptables -A INPUT -p tcp --dport 8856 -j ACCEPT
sudo service iptables save
```

5. **创建系统服务**
```bash
sudo tee /etc/systemd/system/subscription-converter.service > /dev/null << 'EOF'
[Unit]
Description=Subscription Converter Server
After=network.target

[Service]
Type=simple
User=subscription
Group=subscription
WorkingDirectory=/opt/subscription-converter
ExecStart=/opt/subscription-converter/subscription-converter-linux
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

# 安全设置
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/subscription-converter

[Install]
WantedBy=multi-user.target
EOF
```

6. **启动服务**
```bash
sudo systemctl daemon-reload
sudo systemctl enable subscription-converter
sudo systemctl start subscription-converter
```

## 🔧 服务管理

### 常用命令
```bash
# 查看服务状态
sudo systemctl status subscription-converter

# 查看实时日志
sudo journalctl -fu subscription-converter

# 重启服务
sudo systemctl restart subscription-converter

# 停止服务
sudo systemctl stop subscription-converter

# 重新启动服务
sudo systemctl start subscription-converter

# 禁用开机自启
sudo systemctl disable subscription-converter
```

### 配置文件位置
- **可执行文件**: `/opt/subscription-converter/subscription-converter-linux`
- **管理员配置**: `/opt/subscription-converter/admin_config.json`
- **订阅文件**: `/opt/subscription-converter/subscription_*.txt`
- **系统服务**: `/etc/systemd/system/subscription-converter.service`

## 🌐 访问服务

### 本地访问
```bash
curl http://localhost:8856
```

### 远程访问
```bash
# 替换 YOUR_SERVER_IP 为实际服务器IP
http://YOUR_SERVER_IP:8856
```

### 获取服务器IP
```bash
# 内网IP
hostname -I

# 外网IP (如果有)
curl -4 ip.sb
```

## 🔐 首次设置

1. **访问服务器地址**
   - 浏览器打开: `http://YOUR_SERVER_IP:8856`

2. **设置管理员账号**
   - 首次访问会自动跳转到设置页面
   - 输入管理员用户名和密码
   - 完成设置后即可正常使用

3. **功能说明**
   - **主页**: 生成订阅链接
   - **管理后台**: 查看所有订阅记录
   - **智能去重**: 相同配置复用链接

## 🛠️ 故障排除

### 服务无法启动
```bash
# 查看错误日志
sudo journalctl -fu subscription-converter

# 检查文件权限
ls -la /opt/subscription-converter/

# 检查端口占用
sudo netstat -tlpn | grep 8856
```

### 无法访问服务
```bash
# 检查服务状态
sudo systemctl status subscription-converter

# 检查防火墙
sudo firewall-cmd --list-ports

# 测试本地连接
curl -I http://localhost:8856
```

### 重新配置
```bash
# 删除管理员配置文件（重新设置账号）
sudo rm /opt/subscription-converter/admin_config.json

# 重启服务
sudo systemctl restart subscription-converter
```

## 📈 性能优化

### 系统调优
```bash
# 增加文件描述符限制
echo "subscription soft nofile 65536" >> /etc/security/limits.conf
echo "subscription hard nofile 65536" >> /etc/security/limits.conf

# 优化网络参数
echo "net.core.somaxconn = 65535" >> /etc/sysctl.conf
sysctl -p
```

### 日志管理
```bash
# 限制日志大小
sudo mkdir -p /etc/systemd/journald.conf.d/
echo -e "[Journal]\nSystemMaxUse=100M" | sudo tee /etc/systemd/journald.conf.d/subscription.conf
sudo systemctl restart systemd-journald
```

## 🔄 更新程序

```bash
# 停止服务
sudo systemctl stop subscription-converter

# 备份配置文件
sudo cp /opt/subscription-converter/admin_config.json /tmp/

# 替换可执行文件
sudo cp subscription-converter-linux /opt/subscription-converter/
sudo chmod +x /opt/subscription-converter/subscription-converter-linux
sudo chown subscription:subscription /opt/subscription-converter/subscription-converter-linux

# 恢复配置文件
sudo cp /tmp/admin_config.json /opt/subscription-converter/

# 重启服务
sudo systemctl start subscription-converter
```

## 💡 常见问题

**Q: 如何修改端口？**
A: 目前端口固定为8856，如需修改需要重新编译程序。

**Q: 支持HTTPS吗？**
A: 可以通过Nginx反向代理实现HTTPS。

**Q: 如何备份数据？**
A: 备份整个 `/opt/subscription-converter` 目录即可。

**Q: 服务器重启后还能正常工作吗？**
A: 是的，服务已设置为开机自启动。

---

## 📞 技术支持

如果在部署过程中遇到问题，请提供：
- CentOS版本信息
- 错误日志内容
- 详细的操作步骤

祝您使用愉快！ 🎉 