package services

import (
	"errors"
	"ggcode/internal/mocks"
	"ggcode/internal/models"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var _ = Describe("UserService", func() {
	var (
		mockCtrl    *gomock.Controller
		mockRepo    *mocks.MockUserRepository
		userService *UserService
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockRepo = mocks.NewMockUserRepository(mockCtrl)
		userService = &UserService{
			userRepo: mockRepo,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Register", func() {
		Context("when registration is successful", func() {
			It("should create user and return token", func() {
				// Arrange
				username := "testuser"
				email := "test@example.com"
				password := "123456"

				// Mock: 用户名不存在
				mockRepo.EXPECT().
					GetByUsernameOrEmail(username, email).
					Return(nil, gorm.ErrRecordNotFound)

				// Mock: 创建用户成功
				mockRepo.EXPECT().
					Create(gomock.Any()).
					Return(nil)

				// Act
				user, token, err := userService.Register(username, email, password)

				// Assert
				Expect(err).To(BeNil())
				Expect(user).NotTo(BeNil())
				Expect(user.Username).To(Equal(username))
				Expect(user.Email).To(Equal(email))
				Expect(token).NotTo(BeEmpty())

				// 验证密码已加密
				Expect(user.Password).NotTo(Equal(password))
				err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
				Expect(err).To(BeNil())
			})
		})

		Context("when username already exists", func() {
			It("should return error", func() {
				// Arrange
				existingUser := &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				}
				mockRepo.EXPECT().
					GetByUsernameOrEmail("testuser", "test@example.com").
					Return(existingUser, nil)

				// Act
				user, token, err := userService.Register("testuser", "test@example.com", "123456")

				// Assert
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("用户名或邮箱已存在"))
				Expect(user).To(BeNil())
				Expect(token).To(BeEmpty())
			})
		})

		Context("when repository error occurs", func() {
			It("should return error", func() {
				// Arrange
				mockRepo.EXPECT().
					GetByUsernameOrEmail("testuser", "test@example.com").
					Return(nil, errors.New("database error"))

				// Act
				user, token, err := userService.Register("testuser", "test@example.com", "123456")

				// Assert
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("database error"))
				Expect(user).To(BeNil())
				Expect(token).To(BeEmpty())
			})
		})

		Context("when password encryption fails", func() {
			It("should return encryption error", func() {
				// Arrange - 使用一个会导致bcrypt失败的密码（超长密码）
				veryLongPassword := string(make([]byte, 100)) // 100字节的密码
				mockRepo.EXPECT().
					GetByUsernameOrEmail("testuser", "test@example.com").
					Return(nil, gorm.ErrRecordNotFound)

				// Act
				user, token, err := userService.Register("testuser", "test@example.com", veryLongPassword)

				// Assert
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("密码加密失败"))
				Expect(user).To(BeNil())
				Expect(token).To(BeEmpty())
			})
		})

		Context("when user creation fails", func() {
			It("should return creation error", func() {
				// Arrange
				mockRepo.EXPECT().
					GetByUsernameOrEmail("testuser", "test@example.com").
					Return(nil, gorm.ErrRecordNotFound)

				mockRepo.EXPECT().
					Create(gomock.Any()).
					Return(errors.New("creation failed"))

				// Act
				user, token, err := userService.Register("testuser", "test@example.com", "123456")

				// Assert
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("创建用户失败"))
				Expect(user).To(BeNil())
				Expect(token).To(BeEmpty())
			})
		})
	})

	Describe("Login", func() {
		Context("when login is successful", func() {
			It("should return user and token", func() {
				// Arrange
				username := "testuser"
				password := "123456"
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

				existingUser := &models.User{
					ID:       1,
					Username: username,
					Email:    "test@example.com",
					Password: string(hashedPassword),
				}

				mockRepo.EXPECT().
					GetByUsername(username).
					Return(existingUser, nil)

				// Act
				user, token, err := userService.Login(username, password)

				// Assert
				Expect(err).To(BeNil())
				Expect(user).NotTo(BeNil())
				Expect(user.Username).To(Equal(username))
				Expect(token).NotTo(BeEmpty())
			})
		})

		Context("when user not found", func() {
			It("should return error", func() {
				// Arrange
				mockRepo.EXPECT().
					GetByUsername("nonexistent").
					Return(nil, gorm.ErrRecordNotFound)

				// Act
				user, token, err := userService.Login("nonexistent", "123456")

				// Assert
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("用户名或密码错误"))
				Expect(user).To(BeNil())
				Expect(token).To(BeEmpty())
			})
		})

		Context("when password is incorrect", func() {
			It("should return error", func() {
				// Arrange
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.DefaultCost)
				existingUser := &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
					Password: string(hashedPassword),
				}

				mockRepo.EXPECT().
					GetByUsername("testuser").
					Return(existingUser, nil)

				// Act
				user, token, err := userService.Login("testuser", "wrongpass")

				// Assert
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("用户名或密码错误"))
				Expect(user).To(BeNil())
				Expect(token).To(BeEmpty())
			})
		})

		Context("when repository error occurs", func() {
			It("should return error", func() {
				// Arrange
				mockRepo.EXPECT().
					GetByUsername("testuser").
					Return(nil, errors.New("database error"))

				// Act
				user, token, err := userService.Login("testuser", "123456")

				// Assert
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("用户名或密码错误"))
				Expect(user).To(BeNil())
				Expect(token).To(BeEmpty())
			})
		})
	})

	Describe("IsAdmin", func() {
		Context("when user is admin", func() {
			It("should return true", func() {
				// Arrange
				userID := uint(1)
				mockRepo.EXPECT().
					IsAdmin(userID).
					Return(true, nil)

				// Act
				isAdmin, err := userService.IsAdmin(userID)

				// Assert
				Expect(err).To(BeNil())
				Expect(isAdmin).To(BeTrue())
			})
		})

		Context("when user is not admin", func() {
			It("should return false", func() {
				// Arrange
				userID := uint(2)
				mockRepo.EXPECT().
					IsAdmin(userID).
					Return(false, nil)

				// Act
				isAdmin, err := userService.IsAdmin(userID)

				// Assert
				Expect(err).To(BeNil())
				Expect(isAdmin).To(BeFalse())
			})
		})

		Context("when repository error occurs", func() {
			It("should return error", func() {
				// Arrange
				userID := uint(1)
				mockRepo.EXPECT().
					IsAdmin(userID).
					Return(false, errors.New("database error"))

				// Act
				isAdmin, err := userService.IsAdmin(userID)

				// Assert
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("database error"))
				Expect(isAdmin).To(BeFalse())
			})
		})
	})
})
