# GGCode - 智能算法学习平台

<div align="center">


## 📈 GitHub Star History

<div align="center">

![GitHub Star History Chart](https://api.star-history.com/svg?repos=GGboya/ggcode&type=Date)
## 📈 最近开发活动

[![GitHub Activity](https://github-readme-activity-graph.vercel.app/graph?username=GGboya&repo=ggcode&theme=radical&hide_border=true&area=true&custom_title=GGCode%20开发活动)](https://github.com/GGboya/ggcode)

</div>


</div>


## 🎯 项目简介

GGCode 是基于艾宾浩斯遗忘曲线理论的智能算法学习平台，帮助程序员科学高效地掌握算法题目。平台集成了智能学习算法、在线评测系统、社区题库分享等核心功能。

## ✨ 核心功能

### 🔐 用户系统
- **JWT身份认证**：安全的用户注册/登录系统
- **用户权限管理**：基于角色的访问控制
- **会话管理**：智能Token过期处理

### 📚 题库管理
- **官方题库**：LeetCode Hot 100 精选题目
- **个人题库**：创建、编辑、删除自定义题库
- **共享题库**：Star、Fork 优质题库，社区共享
- **竞赛题目导入**：支持从竞赛中批量导入题目，支持分数范围选择

### 🧠 智能学习算法
- **艾宾浩斯遗忘曲线**：基于科学记忆理论的学习算法
- **两种学习模式**：
  - **独立AC模式**：第一次遇到题目就独立AC，后续不再复习
  - **不会做模式**：7次复习（1天→2天→4天→7天→15天→30天→60天）
- **个性化推荐**：每日智能推送学习任务
- **学习断点续传**：自动保存进度，支持学习计划缓存

### 📊 学习统计与可视化
- **学习热力图**：一年学习活跃度可视化展示
- **学习进度追踪**：掌握率统计和进度可视化
- **自动打卡系统**：每日学习打卡，连续学习天数追踪
- **错题本系统**：自动收集错题，独立错题库管理

### ⚡ 在线评测系统
- **多语言支持**：C++、Java、Python、Go
- **高性能引擎**：基于 go-judge 的实时评测引擎
- **智能检测**：时间超限、内存超限、答案错误等状态判定
- **灵活配置**：支持自定义测试用例和运行参数

### 🏝️ 面试岛系统
- **关卡创建**：支持自定义关卡和测试用例
- **萌系视效**：丰富的视觉效果和动画
- **智能评分**：难度和时间加权的智能题目评分

## 🚀 技术栈

### 后端技术
- **语言框架**：Go + Gin + GORM
- **数据库**：MySQL
- **认证**：JWT Token
- **日志系统**：Logrus 结构化日志
- **配置管理**：统一配置系统，支持环境变量覆盖
- **接口文档**：Swagger 自动生成 API 文档

### 前端技术
- **基础技术**：HTML/CSS/JavaScript + Bootstrap 5
- **用户体验**：无感操作，动画过渡
- **PWA支持**：渐进式Web应用
- **响应式设计**：适配多设备

### 部署与运维
- **服务化**：Systemd 服务管理
- **安全加固**：HTTPS、HSTS、CSP、UFW防火墙
- **性能优化**：Nginx HTTP/2、Gzip压缩、静态缓存

## 📖 文档

- **[🚀 快速部署指南](DEPLOY.md)** - Docker 一键启动部署
- **[📋 产品更新日志](CHANGELOG.md)** - 查看最新功能更新和问题修复

## 🌟 特色亮点

- **科学学习算法**：基于艾宾浩斯遗忘曲线的智能学习系统
- **现代化用户体验**：无感操作，动画过渡，响应式设计
- **社区驱动**：题库分享机制，Star/Fork功能
- **完整数据统计**：学习热力图，进度追踪，错题管理
- **高性能评测**：基于go-judge的多语言在线评测系统
- **安全可靠**：HTTPS全站加密，多层安全防护 

## 🛠️ 快速开始

请参考 [快速部署指南](DEPLOY.md) 进行本地环境部署和启动。

## 📂 目录结构

```
ggcode/
├── internal/           # 核心后端代码
│   ├── controllers/    # 路由与控制器
│   ├── services/       # 业务逻辑
│   ├── repositories/   # 数据访问层
│   ├── models/         # 数据模型
│   └── ...             
├── web/                # 前端静态资源与模板
├── docs/               # API 文档
├── config.yaml         # 配置文件
└── ...
```

## 🤝 贡献指南

欢迎任何形式的贡献！你可以：
- 提交 Issue 报告 bug 或建议新功能
- 提交 Pull Request 修复问题或优化功能
- 优化文档或翻译

贡献流程：
1. Fork 本仓库
2. 新建分支 (`git checkout -b feature/xxx`)
3. 提交更改 (`git commit -am 'feat: xxx'`)
4. 推送分支 (`git push origin feature/xxx`)
5. 创建 Pull Request

请遵循 [代码规范](#代码规范) 和 [行为准则](CODE_OF_CONDUCT.md)。

## 📜 行为准则

请阅读 [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md) 以了解参与本项目的行为规范。

## 📝 License

本项目基于 MIT License 开源。详见 [LICENSE](./LICENSE)。

## ❓ 常见问题

- Q: 如何导入自定义题库？
- Q: 评测支持哪些语言？
- Q: 如何贡献新功能？
- Q: 启动报错怎么办？


## 📛 项目徽章

[![MIT License](https://img.shields.io/github/license/GGboya/ggcode)](./LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/GGboya/ggcode)](https://goreportcard.com/report/github.com/GGboya/ggcode) 
