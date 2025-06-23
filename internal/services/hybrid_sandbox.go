package services

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// HybridSandbox 混合沙箱实现 - 根据环境自动选择策略
type HybridSandbox struct {
	workDir     string
	timeLimit   int // 毫秒
	memoryLimit int // KB
	checker     *SandboxEnvironmentChecker
	mode        string // "real", "enhanced", "simple"
}

// NewHybridSandbox 创建混合沙箱
func NewHybridSandbox(workDir string, timeLimit, memoryLimit int) *HybridSandbox {
	checker := &SandboxEnvironmentChecker{}
	status := checker.CheckEnvironment()

	var mode string
	if status.CanUseRealSandbox {
		mode = "real"
	} else if checker.canUseEnhancedMode() {
		mode = "enhanced"
	} else {
		mode = "simple"
	}

	return &HybridSandbox{
		workDir:     workDir,
		timeLimit:   timeLimit,
		memoryLimit: memoryLimit,
		checker:     checker,
		mode:        mode,
	}
}

// Execute 执行命令
func (s *HybridSandbox) Execute(command string, input string) (*SandboxResult, error) {
	switch s.mode {
	case "real":
		return s.executeReal(command, input)
	case "enhanced":
		return s.executeEnhanced(command, input)
	default:
		return s.executeSimple(command, input)
	}
}

// executeReal 完整沙箱执行（需要root权限）
func (s *HybridSandbox) executeReal(command string, input string) (*SandboxResult, error) {
	realSandbox := NewRealSandbox(s.workDir, s.timeLimit, s.memoryLimit)
	return realSandbox.Execute(command, input)
}

// executeEnhanced 增强沙箱执行（普通用户可用）
func (s *HybridSandbox) executeEnhanced(command string, input string) (*SandboxResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.timeLimit+1000)*time.Millisecond)
	defer cancel()

	// 使用用户命名空间（不需要root权限）
	var cmd *exec.Cmd
	if s.canUseUserNamespace() {
		cmd = exec.CommandContext(ctx, "unshare",
			"--user",       // 用户命名空间
			"--pid",        // PID命名空间
			"--fork",       // fork子进程
			"--mount-proc", // 挂载新的/proc
			"--",
			"bash", "-c", command)
	} else {
		// 退化到基础隔离
		cmd = exec.CommandContext(ctx, "bash", "-c", command)
	}

	cmd.Dir = s.workDir
	cmd.Stdin = strings.NewReader(input)

	// 设置进程属性
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// 限制环境变量
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=/tmp",
		"LANG=C",
		"LC_ALL=C",
	}

	return s.executeWithBasicMonitoring(cmd)
}

// executeSimple 简单沙箱执行
func (s *HybridSandbox) executeSimple(command string, input string) (*SandboxResult, error) {
	simpleSandbox := NewSimpleSandbox(s.workDir, s.timeLimit, s.memoryLimit)
	return simpleSandbox.Execute(command, input)
}

// executeWithBasicMonitoring 基础监控执行
func (s *HybridSandbox) executeWithBasicMonitoring(cmd *exec.Cmd) (*SandboxResult, error) {
	result := &SandboxResult{
		Status: "SE",
	}

	// 启动进程
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	endTime := time.Now()

	result.TimeUsed = int(endTime.Sub(startTime).Milliseconds())
	result.Stdout = string(output)

	// 处理结果
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			if result.TimeUsed >= s.timeLimit {
				result.Status = "TLE"
			} else if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
				if result.ExitCode != 0 {
					result.Status = "RE"
				} else {
					result.Status = "OK"
				}
			} else {
				result.Status = "RE"
			}
		} else {
			result.Status = "RE"
		}
	} else {
		result.Status = "OK"
	}

	return result, nil
}

// canUseEnhancedMode 检查是否可以使用增强模式
func (checker *SandboxEnvironmentChecker) canUseEnhancedMode() bool {
	// 检查是否有unshare命令
	if _, err := exec.LookPath("unshare"); err != nil {
		return false
	}

	// 检查是否支持用户命名空间
	if _, err := os.Stat("/proc/self/ns/user"); err != nil {
		return false
	}

	return true
}

// canUseUserNamespace 检查是否可以使用用户命名空间
func (s *HybridSandbox) canUseUserNamespace() bool {
	// 测试用户命名空间是否可用
	cmd := exec.Command("unshare", "--user", "--", "echo", "test")
	return cmd.Run() == nil
}

// GetMode 获取当前沙箱模式
func (s *HybridSandbox) GetMode() string {
	return s.mode
}

// GetModeDescription 获取模式描述
func (s *HybridSandbox) GetModeDescription() string {
	switch s.mode {
	case "real":
		return "完整沙箱模式 - 完全隔离，最高安全性"
	case "enhanced":
		return "增强沙箱模式 - 部分隔离，中等安全性"
	case "simple":
		return "简单沙箱模式 - 基础隔离，基本安全性"
	default:
		return "未知模式"
	}
}

// SandboxModeInfo 沙箱模式信息
type SandboxModeInfo struct {
	Mode        string            `json:"mode"`
	Description string            `json:"description"`
	Features    map[string]string `json:"features"`
	Limitations []string          `json:"limitations"`
}

// GetModeInfo 获取详细的模式信息
func (s *HybridSandbox) GetModeInfo() *SandboxModeInfo {
	info := &SandboxModeInfo{
		Mode:        s.mode,
		Description: s.GetModeDescription(),
		Features:    make(map[string]string),
		Limitations: []string{},
	}

	switch s.mode {
	case "real":
		info.Features = map[string]string{
			"进程隔离":   "完全PID命名空间隔离",
			"网络隔离":   "完全网络命名空间隔离",
			"文件系统隔离": "挂载命名空间隔离",
			"内存限制":   "cgroup内存限制",
			"CPU限制":  "cgroup CPU限制",
			"用户权限":   "降级到nobody用户",
		}
		info.Limitations = []string{"需要root权限"}

	case "enhanced":
		info.Features = map[string]string{
			"进程隔离":  "用户命名空间 + PID命名空间",
			"环境隔离":  "受限的环境变量",
			"进程组隔离": "独立进程组",
			"时间限制":  "基于context的超时",
		}
		info.Limitations = []string{
			"无真正的内存限制",
			"无网络隔离",
			"有限的文件系统保护",
		}

	case "simple":
		info.Features = map[string]string{
			"进程组隔离": "独立进程组",
			"环境隔离":  "受限的环境变量",
			"时间限制":  "基于context的超时",
		}
		info.Limitations = []string{
			"无内存限制",
			"无网络隔离",
			"无文件系统隔离",
			"可访问整个系统",
		}
	}

	return info
}

// TestSandbox 测试沙箱功能
func (s *HybridSandbox) TestSandbox() (*SandboxResult, error) {
	testCode := `
echo "Testing sandbox..."
echo "Current user: $(id)"
echo "Available commands: $(which ls cat echo)"
echo "Process tree: $(ps -ef | head -5)"
echo "Network interfaces: $(ip addr show | grep inet || echo 'No network info')"
echo "File system: $(ls -la / | head -5)"
echo "Memory info: $(cat /proc/meminfo | head -3 || echo 'No memory info')"
echo "Test completed successfully"
`

	return s.Execute(testCode, "")
}
