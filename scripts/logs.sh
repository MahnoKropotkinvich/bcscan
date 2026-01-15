#!/bin/bash

SERVICE=$1

if [ -z "$SERVICE" ]; then
    echo "用法: ./logs.sh <service-name>"
    echo "可用服务: postgres, redis, redpanda, ganache"
    exit 1
fi

cd "$(dirname "$0")/../deployments"

podman-compose -f podman-compose.yml logs -f "$SERVICE"
