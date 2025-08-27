# 发布流程说明

## 概述

本项目使用 GoReleaser + GitHub Actions + Scoop 实现自动化发布流程。

## 发布前准备

### 1. 配置 GitHub Secrets

在 GitHub 仓库设置中添加以下 Secrets（可选）：
- `SCOOP_BUCKET_GITHUB_TOKEN`: 用于推送到 Scoop bucket 仓库的 Personal Access Token

### 2. 创建 Scoop Bucket 仓库（可选）

如果需要支持 Scoop 安装，需要先创建一个 Scoop bucket 仓库：

1. 创建新仓库，命名为 `scoop-bucket`
2. 在仓库中创建 `bucket` 目录
3. 生成 Personal Access Token 并配置到主仓库的 Secrets 中

## 发布流程

### 自动发布

1. 创建并推送版本标签：
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions 会自动触发发布流程：
   - 构建多平台二进制文件
   - 创建 GitHub Release
   - 生成 Changelog
   - 更新 Scoop manifest（如果配置了）

### 手动发布

如果需要手动发布，可以在本地运行：

```bash
# 测试构建
goreleaser build --snapshot --clean

# 正式发布（需要设置 GITHUB_TOKEN）
export GITHUB_TOKEN=your_token_here
goreleaser release --clean
```

## 支持的平台

- **Linux**: amd64, arm64, 386
- **Windows**: amd64, arm64, 386
- **macOS**: amd64, arm64

## 安装方式

### 直接下载

用户可以从 GitHub Releases 页面下载对应平台的二进制文件。

### Scoop (Windows)

```powershell
# 添加 bucket
scoop bucket add sterm https://github.com/huanfeng/scoop-bucket

# 安装
scoop install sterm
```

### 手动安装

#### Linux/macOS
```bash
# 下载
curl -L https://github.com/huanfeng/sterm/releases/latest/download/sterm-v1.0.0-linux-x86_64.tar.gz -o sterm.tar.gz

# 解压
tar xzf sterm.tar.gz

# 安装
sudo mv sterm /usr/local/bin/
sudo chmod +x /usr/local/bin/sterm
```

#### Windows
下载 ZIP 文件，解压后将 `sterm.exe` 添加到系统 PATH。

## 版本号规范

使用语义化版本：
- `v1.0.0`: 主版本.次版本.修订版本
- 主版本：不兼容的 API 修改
- 次版本：向下兼容的功能性新增
- 修订版本：向下兼容的问题修正

## Changelog 规范

提交信息遵循 Conventional Commits 规范：
- `feat:` 新功能
- `fix:` 修复 bug
- `docs:` 文档更新
- `perf:` 性能优化
- `refactor:` 重构
- `test:` 测试
- `chore:` 构建过程或辅助工具的变动

## 故障排除

### GoReleaser 构建失败

1. 检查 Go 版本是否正确
2. 运行 `go mod tidy` 确保依赖正确
3. 本地测试：`goreleaser build --snapshot --clean`

### Scoop 更新失败

1. 检查 `SCOOP_BUCKET_GITHUB_TOKEN` 是否配置
2. 确认 token 有 repo 权限
3. 检查 bucket 仓库是否存在

### GitHub Actions 失败

查看 Actions 日志，常见问题：
- Go 版本不匹配
- 依赖下载失败
- 权限问题