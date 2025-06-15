# GGCode - 基于艾宾浩斯遗忘曲线的算法学习平台

## 项目介绍

GGCode 是一个智能算法学习平台，基于艾宾浩斯遗忘曲线理论，为用户提供科学的算法题目复习计划。现已升级支持MySQL数据库，提供更好的性能和可扩展性。

## 核心功能

### 🔐 用户系统
- 用户注册/登录
- JWT 认证机制

### 📚 题库管理
- 官方题库（LeetCode Hot 100，包含100道精选题目）
- 自定义题库创建
- 题目管理和查看

### 🧠 智能学习
- 基于艾宾浩斯遗忘曲线的复习算法
- 个性化学习计划
- 智能题目推荐
- 两种学习模式：
  - **独立AC**: 简化复习路径（3次复习）
  - **不会做**: 完整复习路径（7次复习）

### 📊 学习统计
- 学习进度追踪
- 完成度统计
- 复习安排可视化
- 自动打卡系统

## 艾宾浩斯遗忘曲线

### 完整复习路径（不会做的题目）
- 第1次复习：1天后
- 第2次复习：2天后
- 第3次复习：4天后
- 第4次复习：7天后
- 第5次复习：15天后
- 第6次复习：30天后
- 第7次复习：60天后

### 简化复习路径（独立AC的题目）
- 第1次复习：1天后
- 第2次复习：4天后
- 第3次复习：15天后

完成所有复习后，题目将被标记为"已掌握"。

## 技术栈

### 后端
- **Go** - 主要编程语言
- **Gin** - Web框架
- **GORM** - ORM框架
- **MySQL** - 数据库（生产环境推荐）
- **JWT** - 身份认证

### 前端
- **HTML/CSS/JavaScript** - 基础技术
- **Bootstrap 5** - UI框架
- **Axios** - HTTP客户端

## 快速开始

### 方法一：自动部署（推荐）

1. **确保MySQL已安装并运行**
   ```bash
   # Ubuntu/Debian
   sudo apt-get install mysql-server
   sudo systemctl start mysql
   
   # CentOS/RHEL
   sudo yum install mysql-server
   sudo systemctl start mysqld
   
   # macOS
   brew install mysql
   brew services start mysql
   ```

2. **运行自动部署脚本**
   ```bash
   ./deploy_mysql.sh
   ```

3. **启动应用**
   ```bash
   ./ggcode
   ```

4. **访问应用**
   打开浏览器访问：http://localhost:8080

### 方法二：手动配置

1. **安装依赖**
   ```bash
   go mod tidy
   ```

2. **创建MySQL数据库**
   ```bash
   mysql -u root -p < setup_mysql.sql
   ```

3. **配置环境变量**
   ```bash
   cp env.example .env
   # 编辑 .env 文件，设置数据库连接信息
   ```

4. **构建项目**
   ```bash
   go build -o ggcode .
   ```

5. **运行服务器**
   ```bash
   ./ggcode
   ```

## 环境配置

创建 `.env` 文件并配置以下参数：

```env
# 数据库配置
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=ggcode

# JWT密钥
JWT_SECRET=your-secret-key

# 服务器配置
SERVER_PORT=8080
```

## 数据迁移

如果你之前使用SQLite版本，可以使用提供的迁移工具：

```bash
# 1. 导出SQLite数据
sqlite3 ggcode.db .dump > ggcode_backup.sql

# 2. 转换为MySQL格式
python3 migrate_to_mysql.py ggcode_backup.sql ggcode_mysql.sql

# 3. 导入到MySQL
mysql -u root -p ggcode < ggcode_mysql.sql
```

## 项目结构

```
ggcode/
├── main.go                 # 程序入口
├── internal/
│   ├── database/          # 数据库相关
│   │   ├── database.go    # 数据库初始化
│   │   └── models.go      # 数据模型
│   ├── handlers/          # HTTP处理器
│   │   └── handlers.go
│   ├── middleware/        # 中间件
│   │   └── auth.go        # 认证中间件
│   ├── server/           # 服务器配置
│   │   └── server.go
│   └── services/         # 业务逻辑
│       └── ebbinghaus.go # 艾宾浩斯算法
├── web/                  # 前端文件
│   ├── static/          # 静态资源
│   └── templates/       # HTML模板
├── deploy_mysql.sh      # MySQL部署脚本
├── setup_mysql.sql      # MySQL初始化脚本
├── migrate_to_mysql.py  # 数据迁移工具
└── env.example         # 环境变量示例
```

## 数据库设计

### 主要表结构
- `users` - 用户信息
- `question_banks` - 题库信息
- `questions` - 题目信息（100道LeetCode Hot题目）
- `user_study_plans` - 用户学习计划
- `user_question_progresses` - 用户题目学习进度
- `user_check_ins` - 用户打卡记录

## 特性亮点

### 🎯 智能学习算法
- 科学的艾宾浩斯遗忘曲线实现
- 根据掌握程度调整复习频率
- 自动安排每日学习任务

### 📈 学习数据分析
- 详细的学习进度统计
- 连续打卡天数记录
- 学习效果可视化

### 🔄 灵活的学习模式
- 支持无限制继续学习
- 已掌握题目重新复习
- 个性化学习计划管理

### 🚀 生产级特性
- MySQL数据库支持
- 环境变量配置
- 容器化部署支持
- 完整的错误处理

## 开发说明

### 本地开发
```bash
# 启动开发模式（自动重载）
go run main.go

# 运行测试
go test ./...

# 代码格式化
go fmt ./...
```

### 生产部署
1. 使用环境变量配置敏感信息
2. 设置强密码和安全的JWT密钥
3. 配置MySQL主从复制（可选）
4. 使用反向代理（Nginx）
5. 启用HTTPS

## 贡献指南

欢迎提交Issue和Pull Request！

## 许可证

MIT License

## 更新日志

### v2.0.0 (最新)
- ✅ 升级到MySQL数据库
- ✅ 增加到100道LeetCode题目
- ✅ 添加环境变量配置支持
- ✅ 优化学习算法
- ✅ 完善部署脚本

### v1.0.0
- ✅ 基础功能实现
- ✅ SQLite数据库支持
- ✅ 艾宾浩斯遗忘曲线算法 