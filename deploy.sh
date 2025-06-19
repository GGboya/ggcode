#!/bin/bash

# GGCode 部署脚本

set -e

echo "🚀 开始部署 GGCode 服务..."

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then
    echo "❌ 请使用root用户运行此脚本"
    exit 1
fi

# 1. 编译程序
echo "📦 编译程序..."
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
go build -o ggcode main.go

if [ ! -f "ggcode" ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"

# 2. 停止现有服务（如果存在）
if systemctl is-active --quiet ggcode; then
    echo "🛑 停止现有服务..."
    systemctl stop ggcode
fi

# 3. 提示设置数据库密码
echo "⚠️  请确保在 ggcode.service 中设置正确的数据库密码"
echo "   当前配置: DB_PASSWORD=your_secure_password"
echo ""

# 4. 复制服务文件
echo "📋 安装systemd服务..."
cp ggcode.service /etc/systemd/system/

# 5. 重新加载systemd
echo "🔄 重新加载systemd..."
systemctl daemon-reload

# 6. 启用并启动服务
echo "🚀 启动服务..."
systemctl enable ggcode
systemctl start ggcode

# 7. 检查服务状态
echo "📊 检查服务状态..."
sleep 3
if systemctl is-active --quiet ggcode; then
    echo "✅ 服务启动成功！"
    systemctl status ggcode --no-pager
else
    echo "❌ 服务启动失败，查看错误日志："
    journalctl -u ggcode -n 10 --no-pager
fi

echo ""
echo "🎯 部署完成！"
echo ""
echo "📋 常用命令："
echo "  查看状态: systemctl status ggcode"
echo "  查看日志: journalctl -u ggcode -f"
echo "  重启服务: systemctl restart ggcode"
echo "  停止服务: systemctl stop ggcode"
echo "  禁用服务: systemctl disable ggcode"
echo ""
echo "🌐 服务地址: http://your-server-ip:8080" 