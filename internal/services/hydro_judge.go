package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"ggcode/internal/repositories"
)

// HydroJudgeService Hydro风格的评测服务
type HydroJudgeService interface {
	// 提交评测
	SubmitForJudge(submission *JudgeSubmission) (*JudgeResult, error)
	// 获取评测结果
	GetJudgeResult(submissionID uint) (*JudgeResult, error)
	// 获取评测队列状态
	GetQueueStatus() *JudgeQueueStatus
	// 停止评测服务
	Stop()
}

// JudgeSubmission 评测提交
type JudgeSubmission struct {
	ID         uint   `json:"id"`
	UserID     uint   `json:"user_id"`
	LevelID    uint   `json:"level_id"`
	Code       string `json:"code"`
	Language   string `json:"language"`
	SubmitTime int    `json:"submit_time"`
	Priority   int    `json:"priority"` // 优先级，数值越大优先级越高
}

// JudgeResult 评测结果
type JudgeResult struct {
	SubmissionID   uint                  `json:"submission_id"`
	Status         JudgeStatus           `json:"status"`
	Score          int                   `json:"score"`
	MaxScore       int                   `json:"max_score"`
	TimeUsed       int                   `json:"time_used"`       // 毫秒
	MemoryUsed     int                   `json:"memory_used"`     // KB
	CompileMessage string                `json:"compile_message"` // 编译信息
	TestCases      []HydroTestCaseResult `json:"test_cases"`      // 测试点结果
	SubtaskResults []HydroSubtaskResult  `json:"subtask_results"` // 子任务结果
	SystemInfo     HydroSystemInfo       `json:"system_info"`     // 系统信息
	JudgeStartTime time.Time             `json:"judge_start_time"`
	JudgeEndTime   time.Time             `json:"judge_end_time"`
	JudgeDuration  time.Duration         `json:"judge_duration"` // 评测用时
	Error          string                `json:"error,omitempty"`
}

// JudgeStatus 评测状态
type JudgeStatus string

const (
	StatusPending             JudgeStatus = "PENDING" // 等待评测
	StatusJudging             JudgeStatus = "JUDGING" // 正在评测
	StatusAccepted            JudgeStatus = "AC"      // 通过
	StatusWrongAnswer         JudgeStatus = "WA"      // 答案错误
	StatusTimeLimitExceeded   JudgeStatus = "TLE"     // 时间超限
	StatusMemoryLimitExceeded JudgeStatus = "MLE"     // 内存超限
	StatusRuntimeError        JudgeStatus = "RE"      // 运行时错误
	StatusCompileError        JudgeStatus = "CE"      // 编译错误
	StatusSystemError         JudgeStatus = "SE"      // 系统错误
	StatusPartiallyCorrect    JudgeStatus = "PC"      // 部分正确
	StatusSkipped             JudgeStatus = "SKIPPED" // 跳过
)

// HydroTestCaseResult 测试点结果
type HydroTestCaseResult struct {
	ID         int         `json:"id"`
	Status     JudgeStatus `json:"status"`
	Score      int         `json:"score"`
	MaxScore   int         `json:"max_score"`
	TimeUsed   int         `json:"time_used"`   // 毫秒
	MemoryUsed int         `json:"memory_used"` // KB
	Input      string      `json:"input,omitempty"`
	Output     string      `json:"output,omitempty"`
	Expected   string      `json:"expected,omitempty"`
	CheckerMsg string      `json:"checker_msg,omitempty"`
	IsHidden   bool        `json:"is_hidden"` // 是否为隐藏测试点
}

// HydroSubtaskResult 子任务结果
type HydroSubtaskResult struct {
	ID        int                   `json:"id"`
	Score     int                   `json:"score"`
	MaxScore  int                   `json:"max_score"`
	Status    JudgeStatus           `json:"status"`
	TestCases []HydroTestCaseResult `json:"test_cases"`
}

// HydroSystemInfo 系统信息
type HydroSystemInfo struct {
	JudgeVersion string `json:"judge_version"`
	CompilerInfo string `json:"compiler_info"`
	SystemLoad   string `json:"system_load"`
	JudgeServer  string `json:"judge_server"`
}

// JudgeQueueStatus 评测队列状态
type JudgeQueueStatus struct {
	PendingCount int `json:"pending_count"`
	JudgingCount int `json:"judging_count"`
	TotalJudged  int `json:"total_judged"`
}

// LanguageConfig 语言配置
type LanguageConfig struct {
	Name        string            `json:"name"`
	CompileCmd  string            `json:"compile_cmd"`
	RunCmd      string            `json:"run_cmd"`
	SourceExt   string            `json:"source_ext"`
	ExecExt     string            `json:"exec_ext"`
	TimeLimit   int               `json:"time_limit"`   // 默认时间限制(秒)
	MemoryLimit int               `json:"memory_limit"` // 默认内存限制(MB)
	CompileTime int               `json:"compile_time"` // 编译时间限制(秒)
	Environment map[string]string `json:"environment"`  // 环境变量
}

// SandboxConfig 沙箱配置
type SandboxConfig struct {
	TimeLimit     int      `json:"time_limit"`     // 毫秒
	MemoryLimit   int      `json:"memory_limit"`   // KB
	OutputLimit   int      `json:"output_limit"`   // 字节数
	ProcessLimit  int      `json:"process_limit"`  // 进程数限制
	EnableNetwork bool     `json:"enable_network"` // 是否允许网络访问
	ReadPaths     []string `json:"read_paths"`     // 允许读取的路径
	WritePaths    []string `json:"write_paths"`    // 允许写入的路径
}

// TestCaseConfig 测试用例配置
type TestCaseConfig struct {
	Input       string `json:"input"`
	Output      string `json:"output"`
	TimeLimit   int    `json:"time_limit"`   // 毫秒
	MemoryLimit int    `json:"memory_limit"` // KB
	Score       int    `json:"score"`
	IsSample    bool   `json:"is_sample"`
	IsHidden    bool   `json:"is_hidden"`
}

// 队列任务
type queueTask struct {
	submission *JudgeSubmission
	result     chan *JudgeResult
}

// hydro评测服务实现
type hydroJudgeService struct {
	// 配置
	languageConfigs map[string]LanguageConfig
	checkerPath     string
	sandboxPath     string
	maxWorkers      int

	// 队列和并发控制
	taskQueue chan queueTask
	results   sync.Map // map[uint]*JudgeResult
	workers   []*worker
	running   bool
	mu        sync.RWMutex

	// 统计信息
	totalJudged  int
	pendingCount int
	judgingCount int

	// 数据访问
	interviewRepo repositories.InterviewRepository
}

// worker 评测工作者
type worker struct {
	id      int
	service *hydroJudgeService
	stop    chan bool
	done    chan bool
}

func NewHydroJudgeService(interviewRepo repositories.InterviewRepository) HydroJudgeService {
	service := &hydroJudgeService{
		languageConfigs: getHydroLanguageConfigs(),
		checkerPath:     "/opt/ggcode/checker",
		sandboxPath:     "/opt/ggcode/sandbox",
		maxWorkers:      4, // 4个并发评测工作者
		taskQueue:       make(chan queueTask, 100),
		running:         true,
		interviewRepo:   interviewRepo,
	}

	// 启动工作者
	for i := 0; i < service.maxWorkers; i++ {
		worker := &worker{
			id:      i,
			service: service,
			stop:    make(chan bool),
			done:    make(chan bool),
		}
		service.workers = append(service.workers, worker)
		go worker.run()
	}

	return service
}

// 获取Hydro风格的语言配置
func getHydroLanguageConfigs() map[string]LanguageConfig {
	return map[string]LanguageConfig{
		"cpp": {
			Name:        "C++ (GCC 9.4.0)",
			CompileCmd:  "g++ -std=c++17 -O2 -Wall -Wextra -static -DONLINE_JUDGE -o {executable} {source}",
			RunCmd:      "{executable}",
			SourceExt:   ".cpp",
			ExecExt:     "",
			TimeLimit:   2,
			MemoryLimit: 256,
			CompileTime: 10,
			Environment: map[string]string{
				"LANG":   "C",
				"LC_ALL": "C",
				"PATH":   "/usr/bin:/bin",
			},
		},
		"python": {
			Name:        "Python 3.9.2",
			CompileCmd:  "",
			RunCmd:      "python3 {source}",
			SourceExt:   ".py",
			ExecExt:     "",
			TimeLimit:   5,
			MemoryLimit: 256,
			CompileTime: 0,
			Environment: map[string]string{
				"PYTHONPATH":       "/opt/python-lib",
				"PYTHONIOENCODING": "utf-8",
				"LANG":             "C.UTF-8",
				"LC_ALL":           "C.UTF-8",
			},
		},
		"java": {
			Name:        "Java (OpenJDK 11.0.11)",
			CompileCmd:  "javac -cp /opt/java-lib -encoding UTF-8 {source}",
			RunCmd:      "java -cp . -Xmx{memory_limit}m -Dfile.encoding=UTF-8 -Djava.security.manager -Djava.security.policy=/opt/java.policy {classname}",
			SourceExt:   ".java",
			ExecExt:     "",
			TimeLimit:   3,
			MemoryLimit: 512,
			CompileTime: 30,
			Environment: map[string]string{
				"LANG":      "C.UTF-8",
				"LC_ALL":    "C.UTF-8",
				"PATH":      "/usr/bin:/bin",
				"JAVA_HOME": "/usr/lib/jvm/java-11-openjdk-amd64",
			},
		},
	}
}

// SubmitForJudge 提交评测
func (hjs *hydroJudgeService) SubmitForJudge(submission *JudgeSubmission) (*JudgeResult, error) {
	hjs.mu.Lock()
	defer hjs.mu.Unlock()

	if !hjs.running {
		return nil, errors.New("评测服务已停止")
	}

	// 创建初始结果
	result := &JudgeResult{
		SubmissionID:   submission.ID,
		Status:         StatusPending,
		JudgeStartTime: time.Now(),
	}

	// 存储结果
	hjs.results.Store(submission.ID, result)

	// 创建任务
	task := queueTask{
		submission: submission,
		result:     make(chan *JudgeResult, 1),
	}

	// 异步处理任务
	go func() {
		select {
		case hjs.taskQueue <- task:
			hjs.pendingCount++
		default:
			// 队列满了
			result.Status = StatusSystemError
			result.Error = "评测队列已满，请稍后再试"
			result.JudgeEndTime = time.Now()
			result.JudgeDuration = time.Since(result.JudgeStartTime)
			hjs.results.Store(submission.ID, result)
		}
	}()

	return result, nil
}

// GetJudgeResult 获取评测结果
func (hjs *hydroJudgeService) GetJudgeResult(submissionID uint) (*JudgeResult, error) {
	if value, ok := hjs.results.Load(submissionID); ok {
		return value.(*JudgeResult), nil
	}
	return nil, errors.New("未找到评测结果")
}

// GetQueueStatus 获取队列状态
func (hjs *hydroJudgeService) GetQueueStatus() *JudgeQueueStatus {
	hjs.mu.RLock()
	defer hjs.mu.RUnlock()

	return &JudgeQueueStatus{
		PendingCount: hjs.pendingCount,
		JudgingCount: hjs.judgingCount,
		TotalJudged:  hjs.totalJudged,
	}
}

// Stop 停止评测服务
func (hjs *hydroJudgeService) Stop() {
	hjs.mu.Lock()
	hjs.running = false
	hjs.mu.Unlock()

	// 停止所有工作者
	for _, w := range hjs.workers {
		w.stop <- true
		<-w.done
	}

	close(hjs.taskQueue)
}

// worker 运行
func (w *worker) run() {
	defer func() {
		w.done <- true
	}()

	for {
		select {
		case <-w.stop:
			return
		case task := <-w.service.taskQueue:
			w.service.pendingCount--
			w.service.judgingCount++

			// 执行评测
			result := w.judgeSubmission(task.submission)

			// 更新结果
			w.service.results.Store(task.submission.ID, result)

			w.service.judgingCount--
			w.service.totalJudged++

			// 发送结果
			select {
			case task.result <- result:
			default:
			}
		}
	}
}

// judgeSubmission 评测提交
func (w *worker) judgeSubmission(submission *JudgeSubmission) *JudgeResult {
	startTime := time.Now()

	result := &JudgeResult{
		SubmissionID:   submission.ID,
		Status:         StatusJudging,
		JudgeStartTime: startTime,
	}

	// 1. 提交解析 - 验证语言和代码
	if err := w.validateSubmission(submission); err != nil {
		result.Status = StatusSystemError
		result.Error = err.Error()
		result.JudgeEndTime = time.Now()
		result.JudgeDuration = time.Since(startTime)
		return result
	}

	// 2. 获取测试用例
	testCases, err := w.getTestCases(submission.LevelID)
	if err != nil {
		result.Status = StatusSystemError
		result.Error = fmt.Sprintf("获取测试用例失败: %v", err)
		result.JudgeEndTime = time.Now()
		result.JudgeDuration = time.Since(startTime)
		return result
	}

	// 3. 创建沙箱环境
	workDir, err := w.createSandbox(submission)
	if err != nil {
		result.Status = StatusSystemError
		result.Error = fmt.Sprintf("创建沙箱失败: %v", err)
		result.JudgeEndTime = time.Now()
		result.JudgeDuration = time.Since(startTime)
		return result
	}
	defer w.cleanupSandbox(workDir)

	// 4. 编译代码
	if err := w.compileCode(submission, workDir, result); err != nil {
		result.Status = StatusCompileError
		result.Error = fmt.Sprintf("编译失败: %v", err)
		result.JudgeEndTime = time.Now()
		result.JudgeDuration = time.Since(startTime)
		return result
	}

	// 5. 运行测试用例
	result.TestCases = make([]HydroTestCaseResult, len(testCases))
	result.MaxScore = w.calculateMaxScore(testCases)

	for i, testCase := range testCases {
		testResult := w.runTestCase(submission, workDir, testCase, i+1)
		result.TestCases[i] = testResult
		result.Score += testResult.Score

		// 更新总体时间和内存使用
		if testResult.TimeUsed > result.TimeUsed {
			result.TimeUsed = testResult.TimeUsed
		}
		if testResult.MemoryUsed > result.MemoryUsed {
			result.MemoryUsed = testResult.MemoryUsed
		}

		// 如果是非样例测试点且失败，可能需要跳过后续测试点
		if !testCase.IsSample && testResult.Status != StatusAccepted {
			// 这里可以根据评测策略决定是否继续
		}
	}

	// 6. 生成评测报告
	result.Status = w.determineOverallStatus(result.TestCases)
	result.SubtaskResults = w.generateSubtaskResults(result.TestCases)
	result.SystemInfo = w.getSystemInfo()
	result.JudgeEndTime = time.Now()
	result.JudgeDuration = time.Since(startTime)

	return result
}

// validateSubmission 验证提交
func (w *worker) validateSubmission(submission *JudgeSubmission) error {
	// 检查语言是否支持
	if _, exists := w.service.languageConfigs[submission.Language]; !exists {
		return errors.New("不支持的编程语言")
	}

	// 检查代码长度
	if len(submission.Code) == 0 {
		return errors.New("代码不能为空")
	}
	if len(submission.Code) > 1024*1024 { // 1MB
		return errors.New("代码长度超过限制")
	}

	// 检查危险代码
	if w.containsDangerousCode(submission.Code, submission.Language) {
		return errors.New("代码包含危险操作")
	}

	return nil
}

// containsDangerousCode 检查危险代码
func (w *worker) containsDangerousCode(code, language string) bool {
	dangerousPatterns := map[string][]string{
		"cpp": {
			`system\s*\(`,
			`exec\w*\s*\(`,
			`fork\s*\(`,
			`#include\s*<windows\.h>`,
			`#include\s*<unistd\.h>`,
			`#include\s*<sys/.*>`,
		},
		"python": {
			`import\s+os`,
			`import\s+subprocess`,
			`import\s+sys`,
			`import\s+socket`,
			`__import__`,
			`eval\s*\(`,
			`exec\s*\(`,
			`compile\s*\(`,
		},
		"java": {
			`Runtime\.getRuntime\(\)\.exec`,
			`ProcessBuilder`,
			`System\.exit`,
			`Thread\.sleep`,
			`java\.io\.File`,
			`java\.net\.`,
		},
	}

	patterns, exists := dangerousPatterns[language]
	if !exists {
		return false
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			return true
		}
	}
	return false
}

// getTestCases 获取测试用例
func (w *worker) getTestCases(levelID uint) ([]TestCaseConfig, error) {
	// 从数据库获取测试用例
	dbTestCases, err := w.service.interviewRepo.GetTestCasesByLevelID(levelID)
	if err != nil {
		return nil, err
	}

	var testCases []TestCaseConfig
	for _, tc := range dbTestCases {
		testCases = append(testCases, TestCaseConfig{
			Input:       tc.Input,
			Output:      tc.Output,
			TimeLimit:   2000,                   // 默认2秒
			MemoryLimit: 256000,                 // 默认256MB
			Score:       100 / len(dbTestCases), // 平均分配分数
			IsSample:    tc.IsSample,
			IsHidden:    !tc.IsSample,
		})
	}

	if len(testCases) == 0 {
		// 如果数据库没有测试用例，返回示例
		return []TestCaseConfig{
			{
				Input:       "1 2\n",
				Output:      "3\n",
				TimeLimit:   1000,
				MemoryLimit: 256000,
				Score:       100,
				IsSample:    true,
				IsHidden:    false,
			},
		}, nil
	}

	return testCases, nil
}

// createSandbox 创建沙箱环境
func (w *worker) createSandbox(submission *JudgeSubmission) (string, error) {
	// 创建临时工作目录
	workDir, err := os.MkdirTemp("", fmt.Sprintf("hydro_judge_%d_*", submission.ID))
	if err != nil {
		return "", fmt.Errorf("创建工作目录失败: %v", err)
	}

	// 设置目录权限
	if err := os.Chmod(workDir, 0755); err != nil {
		os.RemoveAll(workDir)
		return "", fmt.Errorf("设置目录权限失败: %v", err)
	}

	return workDir, nil
}

// compileCode 编译代码
func (w *worker) compileCode(submission *JudgeSubmission, workDir string, result *JudgeResult) error {
	config := w.service.languageConfigs[submission.Language]

	// 如果不需要编译（解释型语言）
	if config.CompileCmd == "" {
		// 直接写入源文件
		sourceFile := filepath.Join(workDir, "Main"+config.SourceExt)
		return os.WriteFile(sourceFile, []byte(submission.Code), 0644)
	}

	// 创建源文件
	sourceFile := filepath.Join(workDir, "Main"+config.SourceExt)
	if err := os.WriteFile(sourceFile, []byte(submission.Code), 0644); err != nil {
		return fmt.Errorf("写入源文件失败: %v", err)
	}

	// 准备编译命令
	execFile := filepath.Join(workDir, "Main"+config.ExecExt)
	compileCmd := strings.ReplaceAll(config.CompileCmd, "{source}", sourceFile)
	compileCmd = strings.ReplaceAll(compileCmd, "{executable}", execFile)

	// 执行编译
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.CompileTime)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", compileCmd)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置环境变量
	cmd.Env = []string{}
	for k, v := range config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := cmd.Run(); err != nil {
		result.CompileMessage = stderr.String()
		return fmt.Errorf("编译失败: %v", err)
	}

	// 保存编译信息
	if stdout.Len() > 0 || stderr.Len() > 0 {
		result.CompileMessage = stdout.String() + stderr.String()
	}

	return nil
}

// runTestCase 运行测试用例
func (w *worker) runTestCase(submission *JudgeSubmission, workDir string, testCase TestCaseConfig, caseID int) HydroTestCaseResult {
	result := HydroTestCaseResult{
		ID:       caseID,
		Status:   StatusSystemError,
		Score:    0,
		MaxScore: testCase.Score,
		Input:    testCase.Input,
		Expected: testCase.Output,
		IsHidden: testCase.IsHidden,
	}

	config := w.service.languageConfigs[submission.Language]

	// 准备运行命令
	var runCmd string
	if config.CompileCmd == "" {
		// 解释型语言
		sourceFile := filepath.Join(workDir, "Main"+config.SourceExt)
		runCmd = strings.ReplaceAll(config.RunCmd, "{source}", sourceFile)
	} else {
		// 编译型语言
		execFile := filepath.Join(workDir, "Main"+config.ExecExt)
		runCmd = strings.ReplaceAll(config.RunCmd, "{executable}", execFile)
	}

	// 替换内存限制参数
	runCmd = strings.ReplaceAll(runCmd, "{memory_limit}", strconv.Itoa(testCase.MemoryLimit/1024))
	runCmd = strings.ReplaceAll(runCmd, "{classname}", "Main")

	// 创建沙箱配置
	sandboxConfig := SandboxConfig{
		TimeLimit:     testCase.TimeLimit,
		MemoryLimit:   testCase.MemoryLimit,
		OutputLimit:   1024 * 1024, // 1MB
		ProcessLimit:  1,
		EnableNetwork: false,
	}

	// 在沙箱中运行
	return w.runInSandbox(runCmd, workDir, testCase.Input, sandboxConfig, result)
}

// runInSandbox 在沙箱中运行程序
func (w *worker) runInSandbox(command, workDir string, input string, config SandboxConfig, result HydroTestCaseResult) HydroTestCaseResult {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeLimit+1000)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workDir

	// 设置资源限制
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(input)

	// 记录开始时间
	startTime := time.Now()
	err := cmd.Run()
	endTime := time.Now()

	result.TimeUsed = int(endTime.Sub(startTime).Milliseconds())

	// 处理运行结果
	if ctx.Err() == context.DeadlineExceeded {
		result.Status = StatusTimeLimitExceeded
		result.Output = "Time Limit Exceeded"
		return result
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					switch status.Signal() {
					case syscall.SIGKILL:
						result.Status = StatusTimeLimitExceeded
						result.Output = "Time Limit Exceeded"
					case syscall.SIGSEGV:
						result.Status = StatusRuntimeError
						result.Output = "Segmentation Fault"
					default:
						result.Status = StatusRuntimeError
						result.Output = fmt.Sprintf("Runtime Error (Signal: %v)", status.Signal())
					}
				} else {
					result.Status = StatusRuntimeError
					result.Output = fmt.Sprintf("Runtime Error (Exit Code: %d)", status.ExitStatus())
				}
			}
		} else {
			result.Status = StatusRuntimeError
			result.Output = err.Error()
		}

		if stderr.Len() > 0 {
			result.Output += "\n" + stderr.String()
		}
		return result
	}

	// 检查输出
	actualOutput := strings.TrimSpace(stdout.String())
	expectedOutput := strings.TrimSpace(result.Expected)

	result.Output = actualOutput

	// 使用校验器比较结果
	if w.checkAnswer(actualOutput, expectedOutput) {
		result.Status = StatusAccepted
		result.Score = result.MaxScore
		result.CheckerMsg = "Accepted"
	} else {
		result.Status = StatusWrongAnswer
		result.CheckerMsg = "Wrong Answer"
	}

	return result
}

// checkAnswer 校验答案（支持多种校验模式）
func (w *worker) checkAnswer(actual, expected string) bool {
	// 1. 精确匹配
	if actual == expected {
		return true
	}

	// 2. 忽略行尾空格的匹配
	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")

	if len(actualLines) != len(expectedLines) {
		return false
	}

	for i := range actualLines {
		if strings.TrimRight(actualLines[i], " \t") != strings.TrimRight(expectedLines[i], " \t") {
			return false
		}
	}

	return true
}

// calculateMaxScore 计算最大分数
func (w *worker) calculateMaxScore(testCases []TestCaseConfig) int {
	total := 0
	for _, tc := range testCases {
		total += tc.Score
	}
	return total
}

// determineOverallStatus 确定总体状态
func (w *worker) determineOverallStatus(testCases []HydroTestCaseResult) JudgeStatus {
	if len(testCases) == 0 {
		return StatusSystemError
	}

	allAC := true
	hasWA := false
	hasTLE := false
	hasMLE := false
	hasRE := false

	for _, tc := range testCases {
		switch tc.Status {
		case StatusAccepted:
			continue
		case StatusWrongAnswer:
			allAC = false
			hasWA = true
		case StatusTimeLimitExceeded:
			allAC = false
			hasTLE = true
		case StatusMemoryLimitExceeded:
			allAC = false
			hasMLE = true
		case StatusRuntimeError:
			allAC = false
			hasRE = true
		default:
			allAC = false
		}
	}

	if allAC {
		return StatusAccepted
	}

	// 按优先级返回状态
	if hasRE {
		return StatusRuntimeError
	}
	if hasTLE {
		return StatusTimeLimitExceeded
	}
	if hasMLE {
		return StatusMemoryLimitExceeded
	}
	if hasWA {
		return StatusWrongAnswer
	}

	return StatusSystemError
}

// generateSubtaskResults 生成子任务结果
func (w *worker) generateSubtaskResults(testCases []HydroTestCaseResult) []HydroSubtaskResult {
	// 简单实现：将所有测试用例作为一个子任务
	subtask := HydroSubtaskResult{
		ID:        1,
		TestCases: testCases,
	}

	for _, tc := range testCases {
		subtask.Score += tc.Score
		subtask.MaxScore += tc.MaxScore
	}

	if subtask.Score == subtask.MaxScore {
		subtask.Status = StatusAccepted
	} else if subtask.Score > 0 {
		subtask.Status = StatusPartiallyCorrect
	} else {
		subtask.Status = StatusWrongAnswer
	}

	return []HydroSubtaskResult{subtask}
}

// getSystemInfo 获取系统信息
func (w *worker) getSystemInfo() HydroSystemInfo {
	return HydroSystemInfo{
		JudgeVersion: "GGCode Hydro Judge v1.0",
		CompilerInfo: w.getCompilerInfo(),
		SystemLoad:   w.getSystemLoad(),
		JudgeServer:  fmt.Sprintf("Worker-%d", w.id),
	}
}

// getCompilerInfo 获取编译器信息
func (w *worker) getCompilerInfo() string {
	var info []string

	// 获取 GCC 版本
	if output, err := exec.Command("gcc", "--version").Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			info = append(info, "GCC: "+lines[0])
		}
	}

	// 获取 Python 版本
	if output, err := exec.Command("python3", "--version").Output(); err == nil {
		info = append(info, "Python: "+strings.TrimSpace(string(output)))
	}

	// 获取 Java 版本
	if output, err := exec.Command("java", "-version").CombinedOutput(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			info = append(info, "Java: "+lines[0])
		}
	}

	return strings.Join(info, "; ")
}

// getSystemLoad 获取系统负载
func (w *worker) getSystemLoad() string {
	if output, err := exec.Command("uptime").Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return "Unknown"
}

// cleanupSandbox 清理沙箱
func (w *worker) cleanupSandbox(workDir string) {
	os.RemoveAll(workDir)
}
