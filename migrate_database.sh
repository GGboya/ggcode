#!/bin/bash

# 数据库迁移脚本
# 使用方法: ./migrate_database.sh [server_host] [server_user] [server_password]

set -e

# 配置
LOCAL_DB="ggcode"
LOCAL_USER="root"
BACKUP_FILE="$HOME/ggcode_migration_$(date +%Y%m%d_%H%M%S).sql"

# 服务器配置（可以通过参数传入）
SERVER_HOST=${1:-"your-server-ip"}
SERVER_USER=${2:-"root"}
SERVER_PASSWORD=${3:-""}

echo "🚀 开始数据库迁移..."

# 1. 导出本地数据库
echo "📦 正在导出本地数据库..."
mysqldump -u $LOCAL_USER -p --single-transaction --routines --triggers --databases $LOCAL_DB > $BACKUP_FILE

if [ $? -eq 0 ]; then
    echo "✅ 本地数据库导出成功: $BACKUP_FILE"
else
    echo "❌ 本地数据库导出失败"
    exit 1
fi

# 2. 显示备份文件信息
echo "📊 备份文件信息:"
ls -lh $BACKUP_FILE

# 3. 提供导入命令
echo ""
echo "🔄 请在服务器上执行以下命令来导入数据库:"
echo "mysql -u $SERVER_USER -p < $BACKUP_FILE"
echo ""

# 4. 如果提供了服务器信息，尝试自动传输
if [ "$SERVER_HOST" != "your-server-ip" ]; then
    echo "📤 正在传输到服务器..."
    scp $BACKUP_FILE $SERVER_USER@$SERVER_HOST:~/
    
    if [ $? -eq 0 ]; then
        echo "✅ 文件传输成功"
        echo "🔄 请在服务器上执行: mysql -u root -p < ~/$BACKUP_FILE"
    else
        echo "❌ 文件传输失败，请手动传输文件"
    fi
fi

echo ""
echo "🎯 迁移准备完成！"
echo "📝 备份文件: $BACKUP_FILE"
echo "💡 提示: 导入前请确保服务器MySQL已安装并运行" 