# 订阅转换工具

一个简单易用的Clash配置转换工具，支持将Clash YAML格式配置转换为通用的订阅链接格式，适用于PassWall、V2rayN等代理客户端。

## 功能特点

- 🌐 **Web界面操作** - 美观易用的前端界面
- 📥 **多种输入方式** - 支持URL链接和直接粘贴配置
- 🔄 **实时转换** - 即时转换Clash配置为订阅链接
- 📱 **跨平台兼容** - 支持Windows、Linux、macOS
- 🚀 **一键部署** - 可生成独立可执行文件
- 🔗 **多协议支持** - 支持SS、VMess、Trojan等协议

## 快速开始

### 方法一：Python环境运行

1. **安装依赖**
```bash
pip install -r requirements.txt
```

2. **启动服务器**
```bash
python start_server.py
```

3. **访问Web界面**
   - 本地访问: http://localhost:5000
   - 局域网访问: http://[你的IP地址]:5000

### 方法二：生成可执行文件

1. **运行构建脚本**
```bash
python build_exe.py
```

2. **运行生成的可执行文件**
```bash
./dist/订阅转换服务器.exe
```

## 使用说明

### Web界面操作

1. 打开浏览器，访问服务器地址
2. 选择配置文件来源：
   - **URL链接**: 输入Clash配置文件的下载地址
   - **直接输入**: 将Clash配置文件内容粘贴到文本框
3. 点击"生成订阅链接"按钮
4. 复制生成的订阅链接到你的代理客户端

### 客户端配置

生成的订阅链接可以直接用于以下客户端：

- **PassWall**: OpenWrt路由器插件
- **V2rayN**: Windows客户端
- **V2rayNG**: Android客户端
- **Clash for Windows**: Windows客户端
- **Clash for Android**: Android客户端

## 服务器部署

### 本地部署

```bash
# 克隆项目
git clone [项目地址]
cd 订阅转换

# 安装依赖
pip install -r requirements.txt

# 启动服务
python start_server.py
```

### 云服务器部署

1. **上传文件到服务器**
```bash
scp -r . user@server:/path/to/subscription-converter/
```

2. **在服务器上安装依赖并启动**
```bash
cd /path/to/subscription-converter/
pip install -r requirements.txt
python start_server.py
```

3. **后台运行（可选）**
```bash
nohup python start_server.py > server.log 2>&1 &
```

### Docker部署（可选）

创建Dockerfile:
```dockerfile
FROM python:3.9-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .
EXPOSE 5000

CMD ["python", "start_server.py"]
```

构建并运行:
```bash
docker build -t subscription-converter .
docker run -p 5000:5000 subscription-converter
```

## API接口

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
  "subscription_url": "http://localhost:5000/subscription",
  "proxy_count": 10
}
```

### 订阅接口

**GET** `/subscription`

返回Base64编码的订阅内容，可直接用作订阅链接。

## 文件结构

```
订阅转换/
├── web_server.py          # Flask Web服务器
├── start_server.py        # 启动脚本
├── build_exe.py           # 可执行文件生成脚本
├── requirements.txt       # Python依赖
├── templates/             # HTML模板目录
│   └── index.html        # 前端界面
├── clash_config.yaml      # 示例配置文件
├── subscription.txt       # 生成的订阅内容
└── README.md             # 项目说明
```

## 技术栈

- **后端**: Flask (Python Web框架)
- **前端**: HTML + CSS + JavaScript
- **配置解析**: PyYAML
- **HTTP请求**: Requests
- **打包工具**: PyInstaller

## 支持的协议

- **Shadowsocks (SS)**: 完整支持
- **VMess**: 完整支持  
- **Trojan**: 基础支持
- **其他协议**: 持续添加中

## 常见问题

### Q: 为什么转换后节点数量少了？
A: 可能的原因：
- 配置文件中包含不支持的协议类型
- 节点配置格式不标准
- 检查控制台错误信息

### Q: 生成的订阅链接无法使用？
A: 请检查：
- 服务器是否可以从客户端网络访问
- 防火墙是否开放5000端口
- 订阅链接格式是否正确

### Q: 如何修改服务器端口？
A: 在`start_server.py`中修改`port=5000`为其他端口。

## 贡献

欢迎提交Issue和Pull Request来改进项目！

## 许可证

MIT License

## 更新日志

### v1.0.0
- 初始版本发布
- 支持Web界面操作
- 支持SS和VMess协议转换
- 支持生成可执行文件 