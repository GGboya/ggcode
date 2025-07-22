package controllers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"

	mockServices "ggcode/internal/mocks/services"
	"ggcode/internal/models"
	"ggcode/internal/repositories"

	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// 在文件顶部添加 mock 结构体定义

var _ = Describe("QuestionBankController", func() {
	var (
		ctrl        *gomock.Controller
		mockService *mockServices.MockQuestionBankServiceInterface
		controller  *QuestionBankController
		w           *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		mockService = mockServices.NewMockQuestionBankServiceInterface(ctrl)
		controller = NewQuestionBankController(mockService)
		w = httptest.NewRecorder()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("GetQuestionBanks", func() {
		It("should return 200 and data", func() {
			mockService.EXPECT().GetQuestionBanks(uint(1), "", "", 1, 10).Return(&services.QuestionBankListResponse{}, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questionbanks", nil)
			c.Set("user_id", uint(1))
			controller.GetQuestionBanks(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 500 on service error", func() {
			mockService.EXPECT().GetQuestionBanks(uint(1), "", "", 1, 10).Return(nil, errors.New("db error"))
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questionbanks", nil)
			c.Set("user_id", uint(1))
			controller.GetQuestionBanks(c)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("CreateQuestionBank", func() {
		It("should create question bank", func() {
			mockService.EXPECT().CreateQuestionBankWithImport("test", "desc", uint(1), "", 0, 0).Return(&models.QuestionBank{ID: 1, Name: "test"}, nil)
			c, _ := gin.CreateTestContext(w)
			body := `{"name":"test","description":"desc"}`
			c.Request, _ = http.NewRequest("POST", "/api/questionbanks", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("user_id", uint(1))
			controller.CreateQuestionBank(c)
			Expect(w.Code).To(Equal(http.StatusCreated))
		})
		It("should return 400 on bad json", func() {
			c, _ := gin.CreateTestContext(w)
			body := `{"name":123}`
			c.Request, _ = http.NewRequest("POST", "/api/questionbanks", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("user_id", uint(1))
			controller.CreateQuestionBank(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("UpdateQuestionBank", func() {
		It("should update question bank", func() {
			mockService.EXPECT().UpdateQuestionBank(uint(1), uint(1), repositories.QuestionBankUpdateData{Name: "test", Description: "desc"}).Return(nil)
			c, _ := gin.CreateTestContext(w)
			body := `{"name":"test","description":"desc"}`
			c.Request, _ = http.NewRequest("PUT", "/api/questionbanks/1", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			c.Set("user_id", uint(1))
			controller.UpdateQuestionBank(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 400 on bad id", func() {
			c, _ := gin.CreateTestContext(w)
			body := `{"name":"test"}`
			c.Request, _ = http.NewRequest("PUT", "/api/questionbanks/abc", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: "abc"}}
			c.Set("user_id", uint(1))
			controller.UpdateQuestionBank(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("DeleteQuestionBank", func() {
		It("should delete question bank", func() {
			mockService.EXPECT().DeleteQuestionBank(uint(1), uint(1)).Return(nil)
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("DELETE", "/api/questionbanks/1", nil)
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			c.Set("user_id", uint(1))
			controller.DeleteQuestionBank(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 400 on bad id", func() {
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("DELETE", "/api/questionbanks/abc", nil)
			c.Params = gin.Params{{Key: "id", Value: "abc"}}
			c.Set("user_id", uint(1))
			controller.DeleteQuestionBank(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GetQuestionBankProgress", func() {
		It("should get progress", func() {
			mockService.EXPECT().GetQuestionBankProgress(uint(1), uint(1)).Return(&models.QuestionBankProgress{QuestionBankID: 1}, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questionbanks/1/progress", nil)
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			c.Set("user_id", uint(1))
			controller.GetQuestionBankProgress(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("GetAllQuestionBanksProgress", func() {
		It("should get all progress", func() {
			mockService.EXPECT().GetAllQuestionBanksProgress(uint(1)).Return([]models.QuestionBankProgress{}, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questionbanks-progress", nil)
			c.Set("user_id", uint(1))
			controller.GetAllQuestionBanksProgress(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})
})
