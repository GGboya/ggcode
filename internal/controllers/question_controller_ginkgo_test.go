package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	mockServices "ggcode/internal/mocks/services"
	"ggcode/internal/models"
	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QuestionController", func() {
	var (
		ctrl        *gomock.Controller
		mockService *mockServices.MockQuestionServiceInterface
		controller  *QuestionController
		w           *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		mockService = mockServices.NewMockQuestionServiceInterface(ctrl)
		controller = NewQuestionController(mockService)
		w = httptest.NewRecorder()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("GetQuestions", func() {
		It("should return 200 and data", func() {
			mockService.EXPECT().GetQuestions(uint(1), 1, 20).Return(&services.QuestionListResponse{}, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questionbanks/1/questions", nil)
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			controller.GetQuestions(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 400 on bad id", func() {
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questionbanks/abc/questions", nil)
			c.Params = gin.Params{{Key: "id", Value: "abc"}}
			controller.GetQuestions(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GetAllQuestions", func() {
		It("should return 200 and data", func() {
			mockService.EXPECT().GetAllQuestions().Return([]models.Question{}, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questions", nil)
			controller.GetAllQuestions(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("CreateQuestion", func() {
		It("should create question", func() {
			mockService.EXPECT().CreateQuestion(uint(1), uint(1), "title", "url", "easy", 10.0).Return(&models.Question{ID: 1, Title: "title"}, nil)
			c, _ := gin.CreateTestContext(w)
			body := `{"title":"title","url":"url","difficulty":"easy","score":10}`
			c.Request, _ = http.NewRequest("POST", "/api/questionbanks/1/questions", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			c.Set("user_id", uint(1))
			controller.CreateQuestion(c)
			Expect(w.Code).To(Equal(http.StatusCreated))
		})
		It("should return 400 on bad id", func() {
			c, _ := gin.CreateTestContext(w)
			body := `{"title":"title","url":"url","difficulty":"easy","score":10}`
			c.Request, _ = http.NewRequest("POST", "/api/questionbanks/abc/questions", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: "abc"}}
			c.Set("user_id", uint(1))
			controller.CreateQuestion(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GetQuestion", func() {
		It("should return 200 and data", func() {
			mockService.EXPECT().GetQuestion(uint(1)).Return(&models.Question{ID: 1, Title: "title"}, nil)
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questions/1", nil)
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			controller.GetQuestion(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 400 on bad id", func() {
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/api/questions/abc", nil)
			c.Params = gin.Params{{Key: "id", Value: "abc"}}
			controller.GetQuestion(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("UpdateQuestion", func() {
		It("should update question", func() {
			mockService.EXPECT().UpdateQuestion(uint(1), uint(1), uint(1), "title", "url", "easy").Return(&models.Question{ID: 1, Title: "title"}, nil)
			c, _ := gin.CreateTestContext(w)
			body := `{"title":"title","url":"url","difficulty":"easy","question_bank_id":1}`
			c.Request, _ = http.NewRequest("PUT", "/api/questions/1", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			c.Set("user_id", uint(1))
			controller.UpdateQuestion(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 400 on bad id", func() {
			c, _ := gin.CreateTestContext(w)
			body := `{"title":"title","url":"url","difficulty":"easy","question_bank_id":1}`
			c.Request, _ = http.NewRequest("PUT", "/api/questions/abc", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: "abc"}}
			c.Set("user_id", uint(1))
			controller.UpdateQuestion(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("DeleteQuestion", func() {
		It("should delete question", func() {
			mockService.EXPECT().DeleteQuestion(uint(1), uint(1), uint(1)).Return(nil)
			c, _ := gin.CreateTestContext(w)
			body := `{"question_bank_id":1}`
			c.Request, _ = http.NewRequest("DELETE", "/api/questions/1", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			c.Set("user_id", uint(1))
			controller.DeleteQuestion(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 400 on bad id", func() {
			c, _ := gin.CreateTestContext(w)
			body := `{"question_bank_id":1}`
			c.Request, _ = http.NewRequest("DELETE", "/api/questions/abc", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = gin.Params{{Key: "id", Value: "abc"}}
			c.Set("user_id", uint(1))
			controller.DeleteQuestion(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})
})
