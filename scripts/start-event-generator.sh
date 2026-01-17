#!/bin/bash

cd "$(dirname "$0")"

echo "=== BCScan 风险事件生成器 ==="
echo ""

# 检查 Ganache 是否运行
if ! curl -s http://localhost:8545 > /dev/null 2>&1; then
    echo "❌ Ganache 未运行，请先启动："
    echo "   cd deployments && podman-compose up ganache"
    exit 1
fi

echo "✅ Ganache 已运行"
echo ""

# 安装依赖
if [ ! -d "node_modules" ]; then
    echo "📦 安装依赖..."
    npm install
    echo ""
fi

# 启动生成器
echo "🚀 启动风险事件生成器..."
echo ""
node generate-risk-events.js
