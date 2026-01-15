#!/bin/bash

# 测试规则热加载功能

API_URL="http://localhost:8080"

echo "=== 测试规则热加载功能 ==="
echo ""

# 1. 获取当前规则
echo "1. 获取当前规则列表..."
curl -s "${API_URL}/api/rules" | jq '.[] | {name: .metadata.name, enabled: .metadata.enabled}'
echo ""

# 2. 触发规则重新加载
echo "2. 触发规则重新加载..."
RESULT=$(curl -s -X POST "${API_URL}/api/rules/reload")
echo "$RESULT" | jq '.'
echo ""

# 3. 再次获取规则确认
echo "3. 确认规则已更新..."
curl -s "${API_URL}/api/rules" | jq '.[] | {name: .metadata.name, enabled: .metadata.enabled}'
echo ""

echo "=== 测试完成 ==="
