package services

import (
	"fmt"
	"ggcode/internal/models"
	"ggcode/internal/repositories"
)

// QuestionBankWithStarStatus 带有Star状态的题库
type QuestionBankWithStarStatus struct {
	models.QuestionBank
	IsStarred bool `json:"is_starred"`
}

// QuestionBankListResponse 题库列表响应
type QuestionBankListResponse struct {
	Data       []QuestionBankWithStarStatus `json:"data"`
	Pagination PaginationResponse           `json:"pagination"`
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasPrev    bool  `json:"has_prev"`
	HasNext    bool  `json:"has_next"`
}

type QuestionBankService struct {
	questionBankRepo   repositories.QuestionBankRepository
	contestProblemRepo repositories.ContestProblemRepository
	questionRepo       repositories.QuestionRepository // 新增
}

func NewQuestionBankService(repos *repositories.Repositories) *QuestionBankService {
	return &QuestionBankService{
		questionBankRepo:   repos.QuestionBank,
		contestProblemRepo: repos.ContestProblem,
		questionRepo:       repos.Question, // 新增
	}
}

// GetQuestionBanks 获取题库列表
func (s *QuestionBankService) GetQuestionBanks(userID uint, bankType, sortBy string, page, limit int) (*QuestionBankListResponse, error) {
	// 构建查询选项
	options := repositories.QuestionBankQueryOptions{
		UserID:   userID,
		BankType: bankType,
		SortBy:   sortBy,
		Page:     page,
		Limit:    limit,
	}

	// 从数据层获取题库列表
	result, err := s.questionBankRepo.GetQuestionBanks(options)
	if err != nil {
		return nil, err
	}

	// 获取用户Star状态（个人题库不需要查询Star状态）
	var starredBankIDs []uint
	if bankType != "personal" && len(result.Data) > 0 {
		// 提取题库ID列表
		bankIDs := make([]uint, len(result.Data))
		for i, bank := range result.Data {
			bankIDs[i] = bank.ID
		}

		// 查询Star状态
		starredBankIDs, err = s.questionBankRepo.GetStarredBankIDs(userID, bankIDs)
		if err != nil {
			return nil, err
		}
	}

	// 构建带有Star状态的题库列表
	banksWithStarStatus := make([]QuestionBankWithStarStatus, len(result.Data))
	banksWithStarMap := make(map[uint]bool)

	for _, bankID := range starredBankIDs {
		banksWithStarMap[bankID] = true
	}

	for i, bank := range result.Data {
		banksWithStarStatus[i] = QuestionBankWithStarStatus{
			QuestionBank: bank,
			IsStarred:    banksWithStarMap[bank.ID],
		}
	}

	// 构建响应
	response := &QuestionBankListResponse{
		Data: banksWithStarStatus,
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

func (s *QuestionBankService) CreateQuestionBank(name, description string, userID uint) (*models.QuestionBank, error) {
	return s.questionBankRepo.CreateQuestionBank(name, description, userID)
}

func (s *QuestionBankService) UpdateQuestionBank(bankID, userID uint, updateData repositories.QuestionBankUpdateData) error {
	return s.questionBankRepo.UpdateQuestionBank(bankID, userID, updateData)
}

func (s *QuestionBankService) DeleteQuestionBank(bankID, userID uint) error {
	return s.questionBankRepo.DeleteQuestionBank(bankID, userID)
}

// GetOrCreateWrongQuestionBook 获取或创建用户的错题本
func (s *QuestionBankService) GetOrCreateWrongQuestionBook(userID uint) (*models.QuestionBank, error) {
	return s.questionBankRepo.GetOrCreateWrongQuestionBook(userID)
}

// AddQuestionToWrongBook 添加题目到错题本
func (s *QuestionBankService) AddQuestionToWrongBook(userID, questionID uint) error {
	return s.questionBankRepo.AddQuestionToWrongBook(userID, questionID)
}

// CreateQuestionBankWithImport 创建题库并可选导入比赛题目
func (s *QuestionBankService) CreateQuestionBankWithImport(name, description string, userID uint, source string, minScore, maxScore int) (*models.QuestionBank, error) {
	bank, err := s.questionBankRepo.CreateQuestionBank(name, description, userID)
	if err != nil {
		return nil, err
	}
	if source != "" && s.contestProblemRepo != nil {
		problems, err := s.contestProblemRepo.ListContestProblems(source, minScore, maxScore)
		if err != nil {
			return bank, nil // 不影响主流程
		}
		var questions []models.Question
		for _, p := range problems {
			questions = append(questions, models.Question{
				Title:          p.Title,
				URL:            p.URL,
				QuestionBankID: bank.ID,
			})
		}
		fmt.Println(len(questions))
		_ = s.questionRepo.BatchCreateQuestions(questions)
	}
	return bank, nil
}
