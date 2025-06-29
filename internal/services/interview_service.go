package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"ggcode/internal/models"
	"ggcode/internal/repositories"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// InterviewService 面试岛服务接口
type InterviewService interface {
	// 岛屿和关卡
	GetIslandMap(userID uint) ([]repositories.IslandProgressInfo, error)
	GetLevelDetail(userID, levelID uint) (*LevelDetailResponse, error)

	// 岛屿管理（管理员）
	CreateIsland(name, description string) (*models.InterviewIsland, error)
	UpdateIsland(id uint, name, description string) error
	DeleteIsland(id uint) error
	CreateLevel(islandID, questionID uint, name, difficulty string) (*models.InterviewLevel, error)

	// 代码执行和判题
	TestCode(userID, levelID uint, code, language string) (*TestResult, error)
	SubmitCode(userID, levelID uint, code, language string, submitTime int) (*SubmissionResult, error)

	// 进度管理
	GetUserProgress(userID uint) (*UserProgressSummary, error)

	// 知识点
	UnlockTags(userID, levelID uint) error

	// 测试用例 CRUD
	GetTestCases(levelID uint) ([]models.InterviewTestCase, error)
	CreateTestCase(levelID uint, input, output string, isSample bool, order int) (*models.InterviewTestCase, error)
	UpdateTestCase(id uint, input, output string, isSample bool, order int) error
	DeleteTestCase(id uint) error
}

// LevelDetailResponse 关卡详情响应
type LevelDetailResponse struct {
	Level          models.InterviewLevel       `json:"level"`
	Question       models.Question             `json:"question"`
	SampleCases    []models.InterviewTestCase  `json:"sample_cases"`
	UserProgress   models.UserLevelProgress    `json:"user_progress"`
	BestSubmission *models.UserLevelSubmission `json:"best_submission,omitempty"`
}

// TestResult 测试结果
type TestResult struct {
	Status    string           `json:"status"` // AC, WA, TLE, MLE, CE, RE
	Message   string           `json:"message"`
	TestCases []TestCaseResult `json:"test_cases,omitempty"`
	Error     string           `json:"error,omitempty"`
}

// TestCaseResult 测试用例结果
type TestCaseResult struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Output   string `json:"output"`
	Status   string `json:"status"`  // AC, WA, TLE, MLE
	Runtime  int    `json:"runtime"` // 毫秒
	Memory   int    `json:"memory"`  // KB
}

// SubmissionResult 提交结果
type SubmissionResult struct {
	ID           uint             `json:"id"`
	Status       string           `json:"status"`
	Stars        int              `json:"stars"`
	UseTime      int              `json:"use_time"`
	Memory       int              `json:"memory"`
	Message      string           `json:"message"`
	TestResults  []TestCaseResult `json:"test_results,omitempty"`
	NextUnlocked bool             `json:"next_unlocked"`
}

// UserProgressSummary 用户进度总结
type UserProgressSummary struct {
	TotalIslands     int `json:"total_islands"`
	CompletedIslands int `json:"completed_islands"`
	TotalLevels      int `json:"total_levels"`
	CompletedLevels  int `json:"completed_levels"`
	TotalStars       int `json:"total_stars"`
	MaxStars         int `json:"max_stars"`
}

// JudgeConfig 判题配置
type JudgeConfig struct {
	Language    string
	CompileCmd  string
	RunCmd      string
	TimeLimit   int // 秒
	MemoryLimit int // MB
	Extension   string
}

type interviewService struct {
	repo repositories.InterviewRepository
}

func NewInterviewService(repo repositories.InterviewRepository) InterviewService {
	return &interviewService{repo: repo}
}

// 支持的编程语言配置
var languageConfigs = map[string]JudgeConfig{
	"cpp": {
		Language:    "cpp",
		CompileCmd:  "g++ -O2 -std=c++17 {source} -o {executable}",
		RunCmd:      "./{executable}",
		TimeLimit:   2,
		MemoryLimit: 256,
		Extension:   ".cpp",
	},
	"python": {
		Language:    "python",
		CompileCmd:  "",
		RunCmd:      "python3 {source}",
		TimeLimit:   5,
		MemoryLimit: 256,
		Extension:   ".py",
	},
	"java": {
		Language:    "java",
		CompileCmd:  "javac {source}",
		RunCmd:      "java {classname}",
		TimeLimit:   3,
		MemoryLimit: 512,
		Extension:   ".java",
	},
	"go": {
		Language:    "go",
		CompileCmd:  "go build -o {executable} {source}",
		RunCmd:      "./{executable}",
		TimeLimit:   3,
		MemoryLimit: 256,
		Extension:   ".go",
	},
}

// GetIslandMap 获取面试岛地图
func (s *interviewService) GetIslandMap(userID uint) ([]repositories.IslandProgressInfo, error) {
	return s.repo.GetUserIslandProgress(userID)
}

// GetLevelDetail 获取关卡详情
func (s *interviewService) GetLevelDetail(userID, levelID uint) (*LevelDetailResponse, error) {
	level, err := s.repo.GetLevelByID(levelID)
	if err != nil {
		return nil, err
	}

	userProgress, err := s.repo.GetUserLevelProgress(userID, levelID)
	if err != nil {
		return nil, err
	}

	// 检查关卡是否解锁
	if userProgress.Status == "locked" {
		return nil, errors.New("关卡未解锁")
	}

	sampleCases, err := s.repo.GetSampleTestCases(levelID)
	if err != nil {
		return nil, err
	}

	bestSubmission, _ := s.repo.GetBestSubmission(userID, levelID)

	return &LevelDetailResponse{
		Level:          *level,
		Question:       level.Question,
		SampleCases:    sampleCases,
		UserProgress:   *userProgress,
		BestSubmission: bestSubmission,
	}, nil
}

// TestCode 测试代码（只运行样例）
func (s *interviewService) TestCode(userID, levelID uint, code, language string) (*TestResult, error) {
	// 获取样例测试用例
	sampleCases, err := s.repo.GetSampleTestCases(levelID)
	if err != nil {
		return nil, err
	}

	return s.executeCode(code, language, sampleCases)
}

// SubmitCode 提交代码（运行所有测试用例）
func (s *interviewService) SubmitCode(userID, levelID uint, code, language string, submitTime int) (*SubmissionResult, error) {
	// 获取所有测试用例
	testCases, err := s.repo.GetTestCasesByLevelID(levelID)
	if err != nil {
		return nil, err
	}

	// 执行代码
	testResult, err := s.executeCode(code, language, testCases)
	if err != nil {
		return nil, err
	}

	// 计算星级
	stars := s.calculateStars(testResult.Status, submitTime)

	// 创建提交记录
	submission := &models.UserLevelSubmission{
		UserID:     userID,
		LevelID:    levelID,
		Code:       code,
		Language:   language,
		Status:     testResult.Status,
		UseTime:    s.getTotalRuntime(testResult.TestCases),
		Memory:     s.getMaxMemory(testResult.TestCases),
		Stars:      stars,
		SubmitTime: submitTime,
	}

	if testResult.Status != "AC" {
		submission.ErrorMsg = testResult.Message
	}

	// 保存测试结果详情
	if testResultJSON, err := json.Marshal(testResult.TestCases); err == nil {
		submission.TestResults = string(testResultJSON)
	}

	err = s.repo.CreateSubmission(submission)
	if err != nil {
		return nil, err
	}

	// 更新用户关卡进度
	var nextUnlocked bool
	if testResult.Status == "AC" {
		progress := &models.UserLevelProgress{
			UserID:   userID,
			LevelID:  levelID,
			Status:   "completed",
			Stars:    stars,
			BestTime: submitTime,
		}

		err = s.repo.CreateOrUpdateLevelProgress(progress)
		if err != nil {
			return nil, err
		}

		// 如果获得2星以上，解锁下一关
		if stars >= 2 {
			err = s.repo.UnlockNextLevel(userID, levelID)
			if err == nil {
				nextUnlocked = true
			}
		}

		// 知识点解锁：获取题目标签并写入 user_unlocked_tags
		if level, _ := s.repo.GetLevelByID(levelID); level != nil {
			for _, tag := range level.Question.Tags {
				_ = s.repo.AddUserUnlockedTag(userID, tag.ID) // 忽略错误即可
			}
		}
	}

	return &SubmissionResult{
		ID:           submission.ID,
		Status:       testResult.Status,
		Stars:        stars,
		UseTime:      submission.UseTime,
		Memory:       submission.Memory,
		Message:      testResult.Message,
		TestResults:  testResult.TestCases,
		NextUnlocked: nextUnlocked,
	}, nil
}

// executeCode 执行代码
func (s *interviewService) executeCode(code, language string, testCases []models.InterviewTestCase) (*TestResult, error) {
	config, exists := languageConfigs[language]
	if !exists {
		return nil, errors.New("不支持的编程语言")
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "ggcode_judge_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// 写入源代码文件
	sourceFile := filepath.Join(tempDir, "solution"+config.Extension)
	err = os.WriteFile(sourceFile, []byte(code), 0644)
	if err != nil {
		return nil, err
	}

	// 编译代码（如果需要）
	var executableFile string
	if config.CompileCmd != "" {
		executableFile = filepath.Join(tempDir, "solution")
		compileCmd := strings.ReplaceAll(config.CompileCmd, "{source}", sourceFile)
		compileCmd = strings.ReplaceAll(compileCmd, "{executable}", executableFile)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "bash", "-c", compileCmd)
		cmd.Dir = tempDir

		var compileOutput bytes.Buffer
		cmd.Stderr = &compileOutput

		err = cmd.Run()
		if err != nil {
			return &TestResult{
				Status:  "CE",
				Message: "编译错误: " + compileOutput.String(),
			}, nil
		}
	}

	// 运行测试用例
	var testCaseResults []TestCaseResult
	allPassed := true

	for _, testCase := range testCases {
		result := s.runSingleTest(sourceFile, executableFile, config, testCase.Input, testCase.Output)
		testCaseResults = append(testCaseResults, result)

		if result.Status != "AC" {
			allPassed = false
		}
	}

	// 确定最终状态
	status := "AC"
	message := "所有测试用例通过"

	if !allPassed {
		// 查找第一个失败的测试用例类型
		for _, result := range testCaseResults {
			if result.Status != "AC" {
				status = result.Status
				switch status {
				case "WA":
					message = "答案错误"
				case "TLE":
					message = "时间超限"
				case "MLE":
					message = "内存超限"
				case "RE":
					message = "运行时错误"
				}
				break
			}
		}
	}

	return &TestResult{
		Status:    status,
		Message:   message,
		TestCases: testCaseResults,
	}, nil
}

// runSingleTest 运行单个测试用例
func (s *interviewService) runSingleTest(sourceFile, executableFile string, config JudgeConfig, input, expectedOutput string) TestCaseResult {
	var runCmd string
	if config.CompileCmd != "" {
		// 编译型语言
		runCmd = strings.ReplaceAll(config.RunCmd, "{executable}", "solution")
	} else {
		// 解释型语言
		runCmd = strings.ReplaceAll(config.RunCmd, "{source}", filepath.Base(sourceFile))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeLimit)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", runCmd)
	cmd.Dir = filepath.Dir(sourceFile)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(input)

	start := time.Now()
	err := cmd.Run()
	runtime := int(time.Since(start).Milliseconds())

	result := TestCaseResult{
		Input:    input,
		Expected: strings.TrimSpace(expectedOutput),
		Runtime:  runtime,
		Memory:   0, // 简化实现，不统计内存
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "TLE"
		result.Output = "时间超限"
		return result
	}

	if err != nil {
		result.Status = "RE"
		result.Output = stderr.String()
		return result
	}

	actualOutput := strings.TrimSpace(stdout.String())
	result.Output = actualOutput

	if actualOutput == result.Expected {
		result.Status = "AC"
	} else {
		result.Status = "WA"
	}

	return result
}

// calculateStars 根据完成时间计算星级
func (s *interviewService) calculateStars(status string, submitTime int) int {
	if status != "AC" {
		return 0
	}

	// 5分钟内完成 = 3星
	if submitTime <= 300 {
		return 3
	}
	// 10分钟内完成 = 2星
	if submitTime <= 600 {
		return 2
	}
	// 15分钟内完成 = 1星
	if submitTime <= 900 {
		return 1
	}
	// 超过15分钟 = 0星
	return 0
}

// getTotalRuntime 获取总运行时间
func (s *interviewService) getTotalRuntime(results []TestCaseResult) int {
	total := 0
	for _, result := range results {
		total += result.Runtime
	}
	return total
}

// getMaxMemory 获取最大内存使用
func (s *interviewService) getMaxMemory(results []TestCaseResult) int {
	max := 0
	for _, result := range results {
		if result.Memory > max {
			max = result.Memory
		}
	}
	return max
}

// GetUserProgress 获取用户总体进度
func (s *interviewService) GetUserProgress(userID uint) (*UserProgressSummary, error) {
	progressInfos, err := s.repo.GetUserIslandProgress(userID)
	if err != nil {
		return nil, err
	}

	summary := &UserProgressSummary{}

	for _, info := range progressInfos {
		summary.TotalIslands++
		summary.TotalLevels += info.TotalCount
		summary.CompletedLevels += info.CompletedCount
		summary.TotalStars += info.TotalStars
		summary.MaxStars += info.MaxStars

		// 如果岛屿所有关卡都完成，认为岛屿完成
		if info.CompletedCount == info.TotalCount && info.TotalCount > 0 {
			summary.CompletedIslands++
		}
	}

	return summary, nil
}

func (s *interviewService) DeleteIsland(id uint) error {
	return s.repo.DeleteIsland(id)
}

func (s *interviewService) CreateIsland(name, description string) (*models.InterviewIsland, error) {
	island := &models.InterviewIsland{
		Name:        name,
		Description: description,
		Difficulty:  "Easy",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.repo.CreateIsland(island); err != nil {
		return nil, err
	}
	return island, nil
}

func (s *interviewService) UpdateIsland(id uint, name, description string) error {
	island, err := s.repo.GetIslandByID(id)
	if err != nil {
		return err
	}
	if name != "" {
		island.Name = name
	}
	island.Description = description
	island.UpdatedAt = time.Now()
	return s.repo.UpdateIsland(island)
}

// CreateLevel 创建新关卡
func (s *interviewService) CreateLevel(islandID, questionID uint, name, difficulty string) (*models.InterviewLevel, error) {
	// 获取当前岛屿的最大关卡号
	maxLevelNum, err := s.repo.GetMaxLevelNum(islandID)
	if err != nil {
		return nil, err
	}

	level := &models.InterviewLevel{
		IslandID:   islandID,
		QuestionID: questionID,
		Name:       name,
		Difficulty: difficulty,
		LevelNum:   maxLevelNum + 1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.CreateLevel(level); err != nil {
		return nil, err
	}
	return level, nil
}

// UnlockTags 根据关卡题目标签为用户解锁知识点
func (s *interviewService) UnlockTags(userID, levelID uint) error {
	level, err := s.repo.GetLevelByID(levelID)
	if err != nil {
		return err
	}
	for _, tag := range level.Question.Tags {
		_ = s.repo.AddUserUnlockedTag(userID, tag.ID)
	}
	return nil
}

// GetTestCases 获取测试用例
func (s *interviewService) GetTestCases(levelID uint) ([]models.InterviewTestCase, error) {
	return s.repo.GetTestCasesByLevelID(levelID)
}

// CreateTestCase 创建测试用例
func (s *interviewService) CreateTestCase(levelID uint, input, output string, isSample bool, order int) (*models.InterviewTestCase, error) {
	tc := &models.InterviewTestCase{LevelID: levelID, Input: input, Output: output, IsSample: isSample, Order: order}
	if err := s.repo.CreateTestCase(tc); err != nil {
		return nil, err
	}
	return tc, nil
}

// UpdateTestCase 更新测试用例
func (s *interviewService) UpdateTestCase(id uint, input, output string, isSample bool, order int) error {
	tc, err := s.repo.GetTestCase(id)
	if err != nil {
		return err
	}
	tc.Input = input
	tc.Output = output
	tc.IsSample = isSample
	tc.Order = order
	return s.repo.UpdateTestCase(tc)
}

// DeleteTestCase 删除测试用例
func (s *interviewService) DeleteTestCase(id uint) error {
	return s.repo.DeleteTestCase(id)
}
