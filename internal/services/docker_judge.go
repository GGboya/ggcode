package services

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DockerJudgeService 基于Docker容器池的评测服务
type DockerJudgeService struct {
	containerImage string
	pool           *SimpleContainerPool
}

// NewDockerJudgeServiceWithPool 创建带容器池的Docker评测服务
func NewDockerJudgeServiceWithPool(pool *SimpleContainerPool) *DockerJudgeService {
	if pool == nil {
		log.Fatal("容器池不能为空，DockerJudgeService 必须使用容器池")
	}

	service := &DockerJudgeService{
		containerImage: "ggcode-judge:latest",
		pool:           pool,
	}

	log.Printf("Docker评测服务已初始化，使用容器池模式")
	return service
}

// DockerJudgeRequest Docker评测请求
type DockerJudgeRequest struct {
	Code        string `json:"code"`
	Language    string `json:"language"`
	Input       string `json:"input"`
	Expected    string `json:"expected"`
	TimeLimit   int    `json:"time_limit"`   // 秒
	MemoryLimit int    `json:"memory_limit"` // MB
}

// DockerJudgeResult Docker评测结果
type DockerJudgeResult struct {
	Status         string `json:"status"` // AC, WA, TLE, MLE, RE, CE
	ExitCode       int    `json:"exit_code"`
	CompileMessage string `json:"compile_message"`
	RuntimeMessage string `json:"runtime_message"`
	ActualOutput   string `json:"actual_output"`
	ExpectedOutput string `json:"expected_output"`
	TimeUsed       int    `json:"time_used"`   // 毫秒
	MemoryUsed     int    `json:"memory_used"` // KB
}

// RunJudge 运行Docker评测（容器池模式）
func (djs *DockerJudgeService) RunJudge(req *DockerJudgeRequest) (*DockerJudgeResult, error) {
	// 获取容器
	container, err := djs.pool.GetContainer(req.Language)
	if err != nil {
		return nil, fmt.Errorf("获取容器失败: %v", err)
	}
	defer djs.pool.ReleaseContainer(container.ID)

	// 在容器中执行评测
	result, err := djs.executeInPoolContainer(container, req)
	if err != nil {
		return nil, fmt.Errorf("容器评测失败: %v", err)
	}

	log.Printf("容器评测完成: 容器=%s, 语言=%s, 状态=%s, 用时=%dms",
		container.ID[:12], req.Language, result.Status, result.TimeUsed)

	return result, nil
}

// executeInPoolContainer 在容器池的容器中执行评测
func (djs *DockerJudgeService) executeInPoolContainer(container *SimpleContainer, req *DockerJudgeRequest) (*DockerJudgeResult, error) {
	startTime := time.Now()

	result := &DockerJudgeResult{
		ExpectedOutput: req.Expected,
	}

	// 1. 确定源文件名
	var filename string
	switch req.Language {
	case "cpp":
		filename = "Main.cpp"
	case "python":
		filename = "Main.py"
	case "java":
		filename = "Main.java"
	case "go":
		filename = "Main.go"
	default:
		result.Status = "SE"
		result.RuntimeMessage = fmt.Sprintf("不支持的语言: %s", req.Language)
		return result, nil
	}
	fmt.Printf("[DEBUG] 确定源文件名: %s\n", filename)

	// 2. 分步骤创建文件，避免复杂的转义问题

	// 2.1 清理工作目录
	cleanCmd := exec.Command("docker", "exec", container.ID, "bash", "-c", "rm -rf /opt/judge/* 2>/dev/null || true")
	cleanCmd.Run()

	// 2.2 创建源文件
	sourceCmd := exec.Command("docker", "exec", "-i", container.ID, "bash", "-c", fmt.Sprintf("cat > /opt/judge/%s", filename))
	sourceCmd.Stdin = strings.NewReader(req.Code)
	if err := sourceCmd.Run(); err != nil {
		return nil, fmt.Errorf("创建源文件失败: %v", err)
	}

	// 2.3 创建输入文件
	inputCmd := exec.Command("docker", "exec", "-i", container.ID, "bash", "-c", "cat > /opt/judge/input.txt")
	inputCmd.Stdin = strings.NewReader(req.Input)
	if err := inputCmd.Run(); err != nil {
		return nil, fmt.Errorf("创建输入文件失败: %v", err)
	}

	// 2.4 创建期望输出文件
	expectedCmd := exec.Command("docker", "exec", "-i", container.ID, "bash", "-c", "cat > /opt/judge/expected.txt")
	expectedCmd.Stdin = strings.NewReader(req.Expected)
	if err := expectedCmd.Run(); err != nil {
		return nil, fmt.Errorf("创建期望输出文件失败: %v", err)
	}

	// 2.5 构建评测命令
	var evalCommand string
	switch req.Language {
	case "cpp":
		evalCommand = fmt.Sprintf(`g++ -O2 -std=c++17 %s -o main 2>compile.log
if [ $? -eq 0 ]; then
    echo "=== TIME START ==="
    # 使用date命令测量时间（容器中通常有date命令）
    START_TIME=$(date +%%s%%3N)
    timeout %d ./main < input.txt > output.txt 2>runtime.log
    RESULT=$?
    END_TIME=$(date +%%s%%3N)
    # 计算时间差（毫秒）
    TIME_DIFF=$((END_TIME - START_TIME))
    echo "执行时间: ${TIME_DIFF}ms"
    echo "=== TIME END ==="
    if [ $RESULT -eq 0 ]; then
        echo "程序正常执行"
        echo "=== OUTPUT START ==="
        cat output.txt 2>/dev/null || echo ""
        echo "=== OUTPUT END ==="
    elif [ $RESULT -eq 124 ]; then
        echo "TLE: 时间超限"
    else
        echo "RE: 运行时错误"
    fi
else
    echo "编译失败"
    cat compile.log
fi`, filename, req.TimeLimit)

	case "python":
		evalCommand = fmt.Sprintf(`echo "=== TIME START ==="
START_TIME=$(date +%%s%%3N)
timeout %d python3 %s < input.txt > output.txt 2>runtime.log
RESULT=$?
END_TIME=$(date +%%s%%3N)
TIME_DIFF=$((END_TIME - START_TIME))
echo "执行时间: ${TIME_DIFF}ms"
echo "=== TIME END ==="
if [ $RESULT -eq 0 ]; then
    echo "程序正常执行"
    echo "=== OUTPUT START ==="
    cat output.txt 2>/dev/null || echo ""
    echo "=== OUTPUT END ==="
elif [ $RESULT -eq 124 ]; then
    echo "TLE: 时间超限"
else
    echo "RE: 运行时错误"
fi`, req.TimeLimit, filename)

	case "java":
		evalCommand = fmt.Sprintf(`javac %s 2>compile.log
if [ $? -eq 0 ]; then
    echo "=== TIME START ==="
    START_TIME=$(date +%%s%%3N)
    timeout %d java Main < input.txt > output.txt 2>runtime.log
    RESULT=$?
    END_TIME=$(date +%%s%%3N)
    TIME_DIFF=$((END_TIME - START_TIME))
    echo "执行时间: ${TIME_DIFF}ms"
    echo "=== TIME END ==="
    if [ $RESULT -eq 0 ]; then
        echo "程序正常执行"
        echo "=== OUTPUT START ==="
        cat output.txt 2>/dev/null || echo ""
        echo "=== OUTPUT END ==="
    elif [ $RESULT -eq 124 ]; then
        echo "TLE: 时间超限"
    else
        echo "RE: 运行时错误"
    fi
else
    echo "编译失败"
    cat compile.log
fi`, filename, req.TimeLimit)

	case "go":
		evalCommand = fmt.Sprintf(`echo "检查Go环境..."
go version
echo "设置Go环境变量..."
export GO111MODULE=off && export GOPATH=/home/judge/go && export GOROOT=/usr/local/go && export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
echo "开始编译Go程序..."
go build -o main %s 2>compile.log
if [ $? -eq 0 ]; then
    echo "编译成功"
    echo "=== TIME START ==="
    START_TIME=$(date +%%s%%3N)
    timeout %d ./main < input.txt > output.txt 2>runtime.log
    RESULT=$?
    END_TIME=$(date +%%s%%3N)
    TIME_DIFF=$((END_TIME - START_TIME))
    echo "执行时间: ${TIME_DIFF}ms"
    echo "=== TIME END ==="
    if [ $RESULT -eq 0 ]; then
        echo "程序正常执行"
        echo "=== OUTPUT START ==="
        cat output.txt 2>/dev/null || echo ""
        echo "=== OUTPUT END ==="
    elif [ $RESULT -eq 124 ]; then
        echo "TLE: 时间超限"
    else
        echo "RE: 运行时错误"
    fi
else
    echo "编译失败"
    cat compile.log
fi`, filename, req.TimeLimit)

	default:
		result.Status = "SE"
		result.RuntimeMessage = fmt.Sprintf("不支持的语言: %s", req.Language)
		return result, nil
	}

	// 3. 执行评测命令
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeLimit+30)*time.Second)
	defer cancel()

	// 使用更稳定的执行方式
	cmd := exec.CommandContext(ctx, "docker", "exec", container.ID, "bash", "-c", fmt.Sprintf("cd /opt/judge && %s", evalCommand))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("执行命令: %s", cmd.String())
	err := cmd.Run()
	log.Printf("执行命令结果: %v", err)

	output := stdout.String()
	errorOutput := stderr.String()

	log.Printf("输出: %s", output)
	log.Printf("错误输出: %s", errorOutput)

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	// 4. 解析输出和状态
	var actualOutput string
	var executionTime float64

	// 从命令输出中提取实际程序输出和执行时间
	if strings.Contains(output, "=== OUTPUT START ===") && strings.Contains(output, "=== OUTPUT END ===") {
		startIdx := strings.Index(output, "=== OUTPUT START ===")
		endIdx := strings.Index(output, "=== OUTPUT END ===")
		if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
			actualOutput = strings.TrimSpace(output[startIdx+len("=== OUTPUT START ===") : endIdx])
		}
	}

	// 提取执行时间
	if strings.Contains(output, "=== TIME START ===") && strings.Contains(output, "=== TIME END ===") {
		timeStartIdx := strings.Index(output, "=== TIME START ===")
		timeEndIdx := strings.Index(output, "=== TIME END ===")
		if timeStartIdx != -1 && timeEndIdx != -1 && timeEndIdx > timeStartIdx {
			timeSection := output[timeStartIdx+len("=== TIME START ===") : timeEndIdx]
			// 查找执行时间信息
			lines := strings.Split(timeSection, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				// 查找我们新添加的时间格式 "执行时间: XXXms"
				if strings.HasPrefix(line, "执行时间: ") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						timeStr := strings.TrimSuffix(parts[1], "ms")
						if time, err := strconv.ParseFloat(timeStr, 64); err == nil {
							executionTime = time / 1000.0 // 转换为秒
							break
						}
					}
				}
				// 兼容旧的 /usr/bin/time 输出格式
				if strings.HasPrefix(line, "real ") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						if time, err := strconv.ParseFloat(parts[1], 64); err == nil {
							executionTime = time
							break
						}
					}
				}
			}
		}
	}

	result.ActualOutput = actualOutput

	// 智能比较输出（在Go中处理，更可靠）
	if strings.Contains(output, "编译失败") {
		result.Status = "CE"
		result.CompileMessage = djs.extractCompileError(container.ID)
	} else if strings.Contains(output, "RE: 运行时错误") {
		result.Status = "RE"
		result.RuntimeMessage = djs.extractRuntimeError(container.ID)
	} else if strings.Contains(output, "TLE: 时间超限") {
		result.Status = "TLE"
		result.RuntimeMessage = "程序执行超时"
	} else {
		// 进行智能输出比较
		if djs.compareOutputs(actualOutput, req.Expected) {
			result.Status = "AC"
		} else {
			result.Status = "WA"
			// 如果有系统错误信息，也记录下来
			if errorOutput != "" {
				result.RuntimeMessage = fmt.Sprintf("系统信息: %s", errorOutput)
			}
		}
	}

	// 使用从容器内测量的实际执行时间
	if executionTime > 0 {
		result.TimeUsed = int(executionTime * 1000) // 转换为毫秒
	} else {
		// 如果无法获取精确时间，使用外部计时作为备选
		result.TimeUsed = int(time.Since(startTime).Milliseconds())
	}

	return result, nil
}

// compareOutputs 智能比较输出结果
func (djs *DockerJudgeService) compareOutputs(actual, expected string) bool {
	// 去除首尾空白字符
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)

	// 直接比较
	if actual == expected {
		return true
	}

	// 忽略行尾空格的比较
	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")

	if len(actualLines) != len(expectedLines) {
		return false
	}

	for i := 0; i < len(actualLines); i++ {
		actualLine := strings.TrimRight(actualLines[i], " \t")
		expectedLine := strings.TrimRight(expectedLines[i], " \t")
		if actualLine != expectedLine {
			return false
		}
	}

	return true
}

// extractRuntimeError 从容器中提取运行时错误信息
func (djs *DockerJudgeService) extractRuntimeError(containerID string) string {
	cmd := exec.Command("docker", "exec", containerID, "cat", "/opt/judge/runtime.log")
	output, err := cmd.Output()
	if err != nil {
		return "无法获取运行时错误详情"
	}
	errorMsg := string(output)
	if errorMsg == "" {
		return "程序运行时出现错误，但无详细信息"
	}
	return fmt.Sprintf("运行时错误:\n%s", errorMsg)
}

// extractCompileError 从容器中提取编译错误信息
func (djs *DockerJudgeService) extractCompileError(containerID string) string {
	cmd := exec.Command("docker", "exec", containerID, "cat", "/opt/judge/compile.log")
	output, err := cmd.Output()
	if err != nil {
		return "无法获取编译错误详情"
	}
	errorMsg := string(output)
	if errorMsg == "" {
		return "编译失败，但无详细信息"
	}
	return fmt.Sprintf("编译错误:\n%s", errorMsg)
}
