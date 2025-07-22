package controllers

import (
	"net/http"
	"net/http/httptest"

	mockServices "ggcode/internal/mocks/services"
	"ggcode/internal/models"
	"ggcode/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("CheckInController", func() {
	var (
		ctrl        *gomock.Controller
		mockService *mockServices.MockCheckInServiceInterface
		controller  *CheckInController
		w           *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		mockService = mockServices.NewMockCheckInServiceInterface(ctrl)
		controller = NewCheckInController(mockService)
		w = httptest.NewRecorder()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("CheckInToday", func() {
		It("should return 200 on success", func() {
			mockService.EXPECT().CheckInToday(uint(1)).Return(nil)
			c, _ := gin.CreateTestContext(w)
			c.Set("user_id", uint(1))
			controller.CheckInToday(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 500 on error", func() {
			mockService.EXPECT().CheckInToday(uint(1)).Return(assert.AnError)
			c, _ := gin.CreateTestContext(w)
			c.Set("user_id", uint(1))
			controller.CheckInToday(c)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("GetCheckInStats", func() {
		It("should return 200 and stats", func() {
			mockStats := &models.CheckInStat{CheckedInToday: true, ConsecutiveDays: 3, BestStreak: 5, TotalCheckInDays: 10}
			mockService.EXPECT().GetCheckInStats(uint(1)).Return(mockStats, nil)
			c, _ := gin.CreateTestContext(w)
			c.Set("user_id", uint(1))
			controller.GetCheckInStats(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 500 on error", func() {
			mockService.EXPECT().GetCheckInStats(uint(1)).Return(nil, assert.AnError)
			c, _ := gin.CreateTestContext(w)
			c.Set("user_id", uint(1))
			controller.GetCheckInStats(c)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("GetStudyHeatmap", func() {
		It("should return 200 and heatmap", func() {
			mockHeatmap := &services.HeatmapResponse{TotalCommits: 10, CurrentStreak: 2, MaxStreak: 5, ThisYear: 8}
			mockService.EXPECT().GetStudyHeatmap(uint(1)).Return(mockHeatmap, nil)
			c, _ := gin.CreateTestContext(w)
			c.Set("user_id", uint(1))
			controller.GetStudyHeatmap(c)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
		It("should return 500 on error", func() {
			mockService.EXPECT().GetStudyHeatmap(uint(1)).Return(nil, assert.AnError)
			c, _ := gin.CreateTestContext(w)
			c.Set("user_id", uint(1))
			controller.GetStudyHeatmap(c)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
