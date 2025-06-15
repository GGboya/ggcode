package main

import (
	"ggcode/internal/database"
	"ggcode/internal/server"
	"log"
	"os"

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

	log.Printf("Server starting on :%s", port)
	log.Printf("Database: MySQL")
	log.Printf("Visit: http://localhost:%s", port)

	if err := srv.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
