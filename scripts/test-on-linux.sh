#!/bin/bash

# Bubbleterm 测试脚本 - 在 Linux 服务器上运行
# 使用方法：通过 SSH 将此脚本和代码上传到服务器，然后执行

set -e

echo "🚀 Bubbleterm 终端模拟器测试"
echo "================================"
echo ""

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "❌ 未安装 Go 环境"
    echo "请先安装 Go: https://go.dev/dl/"
    exit 1
fi

echo "✅ Go 版本: $(go version)"
echo ""

# 设置环境变量
export SSH_MCP_TERMINAL_EMULATOR=bubbleterm

# 编译并运行测试
echo "📦 编译测试程序..."
go build -o /tmp/test-bubbleterm cmd/test-bubbleterm-real/main.go

echo "🚀 运行测试..."
/tmp/test-bubbleterm

echo ""
echo "================================"
echo "✅ 测试完成"
