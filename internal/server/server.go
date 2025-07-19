package server

import (
	"ggcode/internal/config"
	"ggcode/internal/controllers"
	"ggcode/internal/middleware"
	"ggcode/internal/pkg/logger"
	"ggcode/internal/repositories"
	"ggcode/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Server struct {
	router      *gin.Engine
	db          *gorm.DB
	controllers *controllers.Controllers
	config      *config.Config
}

func New(db *gorm.DB, cfg *config.Config) (*Server, error) {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由器，使用自定义恢复处理器
	router := gin.New()

	// 使用自定义错误处理中间件
	router.Use(middleware.ErrorHandler())

	// 创建安全中间件
	securityMiddleware := middleware.NewSecurityMiddleware(cfg)

	// 添加安全中间件
	router.Use(securityMiddleware.RequestID())
	router.Use(securityMiddleware.SecurityHeaders())
	router.Use(securityMiddleware.CORS())
	router.Use(securityMiddleware.RequestSizeLimit())
	router.Use(securityMiddleware.RateLimit())

	// 启动速率限制器清理
	securityMiddleware.CleanupRateLimiters()

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

	// 初始化新架构
	repos := repositories.NewRepositories(db)
	serviceLayer := services.NewServices(repos, db, cfg)
	controllers := controllers.NewControllers(serviceLayer)

	server := &Server{
		router:      router,
		db:          db,
		controllers: controllers,
		config:      cfg,
	}

	server.setupRoutes()
	logger.Info("Server initialized successfully")
	return server, nil
}

func (s *Server) setupRoutes() {
	// 创建处理器
	ctrl := s.controllers
	// 公开路由
	public := s.router.Group("/")
	{
		// 静态页面
		public.GET("/", ctrl.Page.HomePage)
		public.GET("/login", ctrl.Page.LoginPage)
		public.GET("/register", ctrl.Page.RegisterPage)

		// API 路由
		api := public.Group("/api")
		{
			api.POST("/login", ctrl.User.Login)
			api.POST("/register", ctrl.User.Register)
		}
	}

	// 需要认证的路由
	auth := s.router.Group("/")
	auth.Use(middleware.AuthMiddleware(s.config))
	{
		// 页面路由
		auth.GET("/dashboard", ctrl.Page.Dashboard)
		auth.GET("/questionbanks", ctrl.Page.QuestionBanksPage)
		auth.GET("/study-plans", ctrl.Page.StudyPlansPage)
		auth.GET("/study", ctrl.Page.StudyPage)
		auth.GET("/interview-island", ctrl.Page.InterviewIslandPage)
		auth.GET("/level/:levelId", ctrl.Page.LevelPage)

		// API 路由
		api := auth.Group("/api")
		{
			// 用户相关
			api.POST("/logout", ctrl.User.Logout)
			api.GET("/auth/verify", ctrl.User.VerifyToken)

			// 题库相关
			api.GET("/questionbanks", ctrl.QuestionBank.GetQuestionBanks)
			api.POST("/questionbanks", ctrl.QuestionBank.CreateQuestionBank)
			api.PUT("/questionbanks/:id", ctrl.QuestionBank.UpdateQuestionBank)
			api.DELETE("/questionbanks/:id", ctrl.QuestionBank.DeleteQuestionBank)

			// 题库题目相关
			api.GET("/questionbanks/:id/questions", ctrl.Question.GetQuestions)
			api.POST("/questionbanks/:id/questions", ctrl.Question.CreateQuestion)
			api.GET("/questions", ctrl.Question.GetAllQuestions)
			api.GET("/questions/:id", ctrl.Question.GetQuestion)
			api.PUT("/questions/:id", ctrl.Question.UpdateQuestion)
			api.DELETE("/questions/:id", ctrl.Question.DeleteQuestion)

			// 共享题库相关
			api.POST("/questionbanks/:id/share", ctrl.Share.ShareQuestionBank)
			api.DELETE("/questionbanks/:id/share", ctrl.Share.UnshareQuestionBank)
			api.POST("/questionbanks/:id/star", ctrl.Share.StarQuestionBank)
			api.DELETE("/questionbanks/:id/star", ctrl.Share.UnstarQuestionBank)
			api.POST("/questionbanks/:id/fork", ctrl.Share.ForkQuestionBank)
			api.GET("/starred-questionbanks", ctrl.Share.GetUserStarredBanks)

			// 学习计划相关
			api.POST("/study-plan", ctrl.StudyPlan.CreateStudyPlan)
			api.GET("/study-plan/:id", ctrl.StudyPlan.GetStudyPlan)
			api.PUT("/study-plan/:id", ctrl.StudyPlan.UpdateStudyPlan)
			api.DELETE("/study-plan/:id", ctrl.StudyPlan.DeleteStudyPlan)
			api.GET("/study-plans", ctrl.StudyPlan.GetAllStudyPlans)
			api.GET("/study-plan/:id/progress", ctrl.StudyPlan.GetStudyPlanProgress)

			// 艾宾浩斯算法相关
			api.GET("/study-plan/:id/daily-questions", ctrl.StudyPlan.GetDailyQuestions)

			// 学习进度相关
			api.GET("/questionbanks/:id/progress", ctrl.Progress.GetQuestionBankProgress)
			api.GET("/questionbanks-progress", ctrl.Progress.GetAllQuestionBanksProgress)

			// 打卡相关
			api.POST("/checkin", ctrl.Progress.CheckInToday)
			api.GET("/checkin-stats", ctrl.Progress.GetCheckInStats)

			// 学习热力图
			api.GET("/study-heatmap", ctrl.Progress.GetStudyHeatmap)

			// 面试岛相关
			api.GET("/interview-island/map", ctrl.Interview.GetIslandMap)
			api.GET("/interview-island/level/:levelId", ctrl.Interview.GetLevelDetail)
			api.GET("/interview-island/progress", ctrl.Interview.GetUserProgress)

			// 管理员 面试岛CRUD
			api.POST("/interview-island/create", ctrl.Interview.CreateIsland)
			api.POST("/interview-island/:id/edit", ctrl.Interview.EditIsland)
			api.POST("/interview-island/:id/delete", ctrl.Interview.DeleteIsland)
			api.POST("/interview-island/level/create", ctrl.Interview.CreateLevel)
			api.POST("/interview-island/level/:levelId/edit", ctrl.Interview.EditLevel)
			api.PUT("/interview-island/level/:levelId", ctrl.Interview.UpdateLevelDetail)
			api.DELETE("/interview-island/level/:levelId", ctrl.Interview.DeleteLevel)

			// 测试用例接口
			api.GET("/interview-island/level/:levelId/testcases", ctrl.Interview.GetLevelTestCases)
			api.POST("/interview-island/level/:levelId/testcases", ctrl.Interview.AddTestCase)
			api.DELETE("/interview-island/testcases/:id", ctrl.Interview.DeleteTestCase)

			// go-judge 评测系统
			goJudge := api.Group("/go-judge")
			{
				goJudge.POST("/execute", ctrl.GoJudge.ExecuteCode)
				goJudge.GET("/execute", ctrl.GoJudge.ExecuteCodeSimple)
				goJudge.POST("/level/:levelId/test", ctrl.GoJudge.TestCode)
				goJudge.POST("/level/:levelId/submit", ctrl.GoJudge.SubmitCode)
				goJudge.GET("/health", ctrl.GoJudge.HealthCheck)
				goJudge.GET("/languages", ctrl.GoJudge.GetSupportedLanguages)
				goJudge.GET("/system-info", ctrl.GoJudge.GetSystemInfo)
			}

			// 用户-题目相关
			userQuestion := api.Group("/user-question")
			{
				userQuestion.POST(":question_id/complete", ctrl.UserQuestion.CompleteQuestion)
				userQuestion.GET("/stats", ctrl.UserQuestion.GetStudyStats)
			}
		}
	}

	logger.Info("Routes configured successfully")
}

func (s *Server) Run(addr string) error {
	// 创建HTTP服务器，使用配置中的超时设置
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	return httpSrv.ListenAndServe()
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown() error {
	logger.Info("正在优雅关闭服务器...")

	if s.db != nil {
		logger.Info("正在关闭数据库连接...")
		if sqlDB, err := s.db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				logger.Errorf("数据库连接关闭失败: %v", err)
			} else {
				logger.Info("数据库连接已关闭")
			}
		}
	}

	logger.Info("服务器已关闭")
	return nil
}

// GetRouter 获取路由器实例
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
