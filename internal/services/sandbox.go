package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// RealSandbox 真正的沙箱实现
type RealSandbox struct {
	workDir     string
	timeLimit   int // 毫秒
	memoryLimit int // KB
	uid         int // 用户ID
	gid         int // 组ID
}

// SandboxResult 沙箱执行结果
type SandboxResult struct {
	ExitCode   int
	Signal     int
	TimeUsed   int // 毫秒
	MemoryUsed int // KB
	Stdout     string
	Stderr     string
	Status     string // OK, TLE, MLE, RE, SE
}

// NewRealSandbox 创建真正的沙箱
func NewRealSandbox(workDir string, timeLimit, memoryLimit int) *RealSandbox {
	return &RealSandbox{
		workDir:     workDir,
		timeLimit:   timeLimit,
		memoryLimit: memoryLimit,
		uid:         65534, // nobody用户
		gid:         65534, // nobody组
	}
}

// Execute 在沙箱中执行命令
func (s *RealSandbox) Execute(command string, input string) (*SandboxResult, error) {
	// 1. 创建cgroup来限制资源
	cgroupDir, err := s.createCgroup()
	if err != nil {
		return nil, fmt.Errorf("创建cgroup失败: %v", err)
	}
	defer s.cleanupCgroup(cgroupDir)

	// 2. 准备命令
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.timeLimit+1000)*time.Millisecond)
	defer cancel()

	// 使用unshare创建命名空间隔离
	cmd := exec.CommandContext(ctx, "unshare",
		"--pid",        // PID命名空间
		"--net",        // 网络命名空间
		"--mount",      // 挂载命名空间
		"--ipc",        // IPC命名空间
		"--uts",        // UTS命名空间
		"--fork",       // fork子进程
		"--mount-proc", // 挂载新的/proc
		"--",
		"bash", "-c", command)

	cmd.Dir = s.workDir
	cmd.Stdin = strings.NewReader(input)

	// 3. 设置进程属性
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(s.uid),
			Gid: uint32(s.gid),
		},
		Setpgid: true,
	}

	// 4. 设置环境变量
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=/tmp",
		"USER=nobody",
		"LANG=C",
		"LC_ALL=C",
	}

	// 5. 执行命令并监控资源
	result, err := s.executeWithMonitoring(cmd, cgroupDir)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// createCgroup 创建cgroup进行资源限制
func (s *RealSandbox) createCgroup() (string, error) {
	// 生成唯一的cgroup名称
	cgroupName := fmt.Sprintf("ggcode-sandbox-%d", os.Getpid())
	cgroupDir := filepath.Join("/sys/fs/cgroup", cgroupName)

	// 创建cgroup目录
	if err := os.MkdirAll(cgroupDir, 0755); err != nil {
		return "", err
	}

	// 设置内存限制
	memoryFile := filepath.Join(cgroupDir, "memory.max")
	if err := os.WriteFile(memoryFile, []byte(strconv.Itoa(s.memoryLimit*1024)), 0644); err != nil {
		// 尝试旧版本的cgroup
		memoryFile = filepath.Join(cgroupDir, "memory", "memory.limit_in_bytes")
		if err := os.WriteFile(memoryFile, []byte(strconv.Itoa(s.memoryLimit*1024)), 0644); err != nil {
			return cgroupDir, fmt.Errorf("设置内存限制失败: %v", err)
		}
	}

	// 设置CPU限制（防止CPU占用过高）
	cpuFile := filepath.Join(cgroupDir, "cpu.max")
	if err := os.WriteFile(cpuFile, []byte("100000 100000"), 0644); err != nil {
		// 尝试旧版本的cgroup
		cpuFile = filepath.Join(cgroupDir, "cpu", "cpu.cfs_quota_us")
		os.WriteFile(cpuFile, []byte("100000"), 0644)
		cpuPeriodFile := filepath.Join(cgroupDir, "cpu", "cpu.cfs_period_us")
		os.WriteFile(cpuPeriodFile, []byte("100000"), 0644)
	}

	// 限制进程数
	pidsFile := filepath.Join(cgroupDir, "pids.max")
	if err := os.WriteFile(pidsFile, []byte("32"), 0644); err != nil {
		// 尝试旧版本
		pidsFile = filepath.Join(cgroupDir, "pids", "pids.max")
		os.WriteFile(pidsFile, []byte("32"), 0644)
	}

	return cgroupDir, nil
}

// executeWithMonitoring 执行命令并监控资源使用
func (s *RealSandbox) executeWithMonitoring(cmd *exec.Cmd, cgroupDir string) (*SandboxResult, error) {
	result := &SandboxResult{
		Status: "SE", // 默认系统错误
	}

	// 创建管道获取输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return result, err
	}

	// 启动命令
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		result.Status = "SE"
		return result, err
	}

	// 将进程加入cgroup
	if cmd.Process != nil {
		procsFile := filepath.Join(cgroupDir, "cgroup.procs")
		os.WriteFile(procsFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	}

	// 读取输出
	stdoutData := make([]byte, 1024*1024) // 1MB限制
	stderrData := make([]byte, 1024*1024)

	go func() {
		stdout.Read(stdoutData)
	}()
	go func() {
		stderr.Read(stderrData)
	}()

	// 等待命令完成
	err = cmd.Wait()
	endTime := time.Now()

	result.TimeUsed = int(endTime.Sub(startTime).Milliseconds())
	result.Stdout = string(stdoutData)
	result.Stderr = string(stderrData)

	// 检查内存使用
	result.MemoryUsed = s.getMemoryUsage(cgroupDir)

	// 处理退出状态
	if err != nil {
		// 检查是否是超时导致的退出
		if result.TimeUsed >= s.timeLimit {
			result.Status = "TLE"
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
				if status.Signaled() {
					result.Signal = int(status.Signal())
					switch status.Signal() {
					case syscall.SIGKILL:
						if result.MemoryUsed > s.memoryLimit {
							result.Status = "MLE"
						} else {
							result.Status = "TLE"
						}
					case syscall.SIGSEGV:
						result.Status = "RE"
					default:
						result.Status = "RE"
					}
				} else {
					if result.ExitCode != 0 {
						result.Status = "RE"
					} else {
						result.Status = "OK"
					}
				}
			}
		} else {
			result.Status = "RE"
		}
	} else {
		result.ExitCode = 0
		result.Status = "OK"
	}

	return result, nil
}

// getMemoryUsage 获取内存使用量
func (s *RealSandbox) getMemoryUsage(cgroupDir string) int {
	// 尝试读取内存使用情况
	memoryFiles := []string{
		filepath.Join(cgroupDir, "memory.current"),
		filepath.Join(cgroupDir, "memory", "memory.usage_in_bytes"),
	}

	for _, file := range memoryFiles {
		if data, err := os.ReadFile(file); err == nil {
			if usage, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				return usage / 1024 // 转换为KB
			}
		}
	}

	return 0
}

// cleanupCgroup 清理cgroup
func (s *RealSandbox) cleanupCgroup(cgroupDir string) {
	// 杀死所有进程
	procsFile := filepath.Join(cgroupDir, "cgroup.procs")
	if data, err := os.ReadFile(procsFile); err == nil {
		pids := strings.Fields(string(data))
		for _, pidStr := range pids {
			if pid, err := strconv.Atoi(pidStr); err == nil {
				syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	}

	// 删除cgroup目录
	os.RemoveAll(cgroupDir)
}

// SandboxChecker 安全检查器
type SandboxChecker struct{}

// CheckSeccomp 检查是否支持seccomp
func (sc *SandboxChecker) CheckSeccomp() bool {
	if _, err := os.Stat("/proc/sys/kernel/seccomp"); err != nil {
		return false
	}
	return true
}

// CheckNamespaces 检查是否支持命名空间
func (sc *SandboxChecker) CheckNamespaces() bool {
	namespaces := []string{"pid", "net", "mnt", "ipc", "uts"}
	for _, ns := range namespaces {
		if _, err := os.Stat(fmt.Sprintf("/proc/self/ns/%s", ns)); err != nil {
			return false
		}
	}
	return true
}

// CheckCgroups 检查是否支持cgroups
func (sc *SandboxChecker) CheckCgroups() bool {
	if _, err := os.Stat("/sys/fs/cgroup"); err != nil {
		return false
	}
	return true
}

// CheckPermissions 检查是否有足够权限
func (sc *SandboxChecker) CheckPermissions() bool {
	return os.Geteuid() == 0 // 需要root权限
}

// SimpleSandbox 简化的沙箱实现（当系统不支持完整沙箱时）
type SimpleSandbox struct {
	workDir     string
	timeLimit   int
	memoryLimit int
}

// NewSimpleSandbox 创建简化沙箱
func NewSimpleSandbox(workDir string, timeLimit, memoryLimit int) *SimpleSandbox {
	return &SimpleSandbox{
		workDir:     workDir,
		timeLimit:   timeLimit,
		memoryLimit: memoryLimit,
	}
}

// Execute 执行命令（简化版本）
func (s *SimpleSandbox) Execute(command string, input string) (*SandboxResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.timeLimit+1000)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = s.workDir
	cmd.Stdin = strings.NewReader(input)

	// 基础的进程隔离
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

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	endTime := time.Now()

	result := &SandboxResult{
		TimeUsed: int(endTime.Sub(startTime).Milliseconds()),
		Stdout:   string(output),
		Status:   "OK",
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = "TLE"
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Status = "RE"
		} else {
			result.Status = "SE"
		}
	}

	return result, nil
}
