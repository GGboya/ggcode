package services

import (
	"fmt"
	"os"
	"os/exec"
)

// SandboxEnvironmentChecker 检查沙箱环境是否可用
type SandboxEnvironmentChecker struct{}

// CheckResult 检查结果
type CheckResult struct {
	Feature   string `json:"feature"`
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

// EnvironmentStatus 环境状态
type EnvironmentStatus struct {
	CanUseRealSandbox bool          `json:"can_use_real_sandbox"`
	Checks            []CheckResult `json:"checks"`
	Recommendation    string        `json:"recommendation"`
}

// CheckEnvironment 检查沙箱环境
func (checker *SandboxEnvironmentChecker) CheckEnvironment() *EnvironmentStatus {
	status := &EnvironmentStatus{
		CanUseRealSandbox: true,
		Checks:            []CheckResult{},
	}

	// 1. 检查是否为root用户
	rootCheck := CheckResult{
		Feature:   "Root权限",
		Available: os.Geteuid() == 0,
	}
	if rootCheck.Available {
		rootCheck.Message = "✓ 具有root权限，可以使用完整沙箱功能"
	} else {
		rootCheck.Message = "✗ 需要root权限才能使用完整沙箱功能"
		status.CanUseRealSandbox = false
	}
	status.Checks = append(status.Checks, rootCheck)

	// 2. 检查unshare命令
	unshareCheck := CheckResult{
		Feature: "unshare命令",
	}
	if _, err := exec.LookPath("unshare"); err == nil {
		unshareCheck.Available = true
		unshareCheck.Message = "✓ unshare命令可用，支持命名空间隔离"
	} else {
		unshareCheck.Available = false
		unshareCheck.Message = "✗ unshare命令不可用，无法创建命名空间"
		status.CanUseRealSandbox = false
	}
	status.Checks = append(status.Checks, unshareCheck)

	// 3. 检查cgroup支持
	cgroupCheck := CheckResult{
		Feature: "cgroup",
	}
	if _, err := os.Stat("/sys/fs/cgroup"); err == nil {
		cgroupCheck.Available = true
		cgroupCheck.Message = "✓ cgroup文件系统可用，支持资源限制"
	} else {
		cgroupCheck.Available = false
		cgroupCheck.Message = "✗ cgroup不可用，无法进行资源限制"
		status.CanUseRealSandbox = false
	}
	status.Checks = append(status.Checks, cgroupCheck)

	// 4. 检查命名空间支持
	namespaceCheck := CheckResult{
		Feature: "命名空间",
	}
	namespaces := []string{"pid", "net", "mnt", "ipc", "uts"}
	allSupported := true
	for _, ns := range namespaces {
		if _, err := os.Stat(fmt.Sprintf("/proc/self/ns/%s", ns)); err != nil {
			allSupported = false
			break
		}
	}
	namespaceCheck.Available = allSupported
	if allSupported {
		namespaceCheck.Message = "✓ 所有必需的命名空间都支持"
	} else {
		namespaceCheck.Message = "✗ 部分命名空间不支持"
		status.CanUseRealSandbox = false
	}
	status.Checks = append(status.Checks, namespaceCheck)

	// 5. 检查编译器
	compilerCheck := CheckResult{
		Feature: "编译器",
	}
	compilers := map[string]string{
		"g++":     "C++编译器",
		"python3": "Python解释器",
		"javac":   "Java编译器",
	}

	availableCompilers := []string{}
	for cmd, name := range compilers {
		if _, err := exec.LookPath(cmd); err == nil {
			availableCompilers = append(availableCompilers, name)
		}
	}

	compilerCheck.Available = len(availableCompilers) > 0
	if len(availableCompilers) == len(compilers) {
		compilerCheck.Message = "✓ 所有支持的编译器都可用"
	} else if len(availableCompilers) > 0 {
		compilerCheck.Message = fmt.Sprintf("⚠ 部分编译器可用: %v", availableCompilers)
	} else {
		compilerCheck.Message = "✗ 没有可用的编译器"
	}
	status.Checks = append(status.Checks, compilerCheck)

	// 生成建议
	if status.CanUseRealSandbox {
		status.Recommendation = "✓ 环境完全支持，建议使用RealSandbox获得最佳安全性"
	} else {
		status.Recommendation = "⚠ 环境不完全支持，将使用SimpleSandbox（安全性较低）"

		if !rootCheck.Available {
			status.Recommendation += "\n• 建议以root权限运行以获得完整沙箱功能"
		}
		if !unshareCheck.Available {
			status.Recommendation += "\n• 建议安装util-linux包获得unshare命令"
		}
		if !cgroupCheck.Available {
			status.Recommendation += "\n• 建议启用cgroup支持"
		}
	}

	return status
}

// GetSandboxMode 获取推荐的沙箱模式
func (checker *SandboxEnvironmentChecker) GetSandboxMode() string {
	status := checker.CheckEnvironment()
	if status.CanUseRealSandbox {
		return "real"
	}
	return "simple"
}

// PrintEnvironmentStatus 打印环境状态
func (checker *SandboxEnvironmentChecker) PrintEnvironmentStatus() {
	status := checker.CheckEnvironment()

	fmt.Println("=== 沙箱环境检查 ===")
	for _, check := range status.Checks {
		fmt.Printf("%s: %s\n", check.Feature, check.Message)
	}

	fmt.Println("\n=== 建议 ===")
	fmt.Println(status.Recommendation)

	fmt.Printf("\n=== 当前模式 ===\n")
	if status.CanUseRealSandbox {
		fmt.Println("✓ 真正沙箱模式 (RealSandbox)")
	} else {
		fmt.Println("⚠ 简化沙箱模式 (SimpleSandbox)")
	}
}
