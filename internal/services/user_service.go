package services

import (
	"errors"
	"ggcode/internal/config"
	"ggcode/internal/middleware"
	"ggcode/internal/models"
	"ggcode/internal/repositories"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserServiceInterface 定义用户服务接口，便于 controller 解耦和单元测试
//go:generate mockgen -source=user_service.go -destination=../mocks/services/mock_user_service.go -package=services

type UserServiceInterface interface {
	Register(username, email, password string) (*models.User, string, error)
	Login(username, password string) (*models.User, string, error)
	IsAdmin(userID uint) (bool, error)
	// 可以根据需要补充更多接口方法
}

type UserService struct {
	userRepo repositories.UserRepository
	config   *config.Config
}

func NewUserService(userRepo repositories.UserRepository, cfg *config.Config) *UserService {
	return &UserService{
		userRepo: userRepo,
		config:   cfg,
	}
}

// Register 用户注册
func (s *UserService) Register(username, email, password string) (*models.User, string, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsernameOrEmail(username, email)
	if err == nil && existingUser != nil {
		return nil, "", errors.New("用户名或邮箱已存在")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", errors.New("密码加密失败")
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
	}
	// 让repository层创建用户
	err = s.userRepo.Create(user)
	if err != nil {
		return nil, "", errors.New("创建用户失败")
	}

	// 生成token
	token, err := middleware.GenerateToken(user.ID, user.Username, s.config)
	if err != nil {
		return nil, "", errors.New("生成token失败")
	}

	return user, token, nil
}

// Login 用户登录
func (s *UserService) Login(username, password string) (*models.User, string, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	// 生成token
	token, err := middleware.GenerateToken(user.ID, user.Username, s.config)
	if err != nil {
		return nil, "", errors.New("生成token失败")
	}

	return user, token, nil
}

// IsAdmin 判断用户是否为管理员
func (s *UserService) IsAdmin(userID uint) (bool, error) {
	return s.userRepo.IsAdmin(userID)
}

// // GetUserByID 通过ID获取用户信息
// func (s *UserService) GetUserByID(id uint) (*models.User, error) {
// 	return s.userRepo.GetByID(id)
// }

// UserService 实现 UserServiceInterface
var _ UserServiceInterface = (*UserService)(nil)
