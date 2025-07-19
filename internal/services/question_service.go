package services

import (
	"ggcode/internal/models"
	"ggcode/internal/repositories"
)

// QuestionListResponse 题目列表响应
type QuestionListResponse struct {
	Data       []models.Question  `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

type QuestionService struct {
	questionRepo repositories.QuestionRepository
}

func NewQuestionService(repos *repositories.Repositories) *QuestionService {
	return &QuestionService{questionRepo: repos.Question}
}

// GetQuestions 获取题库下的题目列表
func (s *QuestionService) GetQuestions(bankID uint, page, limit int) (*QuestionListResponse, error) {
	// 从数据层获取题目列表
	result, err := s.questionRepo.GetQuestions(bankID, page, limit)
	if err != nil {
		return nil, err
	}

	// 构建响应
	response := &QuestionListResponse{
		Data: result.Data,
		Pagination: PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
			HasPrev:    page > 1,
			HasNext:    page < result.TotalPages,
		},
	}

	return response, nil
}

// GetAllQuestions 获取所有题目
func (s *QuestionService) GetAllQuestions() ([]models.Question, error) {
	return s.questionRepo.GetAllQuestions()
}

// CreateQuestion 在题库中创建题目
func (s *QuestionService) CreateQuestion(userID, bankID uint, title, URL, difficulty string, score float64) (*models.Question, error) {
	return s.questionRepo.CreateQuestion(userID, bankID, title, URL, difficulty, score)
}

// GetQuestion 获取单个题目
func (s *QuestionService) GetQuestion(questionID uint) (*models.Question, error) {
	return s.questionRepo.GetQuestion(questionID)
}

// UpdateQuestion 更新题目信息
func (s *QuestionService) UpdateQuestion(userID, questionID, bankID uint, title, URL, difficulty string) (*models.Question, error) {
	return s.questionRepo.UpdateQuestion(userID, questionID, bankID, title, URL, difficulty)
}

// UpdateQuestionWithDescription 更新题目信息（包含描述）
func (s *QuestionService) UpdateQuestionWithDescription(userID, questionID, bankID uint, title, URL, difficulty, description string) (*models.Question, error) {
	return s.questionRepo.UpdateQuestionWithDescription(userID, questionID, bankID, title, URL, difficulty, description)
}

// DeleteQuestion 删除题目
func (s *QuestionService) DeleteQuestion(userID, questionID, bankID uint) error {
	return s.questionRepo.DeleteQuestion(userID, questionID, bankID)
}
