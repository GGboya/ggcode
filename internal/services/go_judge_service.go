package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ggcode/internal/pkg/logger"
	"io"
	"net/http"
	"strings"
	"time"
)

// GoJudgeRequest 表示我们应用的评测请求
type GoJudgeRequest struct {
	Language    string `json:"language"`
	Code        string `json:"code"`
	Input       string `json:"input,omitempty"`
	TimeLimit   int64  `json:"timeLimit,omitempty"`   // 毫秒
	MemoryLimit int64  `json:"memoryLimit,omitempty"` // KB
}

// GoJudgeResponse 表示我们应用的评测响应
type GoJudgeResponse struct {
	Status     string `json:"status"`
	ExitStatus int    `json:"exitStatus"`
	Time       int64  `json:"time"`   // 纳秒
	Memory     int64  `json:"memory"` // 字节
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Error      string `json:"error,omitempty"`
}

// GoJudgeAPIRequest go-judge 原生 API 请求结构
type GoJudgeAPIRequest struct {
	Cmd []GoJudgeCommand `json:"cmd"`
}

// GoJudgeCommand go-judge 命令结构
type GoJudgeCommand struct {
	Args          []string               `json:"args"`
	Env           []string               `json:"env,omitempty"`
	Files         []interface{}          `json:"files,omitempty"`
	CPULimit      int64                  `json:"cpuLimit,omitempty"`
	MemoryLimit   int64                  `json:"memoryLimit,omitempty"`
	ProcLimit     int                    `json:"procLimit,omitempty"`
	CopyIn        map[string]interface{} `json:"copyIn,omitempty"`
	CopyOut       []string               `json:"copyOut,omitempty"`
	CopyOutCached []string               `json:"copyOutCached,omitempty"`
}

// GoJudgeAPIResponse go-judge 原生 API 响应结构
type GoJudgeAPIResponse []GoJudgeResult

// GoJudgeResult go-judge 单个结果
type GoJudgeResult struct {
	Status     string            `json:"status"`
	ExitStatus int               `json:"exitStatus"`
	Time       int64             `json:"time"`
	Memory     int64             `json:"memory"`
	RunTime    int64             `json:"runTime"`
	Files      map[string]string `json:"files,omitempty"`
	FileIds    map[string]string `json:"fileIds,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// GoJudgeService 提供 go-judge 集成服务
type GoJudgeService struct {
	baseURL    string
	httpClient *http.Client
}

// NewGoJudgeService 创建新的 go-judge 服务
func NewGoJudgeService(baseURL string) *GoJudgeService {
	if baseURL == "" {
		baseURL = "http://localhost:5050" // go-judge 默认地址
	}

	return &GoJudgeService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // 增加超时时间以支持编译
		},
	}
}

// ExecuteCode 执行代码（编译+运行）
func (s *GoJudgeService) ExecuteCode(req *GoJudgeRequest) (*GoJudgeResponse, error) {
	language := strings.ToLower(req.Language)

	// 对于需要编译的语言，执行两步过程
	if s.needsCompilation(language) {
		return s.executeCompiledLanguage(req)
	}

	// 对于解释型语言，直接执行
	return s.executeInterpretedLanguage(req)
}

// needsCompilation 判断语言是否需要编译
func (s *GoJudgeService) needsCompilation(language string) bool {
	compiledLanguages := []string{"cpp", "c++", "c", "java", "go"}
	for _, lang := range compiledLanguages {
		if language == lang {
			return true
		}
	}
	return false
}

// executeCompiledLanguage 执行需要编译的语言
func (s *GoJudgeService) executeCompiledLanguage(req *GoJudgeRequest) (*GoJudgeResponse, error) {
	// 第一步：编译
	compileReq := s.buildCompileRequest(req)
	compileResp, err := s.sendRequest(compileReq)
	if err != nil {
		return nil, fmt.Errorf("编译请求失败: %v", err)
	}

	if len(compileResp) == 0 {
		return nil, fmt.Errorf("编译响应为空")
	}

	compileResult := compileResp[0]

	// 检查编译是否成功
	if compileResult.Status != "Accepted" {
		return &GoJudgeResponse{
			Status:     "Compile Error",
			ExitStatus: compileResult.ExitStatus,
			Time:       compileResult.Time,
			Memory:     compileResult.Memory,
			Stdout:     compileResult.Files["stdout"],
			Stderr:     compileResult.Files["stderr"],
			Error:      compileResult.Error,
		}, nil
	}

	// 获取编译后的文件ID
	var executableFileId string
	for _, fileId := range compileResult.FileIds {
		executableFileId = fileId
		break
	}

	if executableFileId == "" {
		return nil, fmt.Errorf("编译成功但未找到可执行文件")
	}

	// 第二步：运行
	runReq := s.buildRunRequest(req, executableFileId)
	runResp, err := s.sendRequest(runReq)
	if err != nil {
		// 清理编译文件
		s.cleanupFile(executableFileId)
		return nil, fmt.Errorf("运行请求失败: %v", err)
	}

	if len(runResp) == 0 {
		s.cleanupFile(executableFileId)
		return nil, fmt.Errorf("运行响应为空")
	}

	runResult := runResp[0]

	// 清理编译文件
	s.cleanupFile(executableFileId)

	// 转换结果
	return s.convertResult(runResult), nil
}

// executeInterpretedLanguage 执行解释型语言
func (s *GoJudgeService) executeInterpretedLanguage(req *GoJudgeRequest) (*GoJudgeResponse, error) {
	judgeReq := s.buildInterpretedRequest(req)
	resp, err := s.sendRequest(judgeReq)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %v", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("执行响应为空")
	}

	return s.convertResult(resp[0]), nil
}

// buildCompileRequest 构建编译请求
func (s *GoJudgeService) buildCompileRequest(req *GoJudgeRequest) *GoJudgeAPIRequest {
	language := strings.ToLower(req.Language)

	var args []string
	var sourceFile string
	var executableFile string

	switch language {
	case "cpp", "c++":
		args = []string{"/usr/bin/g++", "-O2", "-std=c++17", "main.cpp", "-o", "main"}
		sourceFile = "main.cpp"
		executableFile = "main"
	case "c":
		args = []string{"/usr/bin/gcc", "-O2", "main.c", "-o", "main"}
		sourceFile = "main.c"
		executableFile = "main"
	case "java":
		args = []string{"/usr/bin/javac", "-encoding", "UTF-8", "Main.java"}
		sourceFile = "Main.java"
		executableFile = "Main.class"
	case "go":
		args = []string{"/go/bin/go", "build", "-o", "main", "main.go"}
		sourceFile = "main.go"
		executableFile = "main"
	}

	// 设置环境变量
	env := []string{"PATH=/usr/bin:/bin"}
	if language == "go" {
		env = append(env, "GOCACHE=/tmp", "GOPATH=/tmp", "PATH=/go/bin:/usr/bin:/bin")
	}

	return &GoJudgeAPIRequest{
		Cmd: []GoJudgeCommand{
			{
				Args: args,
				Env:  env,
				Files: []interface{}{
					map[string]string{"content": ""},
					map[string]interface{}{"name": "stdout", "max": 10240},
					map[string]interface{}{"name": "stderr", "max": 10240},
				},
				CPULimit: func() int64 {
					if language == "go" {
						return 15000000000 // 15 秒
					}
					return 10000000000 // 其他语言保持 10 秒
				}(),
				MemoryLimit: func() int64 {
					if language == "go" {
						return 536870912 // 512MB for Go compilation
					} else if language == "java" {
						return 268435456 // 256MB for Java compilation
					}
					return 134217728 // 128MB for other languages
				}(),
				ProcLimit: 50,
				CopyIn: map[string]interface{}{
					sourceFile: map[string]interface{}{
						"content": req.Code,
					},
				},
				CopyOut:       []string{"stdout", "stderr"},
				CopyOutCached: []string{executableFile},
			},
		},
	}
}

// buildRunRequest 构建运行请求
func (s *GoJudgeService) buildRunRequest(req *GoJudgeRequest, executableFileId string) *GoJudgeAPIRequest {
	language := strings.ToLower(req.Language)

	timeLimit := req.TimeLimit
	if timeLimit == 0 {
		timeLimit = 2000 // 默认2秒
	}

	memoryLimit := req.MemoryLimit
	if memoryLimit == 0 {
		memoryLimit = 128 * 1024 // 默认128MB
	}

	var args []string
	var executableFile string

	switch language {
	case "cpp", "c++", "c", "go":
		args = []string{"./main"}
		executableFile = "main"
	case "java":
		args = []string{"/usr/bin/java", "-Xmx128m", "-Xms32m", "Main"}
		executableFile = "Main.class"
	}

	// 构建 files 数组，stdin 必须在第一个位置
	files := []interface{}{
		map[string]interface{}{"name": "stdout", "max": 10240},
		map[string]interface{}{"name": "stderr", "max": 10240},
	}

	// 如果有输入，添加到 files 的开头（作为 stdin）
	if req.Input != "" {
		files = append([]interface{}{map[string]string{"content": req.Input}}, files...)
	} else {
		files = append([]interface{}{map[string]string{"content": ""}}, files...)
	}

	// 为 Java 设置更大的内存限制
	if strings.Contains(strings.Join(args, " "), "java") {
		memoryLimit = 256 * 1024 // 256MB for Java runtime
	}

	return &GoJudgeAPIRequest{
		Cmd: []GoJudgeCommand{
			{
				Args:        args,
				Env:         []string{"PATH=/usr/bin:/bin"},
				Files:       files,
				CPULimit:    timeLimit * 1000000, // 毫秒转纳秒
				MemoryLimit: memoryLimit * 1024,  // KB转字节
				ProcLimit:   50,
				CopyIn: map[string]interface{}{
					executableFile: map[string]interface{}{
						"fileId": executableFileId,
					},
				},
			},
		},
	}
}

// buildInterpretedRequest 构建解释型语言请求
func (s *GoJudgeService) buildInterpretedRequest(req *GoJudgeRequest) *GoJudgeAPIRequest {
	language := strings.ToLower(req.Language)

	timeLimit := req.TimeLimit
	if timeLimit == 0 {
		timeLimit = 2000 // 默认2秒
	}

	memoryLimit := req.MemoryLimit
	if memoryLimit == 0 {
		memoryLimit = 128 * 1024 // 默认128MB
	}

	var args []string
	var sourceFile string

	switch language {
	case "python", "python3":
		args = []string{"/usr/bin/python3", "main.py"}
		sourceFile = "main.py"
	case "javascript", "js":
		args = []string{"/usr/bin/node", "main.js"}
		sourceFile = "main.js"
	}

	// 构建 files 数组，stdin 必须在第一个位置
	files := []interface{}{
		map[string]interface{}{"name": "stdout", "max": 10240},
		map[string]interface{}{"name": "stderr", "max": 10240},
	}

	// 如果有输入，添加到 files 的开头（作为 stdin）
	if req.Input != "" {
		files = append([]interface{}{map[string]string{"content": req.Input}}, files...)
	} else {
		files = append([]interface{}{map[string]string{"content": ""}}, files...)
	}

	return &GoJudgeAPIRequest{
		Cmd: []GoJudgeCommand{
			{
				Args:        args,
				Env:         []string{"PATH=/usr/bin:/bin"},
				Files:       files,
				CPULimit:    timeLimit * 1000000, // 毫秒转纳秒
				MemoryLimit: memoryLimit * 1024,  // KB转字节
				ProcLimit:   50,
				CopyIn: map[string]interface{}{
					sourceFile: map[string]interface{}{
						"content": req.Code,
					},
				},
			},
		},
	}
}

// sendRequest 发送请求到 go-judge
func (s *GoJudgeService) sendRequest(req *GoJudgeAPIRequest) (GoJudgeAPIResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 添加调试日志
	logger.Infof("[GoJudge] 发送请求到 %s/run", s.baseURL)
	logger.Infof("[GoJudge] 请求数据: %s", string(jsonData))

	resp, err := s.httpClient.Post(s.baseURL+"/run", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP 错误 %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result GoJudgeAPIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 添加调试日志
	logger.Infof("[GoJudge] 响应数据: %s", string(body))

	return result, nil
}

// convertResult 转换 go-judge 结果为我们的格式
func (s *GoJudgeService) convertResult(result GoJudgeResult) *GoJudgeResponse {
	return &GoJudgeResponse{
		Status:     result.Status,
		ExitStatus: result.ExitStatus,
		Time:       result.Time,
		Memory:     result.Memory,
		Stdout:     result.Files["stdout"],
		Stderr:     result.Files["stderr"],
		Error:      result.Error,
	}
}

// cleanupFile 清理缓存文件
func (s *GoJudgeService) cleanupFile(fileId string) {
	if fileId == "" {
		return
	}

	req, err := http.NewRequest("DELETE", s.baseURL+"/file/"+fileId, nil)
	if err != nil {
		return
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// HealthCheck 健康检查
func (s *GoJudgeService) HealthCheck() error {
	resp, err := s.httpClient.Get(s.baseURL + "/version")
	if err != nil {
		return fmt.Errorf("go-judge 服务不可用: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("go-judge 服务状态异常: %d", resp.StatusCode)
	}

	return nil
}
