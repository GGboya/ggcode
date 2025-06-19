#!/bin/bash

# Go 代理设置脚本 - 解决服务器网络问题

echo "🚀 设置 Go 代理..."

# 方案1: 使用七牛云代理（推荐）
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn

# 方案2: 使用阿里云代理（备选）
# export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
# export GOSUMDB=sum.golang.google.cn

# 方案3: 使用官方代理（如果网络正常）
# export GOPROXY=https://proxy.golang.org,direct

echo "✅ Go 代理设置完成"
echo "当前 GOPROXY: $GOPROXY"
echo "当前 GOSUMDB: $GOSUMDB"

# 将设置写入 ~/.bashrc 以便永久生效
echo "" >> ~/.bashrc
echo "# Go 代理设置" >> ~/.bashrc
echo "export GOPROXY=https://goproxy.cn,direct" >> ~/.bashrc
echo "export GOSUMDB=sum.golang.google.cn" >> ~/.bashrc

echo ""
echo "🎯 现在可以运行以下命令："
echo "source ~/.bashrc"
echo "go mod tidy"
echo "go run main.go" 