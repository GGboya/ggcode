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

	// 添加UTF-8编码中间件
	router.Use(func(c *gin.Context) {
		// 对于HTML页面请求，设置正确的Content-Type
		accept := c.Request.Header.Get("Accept")
		if accept != "" && (accept == "text/html" ||
			accept == "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8" ||
			accept == "*/*" ||
			c.Request.URL.Path == "/" ||
			c.Request.URL.Path == "/dashboard" ||
			c.Request.URL.Path == "/login" ||
			c.Request.URL.Path == "/register" ||
			c.Request.URL.Path == "/questionbanks" ||
			c.Request.URL.Path == "/study-plans" ||
			c.Request.URL.Path == "/study") {
			c.Header("Content-Type", "text/html; charset=utf-8")
		}
		c.Next()
	})

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
			api.PUT("/questionbanks/:id", h.UpdateQuestionBank)
			api.DELETE("/questionbanks/:id", h.DeleteQuestionBank)
			api.GET("/questionbanks/:id/questions", h.GetQuestions)
			api.POST("/questionbanks/:id/questions", h.CreateQuestion)
			api.GET("/questions/:id", h.GetQuestion)
			api.PUT("/questions/:id", h.UpdateQuestion)
			api.DELETE("/questions/:id", h.DeleteQuestion)

			// 共享题库相关
			api.POST("/questionbanks/:id/share", h.ShareQuestionBank)
			api.DELETE("/questionbanks/:id/share", h.UnshareQuestionBank)
			api.POST("/questionbanks/:id/star", h.StarQuestionBank)
			api.DELETE("/questionbanks/:id/star", h.UnstarQuestionBank)
			api.POST("/questionbanks/:id/fork", h.ForkQuestionBank)
			api.GET("/starred-questionbanks", h.GetUserStarredBanks)

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
