package repositories

import (
	"ggcode/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByUsername(username string) (*models.User, error)
	GetByUsernameOrEmail(username, email string) (*models.User, error)
	IsAdmin(userID uint) (bool, error)
}

type userRepository struct {
	db *gorm.DB
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	user := &models.User{}
	if err := r.db.Where("username = ?", username).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByUsernameOrEmail(username, email string) (*models.User, error) {
	user := &models.User{}
	if err := r.db.Where("username = ? OR email = ?", username, email).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) IsAdmin(userID uint) (bool, error) {
	var user models.User
	if err := r.db.Select("is_admin").Where("id = ?", userID).First(&user).Error; err != nil {
		return false, err
	}
	return user.IsAdmin, nil
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}
