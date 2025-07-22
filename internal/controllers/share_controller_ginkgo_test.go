package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"ggcode/internal/mocks/services"
	"ggcode/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShareController", func() {
	var (
		ctrl          *gomock.Controller
		mockService   *services.MockShareServiceInterface
		controller    *ShareController
		r             *gin.Engine
		w             *httptest.ResponseRecorder
		testUserID    uint = 1
		testBankID    uint = 1
		testBankIDStr      = "1"
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		mockService = services.NewMockShareServiceInterface(ctrl)
		controller = NewShareController(mockService)
		r = gin.New()
		w = httptest.NewRecorder()

		// Setup routes
		r.POST("/api/questionbanks/:id/share", controller.ShareQuestionBank)
		r.POST("/api/questionbanks/:id/unshare", controller.UnshareQuestionBank)
		r.POST("/api/questionbanks/:id/star", controller.StarQuestionBank)
		r.POST("/api/questionbanks/:id/unstar", controller.UnstarQuestionBank)
		r.POST("/api/questionbanks/:id/fork", controller.ForkQuestionBank)
		r.GET("/api/starred-questionbanks", controller.GetUserStarredBanks)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	// Helper to create a request with a user context
	createContext := func(req *http.Request) *gin.Context {
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: testBankIDStr}}
		c.Set("user_id", testUserID)
		return c
	}

	Describe("ShareQuestionBank", func() {
		It("should share a question bank successfully", func() {
			mockService.EXPECT().ShareQuestionBank(testBankID, testUserID).Return(nil)
			req, _ := http.NewRequest(http.MethodPost, "/api/questionbanks/1/share", nil)
			c := createContext(req)
			controller.ShareQuestionBank(c)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("题库已设为共享"))
		})

		It("should return 404 if bank not found or no permission", func() {
			mockService.EXPECT().ShareQuestionBank(testBankID, testUserID).Return(errors.New("题库不存在或无权限操作"))
			req, _ := http.NewRequest(http.MethodPost, "/api/questionbanks/1/share", nil)
			c := createContext(req)
			controller.ShareQuestionBank(c)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("UnshareQuestionBank", func() {
		It("should unshare a question bank successfully", func() {
			mockService.EXPECT().UnshareQuestionBank(testBankID, testUserID).Return(nil)
			req, _ := http.NewRequest(http.MethodPost, "/api/questionbanks/1/unshare", nil)
			c := createContext(req)
			controller.UnshareQuestionBank(c)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("题库已取消共享"))
		})
	})

	Describe("StarQuestionBank", func() {
		It("should star a question bank successfully", func() {
			mockService.EXPECT().StarQuestionBank(testBankID, testUserID).Return(nil)
			req, _ := http.NewRequest(http.MethodPost, "/api/questionbanks/1/star", nil)
			c := createContext(req)
			controller.StarQuestionBank(c)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("Star成功"))
		})

		It("should return 409 if already starred", func() {
			mockService.EXPECT().StarQuestionBank(testBankID, testUserID).Return(errors.New("已经Star过这个题库"))
			req, _ := http.NewRequest(http.MethodPost, "/api/questionbanks/1/star", nil)
			c := createContext(req)
			controller.StarQuestionBank(c)

			Expect(w.Code).To(Equal(http.StatusConflict))
		})
	})

	Describe("UnstarQuestionBank", func() {
		It("should unstar a question bank successfully", func() {
			mockService.EXPECT().UnstarQuestionBank(testBankID, testUserID).Return(nil)
			req, _ := http.NewRequest(http.MethodPost, "/api/questionbanks/1/unstar", nil)
			c := createContext(req)
			controller.UnstarQuestionBank(c)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("取消Star成功"))
		})
	})

	Describe("ForkQuestionBank", func() {
		It("should fork a question bank successfully", func() {
			forkedBank := &models.QuestionBank{ID: 2, Name: "Forked Bank"}
			mockService.EXPECT().ForkQuestionBank(testBankID, testUserID).Return(forkedBank, nil)
			req, _ := http.NewRequest(http.MethodPost, "/api/questionbanks/1/fork", nil)
			c := createContext(req)
			controller.ForkQuestionBank(c)

			Expect(w.Code).To(Equal(http.StatusCreated))
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["message"]).To(Equal("Fork成功"))
		})
	})

	Describe("GetUserStarredBanks", func() {
		It("should get user starred banks successfully", func() {
			starredBanks := []models.QuestionBank{{ID: 1, Name: "Starred Bank"}}
			total := int64(1)
			mockService.EXPECT().GetUserStarredBanks(testUserID, 1, 20).Return(starredBanks, total, nil)
			req, _ := http.NewRequest(http.MethodGet, "/api/starred-questionbanks", nil)
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", testUserID)

			controller.GetUserStarredBanks(c)

			Expect(w.Code).To(Equal(http.StatusOK))
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["data"]).ToNot(BeNil())
		})
	})
})
