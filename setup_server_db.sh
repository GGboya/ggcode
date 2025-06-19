#!/bin/bash

# 服务器数据库配置脚本

echo "🚀 配置服务器数据库..."

# 设置数据库密码（请修改为您的密码）
DB_PASSWORD="your_secure_password"

echo "📋 请选择配置方案："
echo "1. 修复 root 用户权限"
echo "2. 创建新的数据库用户"
echo "3. 设置环境变量"

read -p "请输入选择 (1-3): " choice

case $choice in
    1)
        echo "🔧 修复 root 用户权限..."
        echo "请在MySQL中执行以下命令："
        echo "sudo mysql"
        echo "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '$DB_PASSWORD';"
        echo "FLUSH PRIVILEGES;"
        echo "EXIT;"
        ;;
    2)
        echo "👤 创建新数据库用户..."
        echo "请在MySQL中执行以下命令："
        echo "sudo mysql"
        echo "CREATE USER 'ggcode'@'localhost' IDENTIFIED BY '$DB_PASSWORD';"
        echo "GRANT ALL PRIVILEGES ON ggcode.* TO 'ggcode'@'localhost';"
        echo "FLUSH PRIVILEGES;"
        echo "EXIT;"
        
        # 设置环境变量
        echo "📝 设置环境变量..."
        export DB_USER=ggcode
        export DB_PASSWORD=$DB_PASSWORD
        export DB_HOST=localhost
        export DB_PORT=3306
        export DB_NAME=ggcode
        
        # 写入 .env 文件
        cat > .env << EOF
DB_HOST=localhost
DB_PORT=3306
DB_USER=ggcode
DB_PASSWORD=$DB_PASSWORD
DB_NAME=ggcode
EOF
        echo "✅ 环境变量已设置并写入 .env 文件"
        ;;
    3)
        echo "🔧 设置环境变量..."
        export DB_USER=root
        export DB_PASSWORD=$DB_PASSWORD
        export DB_HOST=localhost
        export DB_PORT=3306
        export DB_NAME=ggcode
        
        # 写入 .env 文件
        cat > .env << EOF
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=$DB_PASSWORD
DB_NAME=ggcode
EOF
        echo "✅ 环境变量已设置"
        ;;
esac

echo ""
echo "🎯 接下来的步骤："
echo "1. 导入数据库: mysql -u root -p < ggcode_backup.sql"
echo "2. 运行应用: go run main.go" 