# 共享题库功能部署指南

## 功能概述

本次更新添加了以下功能：

1. **题库分类**：题库分为共享题库和个人题库
2. **Star功能**：用户可以收藏(Star)共享题库
3. **Fork功能**：用户可以Fork共享题库，创建自己的副本
4. **题库共享**：个人题库可以设置为共享，允许其他用户查看和Fork
5. **排序功能**：支持按Star数、Fork数、创建时间排序
6. **筛选功能**：支持按题库类型筛选（全部、共享、个人、已收藏）

## 部署步骤

### 1. 数据库迁移

**方案A：使用SQL脚本（推荐）**
```bash
# 执行数据库迁移SQL脚本
mysql -u root -p ggcode < shared_questionbank_migration.sql
```

**方案B：使用GORM自动迁移**
新的字段和表会在应用启动时自动创建（如果使用GORM AutoMigrate）

### 2. 部署应用代码

```bash
# 停止服务
sudo systemctl stop ggcode

# 备份当前版本
cp ggcode ggcode.backup.$(date +%Y%m%d_%H%M%S)

# 构建新版本
./build.sh

# 启动服务
sudo systemctl start ggcode

# 检查服务状态
sudo systemctl status ggcode
```

### 3. 验证功能

1. **访问题库页面**：http://your-domain/questionbanks
2. **测试筛选功能**：点击"全部题库"、"共享题库"、"我的题库"、"已收藏"
3. **测试排序功能**：使用排序下拉菜单
4. **测试个人题库操作**：
   - 创建个人题库
   - 设置题库为共享
   - 取消题库共享
5. **测试共享题库操作**：
   - 收藏(Star)共享题库
   - Fork共享题库
   - 取消收藏

## 数据库结构变更

### 新增字段（question_banks表）
- `is_shared`: 是否为共享题库
- `forked_from`: Fork来源题库ID
- `star_count`: Star数量
- `fork_count`: Fork数量

### 新增表
- `question_bank_stars`: 题库Star关系表

## API端点变更

### 新增API端点
- `GET /api/questionbanks?type=shared&sort=star_count` - 获取题库列表（支持筛选和排序）
- `POST /api/questionbanks/:id/share` - 设置题库为共享
- `DELETE /api/questionbanks/:id/share` - 取消题库共享
- `POST /api/questionbanks/:id/star` - 收藏题库
- `DELETE /api/questionbanks/:id/star` - 取消收藏
- `POST /api/questionbanks/:id/fork` - Fork题库
- `GET /api/starred-questionbanks` - 获取用户收藏的题库

### 更新的API端点
- `GET /api/questionbanks` - 现在支持type和sort查询参数

## 前端功能变更

### 新增UI组件
- 题库类型筛选按钮组（全部/共享/个人/已收藏）
- 排序下拉菜单（时间/Star数/Fork数）
- 题库操作按钮（共享/取消共享/收藏/Fork）
- Star和Fork数量显示
- Fork来源信息显示

### 新增JavaScript函数
- `shareQuestionBank()` - 共享题库
- `unshareQuestionBank()` - 取消共享
- `starQuestionBank()` - 收藏题库
- `unstarQuestionBank()` - 取消收藏
- `forkQuestionBank()` - Fork题库
- `loadStarredBanks()` - 加载收藏的题库

## 注意事项

1. **备份数据**：部署前请务必备份数据库
2. **权限检查**：确保MySQL用户有足够权限执行ALTER TABLE操作
3. **服务重启**：更新代码后需要重启服务
4. **用户体验**：新功能可能需要用户重新登录或清除浏览器缓存

## 故障排除

### 常见问题

1. **数据库迁移失败**
   ```bash
   # 检查MySQL错误日志
   sudo tail -f /var/log/mysql/error.log
   
   # 手动检查表结构
   mysql -u root -p -e "DESCRIBE ggcode.question_banks"
   ```

2. **服务启动失败**
   ```bash
   # 查看服务日志
   sudo journalctl -u ggcode -f
   
   # 检查配置文件
   cat /etc/systemd/system/ggcode.service
   ```

3. **前端功能异常**
   - 检查浏览器控制台错误
   - 清除浏览器缓存
   - 检查API响应

### 回滚方案

如果部署出现问题，可以执行以下回滚操作：

1. **回滚应用代码**
   ```bash
   sudo systemctl stop ggcode
   cp ggcode.backup.* ggcode
   sudo systemctl start ggcode
   ```

2. **回滚数据库**（如果需要）
   ```bash
   # 删除新添加的字段和表
   mysql -u root -p ggcode <<EOF
   ALTER TABLE question_banks 
   DROP COLUMN is_shared,
   DROP COLUMN forked_from,
   DROP COLUMN star_count,
   DROP COLUMN fork_count;
   
   DROP TABLE question_bank_stars;
   EOF
   ```

## 联系支持

如果部署过程中遇到问题，请检查：
1. 系统日志
2. 数据库连接
3. 权限设置
4. 网络配置 