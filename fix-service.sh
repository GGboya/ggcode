#!/bin/bash

echo "🔧 修复 GGCode systemd 服务..."

# 停止服务
echo "🛑 停止服务..."
systemctl stop ggcode

# 更新服务文件
echo "📋 更新服务文件..."
cp ggcode.service /etc/systemd/system/

# 重新加载systemd
echo "🔄 重新加载systemd..."
systemctl daemon-reload

# 启动服务
echo "🚀 启动服务..."
systemctl start ggcode

# 检查状态
echo "📊 检查服务状态..."
sleep 3

if systemctl is-active --quiet ggcode; then
    echo "✅ 服务修复成功！"
    systemctl status ggcode --no-pager
    echo ""
    echo "🌐 服务地址: http://$(hostname -I | awk '{print $1}'):8080"
else
    echo "❌ 服务仍然失败，查看详细日志："
    journalctl -u ggcode -n 15 --no-pager
    echo ""
    echo "🔍 尝试手动运行程序测试："
    echo "   cd /root/projects/ggcode && ./ggcode"
fi 