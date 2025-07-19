package main

import (
	"context"
	"ggcode/internal/config"
	"ggcode/internal/database"
	"ggcode/internal/pkg/logger"
	"ggcode/internal/server"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志系统
	if err := logger.InitLogger(cfg); err != nil {
		logger.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Info("Starting GGCode application...")

	// 初始化数据库
	db, err := database.Init(cfg)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}

	logger.Info("Database initialized successfully")

	// 启动服务器
	srv, err := server.New(db, cfg)
	if err != nil {
		logger.Fatalf("Failed to create server: %v", err)
	}

	// 创建HTTP服务器
	var httpSrv *http.Server

	// 在goroutine中启动服务器
	go func() {
		logger.Infof("HTTP Server starting on :%s", cfg.Server.Port)
		logger.Info("Database: MySQL")
		logger.Infof("Visit: http://localhost:%s", cfg.Server.Port)

		httpSrv = &http.Server{
			Addr:         ":" + cfg.Server.Port,
			Handler:      srv.GetRouter(),
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
			IdleTimeout:  cfg.Server.IdleTimeout,
		}

		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 创建信号通道监听退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("服务器已启动，按 Ctrl+C 优雅退出")

	// 等待退出信号
	<-quit
	logger.Info("收到退出信号，正在优雅关闭...")

	// 设置优雅关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 优雅关闭HTTP服务器
	if httpSrv != nil {
		logger.Info("正在关闭HTTP服务器...")
		if err := httpSrv.Shutdown(ctx); err != nil {
			logger.Errorf("HTTP服务器关闭失败: %v", err)
		} else {
			logger.Info("HTTP服务器已关闭")
		}
	}

	// 关闭应用层资源（容器池、数据库等）
	if err := srv.Shutdown(); err != nil {
		logger.Errorf("应用关闭失败: %v", err)
	}

	logger.Info("程序已退出")
}
