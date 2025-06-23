# Docker容器池评测系统设计

## 概述

基于Docker容器池的高性能评测系统，通过预创建容器和智能分配策略，显著减少容器启动/销毁开销，适合高并发评测场景。

## 系统架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   评测任务队列   │    │   容器池管理器    │    │   Docker容器池   │
│                │    │                 │    │                │
│ ┌─────────────┐ │    │ ┌─────────────┐  │    │ ┌─────────────┐ │
│ │   任务1     │ │───▶│ │  分配策略   │  │───▶│ │ Container1  │ │
│ └─────────────┘ │    │ └─────────────┘  │    │ │   (idle)    │ │
│ ┌─────────────┐ │    │ ┌─────────────┐  │    │ └─────────────┘ │
│ │   任务2     │ │    │ │  状态监控   │  │    │ ┌─────────────┐ │
│ └─────────────┘ │    │ └─────────────┘  │    │ │ Container2  │ │
│ ┌─────────────┐ │    │ ┌─────────────┐  │    │ │   (busy)    │ │
│ │   任务3     │ │    │ │  容器清理   │  │    │ └─────────────┘ │
│ └─────────────┘ │    │ └─────────────┘  │    │ ┌─────────────┐ │
└─────────────────┘    └──────────────────┘    │ │ Container3  │ │
                                              │ │   (idle)    │ │
                                              │ └─────────────┘ │
                                              └─────────────────┘
```

## 核心组件设计

### 1. 容器池管理器 (ContainerPool)

```go
type ContainerStatus string

const (
    StatusIdle    ContainerStatus = "idle"
    StatusBusy    ContainerStatus = "busy"
    StatusError   ContainerStatus = "error"
    StatusStarting ContainerStatus = "starting"
)

type Container struct {
    ID       string          `json:"id"`
    Name     string          `json:"name"`
    Status   ContainerStatus `json:"status"`
    LastUsed time.Time       `json:"last_used"`
    TaskID   string          `json:"task_id,omitempty"`
    Language string          `json:"language"`
}

type ContainerPool struct {
    containers    map[string]*Container
    idleQueue     chan string
    taskQueue     chan *JudgeTask
    mutex         sync.RWMutex
    client        *docker.Client
    maxSize       int
    minSize       int
    languages     []string
}
```

### 2. 任务调度器

```go
type JudgeTask struct {
    ID          string    `json:"id"`
    Language    string    `json:"language"`
    Code        string    `json:"code"`
    TestCases   []TestCase `json:"test_cases"`
    Priority    int       `json:"priority"`
    CreatedAt   time.Time `json:"created_at"`
    ResultChan  chan *JudgeResult `json:"-"`
}

type TaskScheduler struct {
    pool          *ContainerPool
    priorityQueue *PriorityQueue
    workers       int
    stopChan      chan struct{}
}
```

### 3. 容器池初始化

```go
func NewContainerPool(config *PoolConfig) (*ContainerPool, error) {
    pool := &ContainerPool{
        containers: make(map[string]*Container),
        idleQueue:  make(chan string, config.MaxSize),
        taskQueue:  make(chan *JudgeTask, config.QueueSize),
        maxSize:    config.MaxSize,
        minSize:    config.MinSize,
        languages:  config.Languages,
    }
    
    // 初始化Docker客户端
    client, err := docker.NewClientFromEnv()
    if err != nil {
        return nil, err
    }
    pool.client = client
    
    // 预创建最小数量的容器
    if err := pool.warmUp(); err != nil {
        return nil, err
    }
    
    // 启动监控和清理goroutine
    go pool.monitor()
    go pool.cleaner()
    
    return pool, nil
}
```

### 4. 容器预创建和热身

```go
func (p *ContainerPool) warmUp() error {
    for i := 0; i < p.minSize; i++ {
        for _, lang := range p.languages {
            container, err := p.createContainer(lang)
            if err != nil {
                return fmt.Errorf("failed to create container for %s: %v", lang, err)
            }
            
            p.containers[container.ID] = container
            p.idleQueue <- container.ID
        }
    }
    return nil
}

func (p *ContainerPool) createContainer(language string) (*Container, error) {
    // 根据语言选择镜像
    image := p.getImageForLanguage(language)
    
    // 创建容器配置
    config := &container.Config{
        Image: image,
        Cmd:   []string{"sleep", "3600"}, // 保持容器运行
        WorkingDir: "/tmp/judge",
        Env: []string{
            "JUDGE_LANGUAGE=" + language,
            "DEBIAN_FRONTEND=noninteractive",
        },
    }
    
    hostConfig := &container.HostConfig{
        Memory:     256 * 1024 * 1024, // 256MB
        CPUQuota:   50000,              // 50% CPU
        NetworkMode: "none",            // 禁用网络
        ReadonlyRootfs: false,
        SecurityOpt: []string{"no-new-privileges"},
        Ulimits: []*units.Ulimit{
            {Name: "nproc", Soft: 64, Hard: 64},
            {Name: "nofile", Soft: 1024, Hard: 1024},
        },
    }
    
    // 创建容器
    resp, err := p.client.ContainerCreate(
        context.Background(),
        config,
        hostConfig,
        nil,
        nil,
        "",
    )
    if err != nil {
        return nil, err
    }
    
    // 启动容器
    if err := p.client.ContainerStart(
        context.Background(),
        resp.ID,
        types.ContainerStartOptions{},
    ); err != nil {
        return nil, err
    }
    
    return &Container{
        ID:       resp.ID,
        Name:     fmt.Sprintf("judge_worker_%s_%d", language, time.Now().Unix()),
        Status:   StatusIdle,
        LastUsed: time.Now(),
        Language: language,
    }, nil
}
```

### 5. 任务分配策略

```go
func (p *ContainerPool) AssignTask(task *JudgeTask) (*Container, error) {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    
    // 1. 优先分配同语言的空闲容器
    for id, container := range p.containers {
        if container.Status == StatusIdle && container.Language == task.Language {
            container.Status = StatusBusy
            container.TaskID = task.ID
            container.LastUsed = time.Now()
            return container, nil
        }
    }
    
    // 2. 如果没有同语言容器，尝试创建新容器
    if len(p.containers) < p.maxSize {
        container, err := p.createContainer(task.Language)
        if err != nil {
            return nil, err
        }
        
        container.Status = StatusBusy
        container.TaskID = task.ID
        p.containers[container.ID] = container
        return container, nil
    }
    
    // 3. 容器池已满，等待空闲容器
    return p.waitForIdleContainer(task)
}

func (p *ContainerPool) waitForIdleContainer(task *JudgeTask) (*Container, error) {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    timeout := time.After(30 * time.Second)
    
    for {
        select {
        case <-timeout:
            return nil, fmt.Errorf("timeout waiting for available container")
        case <-ticker.C:
            p.mutex.Lock()
            for id, container := range p.containers {
                if container.Status == StatusIdle {
                    container.Status = StatusBusy
                    container.TaskID = task.ID
                    container.LastUsed = time.Now()
                    p.mutex.Unlock()
                    return container, nil
                }
            }
            p.mutex.Unlock()
        }
    }
}
```

### 6. 任务执行

```go
func (p *ContainerPool) ExecuteTask(container *Container, task *JudgeTask) *JudgeResult {
    result := &JudgeResult{
        TaskID:    task.ID,
        Status:    "PENDING",
        TestCases: make([]TestCaseResult, len(task.TestCases)),
    }
    
    // 1. 清理容器环境
    if err := p.cleanContainer(container); err != nil {
        result.Status = "SE"
        result.Error = "Container cleanup failed: " + err.Error()
        return result
    }
    
    // 2. 复制代码到容器
    if err := p.copyCodeToContainer(container, task); err != nil {
        result.Status = "SE"
        result.Error = "Failed to copy code: " + err.Error()
        return result
    }
    
    // 3. 编译代码（如果需要）
    if task.Language != "python" {
        if err := p.compileCode(container, task); err != nil {
            result.Status = "CE"
            result.Error = "Compilation failed: " + err.Error()
            return result
        }
    }
    
    // 4. 运行测试用例
    for i, testCase := range task.TestCases {
        caseResult := p.runTestCase(container, task, testCase)
        result.TestCases[i] = caseResult
    }
    
    // 5. 计算总体结果
    result.Status = p.calculateOverallStatus(result.TestCases)
    result.Score = p.calculateScore(result.TestCases)
    
    return result
}

func (p *ContainerPool) cleanContainer(container *Container) error {
    // 清理容器中的临时文件
    exec, err := p.client.ContainerExecCreate(
        context.Background(),
        container.ID,
        types.ExecConfig{
            Cmd: []string{"sh", "-c", "rm -rf /tmp/judge/* && mkdir -p /tmp/judge"},
            AttachStdout: true,
            AttachStderr: true,
        },
    )
    if err != nil {
        return err
    }
    
    return p.client.ContainerExecStart(
        context.Background(),
        exec.ID,
        types.ExecStartCheck{},
    )
}
```

### 7. 容器监控和自动恢复

```go
func (p *ContainerPool) monitor() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            p.healthCheck()
            p.autoScale()
        }
    }
}

func (p *ContainerPool) healthCheck() {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    
    for id, container := range p.containers {
        // 检查容器是否还在运行
        info, err := p.client.ContainerInspect(context.Background(), id)
        if err != nil || !info.State.Running {
            // 容器异常，标记为错误状态
            container.Status = StatusError
            
            // 如果容器没有在执行任务，尝试重启
            if container.Status != StatusBusy {
                go p.recreateContainer(container)
            }
        }
        
        // 检查是否有长时间运行的任务
        if container.Status == StatusBusy {
            if time.Since(container.LastUsed) > 5*time.Minute {
                // 任务超时，强制重置容器
                go p.forceResetContainer(container)
            }
        }
    }
}

func (p *ContainerPool) autoScale() {
    p.mutex.RLock()
    idleCount := 0
    busyCount := 0
    
    for _, container := range p.containers {
        switch container.Status {
        case StatusIdle:
            idleCount++
        case StatusBusy:
            busyCount++
        }
    }
    p.mutex.RUnlock()
    
    totalCount := len(p.containers)
    
    // 扩容逻辑：如果空闲容器不足且未达到最大值
    if idleCount < 2 && totalCount < p.maxSize {
        for _, lang := range p.languages {
            if totalCount >= p.maxSize {
                break
            }
            go p.addContainer(lang)
            totalCount++
        }
    }
    
    // 缩容逻辑：如果空闲容器过多且超过最小值
    if idleCount > 5 && totalCount > p.minSize {
        go p.removeIdleContainers(idleCount - 3)
    }
}
```

### 8. 优雅关闭

```go
func (p *ContainerPool) Shutdown() error {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    
    // 停止接收新任务
    close(p.taskQueue)
    
    // 等待所有任务完成
    timeout := time.After(60 * time.Second)
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-timeout:
            return fmt.Errorf("shutdown timeout")
        case <-ticker.C:
            allIdle := true
            for _, container := range p.containers {
                if container.Status == StatusBusy {
                    allIdle = false
                    break
                }
            }
            if allIdle {
                goto cleanup
            }
        }
    }
    
cleanup:
    // 清理所有容器
    for id := range p.containers {
        p.client.ContainerStop(context.Background(), id, nil)
        p.client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{})
    }
    
    return nil
}
```

## 配置示例

```yaml
container_pool:
  min_size: 6          # 最小容器数 (每种语言2个)
  max_size: 30         # 最大容器数
  queue_size: 1000     # 任务队列大小
  languages:
    - cpp
    - python  
    - java
  
  resources:
    memory_limit: 256MB
    cpu_limit: 50%      # 50% CPU
    timeout: 5m
    
  images:
    cpp: "ggcode/judge-cpp:latest"
    python: "ggcode/judge-python:latest"
    java: "ggcode/judge-java:latest"
    
  monitoring:
    health_check_interval: 30s
    auto_scale_interval: 60s
    max_idle_time: 10m
```

## 性能优势

### 对比分析

| 指标 | 传统方式 | 容器池方式 | 提升 |
|------|----------|------------|------|
| 容器启动时间 | 2-5秒 | 0秒 | 100% |
| 并发处理能力 | 有限 | 高 | 5-10x |
| 资源利用率 | 低 | 高 | 3-5x |
| 响应时间 | 3-8秒 | 0.5-2秒 | 70% |

### 实际收益
- **启动开销消除**：预热容器避免每次创建销毁
- **并发能力提升**：多容器并行处理任务
- **资源复用**：容器可重复使用，减少资源浪费
- **弹性伸缩**：根据负载动态调整容器数量

## 部署建议

1. **资源规划**：根据并发需求规划容器池大小
2. **监控告警**：设置容器健康检查和资源监控
3. **安全隔离**：确保容器间完全隔离
4. **故障恢复**：自动检测和恢复异常容器
5. **性能调优**：根据实际负载调整参数

这个设计能够显著提升评测系统的性能和稳定性，特别适合高并发的在线评测场景。 