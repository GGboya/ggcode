# �� GGCode 快速部署指南

## 前置要求

- Go 1.18 及以上
- MySQL 8.0 及以上（本地或远程均可）
- Git

## 1. 克隆项目

```bash
git clone https://github.com/GGboya/ggcode.git
cd ggcode
```

## 2. 配置数据库

1. 启动本地 MySQL 服务（或准备好远程 MySQL 实例）。
2. 创建数据库和用户：

```sql
CREATE DATABASE ggcode DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'ggcode'@'localhost' IDENTIFIED BY 'ggcode123';
GRANT ALL PRIVILEGES ON ggcode.* TO 'ggcode'@'localhost';
FLUSH PRIVILEGES;
```

3. 修改 `config.yaml` 或 `.env` 文件，确保数据库连接信息正确：

```
DB_HOST=localhost
DB_PORT=3306
DB_USER=ggcode
DB_PASSWORD=ggcode123
DB_NAME=ggcode
```

## 3. 启动后端服务

> **提示：如果没有本地开发环境，可以直接从 GitHub 的 [release](https://github.com/GGboya/ggcode/releases) 页面下载最新版本的二进制文件（目前仅支持 Linux），下载后解压运行，无需本地编译。**

```bash
go run main.go
```

首次启动会自动进行数据库迁移。

## 4. 访问应用

- 打开浏览器访问：[http://localhost:8080](http://localhost:8080)

## 5. 常见问题

### Q: 数据库连接失败？
- 检查 MySQL 是否已启动，端口和账号密码是否正确。
- 检查防火墙或本地端口占用。

### Q: 启动报错？
- 检查 Go 版本是否符合要求。
- 检查依赖是否已安装（可运行 `go mod tidy`）。

### Q: 如何重置数据库？
- 直接删除数据库后重新创建。

## 6. 生产环境建议

- 修改默认数据库密码和 JWT 密钥，避免使用弱密码。
- 推荐使用 Nginx/Apache 反向代理，启用 HTTPS。
- 配置防火墙，限制数据库端口访问。
- 定期备份数据库。

## 7. 参考

- 如需前端自定义开发，请参考 web 目录下静态资源和模板。
- 详细功能和接口文档见 docs 目录。

---

如有更多问题，欢迎提交 Issue 反馈！ 