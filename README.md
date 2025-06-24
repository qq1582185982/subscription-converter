# 订阅转换服务器 (Go版)

[![Build Status](https://github.com/qq1582185982/subscription-converter/workflows/Build%20Multi-Platform%20Releases/badge.svg)](https://github.com/qq1582185982/subscription-converter/actions)
[![Release](https://img.shields.io/github/v/release/qq1582185982/subscription-converter)](https://github.com/qq1582185982/subscription-converter/releases)
[![Docker](https://img.shields.io/badge/docker-ghcr.io-blue)](https://github.com/qq1582185982/subscription-converter/pkgs/container/subscription-converter)
[![Go Report Card](https://goreportcard.com/badge/github.com/qq1582185982/subscription-converter)](https://goreportcard.com/report/github.com/qq1582185982/subscription-converter)

基于Go语言和SQLite数据库的高性能Clash配置转换服务器，支持将Clash YAML格式配置转换为通用的订阅链接格式。

## ✨ 功能特点

- 🚀 **高性能** - Go语言开发，高并发处理能力
- 💾 **SQLite数据库** - 数据持久化，支持事务操作
- 🎯 **智能去重** - 相同配置复用订阅链接，避免重复生成
- 🔐 **管理员后台** - 完整的用户管理和权限控制系统
- 📊 **数据统计** - 实时显示订阅数量、节点统计等信息
- 🔄 **自动更新** - URL配置源支持实时更新
- 🌐 **Web界面** - 现代化响应式设计，支持移动端
- 📱 **跨平台** - 支持Windows、Linux，单文件部署
- 🔗 **多协议支持** - 支持SS、VMess、Trojan等主流协议
- 🛡️ **会话管理** - 安全的登录会话控制

## 🚀 快速开始

### Windows

1. **下载可执行文件**
   ```bash
   # 下载 subscription-converter.exe
   ```

2. **运行程序**
   ```bash
   .\subscription-converter.exe
   ```

3. **访问服务**
   - 本地访问: http://localhost:8856
   - 局域网访问: http://[你的IP]:8856

### Linux (CentOS/RHEL)

1. **一键部署**
   ```bash
   # 下载部署脚本和可执行文件
   bash deploy-centos.sh
   ```

2. **手动部署**
   ```bash
   # 下载可执行文件
   chmod +x subscription-converter-linux
   ./subscription-converter-linux
   ```

## 🎮 使用说明

### 首次设置

1. **访问服务器地址**: http://your-server:8856
2. **设置管理员账号**: 首次访问会自动跳转到设置页面
3. **输入用户名和密码**: 完成管理员账户配置

### 生成订阅链接

1. **在主页输入配置**:
   - URL链接: 输入Clash配置文件下载地址
   - 直接粘贴: 将配置内容粘贴到文本框
2. **点击生成**: 系统会自动转换并生成订阅链接
3. **复制链接**: 将生成的订阅链接添加到代理客户端

### 管理后台

1. **登录管理后台**: 点击页面右上角"管理后台"
2. **查看订阅记录**: 显示所有生成的订阅及统计信息
3. **管理订阅**: 查看创建时间、更新时间、节点数量等

## 🏗️ 部署指南

### CentOS 7/8/9

详细部署文档: [README-CENTOS.md](README-CENTOS.md)

```bash
# 一键部署
bash deploy-centos.sh
```

### Docker 部署

```dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY subscription-converter-linux /app/
EXPOSE 8856
CMD ["./subscription-converter-linux"]
```

```bash
docker build -t subscription-converter .
docker run -d -p 8856:8856 -v $(pwd)/data:/app subscription-converter
```

### 系统服务

程序支持作为系统服务运行：

```bash
# 查看服务状态
systemctl status subscription-converter

# 启动/停止/重启
systemctl start subscription-converter
systemctl stop subscription-converter
systemctl restart subscription-converter

# 查看日志
journalctl -fu subscription-converter
```

## 🔧 构建说明

### 环境要求

- Go 1.19+
- Git

### 编译

```bash
# 克隆项目
git clone https://github.com/qq1582185982/subscription-converter.git
cd subscription-converter

# 安装依赖
go mod tidy

# Windows版本
go build -ldflags "-s -w" -o subscription-converter.exe .

# Linux版本  
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o subscription-converter-linux .
```

### 快速构建

```bash
# Windows
.\build.bat

# Linux/Mac
bash build.sh
```

### GitHub Actions 自动构建

本项目集成了 GitHub Actions，支持自动构建多平台版本：

- **触发条件**: 推送代码、创建标签、提交PR
- **支持平台**: Windows、Linux、macOS (x64/ARM64)  
- **自动发布**: 创建标签时自动创建 GitHub Release
- **Docker 镜像**: 自动构建并推送到 GitHub Container Registry

**使用方法**:
```bash
# 创建新版本标签
git tag v2.1.0
git push origin v2.1.0

# Actions 将自动:
# 1. 构建所有平台版本
# 2. 创建 GitHub Release
# 3. 上传构建产物
# 4. 构建 Docker 镜像
```

详细说明请参考: [README-ACTIONS.md](README-ACTIONS.md)

## 📡 API接口

### 转换接口

**POST** `/api/convert`

```json
{
  "config_source": "url|text",
  "config_url": "https://example.com/config.yaml",
  "config_text": "clash配置内容"
}
```

**响应**:
```json
{
  "success": true,
  "message": "转换成功！找到 10 个代理节点，订阅ID: abc123",
  "subscription_url": "http://localhost:8856/subscription/abc123",
  "subscription_id": "abc123",
  "proxy_count": 10
}
```

### 订阅接口

**GET** `/subscription/{id}`

返回Base64编码的订阅内容。

### 管理接口

**GET** `/api/subscriptions` (需要登录)

返回所有订阅记录和统计信息。

## 🗄️ 数据库结构

程序使用SQLite数据库存储数据：

- **admin_config**: 管理员配置
- **subscriptions**: 订阅配置和内容
- **sessions**: 登录会话管理  
- **config_hash_map**: 配置哈希映射(去重用)

数据库文件: `subscription.db`

## 🎯 技术栈

- **后端**: Go 1.19
- **数据库**: SQLite (modernc.org/sqlite)
- **前端**: HTML5 + CSS3 + JavaScript
- **模板引擎**: Go html/template
- **HTTP服务器**: Go net/http
- **配置解析**: gopkg.in/yaml.v3

## 🔌 支持的协议

| 协议 | 支持状态 | 说明 |
|------|----------|------|
| Shadowsocks (SS) | ✅ 完整支持 | 包括各种加密方式 |
| VMess | ✅ 完整支持 | V2Ray协议 |
| Trojan | ✅ 完整支持 | Trojan-GFW协议 |

## 🎛️ 配置说明

### 端口配置

默认端口: `8856`

如需修改端口，需要重新编译程序。

### 数据库配置

- 数据库文件: `subscription.db`
- 自动创建表结构
- 支持事务操作
- 自动索引优化

## ❓ 常见问题

### Q: 如何重置管理员密码？
A: 删除 `subscription.db` 文件，重启程序会重新进入首次设置。

### Q: 为什么相同配置生成了不同的订阅链接？
A: 程序已实现智能去重，相同配置会复用已有订阅链接。

### Q: 如何备份数据？
A: 备份 `subscription.db` 文件即可。

### Q: 服务器重启后数据会丢失吗？
A: 不会，所有数据都保存在SQLite数据库中。

### Q: 支持HTTPS吗？
A: 程序本身使用HTTP，可通过Nginx反向代理实现HTTPS。

## 📂 项目结构

```
subscription-converter/
├── main.go                    # 主程序源码
├── template.go                # HTML模板源码
├── go.mod                     # Go模块配置
├── go.sum                     # Go依赖锁定
├── build.bat                  # Windows构建脚本
├── build.sh                   # Linux构建脚本
├── deploy-centos.sh           # CentOS部署脚本
├── install_and_build.bat      # Windows安装脚本
├── README.md                  # 项目说明
├── README-GO.md               # Go版本详细说明
├── README-CENTOS.md           # CentOS部署指南
├── subscription-converter.exe # Windows可执行文件
├── subscription-converter-linux # Linux可执行文件
└── subscription.db            # SQLite数据库(运行时生成)
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 开发流程

1. Fork 项目
2. 创建特性分支: `git checkout -b feature/amazing-feature`
3. 提交更改: `git commit -m 'Add amazing feature'`
4. 推送分支: `git push origin feature/amazing-feature`
5. 提交 Pull Request

## 📄 许可证

本项目基于 MIT 许可证开源 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🔄 更新日志

### v2.0.0 (Go版本)
- ✨ 完全重写为Go语言版本
- 💾 集成SQLite数据库存储
- 🎯 实现智能去重功能
- 🔐 添加管理员后台系统
- 📊 增加数据统计功能
- 🔄 支持配置自动更新
- 🛡️ 完善会话管理机制
- 📱 响应式设计优化

### v1.0.0 (Python版本)
- 🎉 初始版本发布
- 🌐 基础Web界面
- 🔗 支持SS和VMess协议转换

## 🙏 致谢

感谢所有为项目做出贡献的开发者！

---

⭐ 如果这个项目对你有帮助，请给个 Star 支持一下！ 