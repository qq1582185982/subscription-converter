# 订阅转换工具 (Go版)

一个使用Go语言开发的独立可执行订阅转换工具，无需任何运行时环境！支持将Clash YAML格式配置转换为通用的订阅链接格式，适用于PassWall、V2rayN等代理客户端。

## 🌟 特色功能

- ✨ **独立可执行文件** - 无需Python环境，直接运行
- 🌐 **美观的Web界面** - 现代化的前端设计
- 📥 **多种输入方式** - 支持URL链接和直接粘贴配置
- 🔄 **实时转换** - 即时转换Clash配置为订阅链接
- 📱 **跨平台支持** - Windows、Linux、macOS一键构建
- 🚀 **高性能** - Go语言原生性能
- 🔗 **多协议支持** - 支持SS、VMess等主流协议

## 📦 快速开始

### 方法一：下载预编译版本

1. 下载对应平台的可执行文件
2. 双击运行（Windows）或 `./subscription-converter`（Linux/Mac）
3. 在浏览器中访问 `http://localhost:8080`

### 方法二：从源码构建

#### Windows用户:
```cmd
# 双击运行构建脚本
build.bat
```

#### Linux/Mac用户:
```bash
# 给脚本执行权限并运行
chmod +x build.sh
./build.sh
```

#### 手动构建:
```bash
# 安装依赖
go mod tidy

# 构建当前平台
go build -ldflags "-s -w" -o subscription-converter .

# 跨平台构建
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o subscription-converter.exe .
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o subscription-converter-linux .
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o subscription-converter-macos .
```

## 🎯 使用说明

### Web界面操作

1. **启动服务器**
   - Windows: 双击 `subscription-converter.exe`
   - Linux/Mac: `./subscription-converter`

2. **访问Web界面**
   - 本地访问: http://localhost:8080
   - 局域网访问: http://[你的IP地址]:8080

3. **转换配置**
   - 选择配置文件来源（URL链接或直接输入）
   - 输入Clash配置文件URL或粘贴配置内容
   - 点击"生成订阅链接"按钮
   - 复制生成的订阅链接到你的代理客户端

### 客户端配置

生成的订阅链接可以直接用于以下客户端：

- **PassWall**: OpenWrt路由器插件
- **V2rayN**: Windows客户端
- **V2rayNG**: Android客户端
- **Clash for Windows**: Windows客户端
- **Clash for Android**: Android客户端
- **Shadowrocket**: iOS客户端

## 🖥️ 服务器部署

### 1. 单机部署

```bash
# 上传可执行文件到服务器
scp subscription-converter-linux user@server:/path/to/app/

# 在服务器上运行
ssh user@server
cd /path/to/app/
chmod +x subscription-converter-linux
./subscription-converter-linux
```

### 2. 后台运行

```bash
# 使用nohup后台运行
nohup ./subscription-converter-linux > server.log 2>&1 &

# 或使用systemd服务
sudo tee /etc/systemd/system/subscription-converter.service << EOF
[Unit]
Description=Subscription Converter
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/path/to/app
ExecStart=/path/to/app/subscription-converter-linux
Restart=always

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable subscription-converter
sudo systemctl start subscription-converter
```

### 3. Docker部署

创建 Dockerfile:
```dockerfile
FROM scratch
COPY subscription-converter-linux /app
EXPOSE 8080
ENTRYPOINT ["/app"]
```

构建并运行:
```bash
docker build -t subscription-converter .
docker run -p 8080:8080 subscription-converter
```

## 📊 API接口

### 转换接口

**POST** `/api/convert`

请求参数:
```json
{
  "config_source": "url|text",
  "config_url": "https://example.com/config.yaml",
  "config_text": "clash配置内容"
}
```

响应示例:
```json
{
  "success": true,
  "message": "转换成功！找到 10 个代理节点",
  "subscription_url": "http://localhost:8080/subscription",
  "proxy_count": 10
}
```

### 订阅接口

**GET** `/subscription`

返回Base64编码的订阅内容，可直接用作订阅链接。

## 📁 文件结构

```
订阅转换/
├── main.go                    # 主程序
├── template.go                # HTML模板
├── go.mod                     # Go模块文件
├── go.sum                     # 依赖校验文件
├── build.bat                  # Windows构建脚本
├── build.sh                   # Linux/Mac构建脚本
├── subscription-converter.exe # Windows可执行文件
├── subscription-converter-linux # Linux可执行文件
├── subscription-converter-macos # macOS可执行文件
└── README-GO.md              # 项目说明
```

## 🛠️ 技术栈

- **语言**: Go 1.19+
- **Web框架**: 标准库 net/http
- **配置解析**: gopkg.in/yaml.v3
- **前端**: HTML + CSS + JavaScript (内嵌)
- **构建工具**: Go原生构建工具

## 🔧 支持的协议

- **Shadowsocks (SS)**: ✅ 完整支持
- **VMess**: ✅ 完整支持  
- **Trojan**: 🚧 计划支持
- **其他协议**: 持续添加中

## ❓ 常见问题

### Q: 如何检查Go环境是否安装？
A: 在命令行中运行 `go version`，如果显示版本信息则已安装。

### Q: 构建时提示找不到模块？
A: 运行 `go mod tidy` 下载依赖包。

### Q: 如何修改服务器端口？
A: 修改 `main.go` 中的 `port := "8080"` 行。

### Q: 生成的可执行文件太大？
A: 已使用 `-ldflags "-s -w"` 参数压缩，如需进一步压缩可使用 UPX 工具。

### Q: 如何在Linux服务器上开机自启？
A: 使用上面的 systemd 服务配置方法。

## 🔗 相关链接

- [Go语言官网](https://golang.org/)
- [Go语言安装指南](https://golang.org/doc/install)
- [Clash配置文档](https://github.com/Dreamacro/clash/wiki/configuration)

## 📈 性能对比

| 版本 | 启动时间 | 内存占用 | 依赖 | 部署难度 |
|------|----------|----------|------|----------|
| Python版 | ~2s | ~50MB | Python环境 | 中等 |
| **Go版** | **~100ms** | **~10MB** | **无** | **极简** |

## 📝 更新日志

### v1.0.0 (Go版)
- ✨ 全新Go语言重写
- 🚀 独立可执行文件，无依赖
- 📱 跨平台支持
- 🎨 全新的现代化界面设计
- ⚡ 高性能，低内存占用
- 🔧 简化的构建和部署流程

## 📜 许可证

MIT License

## 🤝 贡献

欢迎提交Issue和Pull Request来改进项目！

---

**选择Go版本的理由:**
- ✅ 无需运行时环境，部署极简
- ✅ 高性能，低资源占用
- ✅ 跨平台编译，一次构建多平台使用
- ✅ 单文件部署，便于管理和分发 