package repositories

import (
	"ggcode/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	user := &models.User{}
	if err := r.db.Where("username = ?", username).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetByUsernameOrEmail(username, email string) (*models.User, error) {
	user := &models.User{}
	if err := r.db.Where("username = ? OR email = ?", username, email).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}
