# GitHub Release 指南

本指南将帮助你创建多平台的GitHub Release，让不同操作系统的用户都能下载使用。

---

## 🚀 快速开始

### 方式1：使用PowerShell脚本（Windows推荐）

```powershell
# 1. 构建所有平台的二进制文件
.\build.ps1 -Version 1.0.0

# 2. 创建并推送Git标签
git tag v1.0.0
git push origin v1.0.0

# 3. 创建GitHub Release
gh release create v1.0.0 --title "v1.0.0" --notes "See CHANGELOG.md"

# 4. 上传所有二进制文件
gh release upload v1.0.0 dist/*
```

### 方式2：使用Bash脚本（Linux/macOS推荐）

```bash
# 1. 给脚本添加执行权限
chmod +x build.sh

# 2. 构建所有平台的二进制文件
VERSION=1.0.0 ./build.sh

# 3. 创建并推送Git标签
git tag v1.0.0
git push origin v1.0.0

# 4. 创建GitHub Release
gh release create v1.0.0 --title "v1.0.0" --notes "See CHANGELOG.md"

# 5. 上传所有二进制文件
gh release upload v1.0.0 dist/*
```

---

## 📦 生成的平台文件

构建脚本会为以下平台生成二进制文件：

### Windows
- `sshmcp-windows-amd64-v1.0.0.zip` (64位，最常见)
- `sshmcp-windows-386-v1.0.0.zip` (32位)
- `sshmcp-windows-arm64-v1.0.0.zip` (ARM64，如Surface Pro X)

### Linux
- `sshmcp-linux-amd64-v1.0.0.tar.gz` (64位，x86服务器)
- `sshmcp-linux-arm64-v1.0.0.tar.gz` (ARM64，如AWS Graviton)
- `sshmcp-linux-386-v1.0.0.tar.gz` (32位)
- `sshmcp-linux-arm-v1.0.0.tar.gz` (ARM v6/v7，如Raspberry Pi)

### macOS
- `sshmcp-darwin-amd64-v1.0.0.tar.gz` (Intel芯片)
- `sshmcp-darwin-arm64-v1.0.0.tar.gz` (Apple M1/M2/M3)

---

## 📋 生成的文件

构建完成后，`dist/` 目录包含：

```
dist/
├── sshmcp-windows-amd64-v1.0.0.zip
├── sshmcp-linux-amd64-v1.0.0.tar.gz
├── sshmcp-darwin-amd64-v1.0.0.tar.gz
├── ... (其他平台)
└── checksums.txt (SHA256校验和)
```

---

## 🔐 安全校验

每个发布包都包含SHA256校验和，位于 `checksums.txt`。

**用户验证下载文件完整性：**

```bash
# Linux/macOS
sha256sum -c checksums.txt

# Windows (PowerShell)
Get-FileHash sshmcp-windows-amd64-v1.0.0.zip -Algorithm SHA256
```

---

## 🎯 完整Release流程示例

### 1. 准备发布

```bash
# 确保工作目录干净
git status

# 更新版本号（可选）
# 编辑代码中的版本号

# 运行测试
go test ./...
```

### 2. 构建多平台二进制文件

**Windows:**
```powershell
.\build.ps1 -Version 1.0.0
```

**Linux/macOS:**
```bash
chmod +x build.sh
VERSION=1.0.0 ./build.sh
```

### 3. 创建Git标签

```bash
git tag -a v1.0.0 -m "Release v1.0.0: ANSI filtering upgrade + README bilingual rewrite"
git push origin v1.0.0
```

### 4. 创建GitHub Release

```bash
# 方式1：使用gh CLI（推荐）
gh release create v1.0.0 \
  --title "SSH MCP Server v1.0.0" \
  --notes "## 🎉 Release v1.0.0

### ✨ New Features
- ECMA-48 standard ANSI filtering
- Bilingual README (English + Chinese)
- Support for 5 mainstream MCP clients

### 📦 Downloads
Select the appropriate binary for your platform:
- Windows: sshmcp-windows-amd64-v1.0.0.zip
- Linux: sshmcp-linux-amd64-v1.0.0.tar.gz
- macOS: sshmcp-darwin-amd64-v1.0.0.tar.gz

### 🔐 Verification
SHA256 checksums are provided in checksums.txt

See CHANGELOG.md for full details."

# 方式2：手动在GitHub网页创建
# 访问：https://github.com/Cigarliu/ssh-mcp-server/releases/new
# 选择标签：v1.0.0
# 发布标题和说明
```

### 5. 上传二进制文件

```bash
gh release upload v1.0.0 dist/*
```

### 6. 验证Release

访问：https://github.com/Cigarliu/ssh-mcp-server/releases/v1.0.0

检查：
- ✅ 标题和描述正确
- ✅ 所有平台的二进制文件都已上传
- ✅ checksums.txt包含在Release中
- ✅ 下载链接正常工作

---

## 💡 用户体验

用户下载和使用的方式：

### Windows用户
```powershell
# 1. 下载
# 从Release页面下载 sshmcp-windows-amd64-v1.0.0.zip

# 2. 解压
Expand-Archive sshmcp-windows-amd64-v1.0.0.zip

# 3. 配置Claude Desktop
# 将可执行文件路径添加到claude_desktop_config.json

# 4. 使用
Claude会自动调用SSH MCP Server
```

### Linux用户
```bash
# 1. 下载
wget https://github.com/Cigarliu/ssh-mcp-server/releases/download/v1.0.0/sshmcp-linux-amd64-v1.0.0.tar.gz

# 2. 解压
tar xzf sshmcp-linux-amd64-v1.0.0.tar.gz

# 3. 安装
chmod +x sshmcp-linux-amd64
sudo mv sshmcp-linux-amd64 /usr/local/bin/sshmcp

# 4. 配置Claude Desktop
# 添加到claude_desktop_config.json
```

### macOS用户
```bash
# 1. 下载
curl -L -O https://github.com/Cigarliu/ssh-mcp-server/releases/download/v1.0.0/sshmcp-darwin-amd64-v1.0.0.tar.gz

# 2. 解压
tar xzf sshmcp-darwin-amd64-v1.0.0.tar.gz

# 3. 安装
chmod +x sshmcp-darwin-amd64
sudo mv sshmcp-darwin-amd64 /usr/local/bin/sshmcp

# 4. 允许运行（macOS安全限制）
xattr -d /usr/local/bin/sshmcp
```

---

## 🔄 自动化CI/CD（可选）

### 使用GitHub Actions自动构建和发布

创建 `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: go test -v ./...

      - name: Build release binaries
        run: |
          VERSION=${GITHUB_REF#refs/tags/} ./build.sh

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v1
        with:
          files: dist/*
          draft: false
          prerelease: false
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

这样每次推送标签时，GitHub Actions会自动：
1. 运行测试
2. 构建所有平台的二进制文件
3. 创建Release并上传文件

---

## 📊 版本号规范

建议使用语义化版本号：

- **v1.0.0** - 第一个稳定版本
- **v1.1.0** - 新增功能
- **v1.1.1** - Bug修复
- **v2.0.0** - 重大更新或不兼容变更

---

## ⚡ 快速命令参考

```bash
# 构建所有平台
./build.sh                    # Linux/macOS
.\build.ps1                   # Windows

# 创建标签
git tag v1.0.0
git push origin v1.0.0

# 创建Release（需要gh CLI）
gh release create v1.0.0 --title "v1.0.0" --notes "Release notes"
gh release upload v1.0.0 dist/*

# 删除Release（如果出错了）
gh release delete v1.0.0
git push origin :v1.0.0
git tag -d v1.0.0

# 查看所有Release
gh release list
```

---

## 🎉 完成！

现在你的项目支持多平台下载了！用户可以根据自己的操作系统下载对应的二进制文件，无需编译即可使用。

记得在README中添加下载说明和使用指南！
