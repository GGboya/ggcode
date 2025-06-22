# GGCode HTTPS 部署配置指南

本文档记录了为GGCode智能算法学习平台配置HTTPS的完整过程。

## 📋 配置概述

- **域名**: xxx.com
- **服务器**: 腾讯云 Ubuntu 服务器 (xxx.xxx.xxx.xxx)
- **SSL证书**: Let's Encrypt 免费证书
- **Web服务器**: Nginx (反向代理)
- **应用服务**: Go应用 (端口8080)

## 🚀 部署步骤

### 第一步：环境准备

1. **连接服务器**
   ```bash
   ssh ubuntu@xxx.xxx.xxx.xxx
   ```

2. **检查域名解析**
   ```bash
   nslookup xxx.com
   # 确认解析到服务器IP: xxx.xxx.xxx.xxx
   ```

3. **更新系统**
   ```bash
   sudo apt update && sudo apt upgrade -y
   ```

### 第二步：安装必需软件

```bash
# 安装Nginx、Certbot和相关工具
sudo apt install -y nginx certbot python3-certbot-nginx ufw curl

# 启动并启用Nginx
sudo systemctl start nginx
sudo systemctl enable nginx
```

### 第三步：配置防火墙

```bash
# 配置防火墙规则
sudo ufw allow ssh
sudo ufw allow 'Nginx Full'
sudo ufw --force enable

# 检查防火墙状态
sudo ufw status
```

### 第四步：创建Nginx配置

1. **创建站点配置文件**
   ```bash
   sudo nano /etc/nginx/sites-available/ggcode
   ```

2. **添加临时HTTP配置**（用于申请SSL证书）
   ```nginx
   # 临时HTTP配置，用于申请SSL证书
   server {
       listen 80;
       server_name www.xxx.com, xxx.com;
       
       # Let's Encrypt验证路径
       location /.well-known/acme-challenge/ {
           root /var/www/html;
       }
       
       # 其他请求暂时返回200
       location / {
           return 200 'Server is running, configuring SSL...';
           add_header Content-Type text/plain;
       }
   }
   ```

3. **启用站点配置**
   ```bash
   # 启用站点
   sudo ln -s /etc/nginx/sites-available/ggcode /etc/nginx/sites-enabled/
   
   # 删除默认站点
   sudo rm -f /etc/nginx/sites-enabled/default
   
   # 测试配置
   sudo nginx -t
   
   # 重新加载Nginx
   sudo systemctl reload nginx
   ```

### 第五步：申请SSL证书

```bash
# 使用Let's Encrypt申请免费SSL证书
sudo certbot --nginx -d www.xxx.com -d xxx.com
```

**申请过程中的交互**：
- 输入邮箱地址（用于证书到期通知）
- 同意服务条款（输入 Y）
- 是否分享邮箱给EFF（可选择 N）

### 第六步：配置完整的HTTPS

1. **更新Nginx配置为完整HTTPS配置**
   ```bash
   sudo vim /etc/nginx/sites-available/ggcode
   ```

2. **使用以下完整配置**
   ```nginx
   # HTTP重定向到HTTPS
   server {
       listen 80;
       server_name www.xxx.com, xxx.com;
       return 301 https://$server_name$request_uri;
   }
   
   # HTTPS配置
   server {
       listen 443 ssl http2;
       server_name www.xxx.com xxx.com;
   
       # SSL证书配置
       ssl_certificate /etc/letsencrypt/live/www.xxx.com/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/www.xxx.com/privkey.pem;
       
       # SSL安全配置
       ssl_protocols TLSv1.2 TLSv1.3;
       ssl_prefer_server_ciphers on;
       ssl_ciphers 'ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256';
       
       # HSTS
       add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload";
       
       # 安全头
       add_header X-Frame-Options DENY;
       add_header X-Content-Type-Options nosniff;
       add_header X-XSS-Protection "1; mode=block";
       
       # Gzip压缩
       gzip on;
       gzip_vary on;
       gzip_min_length 1024;
       gzip_proxied any;
       gzip_types text/plain text/css text/xml text/javascript application/javascript application/json;
   
       # 静态文件
       location /static/ {
           proxy_pass http://127.0.0.1:8080;
           expires 1y;
           add_header Cache-Control "public, immutable";
       }
   
       # 反向代理到Go应用
       location / {
           proxy_pass http://127.0.0.1:8080;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
           proxy_cache_bypass $http_upgrade;
       }
   
       # 日志
       access_log /var/log/nginx/ggcode_access.log;
       error_log /var/log/nginx/ggcode_error.log;
   }
   ```

3. **重新加载配置**
   ```bash
   sudo nginx -t
   sudo systemctl reload nginx
   ```

### 第七步：配置Go应用服务

1. **创建systemd服务文件**
   ```bash
   sudo vim /etc/systemd/system/ggcode.service
   ```

2. **添加服务配置**
   ```ini
   [Unit]
   Description=GGCode - 智能算法学习平台
   After=network.target
   
   [Service]
   Type=simple
   User=ubuntu
   Group=ubuntu
   WorkingDirectory=/home/ubuntu/ggcode
   ExecStart=/home/ubuntu/ggcode/ggcode
   Restart=on-failure
   RestartSec=5
   
   Environment=GIN_MODE=release
   Environment=PORT=8080
   
   [Install]
   WantedBy=multi-user.target
   ```

3. **启动服务**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable ggcode
   sudo systemctl start ggcode
   ```


## ✅ 验证配置

### 1. 检查服务状态
```bash
# 检查Nginx状态
sudo systemctl status nginx

# 检查Go应用状态
sudo systemctl status ggcode

# 检查端口监听
sudo netstat -tlnp | grep -E ':80|:443|:8080'
```

### 2. 测试HTTPS访问
```bash
# 测试HTTP重定向
curl -I http://www.xxx.com

# 测试HTTPS访问
curl -I https://www.xxx.com

# 检查SSL证书
openssl s_client -connect www.xxx.com:443 -servername www.xxx.com
```


## 🔧 常用维护命令

### 查看日志
```bash
# Nginx访问日志
sudo tail -f /var/log/nginx/ggcode_access.log

# Nginx错误日志
sudo tail -f /var/log/nginx/ggcode_error.log

# Go应用日志
sudo journalctl -u ggcode -f
```

### 重启服务
```bash
# 重启Nginx
sudo systemctl restart nginx

# 重启Go应用
sudo systemctl restart ggcode

# 重新加载Nginx配置（无需重启）
sudo systemctl reload nginx
```

### 证书管理
```bash
# 查看证书信息
sudo certbot certificates

# 手动续期证书
sudo certbot renew

# 测试续期
sudo certbot renew --dry-run
```

## 🚨 故障排除

### 常见问题

1. **Nginx配置错误**
   ```bash
   # 测试配置语法
   sudo nginx -t
   
   # 查看详细错误
   sudo nginx -t -c /etc/nginx/nginx.conf
   ```

2. **SSL证书申请失败**
   - 检查域名解析是否正确
   - 确认80端口未被其他服务占用
   - 查看详细错误日志

3. **Go应用无法访问**
   ```bash
   # 检查应用是否运行
   sudo systemctl status ggcode
   
   # 检查端口监听
   sudo netstat -tlnp | grep :8080
   
   # 查看应用日志
   sudo journalctl -u ggcode -n 50
   ```

## 📈 性能优化建议

1. **启用HTTP/2**（已配置）
2. **配置Gzip压缩**（已配置）
3. **设置静态文件缓存**（已配置）
4. **启用HSTS**（已配置）

## 🔒 安全最佳实践

1. **定期更新系统和软件包**
2. **监控SSL证书到期时间**
3. **定期检查安全头配置**
4. **启用访问日志监控**
5. **配置防火墙规则**

## 📞 技术支持

如遇到问题，请检查：
1. 系统日志：`sudo journalctl -xe`
2. Nginx日志：`sudo tail -f /var/log/nginx/error.log`
3. 应用日志：`sudo journalctl -u ggcode -f`
