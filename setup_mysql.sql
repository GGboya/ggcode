-- 创建数据库
CREATE DATABASE IF NOT EXISTS ggcode CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户（可选，如果需要专门的数据库用户）
-- CREATE USER IF NOT EXISTS 'ggcode_user'@'localhost' IDENTIFIED BY 'your_password';
-- GRANT ALL PRIVILEGES ON ggcode.* TO 'ggcode_user'@'localhost';
-- FLUSH PRIVILEGES;

-- 使用数据库
USE ggcode; 