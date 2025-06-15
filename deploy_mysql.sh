#!/bin/bash

echo "🚀 GGCode MySQL 部署脚本"
echo "========================"

# 检查MySQL是否安装
if ! command -v mysql &> /dev/null; then
    echo "❌ MySQL未安装，请先安装MySQL"
    echo "Ubuntu/Debian: sudo apt-get install mysql-server"
    echo "CentOS/RHEL: sudo yum install mysql-server"
    echo "macOS: brew install mysql"
    exit 1
fi

# 检查MySQL服务是否运行
if ! systemctl is-active --quiet mysql 2>/dev/null && ! brew services list | grep mysql | grep started &> /dev/null; then
    echo "⚠️  MySQL服务未运行，尝试启动..."
    if command -v systemctl &> /dev/null; then
        sudo systemctl start mysql
    elif command -v brew &> /dev/null; then
        brew services start mysql
    else
        echo "❌ 无法启动MySQL服务，请手动启动"
        exit 1
    fi
fi

# 提示用户输入MySQL root密码
echo "📝 请输入MySQL root密码（如果没有设置密码，直接按回车）:"
read -s MYSQL_ROOT_PASSWORD

# 测试MySQL连接
if [ -z "$MYSQL_ROOT_PASSWORD" ]; then
    mysql -u root -e "SELECT 1;" &> /dev/null
else
    mysql -u root -p"$MYSQL_ROOT_PASSWORD" -e "SELECT 1;" &> /dev/null
fi

if [ $? -ne 0 ]; then
    echo "❌ MySQL连接失败，请检查密码"
    exit 1
fi

echo "✅ MySQL连接成功"

# 创建数据库
echo "📊 创建数据库..."
if [ -z "$MYSQL_ROOT_PASSWORD" ]; then
    mysql -u root < setup_mysql.sql
else
    mysql -u root -p"$MYSQL_ROOT_PASSWORD" < setup_mysql.sql
fi

if [ $? -eq 0 ]; then
    echo "✅ 数据库创建成功"
else
    echo "❌ 数据库创建失败"
    exit 1
fi

# 检查是否存在SQLite数据库文件，如果存在则进行数据迁移
if [ -f "ggcode.db" ]; then
    echo "📦 发现SQLite数据库，开始数据迁移..."
    
    # 检查Python是否可用
    if command -v python3 &> /dev/null; then
        # 导出SQLite数据
        echo "📤 导出SQLite数据..."
        sqlite3 ggcode.db .dump > ggcode_backup.sql
        
        if [ $? -eq 0 ]; then
            echo "✅ SQLite数据导出成功"
            
            # 转换为MySQL格式
            echo "🔄 转换数据格式..."
            python3 migrate_to_mysql.py ggcode_backup.sql ggcode_mysql.sql
            
            if [ $? -eq 0 ]; then
                echo "✅ 数据格式转换成功"
                
                # 导入到MySQL
                echo "📥 导入数据到MySQL..."
                if [ -z "$MYSQL_ROOT_PASSWORD" ]; then
                    mysql -u root ggcode < ggcode_mysql.sql
                else
                    mysql -u root -p"$MYSQL_ROOT_PASSWORD" ggcode < ggcode_mysql.sql
                fi
                
                if [ $? -eq 0 ]; then
                    echo "✅ 数据迁移成功！"
                    echo "📊 数据迁移统计："
                    
                    # 显示迁移的数据统计
                    if [ -z "$MYSQL_ROOT_PASSWORD" ]; then
                        mysql -u root ggcode -e "
                        SELECT 'Users' as Table_Name, COUNT(*) as Count FROM users
                        UNION ALL
                        SELECT 'Question Banks', COUNT(*) FROM question_banks
                        UNION ALL
                        SELECT 'Questions', COUNT(*) FROM questions
                        UNION ALL
                        SELECT 'Study Plans', COUNT(*) FROM user_study_plans
                        UNION ALL
                        SELECT 'Progress Records', COUNT(*) FROM user_question_progresses
                        UNION ALL
                        SELECT 'Check-ins', COUNT(*) FROM user_check_ins;
                        "
                    else
                        mysql -u root -p"$MYSQL_ROOT_PASSWORD" ggcode -e "
                        SELECT 'Users' as Table_Name, COUNT(*) as Count FROM users
                        UNION ALL
                        SELECT 'Question Banks', COUNT(*) FROM question_banks
                        UNION ALL
                        SELECT 'Questions', COUNT(*) FROM questions
                        UNION ALL
                        SELECT 'Study Plans', COUNT(*) FROM user_study_plans
                        UNION ALL
                        SELECT 'Progress Records', COUNT(*) FROM user_question_progresses
                        UNION ALL
                        SELECT 'Check-ins', COUNT(*) FROM user_check_ins;
                        "
                    fi
                    
                    # 清理临时文件
                    rm -f ggcode_backup.sql ggcode_mysql.sql
                    
                    # 备份原SQLite文件
                    mv ggcode.db ggcode.db.backup
                    echo "📁 原SQLite文件已备份为 ggcode.db.backup"
                else
                    echo "❌ 数据导入失败"
                    echo "💡 你可以稍后手动导入：mysql -u root -p ggcode < ggcode_mysql.sql"
                fi
            else
                echo "❌ 数据格式转换失败"
            fi
        else
            echo "❌ SQLite数据导出失败"
        fi
    else
        echo "⚠️  未找到Python3，跳过自动数据迁移"
        echo "💡 你可以手动迁移数据："
        echo "   1. sqlite3 ggcode.db .dump > ggcode_backup.sql"
        echo "   2. python3 migrate_to_mysql.py ggcode_backup.sql ggcode_mysql.sql"
        echo "   3. mysql -u root -p ggcode < ggcode_mysql.sql"
    fi
else
    echo "ℹ️  未发现SQLite数据库文件，跳过数据迁移"
fi

# 创建环境配置文件
echo "⚙️  创建环境配置..."
if [ ! -f ".env" ]; then
    cp env.example .env
    echo "📝 请编辑 .env 文件，设置正确的数据库密码"
    echo "   DB_PASSWORD=$MYSQL_ROOT_PASSWORD"
    
    # 自动设置密码
    if [ ! -z "$MYSQL_ROOT_PASSWORD" ]; then
        sed -i "s/DB_PASSWORD=your_password/DB_PASSWORD=$MYSQL_ROOT_PASSWORD/" .env
    fi
else
    echo "⚠️  .env 文件已存在，跳过创建"
fi


echo ""
echo "🎉 MySQL迁移完成！"
echo "========================"
echo "📋 接下来的步骤："
echo "1. 检查并编辑 .env 文件中的数据库配置"
echo "2. 运行程序: ./ggcode"
echo "3. 访问: http://localhost:8080"
echo ""
echo "💡 提示："
if [ -f "ggcode.db.backup" ]; then
    echo "- 原SQLite数据已成功迁移到MySQL"
    echo "- SQLite备份文件: ggcode.db.backup"
fi
echo "- 程序会自动创建表结构和初始化数据"
echo "- 如果遇到问题，请查看日志或联系技术支持"
echo ""

