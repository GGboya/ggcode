package services

import (
	"errors"
	"time"

	"ggcode/internal/mocks/repositories"
	"ggcode/internal/models"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var _ = Describe("StudyPlanService", func() {
	var (
		ctrl                 *gomock.Controller
		mockStudyPlanRepo    *repositories.MockStudyPlanRepository
		mockUserQuestionRepo *repositories.MockUserQuestionRepository
		mockQuestionRepo     *repositories.MockQuestionRepository
		service              *StudyPlanService
		userID               uint = 1
		questionBankID       uint = 1
		planID               uint = 1
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockStudyPlanRepo = repositories.NewMockStudyPlanRepository(ctrl)
		mockUserQuestionRepo = repositories.NewMockUserQuestionRepository(ctrl)
		mockQuestionRepo = repositories.NewMockQuestionRepository(ctrl)
		service = NewStudyPlanService(mockStudyPlanRepo, mockUserQuestionRepo, mockQuestionRepo)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("CreateStudyPlan", func() {
		Context("成功创建学习计划", func() {
			It("应该返回创建的学习计划", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					CheckStudyPlanExists(userID, questionBankID).
					Return(false, nil)

				expectedPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					DailyCount:     5,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}
				mockStudyPlanRepo.EXPECT().
					CreateStudyPlan(gomock.Any()).
					Return(expectedPlan, nil)

				// 执行测试
				result, err := service.CreateStudyPlan(userID, questionBankID, 5)

				// 验证结果
				Expect(err).To(BeNil())
				Expect(result).To(Equal(expectedPlan))
				Expect(result.UserID).To(Equal(userID))
				Expect(result.QuestionBankID).To(Equal(questionBankID))
				Expect(result.DailyCount).To(Equal(5))
			})
		})

		Context("学习计划已存在", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					CheckStudyPlanExists(userID, questionBankID).
					Return(true, nil)

				// 执行测试
				result, err := service.CreateStudyPlan(userID, questionBankID, 5)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("您已经为该题库创建了学习计划，一个题库只能创建一个学习计划"))
				Expect(result).To(BeNil())
			})
		})

		Context("检查学习计划存在时出错", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					CheckStudyPlanExists(userID, questionBankID).
					Return(false, errors.New("数据库错误"))

				// 执行测试
				result, err := service.CreateStudyPlan(userID, questionBankID, 5)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("数据库错误"))
				Expect(result).To(BeNil())
			})
		})

		Context("创建学习计划时出错", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					CheckStudyPlanExists(userID, questionBankID).
					Return(false, nil)

				mockStudyPlanRepo.EXPECT().
					CreateStudyPlan(gomock.Any()).
					Return(nil, errors.New("创建失败"))

				// 执行测试
				result, err := service.CreateStudyPlan(userID, questionBankID, 5)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("创建失败"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("GetStudyPlan", func() {
		Context("成功获取学习计划", func() {
			It("应该返回学习计划", func() {
				// 设置 mock 期望
				expectedPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					DailyCount:     5,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(expectedPlan, nil)

				// 执行测试
				result, err := service.GetStudyPlan(planID, userID)

				// 验证结果
				Expect(err).To(BeNil())
				Expect(result).To(Equal(expectedPlan))
			})
		})

		Context("学习计划不存在", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(nil, errors.New("学习计划不存在"))

				// 执行测试
				result, err := service.GetStudyPlan(planID, userID)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("学习计划不存在"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("UpdateStudyPlan", func() {
		Context("成功更新学习计划", func() {
			It("应该返回 nil", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					UpdateStudyPlan(planID, userID, 10).
					Return(nil)

				// 执行测试
				err := service.UpdateStudyPlan(planID, userID, 10)

				// 验证结果
				Expect(err).To(BeNil())
			})
		})

		Context("更新失败", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					UpdateStudyPlan(planID, userID, 10).
					Return(errors.New("更新失败"))

				// 执行测试
				err := service.UpdateStudyPlan(planID, userID, 10)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("更新失败"))
			})
		})
	})

	Describe("DeleteStudyPlan", func() {
		Context("成功删除学习计划", func() {
			It("应该返回 nil", func() {
				// 设置 mock 期望
				studyPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					QuestionBank: models.QuestionBank{
						ID:         questionBankID,
						ForkedFrom: nil,
					},
				}
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(studyPlan, nil)

				mockStudyPlanRepo.EXPECT().
					DeleteStudyPlanWithProgress(planID, userID, questionBankID).
					Return(nil)

				// 执行测试
				err := service.DeleteStudyPlan(planID, userID)

				// 验证结果
				Expect(err).To(BeNil())
			})
		})

		Context("学习计划不存在", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(nil, errors.New("学习计划不存在"))

				// 执行测试
				err := service.DeleteStudyPlan(planID, userID)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("学习计划不存在"))
			})
		})

		Context("删除失败", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				studyPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					QuestionBank: models.QuestionBank{
						ID:         questionBankID,
						ForkedFrom: nil,
					},
				}
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(studyPlan, nil)

				mockStudyPlanRepo.EXPECT().
					DeleteStudyPlanWithProgress(planID, userID, questionBankID).
					Return(errors.New("删除失败"))

				// 执行测试
				err := service.DeleteStudyPlan(planID, userID)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("删除失败"))
			})
		})
	})

	Describe("GetAllStudyPlans", func() {
		Context("成功获取所有学习计划", func() {
			It("应该返回学习计划列表", func() {
				// 设置 mock 期望
				expectedPlans := []models.UserStudyPlan{
					{
						ID:             1,
						UserID:         userID,
						QuestionBankID: questionBankID,
						DailyCount:     5,
					},
					{
						ID:             2,
						UserID:         userID,
						QuestionBankID: 2,
						DailyCount:     10,
					},
				}
				mockStudyPlanRepo.EXPECT().
					GetAllStudyPlans(userID, 1, 20).
					Return(expectedPlans, int64(2), nil)

				// 执行测试
				result, total, err := service.GetAllStudyPlans(userID, 1, 20)

				// 验证结果
				Expect(err).To(BeNil())
				Expect(result).To(Equal(expectedPlans))
				Expect(total).To(Equal(int64(2)))
			})
		})

		Context("获取失败", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					GetAllStudyPlans(userID, 1, 20).
					Return(nil, int64(0), errors.New("获取失败"))

				// 执行测试
				result, total, err := service.GetAllStudyPlans(userID, 1, 20)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("获取失败"))
				Expect(result).To(BeNil())
				Expect(total).To(Equal(int64(0)))
			})
		})
	})

	Describe("GetStudyPlanProgress", func() {
		Context("成功获取学习计划进度", func() {
			It("应该返回进度信息", func() {
				// 设置 mock 期望
				studyPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					QuestionBank: models.QuestionBank{
						ID:         questionBankID,
						ForkedFrom: nil,
					},
				}
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(studyPlan, nil)

				mockQuestionRepo.EXPECT().
					GetQuestionCount(questionBankID).
					Return(int64(100), nil)

				mockUserQuestionRepo.EXPECT().
					GetStudiedQuestionCount(userID, questionBankID).
					Return(int64(50), nil)

				mockUserQuestionRepo.EXPECT().
					GetCompletedQuestionCount(userID, questionBankID).
					Return(int64(30), nil)

				mockUserQuestionRepo.EXPECT().
					GetReviewQuestionCount(userID, questionBankID).
					Return(int64(10), nil)

				// 执行测试
				result, err := service.GetStudyPlanProgress(userID, planID)

				// 验证结果
				Expect(err).To(BeNil())
				Expect(result).NotTo(BeNil())
				Expect(result.StudyPlanID).To(Equal(planID))
				Expect(result.TotalQuestions).To(Equal(int64(100)))
				Expect(result.StudiedCount).To(Equal(int64(50)))
				Expect(result.CompletedCount).To(Equal(int64(30)))
				Expect(result.ReviewCount).To(Equal(int64(10)))
				Expect(result.ProgressRate).To(Equal(50))
				Expect(result.MasteryRate).To(Equal(30))
			})
		})

		Context("学习计划不存在", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(nil, errors.New("学习计划不存在"))

				// 执行测试
				result, err := service.GetStudyPlanProgress(userID, planID)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("学习计划不存在"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("GetDailyQuestions", func() {
		Context("成功获取每日学习题目（新缓存）", func() {
			It("应该返回题目列表", func() {
				// 设置 mock 期望
				studyPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					DailyCount:     5,
					QuestionBank: models.QuestionBank{
						ID:         questionBankID,
						ForkedFrom: nil,
					},
				}
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(studyPlan, nil)

				mockStudyPlanRepo.EXPECT().
					GetDailyCache(userID, planID, gomock.Any()).
					Return(nil, gorm.ErrRecordNotFound)

				// 模拟生成每日题目
				expectedQuestions := []models.QuestionWithProgress{
					{
						Question: models.Question{
							ID:         1,
							Title:      "两数之和",
							Difficulty: "Easy",
						},
						Progress: models.UserQuestionProgress{
							UserID:     userID,
							QuestionID: 1,
						},
						IsReview: false,
						Score:    100,
					},
				}

				// 模拟 GetReviewQuestionProgresses
				mockUserQuestionRepo.EXPECT().
					GetReviewQuestionProgresses(userID, questionBankID, gomock.Any()).
					Return([]models.UserQuestionProgress{}, nil)

				// 模拟 GetNewQuestions
				mockUserQuestionRepo.EXPECT().
					GetNewQuestions(userID, questionBankID, 5, nil).
					Return([]models.Question{
						{ID: 1, Title: "两数之和", Difficulty: "Easy"},
					}, nil)

				// 模拟 CacheDailyQuestions
				mockStudyPlanRepo.EXPECT().
					CacheDailyQuestions(userID, planID, gomock.Any(), gomock.Any()).
					Return(nil)

				// 模拟 GetQuestionsByIDs
				mockUserQuestionRepo.EXPECT().
					GetQuestionsByIDs(gomock.Any(), questionBankID).
					Return(expectedQuestions, nil)

				// 模拟 CalculateStartPosition
				mockUserQuestionRepo.EXPECT().
					CalculateStartPosition(userID, gomock.Any(), gomock.Any()).
					Return(0, nil)

				// 执行测试
				result, err := service.GetDailyQuestions(userID, planID)

				// 验证结果
				Expect(err).To(BeNil())
				Expect(result).NotTo(BeNil())
				Expect(result.Questions).To(HaveLen(1))
				Expect(result.Start).To(Equal(0))
				Expect(result.Total).To(Equal(1))
			})
		})

		Context("从缓存获取每日学习题目", func() {
			It("应该返回缓存的题目列表", func() {
				// 设置 mock 期望
				studyPlan := &models.UserStudyPlan{
					ID:             planID,
					UserID:         userID,
					QuestionBankID: questionBankID,
					DailyCount:     5,
					QuestionBank: models.QuestionBank{
						ID:         questionBankID,
						ForkedFrom: nil,
					},
				}
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(studyPlan, nil)

				cache := &models.DailyStudyPlanCache{
					QuestionIDs: "[1,2,3]",
				}
				mockStudyPlanRepo.EXPECT().
					GetDailyCache(userID, planID, gomock.Any()).
					Return(cache, nil)

				expectedQuestions := []models.QuestionWithProgress{
					{
						Question: models.Question{
							ID:         1,
							Title:      "两数之和",
							Difficulty: "Easy",
						},
						Progress: models.UserQuestionProgress{
							UserID:     userID,
							QuestionID: 1,
						},
						IsReview: false,
						Score:    100,
					},
				}

				// 模拟 GetQuestionsByIDs
				mockUserQuestionRepo.EXPECT().
					GetQuestionsByIDs(gomock.Any(), questionBankID).
					Return(expectedQuestions, nil)

				// 模拟 CalculateStartPosition
				mockUserQuestionRepo.EXPECT().
					CalculateStartPosition(userID, gomock.Any(), gomock.Any()).
					Return(0, nil)

				// 执行测试
				result, err := service.GetDailyQuestions(userID, planID)

				// 验证结果
				Expect(err).To(BeNil())
				Expect(result).NotTo(BeNil())
				Expect(result.Questions).To(HaveLen(1))
				Expect(result.Start).To(Equal(0))
				Expect(result.Total).To(Equal(1))
			})
		})

		Context("学习计划不存在", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockStudyPlanRepo.EXPECT().
					GetStudyPlan(planID, userID).
					Return(nil, errors.New("学习计划不存在"))

				// 执行测试
				result, err := service.GetDailyQuestions(userID, planID)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("学习计划不存在"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("GenerateDailyQuestions", func() {
		Context("成功生成每日学习题目", func() {
			It("应该返回题目列表", func() {
				// 设置 mock 期望
				reviewProgresses := []models.UserQuestionProgress{
					{
						UserID:      userID,
						QuestionID:  1,
						IsCompleted: false,
					},
				}
				mockUserQuestionRepo.EXPECT().
					GetReviewQuestionProgresses(userID, questionBankID, gomock.Any()).
					Return(reviewProgresses, nil)

				question := &models.Question{
					ID:         1,
					Title:      "两数之和",
					Difficulty: "Easy",
				}
				mockQuestionRepo.EXPECT().
					GetQuestion(uint(1)).
					Return(question, nil)

				// 模拟 GetNewQuestions（如果需要更多题目）
				mockUserQuestionRepo.EXPECT().
					GetNewQuestions(userID, questionBankID, 4, nil).
					Return([]models.Question{}, nil)

				// 执行测试
				result, err := service.GenerateDailyQuestions(userID, questionBankID, 5)

				// 验证结果
				Expect(err).To(BeNil())
				Expect(result).NotTo(BeNil())
				Expect(result).To(HaveLen(1))
				Expect(result[0].Question.ID).To(Equal(uint(1)))
				Expect(result[0].Question.Title).To(Equal("两数之和"))
				Expect(result[0].IsReview).To(BeTrue())
			})
		})

		Context("获取复习题目失败", func() {
			It("应该返回错误", func() {
				// 设置 mock 期望
				mockUserQuestionRepo.EXPECT().
					GetReviewQuestionProgresses(userID, questionBankID, gomock.Any()).
					Return(nil, errors.New("获取复习题目失败"))

				// 执行测试
				result, err := service.GenerateDailyQuestions(userID, questionBankID, 5)

				// 验证结果
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("获取复习题目失败"))
				Expect(result).To(BeNil())
			})
		})
	})
})
