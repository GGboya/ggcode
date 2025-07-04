# 文件共享评测系统优化方案

## 概述

基于用户需求，将原有的HTTP包传输代码提交方式优化为基于文件共享的方式，通过预置测试数据文件和标准输入重定向实现更高效的代码评测。

## 架构对比

### 优化前（HTTP传输方式）
```
用户提交 -> HTTP JSON -> 临时文件 -> 编译执行 -> 清理 -> 返回结果
```

### 优化后（文件共享方式）
```
用户提交 -> 预置测试数据 -> 文件重定向 -> 编译执行 -> 返回结果
```

## 核心思想

1. **预置测试数据**：管理员预先将测试用例保存为文件
2. **标准输入重定向**：`./user_program < /judge/data/1000/1.in`
3. **基于现有架构**：优化现有面试岛服务，而非创建新服务

## 文件系统结构

```
/judge/data/
├── 1000/                      # 关卡ID 1000
│   ├── 1.in                   # 测试用例1输入
│   ├── 1.ans                  # 测试用例1期望输出
│   ├── 2.in                   # 测试用例2输入
│   ├── 2.ans                  # 测试用例2期望输出
│   └── config.yaml            # 配置文件（时空限制等）
├── 1001/                      # 关卡ID 1001
│   ├── 1.in
│   ├── 1.ans
│   └── config.yaml
└── ...
```

## 核心实现

### 1. TestDataService (测试数据管理服务)

**文件**: `internal/services/test_data_service.go`

**主要功能**:
- 管理测试数据文件的存储和组织
- 支持从数据库数据初始化文件
- 提供测试用例文件的CRUD操作
- 配置文件管理（时空限制等）

**关键方法**:
```go
func (tds *TestDataService) SaveTestCase(levelID uint, caseNum int, inputData, outputData string) error
func (tds *TestDataService) GetTestCaseData(levelID uint, caseNum int) (inputData, outputData string, err error)
func (tds *TestDataService) InitializeFromDatabase(levelID uint, testCases []struct{...}) error
func (tds *TestDataService) SaveConfig(levelID uint, config *TestDataConfig) error
```

### 2. 优化后的InterviewService

**文件**: `internal/services/interview_service.go`

**主要优化**:
- 添加文件重定向支持
- 新增 `runSingleTestWithFileRedirect` 方法
- 兼容原有数据库存储方式
- 优先使用文件路径，fallback到数据库内容

**核心方法**:
```go
func (s *interviewService) runSingleTestWithFileRedirect(sourceFile, executableFile string, config JudgeConfig, testCase models.InterviewTestCase) TestCaseResult
```

### 3. 数据模型扩展

**文件**: `internal/models/interview.go`

**新增字段**:
- `InterviewLevel.TestDataPath`: 测试数据目录路径
- `InterviewTestCase.InputFilePath`: 输入文件路径
- `InterviewTestCase.OutputFilePath`: 输出文件路径

## 优势分析

### 1. 性能优势
- **零HTTP传输开销**: 代码直接写入文件系统，无需JSON序列化
- **内存效率**: 避免了HTTP包在内存中的缓存
- **并发友好**: 文件系统操作天然支持并发

### 2. 可维护性优势
- **完整的执行环境**: 每个提交都有独立的工作空间
- **详细的日志记录**: 编译和运行日志分离保存
- **易于调试**: 可以直接查看生成的中间文件
- **可追溯性**: 所有执行过程都有文件记录

### 3. 扩展性优势
- **支持大文件**: 无HTTP包大小限制
- **多语言统一**: 统一的文件组织方式适用于所有语言
- **批量处理**: 可以批量处理多个测试用例

### 4. 安全性优势
- **隔离性**: 每个提交在独立目录中执行
- **权限控制**: 可以精确控制文件访问权限
- **资源管理**: 便于实现资源使用统计和限制

## 使用示例

### 管理员操作：初始化测试数据
```go
// 从数据库初始化测试数据到文件系统
testCases := []struct {
    Input    string
    Output   string
    IsSample bool
    Order    int
}{
    {Input: "3 5", Output: "8", IsSample: true, Order: 1},
    {Input: "10 20", Output: "30", IsSample: false, Order: 2},
}

err := testDataService.InitializeFromDatabase(1000, testCases)
```

### 用户操作：代码提交（无变化）
```bash
# 现有的API保持不变，用户体验无感知
curl -X POST http://localhost:8080/api/interview-island/level/1/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_token" \
  -d '{
    "code": "#include <iostream>\nusing namespace std;\nint main() {\n    int a, b;\n    cin >> a >> b;\n    cout << a + b << endl;\n    return 0;\n}",
    "language": "cpp"
  }'
```

### 系统执行：文件重定向
```bash
# 系统内部自动执行：
./user_program < /judge/data/1000/1.in > output.txt
# 然后比较 output.txt 和 /judge/data/1000/1.ans
```

## 语言支持

目前支持的编程语言和配置：

| 语言 | 扩展名 | 编译命令 | 执行命令 |
|------|--------|----------|----------|
| C++ | .cpp | `g++ -std=c++17 -O2 -o {executable} {source}` | `{executable}` |
| C | .c | `gcc -O2 -o {executable} {source}` | `{executable}` |
| Java | .java | `javac -encoding UTF-8 {source}` | `java -Xmx{memory}m -cp {workdir} Main` |
| Python | .py | - | `python3 {source}` |
| Go | .go | `go build -o {executable} {source}` | `{executable}` |
| JavaScript | .js | - | `node {source}` |

## 配置说明

### 工作空间配置
- **默认路径**: `/tmp/ggcode_workspace`
- **权限**: `0755` (目录), `0644` (文件)
- **清理策略**: 可配置自动清理时间

### 资源限制
- **默认时间限制**: 5秒 (编译型语言), 10秒 (解释型语言)
- **默认内存限制**: 128MB (通用), 256MB (Java)
- **编译超时**: 30秒

## 部署注意事项

### 1. 文件系统要求
- 确保工作空间目录有足够的磁盘空间
- 考虑使用SSD以提高I/O性能
- 定期清理过期的提交文件

### 2. 权限配置
- 确保应用有创建和删除文件的权限
- 考虑使用专用的用户账户运行服务
- 设置适当的文件系统权限

### 3. 监控和维护
- 监控磁盘使用情况
- 设置自动清理策略
- 记录评测统计信息

## 兼容性

- **向下兼容**: 原有的HTTP接口保持不变
- **渐进式迁移**: 可以同时支持两种评测方式
- **配置切换**: 通过配置文件选择评测方式

## 未来扩展

1. **分布式支持**: 支持多节点文件共享
2. **缓存优化**: 编译结果缓存机制
3. **实时监控**: 提交执行状态实时查看
4. **批量评测**: 支持多个测试用例并行执行
5. **沙箱增强**: 集成更强的安全沙箱机制

## 总结

文件共享评测系统通过消除HTTP传输开销，提供了更高效、更可维护的代码评测解决方案。该优化保持了系统的灵活性和扩展性，同时显著提升了性能和调试体验。 