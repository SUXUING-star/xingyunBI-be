// middleware/cors.go
package middleware

import (
	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	allowedOrigins := []string{
		"http://localhost:5173",             // 开发环境
		"https://www.xingyunbi.site",        // 生产环境
		"https://xingyunbi.vercel.app",      // Vercel 域名
		"https://xingyunbi.site",            // 不带 www 的域名
		"https://xingyunbi-be.onrender.com", // 新添加的 Render 后端域名
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查 origin 是否在允许列表中
		isAllowedOrigin := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowedOrigin = true
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// 如果是开发模式，允许所有源
		if gin.Mode() == gin.DebugMode {
			c.Header("Access-Control-Allow-Origin", origin)
			isAllowedOrigin = true
		}

		// 只有在允许的源时才设置其他 CORS 头
		if isAllowedOrigin {
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Origin, X-Requested-With, Content-Type, Accept, Access-Control-Request-Method, Access-Control-Request-Headers")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// 如果是预检请求，立即返回204
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
