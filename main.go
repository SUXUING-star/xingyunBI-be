package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"bi-backend/config"
	"bi-backend/db"
	"bi-backend/handlers"
	"bi-backend/middleware"
	"bi-backend/storage"
)

func init() {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// 加载配置
	if err := config.Init(); err != nil {
		log.Fatal("Failed to load config:", err)
	}
}
func setupRouter() *gin.Engine {
	r := gin.Default()

	// 中间件
	r.Use(middleware.Cors())
	r.Use(middleware.Logger())

	// 添加一个测试路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// API路由
	api := r.Group("/api")
	{
		// 添加测试路由
		api.GET("/test-oss", func(c *gin.Context) {
			if storage.GetCloudStorage() == nil {
				c.JSON(500, gin.H{"error": "Storage not initialized"})
				return
			}
			c.JSON(200, gin.H{"status": "OSS connection OK"})
		})
		// 认证相关
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.GET("/verify-email", handlers.VerifyEmail)
			auth.POST("/forgot-password", handlers.ForgotPassword)
			auth.POST("/reset-password", handlers.ResetPassword)
		}

		// 需要认证的路由
		authorized := api.Group("")
		authorized.Use(middleware.Auth())
		{
			// 用户相关
			user := authorized.Group("/user")
			{
				user.GET("/profile", handlers.GetProfile)
				user.PUT("/profile", handlers.UpdateProfile)
				user.PUT("/password", handlers.UpdatePassword)
				user.GET("/stats", handlers.GetUserStats) // 添加新的统计接口
			}

			// 数据源相关
			datasource := authorized.Group("/datasources")
			{
				// 使用正确的 UploadDataSource 处理函数
				datasource.POST("", handlers.UploadDataSource)                     // 文件上传
				datasource.GET("", handlers.GetDataSources)                        // 获取列表
				datasource.GET("/:id", handlers.GetDataSource)                     // 获取单个
				datasource.PUT("/:id", handlers.UpdateDataSource)                  // 更新
				datasource.DELETE("/:id", handlers.DeleteDataSource)               // 删除
				datasource.PUT("/:id/preprocessing", handlers.UpdatePreprocessing) //预处理
			}
			// 仪表盘相关
			dashboard := authorized.Group("/dashboards")
			{
				dashboard.POST("", handlers.CreateDashboard)
				dashboard.GET("", handlers.GetDashboards)
				dashboard.GET("/:id", handlers.GetDashboard)
				dashboard.PUT("/:id", handlers.UpdateDashboard)
				dashboard.DELETE("/:id", handlers.DeleteDashboard)
			}

			// 图表相关
			chart := authorized.Group("/charts")
			{
				chart.POST("", handlers.CreateChart)
				chart.GET("/:id", handlers.GetChart)
				chart.GET("", handlers.GetCharts)
				chart.PUT("/:id", handlers.UpdateChart)
				chart.PUT("/:id/config", handlers.UpdateChartConfig)
				chart.DELETE("/:id", handlers.DeleteChart)
			}
			// 机器学习模型相关
			mlmodel := authorized.Group("/mlmodels")
			{
				mlmodel.POST("", handlers.CreateMLModel)
				mlmodel.GET("", handlers.GetMLModels)
				mlmodel.GET("/:id", handlers.GetMLModel)
				mlmodel.PUT("/:id", handlers.UpdateMLModel)
				mlmodel.PUT("/:id/result", handlers.UpdateMLModelResult)
				mlmodel.DELETE("/:id", handlers.DeleteMLModel)
			}
		}
	}
	return r
}

func main() {
	// 根据环境变量设置 gin 模式
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 初始化数据库连接
	if err := db.Init(ctx); err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer db.Close(ctx)

	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
	// 初始化 OSS
	if err := storage.InitCloudStorage(); err != nil {
		log.Fatalf("Failed to initialize cloud storage: %v", err)
	}
	// 初始化路由
	router := setupRouter()
	// 打印所有注册的路由
	log.Println("=== Registered Routes ===")
	routes := router.Routes()
	for _, route := range routes {
		log.Printf("%s %s", route.Method, route.Path)
	}
	log.Println("=======================")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	router.Run(":" + port)
}
