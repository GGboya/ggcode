package main

import (
	"context"
	"ggcode/internal/database"
	"ggcode/internal/server"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量文件
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables or defaults")
	}

	// 初始化数据库
	db, err := database.Init()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 启动服务器
	srv, err := server.New(db)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	// 从环境变量获取端口，默认为8080
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	// 检查是否启用TLS
	tlsCert := os.Getenv("TLS_CERT_FILE")
	tlsKey := os.Getenv("TLS_KEY_FILE")
	enableTLS := os.Getenv("ENABLE_TLS")

	// 创建HTTP服务器
	var httpSrv *http.Server

	// 在goroutine中启动服务器
	go func() {
		if enableTLS == "true" && tlsCert != "" && tlsKey != "" {
			log.Printf("HTTPS Server starting on :%s", port)
			log.Printf("Database: MySQL")
			log.Printf("TLS Certificate: %s", tlsCert)
			log.Printf("TLS Key: %s", tlsKey)
			log.Printf("Visit: https://localhost:%s", port)
			log.Printf("容器池正在启动，请稍候...")

			if err := srv.RunTLS(":"+port, tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
				log.Fatal("Failed to start HTTPS server:", err)
			}
		} else {
			log.Printf("HTTP Server starting on :%s", port)
			log.Printf("Database: MySQL")
			log.Printf("Visit: http://localhost:%s", port)

			httpSrv = &http.Server{
				Addr:    ":" + port,
				Handler: srv.GetRouter(),
			}

			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal("Failed to start HTTP server:", err)
			}
		}
	}()

	// 创建信号通道监听退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("服务器已启动，按 Ctrl+C 优雅退出")

	// 等待退出信号
	<-quit
	log.Printf("收到退出信号，正在优雅关闭...")

	// 设置5分钟的优雅关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 优雅关闭HTTP服务器
	if httpSrv != nil {
		log.Printf("正在关闭HTTP服务器...")
		if err := httpSrv.Shutdown(ctx); err != nil {
			log.Printf("HTTP服务器关闭失败: %v", err)
		} else {
			log.Printf("HTTP服务器已关闭")
		}
	}

	// 关闭应用层资源（容器池、数据库等）
	if err := srv.Shutdown(); err != nil {
		log.Printf("应用关闭失败: %v", err)
	}

	log.Printf("程序已退出")
}
