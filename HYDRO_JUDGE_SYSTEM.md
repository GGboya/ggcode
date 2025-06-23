# GGCode Hydro 评测系统

## 概述

GGCode Hydro 评测系统是一个模拟 [Hydro](https://hydro.js.org/) 评测机制的高级在线评测系统，提供了完整的代码提交、沙箱执行、结果校验和报告生成功能。

## 核心特性

### 1. 提交解析
- **代码验证**: 检查代码长度、语言支持、危险代码模式
- **语言支持**: C++17、Python3、Java17
- **任务队列**: 支持优先级队列管理，确保公平评测

### 2. 沙箱执行
- **安全隔离**: 每个提交在独立的临时目录中执行
- **资源限制**: 
  - 时间限制：可配置（默认C++: 2s, Python: 5s, Java: 3s）
  - 内存限制：可配置（默认C++: 256MB, Python: 256MB, Java: 512MB）
  - 进程限制：防止fork炸弹攻击
- **环境控制**: 严格的环境变量和路径控制

### 3. 结果校验
- **精确匹配**: 支持逐字符精确匹配
- **智能比较**: 忽略行尾空格的比较模式
- **扩展支持**: 可配置自定义校验器（未来支持）

### 4. 详细报告
- **测试点详情**: 每个测试点的状态、时间、内存使用
- **子任务结果**: 支持子任务分组和依赖关系
- **系统信息**: 包含编译器版本、系统负载等信息
- **性能统计**: 完整的评测时间和资源使用统计

## API 接口

### 基础路径
所有 Hydro 评测接口都在 `/api/hydro-judge` 路径下。

### 接口列表

#### 1. 测试代码
```http
POST /api/hydro-judge/level/{levelId}/test
Content-Type: application/json
Authorization: Bearer <token>

{
    "code": "代码内容",
    "language": "cpp|python|java"
}
```

#### 2. 提交代码
```http
POST /api/hydro-judge/level/{levelId}/submit
Content-Type: application/json
Authorization: Bearer <token>

{
    "code": "代码内容",
    "language": "cpp|python|java", 
    "submit_time": 300,  // 提交时间（秒）
    "priority": 1        // 优先级（可选）
}
```

#### 3. 获取评测结果
```http
GET /api/hydro-judge/result/{submissionId}
Authorization: Bearer <token>
```

#### 4. 查看队列状态
```http
GET /api/hydro-judge/queue-status
Authorization: Bearer <token>
```

#### 5. 获取系统信息
```http
GET /api/hydro-judge/system-info
Authorization: Bearer <token>
```

## 评测流程

### 1. 提交解析阶段
```
用户提交代码
    ↓
验证语言支持
    ↓
检查代码安全性
    ↓
创建评测任务
    ↓
加入评测队列
```

### 2. 沙箱执行阶段
```
从队列获取任务
    ↓
创建隔离环境
    ↓
编译代码（如需要）
    ↓
运行测试用例
    ↓
收集运行结果
```

### 3. 结果校验阶段
```
比较输出结果
    ↓
计算测试点分数
    ↓
确定总体状态
    ↓
生成子任务结果
```

### 4. 报告生成阶段
```
汇总所有结果
    ↓
生成详细报告
    ↓
更新系统统计
    ↓
返回最终结果
```

## 评测状态

| 状态码 | 含义 | 说明 |
|--------|------|------|
| PENDING | 等待评测 | 任务已提交，等待处理 |
| JUDGING | 正在评测 | 正在执行评测 |
| AC | 通过 | 所有测试点通过 |
| WA | 答案错误 | 输出与期望不符 |
| TLE | 时间超限 | 超过时间限制 |
| MLE | 内存超限 | 超过内存限制 |
| RE | 运行时错误 | 程序异常退出 |
| CE | 编译错误 | 编译失败 |
| SE | 系统错误 | 评测系统错误 |
| PC | 部分正确 | 部分测试点通过 |

## 语言配置

### C++17
```yaml
编译命令: g++ -std=c++17 -O2 -Wall -Wextra -static -DONLINE_JUDGE -o {executable} {source}
运行命令: {executable}
时间限制: 2秒
内存限制: 256MB
编译时限: 10秒
```

### Python3
```yaml
编译命令: 无（解释型语言）
运行命令: python3 {source}
时间限制: 5秒
内存限制: 256MB
编译时限: 0秒
```

### Java17
```yaml
编译命令: javac -cp /opt/java-lib -encoding UTF-8 {source}
运行命令: java -cp . -Xmx{memory_limit}m -Dfile.encoding=UTF-8 {classname}
时间限制: 3秒
内存限制: 512MB
编译时限: 30秒
```

## 安全机制

### 代码安全检查
- **C++**: 禁止 `system()`, `exec*()`, `fork()` 等危险函数
- **Python**: 禁止 `import os`, `subprocess`, `eval()` 等危险操作
- **Java**: 禁止 `Runtime.exec()`, `ProcessBuilder` 等系统调用

### 沙箱限制
- **文件系统**: 只能访问指定的读写路径
- **网络访问**: 默认禁用网络连接
- **进程数量**: 限制同时运行的进程数
- **输出大小**: 限制程序输出大小

## 性能特性

### 并发处理
- **多工作者**: 支持多个并发评测工作者
- **队列管理**: 基于优先级的任务队列
- **负载均衡**: 自动分配任务到空闲工作者

### 资源监控
- **实时统计**: 实时队列状态和系统负载
- **性能分析**: 详细的评测时间分析
- **资源使用**: 准确的时间和内存统计

## 扩展性

### 自定义校验器
系统预留了自定义校验器接口，支持：
- Special Judge (SPJ)
- 交互题评测
- 多组数据比较
- 近似值比较

### 子任务支持
- **任务分组**: 将测试点分组为子任务
- **依赖关系**: 支持子任务间的依赖
- **部分分**: 支持子任务部分得分

## 部署配置

### 环境要求
- Go 1.19+
- GCC 9.4+ (C++支持)
- Python 3.9+ 
- OpenJDK 11+ (Java支持)
- Linux 操作系统（推荐Ubuntu 20.04+）

### 配置文件
评测系统支持通过环境变量配置：
```bash
# 工作者数量
HYDRO_WORKERS=4

# 队列大小
HYDRO_QUEUE_SIZE=100

# 沙箱路径
HYDRO_SANDBOX_PATH=/opt/ggcode/sandbox

# 校验器路径
HYDRO_CHECKER_PATH=/opt/ggcode/checker
```

## 监控和调试

### 系统监控
- 队列状态实时查看
- 工作者负载监控
- 评测性能统计

### 调试信息
- 详细的编译信息
- 运行时错误捕获
- 系统调用跟踪

## 与原版评测的区别

| 特性 | 原版评测 | Hydro评测 |
|------|----------|-----------|
| 并发处理 | 单线程 | 多工作者并发 |
| 队列管理 | 无队列 | 优先级队列 |
| 资源监控 | 基础 | 详细监控 |
| 安全性 | 基础检查 | 严格沙箱 |
| 扩展性 | 有限 | 高度可扩展 |
| 报告详情 | 简单 | 详细完整 |

## 使用示例

### 前端集成
```javascript
// 提交代码评测
async function submitCode(levelId, code, language) {
    const response = await fetch(`/api/hydro-judge/level/${levelId}/submit`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
            code: code,
            language: language,
            submit_time: Math.floor(Date.now() / 1000)
        })
    });
    
    const result = await response.json();
    return result;
}

// 轮询评测结果
async function pollResult(submissionId) {
    const response = await fetch(`/api/hydro-judge/result/${submissionId}`, {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    });
    
    const result = await response.json();
    return result.data;
}
```

## 未来计划

- [ ] 支持更多编程语言
- [ ] 实现真正的go-sandbox集成
- [ ] 添加交互题支持
- [ ] 实现分布式评测
- [ ] 添加评测数据分析
- [ ] 支持用户自定义测试数据

## 开发指南

### 添加新语言支持
1. 在 `getHydroLanguageConfigs()` 中添加语言配置
2. 在 `containsDangerousCode()` 中添加安全检查规则
3. 测试编译和运行命令
4. 更新文档

### 自定义校验器
1. 实现 `Checker` 接口
2. 在 `checkAnswer()` 方法中集成
3. 配置校验器路径
4. 测试校验逻辑

---

## 联系和支持

如有问题或建议，请提交 Issue 或 Pull Request。 