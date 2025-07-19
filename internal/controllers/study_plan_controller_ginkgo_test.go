package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"ggcode/internal/models"
	"ggcode/internal/services"

	mockservices "ggcode/internal/mocks/services"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StudyPlanController", func() {
	var (
		ctrl           *gomock.Controller
		mockService    *mockservices.MockStudyPlanServiceInterface
		controller     *StudyPlanController
		router         *gin.Engine
		recorder       *httptest.ResponseRecorder
		userID         uint = 1
		questionBankID uint = 1
		planID         uint = 1
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockService = mockservices.NewMockStudyPlanServiceInterface(ctrl)
		controller = NewStudyPlanController(mockService)

		// 设置 Gin 为测试模式
		gin.SetMode(gin.TestMode)
		router = gin.New()

		// 添加中间件来设置 user_id
		router.Use(func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		})

		recorder = httptest.NewRecorder()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("CreateStudyPlan", func() {
		Context("成功创建学习计划", func() {
			It("应该返回 201 状态码和创建的学习计划", func() {
				// 准备请求数据
				reqData := map[string]interface{}{
					"question_bank_id": questionBankID,
					"daily_count":      5,
				}
				reqBody, _ := json.Marshal(reqData)

				// 设置 mock 期望
				expectedPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					DailyCount:     5,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}
				mockService.EXPECT().
					CreateStudyPlan(userID, questionBankID, 5).
					Return(expectedPlan, nil)

				// 设置路由
				router.POST("/study-plans", controller.CreateStudyPlan)

				// 发送请求
				req := httptest.NewRequest("POST", "/study-plans", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusCreated))

				var response models.UserStudyPlan
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				Expect(err).To(BeNil())
				Expect(response.ID).To(Equal(planID))
				Expect(response.UserID).To(Equal(userID))
				Expect(response.QuestionBankID).To(Equal(questionBankID))
				Expect(response.DailyCount).To(Equal(5))
			})
		})

		Context("请求参数无效", func() {
			It("应该返回 400 状态码", func() {
				// 准备无效的请求数据
				reqData := map[string]interface{}{
					"question_bank_id": questionBankID,
					// 缺少 daily_count
				}
				reqBody, _ := json.Marshal(reqData)

				// 设置路由
				router.POST("/study-plans", controller.CreateStudyPlan)

				// 发送请求
				req := httptest.NewRequest("POST", "/study-plans", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("学习计划已存在", func() {
			It("应该返回 409 状态码", func() {
				// 准备请求数据
				reqData := map[string]interface{}{
					"question_bank_id": questionBankID,
					"daily_count":      5,
				}
				reqBody, _ := json.Marshal(reqData)

				// 设置 mock 期望
				mockService.EXPECT().
					CreateStudyPlan(userID, questionBankID, 5).
					Return(nil, errors.New("您已经为该题库创建了学习计划，一个题库只能创建一个学习计划"))

				// 设置路由
				router.POST("/study-plans", controller.CreateStudyPlan)

				// 发送请求
				req := httptest.NewRequest("POST", "/study-plans", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusConflict))
			})
		})

		Context("服务层错误", func() {
			It("应该返回 500 状态码", func() {
				// 准备请求数据
				reqData := map[string]interface{}{
					"question_bank_id": questionBankID,
					"daily_count":      5,
				}
				reqBody, _ := json.Marshal(reqData)

				// 设置 mock 期望
				mockService.EXPECT().
					CreateStudyPlan(userID, questionBankID, 5).
					Return(nil, errors.New("数据库错误"))

				// 设置路由
				router.POST("/study-plans", controller.CreateStudyPlan)

				// 发送请求
				req := httptest.NewRequest("POST", "/study-plans", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("GetStudyPlan", func() {
		Context("成功获取学习计划", func() {
			It("应该返回 200 状态码和学习计划", func() {
				// 设置 mock 期望
				expectedPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					DailyCount:     5,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}
				mockService.EXPECT().
					GetStudyPlan(planID, userID).
					Return(expectedPlan, nil)

				// 设置路由
				router.GET("/study-plans/:id", controller.GetStudyPlan)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/"+strconv.FormatUint(uint64(planID), 10), nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusOK))

				var response models.UserStudyPlan
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				Expect(err).To(BeNil())
				Expect(response.ID).To(Equal(planID))
			})
		})

		Context("无效的计划ID", func() {
			It("应该返回 400 状态码", func() {
				// 设置路由
				router.GET("/study-plans/:id", controller.GetStudyPlan)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/invalid", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("学习计划不存在", func() {
			It("应该返回 404 状态码", func() {
				// 设置 mock 期望
				mockService.EXPECT().
					GetStudyPlan(planID, userID).
					Return(nil, errors.New("学习计划不存在"))

				// 设置路由
				router.GET("/study-plans/:id", controller.GetStudyPlan)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/"+strconv.FormatUint(uint64(planID), 10), nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})
	})

	Describe("UpdateStudyPlan", func() {
		Context("成功更新学习计划", func() {
			It("应该返回 200 状态码", func() {
				// 准备请求数据
				reqData := map[string]interface{}{
					"daily_count": 10,
				}
				reqBody, _ := json.Marshal(reqData)

				// 设置 mock 期望
				mockService.EXPECT().
					UpdateStudyPlan(planID, userID, 10).
					Return(nil)

				// 设置路由
				router.PUT("/study-plans/:id", controller.UpdateStudyPlan)

				// 发送请求
				req := httptest.NewRequest("PUT", "/study-plans/"+strconv.FormatUint(uint64(planID), 10), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
		})

		Context("无效的计划ID", func() {
			It("应该返回 400 状态码", func() {
				// 准备请求数据
				reqData := map[string]interface{}{
					"daily_count": 10,
				}
				reqBody, _ := json.Marshal(reqData)

				// 设置路由
				router.PUT("/study-plans/:id", controller.UpdateStudyPlan)

				// 发送请求
				req := httptest.NewRequest("PUT", "/study-plans/invalid", bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("请求参数无效", func() {
			It("应该返回 400 状态码", func() {
				// 准备无效的请求数据
				reqData := map[string]interface{}{
					"daily_count": 0, // 无效的每日数量
				}
				reqBody, _ := json.Marshal(reqData)

				// 设置路由
				router.PUT("/study-plans/:id", controller.UpdateStudyPlan)

				// 发送请求
				req := httptest.NewRequest("PUT", "/study-plans/"+strconv.FormatUint(uint64(planID), 10), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("服务层错误", func() {
			It("应该返回 500 状态码", func() {
				// 准备请求数据
				reqData := map[string]interface{}{
					"daily_count": 10,
				}
				reqBody, _ := json.Marshal(reqData)

				// 设置 mock 期望
				mockService.EXPECT().
					UpdateStudyPlan(planID, userID, 10).
					Return(errors.New("更新失败"))

				// 设置路由
				router.PUT("/study-plans/:id", controller.UpdateStudyPlan)

				// 发送请求
				req := httptest.NewRequest("PUT", "/study-plans/"+strconv.FormatUint(uint64(planID), 10), bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("DeleteStudyPlan", func() {
		Context("成功删除学习计划", func() {
			It("应该返回 200 状态码", func() {
				// 设置 mock 期望
				mockService.EXPECT().
					DeleteStudyPlan(planID, userID).
					Return(nil)

				// 设置路由
				router.DELETE("/study-plans/:id", controller.DeleteStudyPlan)

				// 发送请求
				req := httptest.NewRequest("DELETE", "/study-plans/"+strconv.FormatUint(uint64(planID), 10), nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
		})

		Context("无效的计划ID", func() {
			It("应该返回 400 状态码", func() {
				// 设置路由
				router.DELETE("/study-plans/:id", controller.DeleteStudyPlan)

				// 发送请求
				req := httptest.NewRequest("DELETE", "/study-plans/invalid", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("学习计划不存在", func() {
			It("应该返回 404 状态码", func() {
				// 设置 mock 期望
				mockService.EXPECT().
					DeleteStudyPlan(planID, userID).
					Return(errors.New("学习计划不存在"))

				// 设置路由
				router.DELETE("/study-plans/:id", controller.DeleteStudyPlan)

				// 发送请求
				req := httptest.NewRequest("DELETE", "/study-plans/"+strconv.FormatUint(uint64(planID), 10), nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})

		Context("服务层错误", func() {
			It("应该返回 500 状态码", func() {
				// 设置 mock 期望
				mockService.EXPECT().
					DeleteStudyPlan(planID, userID).
					Return(errors.New("删除失败"))

				// 设置路由
				router.DELETE("/study-plans/:id", controller.DeleteStudyPlan)

				// 发送请求
				req := httptest.NewRequest("DELETE", "/study-plans/"+strconv.FormatUint(uint64(planID), 10), nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("GetAllStudyPlans", func() {
		Context("成功获取所有学习计划", func() {
			It("应该返回 200 状态码和分页数据", func() {
				// 设置 mock 期望
				expectedPlans := []models.UserStudyPlan{
					{
						ID:             1,
						UserID:         userID,
						QuestionBankID: questionBankID,
						DailyCount:     5,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					},
					{
						ID:             2,
						UserID:         userID,
						QuestionBankID: 2,
						DailyCount:     10,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					},
				}
				mockService.EXPECT().
					GetAllStudyPlans(userID, 1, 20).
					Return(expectedPlans, int64(2), nil)

				// 设置路由
				router.GET("/study-plans", controller.GetAllStudyPlans)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusOK))

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				Expect(err).To(BeNil())
				Expect(response).To(HaveKey("data"))
				Expect(response).To(HaveKey("pagination"))
			})
		})

		Context("带分页参数", func() {
			It("应该正确处理分页参数", func() {
				// 设置 mock 期望
				expectedPlans := []models.UserStudyPlan{}
				mockService.EXPECT().
					GetAllStudyPlans(userID, 2, 10).
					Return(expectedPlans, int64(0), nil)

				// 设置路由
				router.GET("/study-plans", controller.GetAllStudyPlans)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans?page=2&limit=10", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
		})

		Context("服务层错误", func() {
			It("应该返回 500 状态码", func() {
				// 设置 mock 期望
				mockService.EXPECT().
					GetAllStudyPlans(userID, 1, 20).
					Return(nil, int64(0), errors.New("获取失败"))

				// 设置路由
				router.GET("/study-plans", controller.GetAllStudyPlans)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("GetStudyPlanProgress", func() {
		Context("成功获取学习计划进度", func() {
			It("应该返回 200 状态码和进度数据", func() {
				// 设置 mock 期望
				expectedProgress := &services.StudyPlanProgress{
					StudyPlanID:    planID,
					TotalQuestions: 100,
					StudiedCount:   50,
					CompletedCount: 30,
					ReviewCount:    10,
					ProgressRate:   50,
					MasteryRate:    30,
				}
				mockService.EXPECT().
					GetStudyPlanProgress(userID, planID).
					Return(expectedProgress, nil)

				// 设置路由
				router.GET("/study-plans/:id/progress", controller.GetStudyPlanProgress)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/"+strconv.FormatUint(uint64(planID), 10)+"/progress", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusOK))

				var response services.StudyPlanProgress
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				Expect(err).To(BeNil())
				Expect(response.StudyPlanID).To(Equal(planID))
				Expect(response.TotalQuestions).To(Equal(int64(100)))
			})
		})

		Context("无效的计划ID", func() {
			It("应该返回 400 状态码", func() {
				// 设置路由
				router.GET("/study-plans/:id/progress", controller.GetStudyPlanProgress)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/invalid/progress", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("服务层错误", func() {
			It("应该返回 500 状态码", func() {
				// 设置 mock 期望
				mockService.EXPECT().
					GetStudyPlanProgress(userID, planID).
					Return(nil, errors.New("获取进度失败"))

				// 设置路由
				router.GET("/study-plans/:id/progress", controller.GetStudyPlanProgress)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/"+strconv.FormatUint(uint64(planID), 10)+"/progress", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("GetDailyQuestions", func() {
		Context("成功获取每日学习题目", func() {
			It("应该返回 200 状态码和题目数据", func() {
				// 设置 mock 期望
				expectedQuestions := &models.DailyQuestionsResponse{
					Questions: []models.QuestionWithProgress{
						{
							Question: models.Question{
								ID:         1,
								Title:      "两数之和",
								URL:        "https://leetcode.com/problems/two-sum",
								Difficulty: "Easy",
							},
							Progress: models.UserQuestionProgress{
								UserID:     userID,
								QuestionID: 1,
							},
							IsReview: false,
							Score:    100,
						},
					},
					Start: 0,
					Total: 1,
				}
				mockService.EXPECT().
					GetDailyQuestions(userID, planID).
					Return(expectedQuestions, nil)

				// 设置路由
				router.GET("/study-plans/:id/daily-questions", controller.GetDailyQuestions)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/"+strconv.FormatUint(uint64(planID), 10)+"/daily-questions", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusOK))

				var response models.DailyQuestionsResponse
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				Expect(err).To(BeNil())
				Expect(response.Questions).To(HaveLen(1))
				Expect(response.Questions[0].Question.Title).To(Equal("两数之和"))
			})
		})

		Context("无效的计划ID", func() {
			It("应该返回 400 状态码", func() {
				// 设置路由
				router.GET("/study-plans/:id/daily-questions", controller.GetDailyQuestions)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/invalid/daily-questions", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("服务层错误", func() {
			It("应该返回 500 状态码", func() {
				// 设置 mock 期望
				mockService.EXPECT().
					GetDailyQuestions(userID, planID).
					Return(nil, errors.New("获取题目失败"))

				// 设置路由
				router.GET("/study-plans/:id/daily-questions", controller.GetDailyQuestions)

				// 发送请求
				req := httptest.NewRequest("GET", "/study-plans/"+strconv.FormatUint(uint64(planID), 10)+"/daily-questions", nil)
				router.ServeHTTP(recorder, req)

				// 验证响应
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})
