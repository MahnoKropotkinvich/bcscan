#!/bin/bash

set -e

echo "🚀 启动 bcscan 开发环境..."

# 检查 podman-compose 是否安装
if ! command -v podman-compose &> /dev/null; then
    echo "❌ podman-compose 未安装"
    echo "请运行: pip3 install podman-compose"
    exit 1
fi

# 进入部署目录
cd "$(dirname "$0")/../deployments"

# 创建数据目录
echo "📁 创建数据目录..."
mkdir -p ../data/{postgres,redis,redpanda}

# 启动服务
echo "🐳 启动容器服务..."
podman-compose -f podman-compose.yml up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 15

# 检查服务状态
echo ""
echo "📊 服务状态："
podman-compose -f podman-compose.yml ps

echo ""
echo "✅ 开发环境启动完成！"
echo ""
echo "服务访问地址："
echo "  - PostgreSQL: localhost:5432"
echo "  - Redis: localhost:6379"
echo "  - Redpanda: localhost:9092"
echo "  - Ganache: http://localhost:8545"
