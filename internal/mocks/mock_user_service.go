package mocks

import (
	"ggcode/internal/models"

	"github.com/stretchr/testify/mock"
)

// MockUserServiceInterface 是 UserServiceInterface 的 mock 实现
type MockUserServiceInterface struct {
	mock.Mock
}

func (m *MockUserServiceInterface) Register(username, email, password string) (*models.User, string, error) {
	args := m.Called(username, email, password)
	user, _ := args.Get(0).(*models.User)
	return user, args.String(1), args.Error(2)
}

func (m *MockUserServiceInterface) Login(username, password string) (*models.User, string, error) {
	args := m.Called(username, password)
	user, _ := args.Get(0).(*models.User)
	return user, args.String(1), args.Error(2)
}

func (m *MockUserServiceInterface) IsAdmin(userID uint) (bool, error) {
	args := m.Called(userID)
	return args.Bool(0), args.Error(1)
}
