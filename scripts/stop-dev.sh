#!/bin/bash

set -e

echo "🛑 停止 bcscan 开发环境..."

cd "$(dirname "$0")/../deployments"

podman-compose -f docker-compose.yml down

echo "✅ 开发环境已停止"
