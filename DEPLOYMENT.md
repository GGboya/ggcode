# GGCode MySQL 部署指南

## 🚀 快速部署

### 方法一：Docker Compose（推荐）

最简单的部署方式，一键启动MySQL和应用：

```bash
# 1. 克隆项目
git clone <your-repo-url>
cd ggcode

# 2. 启动服务
docker-compose up -d

# 3. 查看日志
docker-compose logs -f

# 4. 访问应用
# http://localhost:8080
```

### 方法二：自动脚本部署

适用于已有MySQL环境：

```bash
# 1. 确保MySQL已安装并运行
sudo systemctl start mysql  # Linux
brew services start mysql   # macOS

# 2. 运行部署脚本
./deploy_mysql.sh

# 3. 启动应用
./ggcode
```

### 方法三：手动部署

完全手动控制的部署方式：

```bash
# 1. 安装依赖
go mod tidy

# 2. 创建数据库
mysql -u root -p < setup_mysql.sql

# 3. 配置环境变量
cp env.example .env
# 编辑 .env 文件

# 4. 编译运行
go build -o ggcode .
./ggcode
```

## 📋 环境要求

### 系统要求
- **操作系统**: Linux, macOS, Windows
- **Go版本**: 1.19+
- **MySQL版本**: 5.7+ 或 8.0+
- **内存**: 最少512MB，推荐1GB+
- **磁盘**: 最少100MB

### 依赖软件
- Go 1.19+
- MySQL 5.7+/8.0+
- Git（用于克隆代码）

## ⚙️ 配置说明

### 环境变量配置

创建 `.env` 文件：

```env
# 数据库配置
DB_HOST=localhost          # 数据库主机
DB_PORT=3306              # 数据库端口
DB_USER=root              # 数据库用户名
DB_PASSWORD=your_password # 数据库密码
DB_NAME=ggcode            # 数据库名称

# 应用配置
JWT_SECRET=your-secret-key # JWT密钥（生产环境必须修改）
SERVER_PORT=8080          # 服务器端口

# 可选配置
GIN_MODE=release          # 生产模式
```

### MySQL配置建议

#### 生产环境配置
```sql
-- 创建专用用户
CREATE USER 'ggcode_user'@'localhost' IDENTIFIED BY 'strong_password';
GRANT ALL PRIVILEGES ON ggcode.* TO 'ggcode_user'@'localhost';
FLUSH PRIVILEGES;

-- 优化配置
SET GLOBAL innodb_buffer_pool_size = 128M;
SET GLOBAL max_connections = 200;
```

#### 性能优化
```ini
# my.cnf 配置建议
[mysqld]
innodb_buffer_pool_size = 256M
max_connections = 200
query_cache_size = 32M
innodb_log_file_size = 64M
```

## 🐳 Docker 部署详解

### Docker Compose 配置

```yaml
version: '3.8'
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ggcode123
      MYSQL_DATABASE: ggcode
    volumes:
      - mysql_data:/var/lib/mysql
    
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      DB_HOST: mysql
      DB_PASSWORD: ggcode123
    depends_on:
      - mysql
```

### 常用Docker命令

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f app
docker-compose logs -f mysql

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 完全清理（包括数据）
docker-compose down -v
```

## 🔧 故障排除

### 常见问题

#### 1. MySQL连接失败
```bash
# 检查MySQL服务状态
systemctl status mysql

# 检查端口占用
netstat -tlnp | grep 3306

# 测试连接
mysql -u root -p -h localhost
```

#### 2. 端口被占用
```bash
# 查看端口占用
lsof -i :8080

# 修改端口
export SERVER_PORT=8081
```

#### 3. 权限问题
```bash
# 给脚本执行权限
chmod +x deploy_mysql.sh

# 检查文件权限
ls -la ggcode
```

#### 4. 数据库初始化失败
```sql
-- 手动创建数据库
CREATE DATABASE ggcode CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE ggcode;

-- 检查表是否创建
SHOW TABLES;
```

### 日志调试

```bash
# 查看应用日志
./ggcode 2>&1 | tee ggcode.log

# MySQL错误日志
tail -f /var/log/mysql/error.log

# Docker日志
docker-compose logs -f --tail=100
```

## 🚀 生产环境部署

### 安全配置

1. **修改默认密码**
   ```bash
   # 生成强密码
   openssl rand -base64 32
   ```

2. **配置防火墙**
   ```bash
   # 只允许必要端口
   ufw allow 8080
   ufw allow 3306  # 仅内网访问
   ```

3. **使用HTTPS**
   ```bash
   # 使用Nginx反向代理
   server {
       listen 443 ssl;
       server_name your-domain.com;
       
       location / {
           proxy_pass http://localhost:8080;
       }
   }
   ```

### 性能优化

1. **数据库优化**
   - 配置合适的缓冲池大小
   - 启用查询缓存
   - 定期优化表

2. **应用优化**
   - 设置 `GIN_MODE=release`
   - 配置适当的连接池
   - 启用gzip压缩

3. **监控配置**
   ```bash
   # 系统监控
   htop
   iostat -x 1
   
   # MySQL监控
   mysqladmin processlist
   mysqladmin status
   ```

## 📊 数据备份

### 自动备份脚本

```bash
#!/bin/bash
# backup.sh
DATE=$(date +%Y%m%d_%H%M%S)
mysqldump -u root -p ggcode > backup_${DATE}.sql
```

### 恢复数据

```bash
# 从备份恢复
mysql -u root -p ggcode < backup_20231215_120000.sql
```

## 🔄 更新升级

### 应用更新

```bash
# 1. 备份数据
mysqldump -u root -p ggcode > backup_before_update.sql

# 2. 停止服务
docker-compose down  # Docker方式
# 或
pkill ggcode         # 直接运行方式

# 3. 更新代码
git pull origin main

# 4. 重新构建
go build -o ggcode .

# 5. 启动服务
docker-compose up -d  # Docker方式
# 或
./ggcode             # 直接运行方式
```

## 📞 技术支持

如果遇到问题，请：

1. 查看本文档的故障排除部分
2. 检查应用和数据库日志
3. 在GitHub提交Issue
4. 联系技术支持

---

**祝您部署顺利！🎉** 