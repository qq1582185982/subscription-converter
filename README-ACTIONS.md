# 🚀 GitHub Actions 自动构建指南

本项目使用 GitHub Actions 自动构建多平台版本和 Docker 镜像。

## 📦 支持的平台

自动构建以下平台的二进制文件：

### Windows
- `windows-amd64.exe` - Windows 64位
- `windows-386.exe` - Windows 32位

### Linux
- `linux-amd64` - Linux 64位
- `linux-386` - Linux 32位
- `linux-arm64` - Linux ARM64

### macOS
- `darwin-amd64` - macOS Intel
- `darwin-arm64` - macOS Apple Silicon (M1/M2)

## 🔄 触发条件

GitHub Actions 会在以下情况下自动触发：

1. **推送到主分支** - 代码推送到 `main` 或 `master` 分支
2. **创建标签** - 创建以 `v` 开头的标签（如 `v1.0.0`）
3. **Pull Request** - 创建或更新 PR
4. **手动触发** - 在 GitHub 仓库的 Actions 页面手动运行

## 🏷️ 发布版本

### 创建新版本

1. **本地标记版本**：
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub 自动发布**：
   - Actions 会自动构建所有平台版本
   - 创建 GitHub Release
   - 上传所有二进制文件

### 版本号格式
- `v1.0.0` - 正式版本
- `v1.0.0-beta.1` - 测试版本
- `v1.0.0-rc.1` - 候选版本

## 🐳 Docker 镜像

### 自动构建
- 推送到主分支时自动构建 Docker 镜像
- 支持 `linux/amd64` 和 `linux/arm64` 架构
- 推送到 GitHub Container Registry

### 镜像标签
- `ghcr.io/用户名/仓库名:latest` - 最新版本
- `ghcr.io/用户名/仓库名:v1.0.0` - 特定版本
- `ghcr.io/用户名/仓库名:main` - 主分支

### 使用 Docker 镜像
```bash
# 拉取最新镜像
docker pull ghcr.io/qq1582185982/subscription-converter:latest

# 运行容器
docker run -d \
  --name subscription-converter \
  -p 8856:8856 \
  -v $(pwd)/data:/app/data \
  ghcr.io/qq1582185982/subscription-converter:latest
```

## 📁 构建产物

### Actions Artifacts
每次构建都会生成 Artifacts，包含：
- 所有平台的二进制文件
- 保存90天
- 可从 Actions 页面下载

### Release Assets
创建标签时会生成 Release，包含：
- 所有平台的二进制文件
- 永久保存
- 可从 Releases 页面下载

## 🛠️ 自定义构建

### 修改构建平台
编辑 `.github/workflows/build.yml` 文件中的 `matrix` 部分：

```yaml
strategy:
  matrix:
    include:
      - os: windows
        arch: amd64
        ext: .exe
      # 添加或删除其他平台
```

### 修改构建参数
调整 `go build` 命令的参数：

```yaml
run: |
  go build -ldflags "-s -w -X main.version=${GITHUB_REF#refs/tags/}" \
    -o dist/subscription-converter-${{ matrix.os }}-${{ matrix.arch }}${{ matrix.ext }} .
```

## 🔍 查看构建状态

1. **Actions 页面**：访问仓库的 Actions 标签页
2. **构建状态徽章**：在 README 中添加状态徽章
3. **构建日志**：点击具体的构建查看详细日志

## 📊 构建缓存

- **Go modules 缓存**：加速依赖下载
- **Docker 层缓存**：加速镜像构建
- **自动清理**：过期缓存自动清理

## 🔒 权限设置

确保仓库设置正确的权限：

1. **Settings → Actions → General**
2. **Workflow permissions** 设置为 "Read and write permissions"
3. **允许 GitHub Actions 创建和批准 pull requests**

## 🚨 故障排除

### 常见问题

1. **构建失败**：
   - 检查 go.mod 和 go.sum 文件
   - 确保代码可以正常编译

2. **发布失败**：
   - 检查 GITHUB_TOKEN 权限
   - 确保标签格式正确

3. **Docker 构建失败**：
   - 检查 Dockerfile 语法
   - 确保基础镜像可用

### 调试方法

1. **启用调试日志**：
   ```yaml
   - name: Enable debug logging
     run: echo "ACTIONS_STEP_DEBUG=true" >> $GITHUB_ENV
   ```

2. **SSH 调试**：
   ```yaml
   - name: Setup tmate session
     uses: mxschmitt/action-tmate@v3
   ```

## 📚 参考资料

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [Go 交叉编译](https://golang.org/doc/install/source#environment)
- [Docker 多平台构建](https://docs.docker.com/build/building/multi-platform/) 