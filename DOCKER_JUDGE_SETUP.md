# GGCode Docker 评测系统设置指南

## 概述

Docker 评测系统通过容器化技术提供安全、隔离的代码执行环境，支持 C++、Python 和 Java 等多种编程语言。

## 优势

✅ **完全隔离**: 每个代码提交在独立容器中运行  
✅ **环境一致**: 预配置好所有编译器和运行时  
✅ **高安全性**: 容器级别的沙箱隔离  
✅ **易于扩展**: 添加新语言只需更新镜像  
✅ **资源控制**: 精确限制 CPU、内存等资源  

## 前置要求

### Windows + WSL2 环境

1. **安装 Docker Desktop**
   - 下载并安装 [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop)
   - 确保系统开启了 Hyper-V 或 WSL2

2. **启用 WSL2 集成**
   - 打开 Docker Desktop
   - 进入 Settings > Resources > WSL Integration
   - 启用 "Enable integration with my default WSL distro"
   - 选择您的 Ubuntu WSL2 发行版
   - 点击 "Apply & Restart"

3. **验证 Docker 可用**
   ```bash
   docker --version
   docker run hello-world
   ```

### Linux 环境

1. **安装 Docker**
   ```bash
   # Ubuntu/Debian
   sudo apt update
   sudo apt install docker.io
   sudo systemctl start docker
   sudo systemctl enable docker
   sudo usermod -aG docker $USER
   
   # 重新登录以应用用户组更改
   ```

## 快速设置

### 自动设置（推荐）

```bash
# 给脚本添加执行权限
chmod +x scripts/setup-docker-judge.sh

# 运行设置脚本
./scripts/setup-docker-judge.sh
```

### 手动设置

1. **创建目录结构**
   ```bash
   mkdir -p judge-data judge-temp docker/judge
   ```

2. **构建评测镜像**
   ```bash
   docker build -t ggcode-judge:latest ./docker/judge
   ```

3. **验证镜像**
   ```bash
   docker run --rm ggcode-judge:latest /bin/bash -c "
       echo '=== 编译器版本 ===' &&
       g++ --version &&
       python3 --version &&
       java -version
   "
   ```

## 测试评测系统

### C++ 测试

```bash
# 创建测试文件
cat > judge-temp/Main.cpp << 'EOF'
#include <iostream>
using namespace std;
int main() {
    int a, b;
    cin >> a >> b;
    cout << a + b << endl;
    return 0;
}
EOF

echo "1 2" > judge-temp/input.txt
echo "3" > judge-temp/expected.txt

# 运行评测
docker run --rm \
    --network none \
    --memory 256m \
    --cpus 1 \
    --user judge \
    -v "$(pwd)/judge-temp:/opt/judge" \
    --workdir /opt/judge \
    ggcode-judge:latest \
    /opt/judge/judge-script.sh cpp 2 256 input.txt expected.txt
```

### Python 测试

```bash
# 创建测试文件
cat > judge-temp/Main.py << 'EOF'
a, b = map(int, input().split())
print(a + b)
EOF

# 运行评测
docker run --rm \
    --network none \
    --memory 256m \
    --cpus 1 \
    --user judge \
    -v "$(pwd)/judge-temp:/opt/judge" \
    --workdir /opt/judge \
    ggcode-judge:latest \
    /opt/judge/judge-script.sh python 5 256 input.txt expected.txt
```

### Java 测试

```bash
# 创建测试文件
cat > judge-temp/Main.java << 'EOF'
import java.util.Scanner;
public class Main {
    public static void main(String[] args) {
        Scanner sc = new Scanner(System.in);
        int a = sc.nextInt();
        int b = sc.nextInt();
        System.out.println(a + b);
        sc.close();
    }
}
EOF

# 运行评测
docker run --rm \
    --network none \
    --memory 512m \
    --cpus 1 \
    --user judge \
    -v "$(pwd)/judge-temp:/opt/judge" \
    --workdir /opt/judge \
    ggcode-judge:latest \
    /opt/judge/judge-script.sh java 3 512 input.txt expected.txt
```

## 集成到应用

### 修改 Hydro 评测服务

在现有的 Hydro 评测服务中集成 Docker 评测：

1. **检查 Docker 可用性**
   ```go
   dockerService := NewDockerJudgeService()
   if dockerService.IsDockerAvailable() {
       // 使用 Docker 评测
   } else {
       // 降级到原有评测方式
   }
   ```

2. **运行 Docker 评测**
   ```go
   req := &DockerJudgeRequest{
       Code:        submission.Code,
       Language:    submission.Language,
       Input:       testCase.Input,
       Expected:    testCase.Output,
       TimeLimit:   config.TimeLimit,
       MemoryLimit: config.MemoryLimit,
   }
   
   result, err := dockerService.RunJudge(req)
   ```

## 配置参数

### 资源限制

| 语言   | 默认时间限制 | 默认内存限制 | 推荐配置 |
|--------|--------------|--------------|----------|
| C++    | 2秒          | 256MB       | 适中     |
| Python | 5秒          | 256MB       | 较宽松   |
| Java   | 3秒          | 512MB       | 较宽松   |

### Docker 运行参数

```bash
docker run \
    --rm                    # 自动删除容器
    --network none          # 网络隔离
    --memory 256m           # 内存限制
    --cpus 1               # CPU限制
    --user judge           # 非root用户
    --read-only            # 只读文件系统（可选）
    --tmpfs /tmp:size=100M # 临时文件系统
    -v workdir:/opt/judge  # 挂载工作目录
```

## 故障排除

### 常见问题

1. **Docker 命令未找到**
   ```
   解决方案：确保 Docker Desktop 已安装并启用 WSL2 集成
   ```

2. **权限错误**
   ```bash
   # 将用户添加到 docker 组
   sudo usermod -aG docker $USER
   # 重新登录
   ```

3. **镜像构建失败**
   ```bash
   # 检查网络连接
   # 清理 Docker 缓存
   docker system prune -a
   ```

4. **容器运行失败**
   ```bash
   # 检查工作目录权限
   ls -la judge-temp/
   
   # 手动测试容器
   docker run -it --rm ggcode-judge:latest /bin/bash
   ```

### 调试模式

```bash
# 以交互模式运行容器进行调试
docker run -it --rm \
    -v "$(pwd)/judge-temp:/opt/judge" \
    --workdir /opt/judge \
    ggcode-judge:latest \
    /bin/bash

# 在容器内手动执行评测命令
./judge-script.sh cpp 2 256 input.txt expected.txt
```

## 性能优化

### 镜像优化

1. **多阶段构建**（高级）
2. **缓存优化**
3. **镜像压缩**

### 运行时优化

1. **容器重用**（开发中）
2. **并发控制**
3. **资源池管理**

## 安全考虑

✅ **网络隔离**: `--network none`  
✅ **用户权限**: 非 root 用户运行  
✅ **文件系统**: 限制读写权限  
✅ **资源限制**: CPU 和内存限制  
✅ **进程隔离**: 容器级别隔离  

## 扩展功能

### 添加新语言

1. 修改 `Dockerfile` 安装编译器
2. 更新 `judge-script.sh` 添加编译运行逻辑
3. 重新构建镜像

### 自定义判题

1. 实现特殊判题器（SPJ）
2. 支持交互题评测
3. 添加性能分析

## 监控和日志

### 容器监控

```bash
# 查看运行中的容器
docker ps

# 查看容器资源使用
docker stats

# 查看容器日志
docker logs <container_id>
```

### 评测统计

- 成功率统计
- 性能指标
- 错误日志分析

---

## 支持

如有问题，请：
1. 检查上述故障排除部分
2. 查看 Docker 和应用日志
3. 提交 Issue 或联系维护人员

祝您使用愉快！ 🚀 