#!/bin/bash

# 编译脚本

echo "🚀 开始编译 GGCode..."

# 设置Go代理（如果需要）
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn

# 编译程序
echo "📦 正在编译..."
go build -o ggcode main.go

if [ $? -eq 0 ]; then
    echo "✅ 编译成功！"
    echo "📁 可执行文件: ./ggcode"
    ls -la ggcode
else
    echo "❌ 编译失败"
    exit 1
fi

echo ""
echo "🎯 接下来可以："
echo "1. 测试运行: ./ggcode"
echo "2. 创建systemd服务" 