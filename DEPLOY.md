# 🚀 GGCode 快速部署指南

## 一键启动（推荐）

### 前置要求
- Docker
- Docker Compose

### 启动步骤

1. **克隆项目**
   ```bash
   git clone <your-repo-url>
   cd ggcode
   ```

2. **一键启动**
   ```bash
   # 使用 docker-compose
   docker-compose up -d
   ```

3. **访问应用**
   - 打开浏览器访问：http://localhost:8080
   - 数据库会自动初始化（GORM 自动迁移）

### 常用命令

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f ggcode

# 停止服务  
docker-compose down

# 重启服务
docker-compose restart

# 完全清理（包括数据）
docker-compose down -v
```

---

## 🔧 配置说明

### 默认配置
- **应用端口**：8080
- **数据库**：MySQL 8.0
- **数据库端口**：3306
- **数据库用户**：ggcode
- **数据库密码**：ggcode123

### 自定义配置
如需修改配置，编辑 `docker-compose.yml` 文件中的环境变量：

```yaml
environment:
  DB_HOST: mysql
  DB_PORT: 3306
  DB_USER: ggcode
  DB_PASSWORD: your-password  # 修改密码
  DB_NAME: ggcode
  JWT_SECRET: your-secret-key  # 修改JWT密钥
  SERVER_PORT: 8080
```

---

## 📊 服务状态检查

```bash
# 检查容器状态
docker-compose ps

# 检查应用健康状态
curl http://localhost:8080

# 查看数据库连接
docker-compose exec mysql mysql -u ggcode -p ggcode
```

---

## 🐛 故障排除

### 端口被占用
```bash
# 修改 docker-compose.yml 中的端口映射
ports:
  - "8081:8080"  # 修改左侧端口号
```

### 数据库连接失败
```bash
# 查看MySQL容器日志
docker-compose logs mysql

# 重启MySQL容器
docker-compose restart mysql
```

### 重置数据
```bash
# 停止服务并删除数据卷
docker-compose down -v

# 重新启动
docker-compose up -d
```

---

## 💡 生产环境部署

1. **修改默认密码**
   ```yaml
   environment:
     MYSQL_ROOT_PASSWORD: your-secure-root-password
     MYSQL_PASSWORD: your-secure-password
     JWT_SECRET: your-production-jwt-secret
   ```

2. **使用外部MySQL**
   ```yaml
   ggcode:
     environment:
       DB_HOST: your-mysql-host
       DB_PORT: 3306
       DB_USER: your-username
       DB_PASSWORD: your-password
   ```

3. **反向代理**
   ```nginx
   # Nginx 配置示例
   location / {
       proxy_pass http://localhost:8080;
       proxy_set_header Host $host;
       proxy_set_header X-Real-IP $remote_addr;
   }
   ```

---

## ✨ 特性

- 🐳 **Docker 化部署**：开箱即用
- 🗄️ **自动数据库迁移**：GORM 自动创建表结构
- 🔄 **健康检查**：确保服务正常启动
- 📁 **数据持久化**：MySQL 数据自动备份
- �� **零配置**：默认配置即可运行 