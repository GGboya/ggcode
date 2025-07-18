package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	mocks "ggcode/internal/mocks/services"
	"ggcode/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserController", func() {
	var (
		mockCtrl    *gomock.Controller
		mockService *mocks.MockUserServiceInterface
		controller  *UserController
		router      *gin.Engine
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		mockCtrl = gomock.NewController(GinkgoT())
		mockService = mocks.NewMockUserServiceInterface(mockCtrl)
		controller = NewUserController(mockService)
		router = gin.Default()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Register", func() {
		Context("when registration is successful", func() {
			It("should return 201 status and token", func() {
				// Arrange
				router.POST("/register", controller.Register)

				expectedUser := &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				}
				expectedToken := "mock-token"

				mockService.EXPECT().
					Register("testuser", "test@example.com", "123456").
					Return(expectedUser, expectedToken, nil)

				// Act
				w := httptest.NewRecorder()
				body := `{"username": "testuser", "email": "test@example.com", "password": "123456"}`
				req, _ := http.NewRequest("POST", "/register", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				router.ServeHTTP(w, req)

				// Assert
				Expect(w.Code).To(Equal(http.StatusCreated))
				Expect(w.Body.String()).To(ContainSubstring("mock-token"))
			})
		})

		Context("when request has invalid parameters", func() {
			It("should return 400 status", func() {
				// Arrange
				router.POST("/register", controller.Register)

				// Act
				w := httptest.NewRecorder()
				body := `{"username": "", "email": "invalid-email", "password": "123"}`
				req, _ := http.NewRequest("POST", "/register", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				router.ServeHTTP(w, req)

				// Assert
				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("when service returns error", func() {
			It("should return 500 status with error message", func() {
				// Arrange
				router.POST("/register", controller.Register)

				mockService.EXPECT().
					Register("user1", "user1@example.com", "123456").
					Return(nil, "", errors.New("注册失败"))

				// Act
				w := httptest.NewRecorder()
				body := `{"username": "user1", "email": "user1@example.com", "password": "123456"}`
				req, _ := http.NewRequest("POST", "/register", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				router.ServeHTTP(w, req)

				// Assert
				Expect(w.Code).To(Equal(http.StatusInternalServerError))
				Expect(w.Body.String()).To(ContainSubstring("注册失败"))
			})
		})
	})

	Describe("Login", func() {
		Context("when login is successful", func() {
			It("should return 200 status and token", func() {
				// Arrange
				router.POST("/login", controller.Login)

				expectedUser := &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				}
				expectedToken := "login-token"

				mockService.EXPECT().
					Login("testuser", "123456").
					Return(expectedUser, expectedToken, nil)

				// Act
				w := httptest.NewRecorder()
				body := `{"username": "testuser", "password": "123456"}`
				req, _ := http.NewRequest("POST", "/login", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				router.ServeHTTP(w, req)

				// Assert
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring("login-token"))
			})
		})

		Context("when credentials are invalid", func() {
			It("should return 401 status with error message", func() {
				// Arrange
				router.POST("/login", controller.Login)

				mockService.EXPECT().
					Login("wronguser", "wrongpass").
					Return(nil, "", errors.New("用户名或密码错误"))

				// Act
				w := httptest.NewRecorder()
				body := `{"username": "wronguser", "password": "wrongpass"}`
				req, _ := http.NewRequest("POST", "/login", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				router.ServeHTTP(w, req)

				// Assert
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
				Expect(w.Body.String()).To(ContainSubstring("用户名或密码错误"))
			})
		})
	})

	Describe("Logout", func() {
		It("should return 200 status with success message", func() {
			// Arrange
			router.POST("/logout", controller.Logout)

			// Act
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/logout", nil)

			router.ServeHTTP(w, req)

			// Assert
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("退出登录成功"))
		})
	})

	Describe("VerifyToken", func() {
		It("should return 200 status with user info", func() {
			// Arrange
			router.Use(func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("username", "testuser")
				c.Next()
			})
			router.GET("/verify", controller.VerifyToken)

			// Act
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/verify", nil)

			router.ServeHTTP(w, req)

			// Assert
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("testuser"))
		})
	})
})
