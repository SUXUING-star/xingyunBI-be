// middleware/logger.go
package middleware

import (
	"bi-backend/db"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LogData struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"user_id,omitempty"`
	Method    string             `bson:"method"`
	Path      string             `bson:"path"`
	Status    int                `bson:"status"`
	Latency   time.Duration      `bson:"latency"`
	IP        string             `bson:"ip"`
	UserAgent string             `bson:"user_agent"`
	CreatedAt time.Time          `bson:"created_at"`
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// 获取用户ID
		var userID primitive.ObjectID
		if id, exists := c.Get("user_id"); exists {
			userID = id.(primitive.ObjectID)
		}

		// 创建日志记录
		logData := LogData{
			ID:        primitive.NewObjectID(),
			UserID:    userID,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
			Latency:   time.Since(start),
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			CreatedAt: time.Now(),
		}

		// 异步写入日志
		go func(data LogData) {
			collection := db.GetClient().Database("bi_platform").Collection("logs")
			collection.InsertOne(context.Background(), data)
		}(logData)
	}
}
