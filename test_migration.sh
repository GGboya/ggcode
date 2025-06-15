#!/bin/bash

echo "🧪 数据迁移测试脚本"
echo "==================="

# 检查SQLite数据库是否存在
if [ ! -f "ggcode.db" ]; then
    echo "❌ 未找到 ggcode.db 文件"
    exit 1
fi

echo "📊 SQLite数据库统计："
sqlite3 ggcode.db "
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

echo ""
echo "📤 导出SQLite数据..."
sqlite3 ggcode.db .dump > migration_test_backup.sql

if [ $? -eq 0 ]; then
    echo "✅ SQLite数据导出成功"
else
    echo "❌ SQLite数据导出失败"
    exit 1
fi

echo "🔄 转换数据格式..."
python3 migrate_to_mysql.py migration_test_backup.sql migration_test_mysql.sql

if [ $? -eq 0 ]; then
    echo "✅ 数据格式转换成功"
    echo ""
    echo "📁 生成的文件："
    echo "  - migration_test_backup.sql (SQLite备份)"
    echo "  - migration_test_mysql.sql (MySQL格式)"
    echo ""
    echo "💡 你可以查看这些文件来验证数据迁移是否正确"
    echo "💡 如果满意，可以运行 ./deploy_mysql.sh 进行完整部署"
else
    echo "❌ 数据格式转换失败"
    exit 1
fi

echo ""
echo "🎉 数据迁移测试完成！" 