#!/usr/bin/env python3
"""
SQLite to MySQL 数据迁移脚本
将SQLite的数据导出为MySQL兼容的SQL格式
"""

import re
import sys

def convert_sqlite_to_mysql(sqlite_dump_file, mysql_output_file):
    """将SQLite dump文件转换为MySQL兼容格式"""
    
    with open(sqlite_dump_file, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 移除SQLite特有的语句
    content = re.sub(r'PRAGMA.*?;', '', content, flags=re.IGNORECASE)
    content = re.sub(r'BEGIN TRANSACTION;', '', content, flags=re.IGNORECASE)
    content = re.sub(r'COMMIT;', '', content, flags=re.IGNORECASE)
    
    # 提取所有INSERT语句（包括多行的）
    insert_pattern = r'INSERT INTO\s+[^;]+;'
    insert_matches = re.findall(insert_pattern, content, flags=re.IGNORECASE | re.DOTALL)
    
    mysql_inserts = []
    
    for insert_stmt in insert_matches:
        # 清理INSERT语句
        insert_stmt = insert_stmt.strip()
        
        # 跳过sqlite_sequence表（MySQL不需要）
        if 'sqlite_sequence' in insert_stmt.lower():
            continue
        
        # 替换表名的引号格式
        insert_stmt = re.sub(r'INSERT INTO\s+`([^`]+)`', r'INSERT INTO `\1`', insert_stmt, flags=re.IGNORECASE)
        insert_stmt = re.sub(r'INSERT INTO\s+([a-zA-Z_][a-zA-Z0-9_]*)', r'INSERT INTO `\1`', insert_stmt, flags=re.IGNORECASE)
        
        # 处理布尔值
        insert_stmt = re.sub(r"'t'", '1', insert_stmt)
        insert_stmt = re.sub(r"'f'", '0', insert_stmt)
        insert_stmt = re.sub(r'\b1\b(?=\s*,)', '1', insert_stmt)  # true
        insert_stmt = re.sub(r'\b0\b(?=\s*,)', '0', insert_stmt)  # false
        
        # 处理时间戳格式 - 移除时区信息
        insert_stmt = re.sub(r"'(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\.\d+\+\d{2}:\d{2}'", r"'\1'", insert_stmt)
        
        # 移除换行符，保持为单行
        insert_stmt = ' '.join(insert_stmt.split())
        
        mysql_inserts.append(insert_stmt)
    
    # 添加MySQL特有的设置
    mysql_content = """-- MySQL数据迁移文件（仅数据，不包含表结构）
-- 表结构由GORM自动创建
SET FOREIGN_KEY_CHECKS = 0;
SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET AUTOCOMMIT = 0;
START TRANSACTION;

"""
    
    # 添加所有INSERT语句
    for insert_stmt in mysql_inserts:
        mysql_content += insert_stmt + '\n'
    
    mysql_content += """
SET FOREIGN_KEY_CHECKS = 1;
COMMIT;
"""
    
    # 写入MySQL文件
    with open(mysql_output_file, 'w', encoding='utf-8') as f:
        f.write(mysql_content)
    
    print(f"转换完成！MySQL数据文件已保存为: {mysql_output_file}")
    print(f"包含 {len(mysql_inserts)} 条INSERT语句")
    
    # 显示每个表的INSERT语句数量
    table_counts = {}
    for insert_stmt in mysql_inserts:
        match = re.search(r'INSERT INTO `([^`]+)`', insert_stmt, flags=re.IGNORECASE)
        if match:
            table_name = match.group(1)
            table_counts[table_name] = table_counts.get(table_name, 0) + 1
    
    print("\n📊 各表数据统计:")
    for table, count in table_counts.items():
        print(f"  {table}: {count} 条记录")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("使用方法: python3 migrate_to_mysql.py <sqlite_dump_file> <mysql_output_file>")
        print("示例: python3 migrate_to_mysql.py ggcode_backup.sql ggcode_mysql.sql")
        sys.exit(1)
    
    sqlite_file = sys.argv[1]
    mysql_file = sys.argv[2]
    
    try:
        convert_sqlite_to_mysql(sqlite_file, mysql_file)
    except Exception as e:
        print(f"转换失败: {e}")
        sys.exit(1) 