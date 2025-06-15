package server

import (
	"ggcode/internal/handlers"
	"ggcode/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Server struct {
	router *gin.Engine
	db     *gorm.DB
}

func New(db *gorm.DB) (*Server, error) {
	router := gin.Default()

	// 设置静态文件服务
	router.Static("/static", "./web/static")

	// 使用glob模式加载模板，强制每次重新加载
	router.LoadHTMLGlob("web/templates/*.html")

	server := &Server{
		router: router,
		db:     db,
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	// 创建处理器
	h := handlers.New(s.db)

	// 公开路由
	public := s.router.Group("/")
	{
		// 静态页面
		public.GET("/", h.HomePage)
		public.GET("/login", h.LoginPage)
		public.GET("/register", h.RegisterPage)

		// API 路由
		api := public.Group("/api")
		{
			api.POST("/login", h.Login)
			api.POST("/register", h.Register)
		}
	}

	// 需要认证的路由
	auth := s.router.Group("/")
	auth.Use(middleware.AuthMiddleware())
	{
		// 页面路由
		auth.GET("/dashboard", h.Dashboard)
		auth.GET("/questionbanks", h.QuestionBanksPage)
		auth.GET("/study-plans", h.StudyPlansPage)
		auth.GET("/study", h.StudyPage)

		// API 路由
		api := auth.Group("/api")
		{
			// 题库相关
			api.GET("/questionbanks", h.GetQuestionBanks)
			api.POST("/questionbanks", h.CreateQuestionBank)
			api.GET("/questionbanks/:id/questions", h.GetQuestions)
			api.POST("/questionbanks/:id/questions", h.CreateQuestion)

			// 学习计划相关
			api.POST("/study-plan", h.CreateStudyPlan)
			api.GET("/study-plan/:id", h.GetStudyPlan)
			api.PUT("/study-plan/:id", h.UpdateStudyPlan)
			api.DELETE("/study-plan/:id", h.DeleteStudyPlan)
			api.GET("/study-plans", h.GetAllStudyPlans)
			api.GET("/study-plan/:id/progress", h.GetStudyPlanProgress)

			// 艾宾浩斯算法相关
			api.GET("/study-plan/:id/daily-questions", h.GetDailyQuestions)
			api.GET("/study-plan/:id/random-mastered-questions", h.GetRandomMasteredQuestions)
			api.POST("/complete-question", h.CompleteQuestion)
			api.GET("/study-stats", h.GetStudyStats)

			// 学习进度相关
			api.GET("/questionbanks/:id/progress", h.GetQuestionBankProgress)
			api.GET("/questionbanks-progress", h.GetAllQuestionBanksProgress)

			// 打卡相关
			api.POST("/checkin", h.CheckInToday)
			api.GET("/checkin-stats", h.GetCheckInStats)
		}
	}
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
