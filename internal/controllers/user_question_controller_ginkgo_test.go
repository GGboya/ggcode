package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ggcode/internal/mocks/services"
	"ggcode/internal/models"
)

var _ = Describe("UserQuestionController", func() {
	var (
		ctrl        *gomock.Controller
		mockService *services.MockUserQuestionServiceInterface
		controller  *UserQuestionController
		r           *gin.Engine
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		mockService = services.NewMockUserQuestionServiceInterface(ctrl)
		controller = NewUserQuestionController(mockService)
		r = gin.New()
		r.POST("/questions/:question_id/complete", controller.CompleteQuestion)
		r.GET("/study_stats", controller.GetStudyStats)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("CompleteQuestion", func() {
		It("should return 200 when complete success", func() {
			mockService.EXPECT().CompleteQuestion(uint(1), uint(2), "success").Return(nil)
			w := httptest.NewRecorder()
			body, _ := json.Marshal(map[string]string{"result_type": "success"})
			req, _ := http.NewRequest("POST", "/questions/2/complete", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "question_id", Value: "2"}}
			c.Set("user_id", uint(1))
			controller.CompleteQuestion(c)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("完成学习记录"))
		})

		It("should return 400 for invalid question_id", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(map[string]string{"result_type": "success"})
			req, _ := http.NewRequest("POST", "/questions/abc/complete", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "question_id", Value: "abc"}}
			c.Set("user_id", uint(1))
			controller.CompleteQuestion(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 for missing result_type", func() {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(map[string]string{})
			req, _ := http.NewRequest("POST", "/questions/2/complete", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "question_id", Value: "2"}}
			c.Set("user_id", uint(1))
			controller.CompleteQuestion(c)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 500 if service returns error", func() {
			mockService.EXPECT().CompleteQuestion(uint(1), uint(2), "success").Return(errors.New("db error"))
			w := httptest.NewRecorder()
			body, _ := json.Marshal(map[string]string{"result_type": "success"})
			req, _ := http.NewRequest("POST", "/questions/2/complete", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "question_id", Value: "2"}}
			c.Set("user_id", uint(1))
			controller.CompleteQuestion(c)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("GetStudyStats", func() {
		It("should return 200 and stats", func() {
			mockStats := &models.StudyStats{
				TotalStudied: 10,
				Completed:    5,
				TodayReview:  2,
			}
			mockService.EXPECT().GetStudyStats(uint(1)).Return(mockStats, nil)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/study_stats", nil)
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", uint(1))
			controller.GetStudyStats(c)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring("total_studied"))
		})

		It("should return 500 if service returns error", func() {
			mockService.EXPECT().GetStudyStats(uint(1)).Return(nil, errors.New("db error"))
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/study_stats", nil)
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", uint(1))
			controller.GetStudyStats(c)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
