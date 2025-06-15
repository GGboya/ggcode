#!/bin/bash

echo "🔧 MySQL权限修复脚本"
echo "==================="

echo "这个脚本将帮助你修复MySQL root用户的权限问题"
echo ""

# 检查MySQL是否运行
if ! systemctl is-active --quiet mysql 2>/dev/null; then
    echo "⚠️  MySQL服务未运行，尝试启动..."
    sudo systemctl start mysql
    if [ $? -ne 0 ]; then
        echo "❌ 无法启动MySQL服务"
        exit 1
    fi
fi

echo "方法1: 使用sudo连接MySQL并重置权限"
echo "=================================="
echo "请输入以下命令来修复权限："
echo ""
echo "sudo mysql -u root"
echo ""
echo "然后在MySQL命令行中执行："
echo "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY 'your_new_password';"
echo "FLUSH PRIVILEGES;"
echo "EXIT;"
echo ""

echo "方法2: 重置MySQL root密码"
echo "========================"
echo "如果方法1不工作，可以尝试重置密码："
echo ""
echo "1. 停止MySQL服务:"
echo "   sudo systemctl stop mysql"
echo ""
echo "2. 跳过权限启动MySQL:"
echo "   sudo mysqld_safe --skip-grant-tables --skip-networking &"
echo ""
echo "3. 连接MySQL:"
echo "   mysql -u root"
echo ""
echo "4. 重置密码:"
echo "   USE mysql;"
echo "   ALTER USER 'root'@'localhost' IDENTIFIED BY 'new_password';"
echo "   FLUSH PRIVILEGES;"
echo "   EXIT;"
echo ""
echo "5. 重启MySQL:"
echo "   sudo pkill mysqld"
echo "   sudo systemctl start mysql"
echo ""

echo "方法3: 使用Docker (最简单)"
echo "========================="
echo "如果上述方法都不工作，建议使用Docker:"
echo ""
echo "docker-compose up -d"
echo ""
echo "这将启动一个全新的MySQL容器，密码已预设为: ggcode123"
echo ""

echo "💡 建议:"
echo "1. 先尝试方法1"
echo "2. 如果不行，使用方法3 (Docker)"
echo "3. 方法2作为最后手段"
echo ""

read -p "是否要我帮你自动尝试方法1? (y/n): " choice
if [ "$choice" = "y" ] || [ "$choice" = "Y" ]; then
    echo ""
    echo "🔄 尝试使用sudo连接MySQL..."
    
    # 创建临时SQL文件
    cat > /tmp/fix_mysql.sql << 'EOF'
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY 'ggcode123';
FLUSH PRIVILEGES;
SELECT 'MySQL权限修复完成!' as Status;
EOF

    # 尝试执行
    if sudo mysql -u root < /tmp/fix_mysql.sql; then
        echo "✅ MySQL权限修复成功!"
        echo "新密码: ggcode123"
        
        # 更新.env文件
        if [ -f ".env" ]; then
            sed -i 's/DB_PASSWORD=.*/DB_PASSWORD=ggcode123/' .env
            echo "✅ .env文件已更新"
        else
            cp env.example .env
            sed -i 's/DB_PASSWORD=your_password/DB_PASSWORD=ggcode123/' .env
            echo "✅ 已创建.env文件"
        fi
        
        # 清理临时文件
        rm -f /tmp/fix_mysql.sql
        
        echo ""
        echo "🎉 现在可以运行程序了:"
        echo "go run main.go"
        
    else
        echo "❌ 自动修复失败，请手动尝试上述方法"
        rm -f /tmp/fix_mysql.sql
    fi
else
    echo "请手动执行上述方法之一"
fi 