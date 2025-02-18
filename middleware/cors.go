package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"https://www.xingyunbi.site",
			"https://xingyunbi.vercel.app",
			"https://xingyunbi.site",
			"https://xingyunbi-be.onrender.com",
		},

		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"HEAD",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Cache-Control",
			"Accept",
			"X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// 在开发模式下允许所有来源
	if gin.Mode() != gin.ReleaseMode {
		config.AllowAllOrigins = true
	}

	return cors.New(config)
}
