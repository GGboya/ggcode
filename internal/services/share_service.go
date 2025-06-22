package services

import (
	"errors"
	"ggcode/internal/database"
	"ggcode/internal/repositories"
)

type ShareService struct {
	shareRepo repositories.ShareRepository
}

func NewShareService(repos *repositories.Repositories) *ShareService {
	return &ShareService{
		shareRepo: repos.Share,
	}
}

// ShareQuestionBank 共享题库
func (s *ShareService) ShareQuestionBank(bankID, userID uint) error {
	// 检查题库权限
	hasPermission, err := s.shareRepo.CheckQuestionBankOwnership(bankID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return errors.New("题库不存在或无权限操作")
	}

	return s.shareRepo.ShareQuestionBank(bankID)
}

// UnshareQuestionBank 取消共享题库
func (s *ShareService) UnshareQuestionBank(bankID, userID uint) error {
	// 检查题库权限
	hasPermission, err := s.shareRepo.CheckQuestionBankOwnership(bankID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return errors.New("题库不存在或无权限操作")
	}

	return s.shareRepo.UnshareQuestionBank(bankID)
}

// StarQuestionBank 收藏题库
func (s *ShareService) StarQuestionBank(bankID, userID uint) error {
	// 检查题库是否存在且为共享状态
	isShared, err := s.shareRepo.CheckQuestionBankShared(bankID)
	if err != nil {
		return err
	}
	if !isShared {
		return errors.New("题库不存在或未共享")
	}

	// 检查是否已经Star
	isStarred, err := s.shareRepo.CheckQuestionBankStarred(bankID, userID)
	if err != nil {
		return err
	}
	if isStarred {
		return errors.New("已经Star过这个题库")
	}

	return s.shareRepo.StarQuestionBank(bankID, userID)
}

// UnstarQuestionBank 取消收藏题库
func (s *ShareService) UnstarQuestionBank(bankID, userID uint) error {
	// 检查是否已经Star
	isStarred, err := s.shareRepo.CheckQuestionBankStarred(bankID, userID)
	if err != nil {
		return err
	}
	if !isStarred {
		return errors.New("尚未Star该题库")
	}

	return s.shareRepo.UnstarQuestionBank(bankID, userID)
}

// ForkQuestionBank Fork题库
func (s *ShareService) ForkQuestionBank(bankID, userID uint) (*database.QuestionBank, error) {
	// 检查题库是否存在且为共享状态
	isShared, err := s.shareRepo.CheckQuestionBankShared(bankID)
	if err != nil {
		return nil, err
	}
	if !isShared {
		return nil, errors.New("题库不存在或未共享")
	}

	// 检查用户是否已经Fork过这个题库
	isForked, err := s.shareRepo.CheckQuestionBankForked(bankID, userID)
	if err != nil {
		return nil, err
	}
	if isForked {
		return nil, errors.New("已经Fork过这个题库")
	}

	return s.shareRepo.ForkQuestionBank(bankID, userID)
}

// GetUserStarredBanks 获取用户收藏的题库
func (s *ShareService) GetUserStarredBanks(userID uint, page, limit int) ([]database.QuestionBank, int64, error) {
	return s.shareRepo.GetUserStarredBanks(userID, page, limit)
}
