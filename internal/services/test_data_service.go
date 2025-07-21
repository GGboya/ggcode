package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// TestDataService 测试数据管理服务
type TestDataService struct {
	judgeDataRoot string // 评测数据根目录，如：/judge/data/
}

// TestDataConfig 测试数据配置
type TestDataConfig struct {
	LevelID     uint `yaml:"level_id"`
	TimeLimit   int  `yaml:"time_limit"`   // 时间限制（秒）
	MemoryLimit int  `yaml:"memory_limit"` // 内存限制（MB）
	TestCases   []struct {
		InputFile  string `yaml:"input_file"`
		OutputFile string `yaml:"output_file"`
		IsSample   bool   `yaml:"is_sample"`
	} `yaml:"test_cases"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

// NewTestDataService 创建测试数据管理服务
func NewTestDataService(judgeDataRoot string) *TestDataService {
	if judgeDataRoot == "" {
		judgeDataRoot = "/judge/data"
	}

	// 确保数据根目录存在
	if err := os.MkdirAll(judgeDataRoot, 0755); err != nil {
		return nil
	}

	return &TestDataService{
		judgeDataRoot: judgeDataRoot,
	}
}

// GetLevelDataPath 获取关卡测试数据目录路径
func (tds *TestDataService) GetLevelDataPath(levelID uint) string {
	return filepath.Join(tds.judgeDataRoot, strconv.Itoa(int(levelID)))
}

// CreateLevelDataDir 创建关卡测试数据目录
func (tds *TestDataService) CreateLevelDataDir(levelID uint) error {
	dataPath := tds.GetLevelDataPath(levelID)
	return os.MkdirAll(dataPath, 0755)
}

// SaveTestCase 保存测试用例文件
func (tds *TestDataService) SaveTestCase(levelID uint, caseNum int, inputData, outputData string) error {
	dataPath := tds.GetLevelDataPath(levelID)

	// 确保目录存在
	if err := tds.CreateLevelDataDir(levelID); err != nil {
		return err
	}

	// 保存输入文件
	inputFile := filepath.Join(dataPath, fmt.Sprintf("%d.in", caseNum))
	if err := os.WriteFile(inputFile, []byte(inputData), 0644); err != nil {
		return fmt.Errorf("保存输入文件失败: %v", err)
	}

	// 保存输出文件
	outputFile := filepath.Join(dataPath, fmt.Sprintf("%d.ans", caseNum))
	if err := os.WriteFile(outputFile, []byte(outputData), 0644); err != nil {
		return fmt.Errorf("保存输出文件失败: %v", err)
	}

	return nil
}

// SaveTestCaseFromFile 从上传的文件保存测试用例
func (tds *TestDataService) SaveTestCaseFromFile(levelID uint, caseNum int, inputFileData, outputFileData []byte) error {
	dataPath := tds.GetLevelDataPath(levelID)

	// 确保目录存在
	if err := tds.CreateLevelDataDir(levelID); err != nil {
		return err
	}

	// 保存输入文件
	inputFile := filepath.Join(dataPath, fmt.Sprintf("%d.in", caseNum))
	if err := os.WriteFile(inputFile, inputFileData, 0644); err != nil {
		return fmt.Errorf("保存输入文件失败: %v", err)
	}

	// 保存输出文件
	outputFile := filepath.Join(dataPath, fmt.Sprintf("%d.ans", caseNum))
	if err := os.WriteFile(outputFile, outputFileData, 0644); err != nil {
		return fmt.Errorf("保存输出文件失败: %v", err)
	}

	return nil
}

// SaveConfig 保存配置文件
func (tds *TestDataService) SaveConfig(levelID uint, config *TestDataConfig) error {
	dataPath := tds.GetLevelDataPath(levelID)
	configFile := filepath.Join(dataPath, "config.yaml")

	config.UpdatedAt = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	return os.WriteFile(configFile, data, 0644)
}

// LoadConfig 加载配置文件
func (tds *TestDataService) LoadConfig(levelID uint) (*TestDataConfig, error) {
	dataPath := tds.GetLevelDataPath(levelID)
	configFile := filepath.Join(dataPath, "config.yaml")

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config TestDataConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// GetTestCaseFiles 获取测试用例文件列表
func (tds *TestDataService) GetTestCaseFiles(levelID uint) ([]struct {
	CaseNum    int
	InputFile  string
	OutputFile string
}, error) {
	dataPath := tds.GetLevelDataPath(levelID)

	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}

	testCases := make(map[int]struct {
		CaseNum    int
		InputFile  string
		OutputFile string
	})

	// 扫描所有 .in 和 .ans 文件
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) == ".in" {
			base := name[:len(name)-3] // 去掉 .in
			if caseNum, err := strconv.Atoi(base); err == nil {
				tc := testCases[caseNum]
				tc.CaseNum = caseNum
				tc.InputFile = filepath.Join(dataPath, name)
				testCases[caseNum] = tc
			}
		} else if filepath.Ext(name) == ".ans" {
			base := name[:len(name)-4] // 去掉 .ans
			if caseNum, err := strconv.Atoi(base); err == nil {
				tc := testCases[caseNum]
				tc.CaseNum = caseNum
				tc.OutputFile = filepath.Join(dataPath, name)
				testCases[caseNum] = tc
			}
		}
	}

	// 转换为切片
	var result []struct {
		CaseNum    int
		InputFile  string
		OutputFile string
	}

	for _, tc := range testCases {
		if tc.InputFile != "" && tc.OutputFile != "" {
			result = append(result, tc)
		}
	}

	return result, nil
}

// GetTestCaseData 读取测试用例数据
func (tds *TestDataService) GetTestCaseData(levelID uint, caseNum int) (inputData, outputData string, err error) {
	dataPath := tds.GetLevelDataPath(levelID)

	inputFile := filepath.Join(dataPath, fmt.Sprintf("%d.in", caseNum))
	outputFile := filepath.Join(dataPath, fmt.Sprintf("%d.ans", caseNum))

	// 读取输入数据
	inputBytes, err := os.ReadFile(inputFile)
	if err != nil {
		return "", "", fmt.Errorf("读取输入文件失败: %v", err)
	}

	// 读取输出数据
	outputBytes, err := os.ReadFile(outputFile)
	if err != nil {
		return "", "", fmt.Errorf("读取输出文件失败: %v", err)
	}

	return string(inputBytes), string(outputBytes), nil
}

// DeleteLevelData 删除关卡测试数据
func (tds *TestDataService) DeleteLevelData(levelID uint) error {
	dataPath := tds.GetLevelDataPath(levelID)
	return os.RemoveAll(dataPath)
}

// DeleteTestCase 删除单个测试用例
func (tds *TestDataService) DeleteTestCase(levelID uint, caseNum int) error {
	dataPath := tds.GetLevelDataPath(levelID)

	inputFile := filepath.Join(dataPath, fmt.Sprintf("%d.in", caseNum))
	outputFile := filepath.Join(dataPath, fmt.Sprintf("%d.ans", caseNum))

	// 删除输入文件
	if err := os.Remove(inputFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除输入文件失败: %v", err)
	}

	// 删除输出文件
	if err := os.Remove(outputFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除输出文件失败: %v", err)
	}

	return nil
}

// InitializeFromDatabase 从数据库初始化测试数据
func (tds *TestDataService) InitializeFromDatabase(levelID uint, testCases []struct {
	Input    string
	Output   string
	IsSample bool
	Order    int
}) error {
	// 创建配置
	config := &TestDataConfig{
		LevelID:     levelID,
		TimeLimit:   5,   // 默认5秒
		MemoryLimit: 128, // 默认128MB
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存测试用例文件
	for i, testCase := range testCases {
		caseNum := i + 1

		if err := tds.SaveTestCase(levelID, caseNum, testCase.Input, testCase.Output); err != nil {
			return fmt.Errorf("保存测试用例 %d 失败: %v", caseNum, err)
		}

		// 添加到配置
		config.TestCases = append(config.TestCases, struct {
			InputFile  string `yaml:"input_file"`
			OutputFile string `yaml:"output_file"`
			IsSample   bool   `yaml:"is_sample"`
		}{
			InputFile:  fmt.Sprintf("%d.in", caseNum),
			OutputFile: fmt.Sprintf("%d.ans", caseNum),
			IsSample:   testCase.IsSample,
		})
	}

	// 保存配置文件
	return tds.SaveConfig(levelID, config)
}
