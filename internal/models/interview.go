package models

import "time"

// InterviewIsland 面试岛模型
type InterviewIsland struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`          // 面试岛名称，如"算法基础岛"
	Description string    `json:"description"`                   // 面试岛描述
	Difficulty  string    `json:"difficulty" gorm:"not null"`    // Easy, Medium, Hard
	IsActive    bool      `json:"is_active" gorm:"default:true"` // 是否激活
	Order       int       `json:"order" gorm:"default:0"`        // 显示顺序
	ImageURL    string    `json:"image_url"`                     // 岛屿图片URL
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// InterviewLevel 面试关卡模型
type InterviewLevel struct {
	ID         uint            `json:"id" gorm:"primaryKey"`
	IslandID   uint            `json:"island_id"`
	Island     InterviewIsland `json:"island" gorm:"foreignKey:IslandID"`
	LevelNum   int             `json:"level_num" gorm:"not null"`        // 关卡序号
	Name       string          `json:"name" gorm:"not null"`             // 关卡名称
	Difficulty string          `json:"difficulty" gorm:"not null"`       // Easy, Medium, Hard
	TimeLimit  int             `json:"time_limit" gorm:"default:900"`    // 时间限制（秒），默认15分钟
	IsUnlocked bool            `json:"is_unlocked" gorm:"default:false"` // 是否解锁
	Position   string          `json:"position"`                         // 地图上的位置坐标 "x,y"

	// 题目内容（直接在关卡中存储）
	QuestionTitle       string `json:"question_title" gorm:"not null"`        // 题目标题
	QuestionDescription string `json:"question_description" gorm:"type:text"` // 题目描述
	QuestionLeetcodeURL string `json:"question_leetcode_url"`                 // LeetCode链接（可选）

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserLevelProgress 用户关卡进度
type UserLevelProgress struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"user_id"`
	User         User           `json:"user" gorm:"foreignKey:UserID"`
	LevelID      uint           `json:"level_id"`
	Level        InterviewLevel `json:"level" gorm:"foreignKey:LevelID"`
	Status       string         `json:"status" gorm:"default:locked"`   // locked, unlocked, completed
	Stars        int            `json:"stars" gorm:"default:0"`         // 获得的星数 (0-3)
	BestTime     int            `json:"best_time" gorm:"default:0"`     // 最佳完成时间（秒）
	CompletedAt  *time.Time     `json:"completed_at"`                   // 完成时间
	AttemptCount int            `json:"attempt_count" gorm:"default:0"` // 尝试次数
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// UserLevelSubmission 用户关卡提交记录
type UserLevelSubmission struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id"`
	User        User           `json:"user" gorm:"foreignKey:UserID"`
	LevelID     uint           `json:"level_id"`
	Level       InterviewLevel `json:"level" gorm:"foreignKey:LevelID"`
	Code        string         `json:"code" gorm:"type:text"`         // 提交的代码
	Language    string         `json:"language" gorm:"not null"`      // 编程语言
	Status      string         `json:"status" gorm:"not null"`        // AC, WA, TLE, MLE, CE, RE
	UseTime     int            `json:"use_time" gorm:"default:0"`     // 用时（秒）
	Memory      int            `json:"memory" gorm:"default:0"`       // 内存使用（KB）
	Score       int            `json:"score" gorm:"default:0"`        // 得分
	ErrorMsg    string         `json:"error_msg"`                     // 错误信息
	TestResults string         `json:"test_results" gorm:"type:text"` // 测试结果详情(JSON格式)
	Stars       int            `json:"stars" gorm:"default:0"`        // 本次提交获得的星数
	SubmitTime  int            `json:"submit_time" gorm:"default:0"`  // 提交时距离开始的时间（秒）
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// InterviewTestCase 面试题测试用例
type InterviewTestCase struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	LevelID   uint           `json:"level_id"`
	Level     InterviewLevel `json:"level" gorm:"foreignKey:LevelID"`
	Input     string         `json:"input" gorm:"type:text"`         // 输入数据
	Output    string         `json:"output" gorm:"type:text"`        // 期望输出
	IsSample  bool           `json:"is_sample" gorm:"default:false"` // 是否为样例
	Order     int            `json:"order" gorm:"default:0"`         // 顺序
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
