package handlers

import (
	"bi-backend/db"
	"bi-backend/models"
	"bi-backend/utils"
	"context"
	"log"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 用户统计
func GetUserStats(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)
	client := db.GetClient()
	ctx := context.TODO()

	// 初始化 stats 结构
	stats := models.UserStats{
		MLModelStats: models.MLModelStats{
			ModelTypes:     make(map[string]int64),
			AverageMetrics: make(map[string]float64),
		},
		RecentActivity: make([]map[string]interface{}, 0),
		UsageStats:     make([]models.UsageStat, 0),
	}

	// 添加错误处理和日志
	//log.Printf("Fetching stats for user: %s", userID.Hex())

	// 1. 获取仪表盘总数
	dashboardColl := client.Database("bi_platform").Collection("dashboards")
	dashboards, err := dashboardColl.CountDocuments(ctx, bson.M{"created_by": userID})
	if err != nil {
		log.Printf("Error counting dashboards: %v", err)
		dashboards = 0 // 出错时设置为默认值
	}
	stats.TotalDashboards = dashboards

	// 2. 获取数据源总数
	dataSourceColl := client.Database("bi_platform").Collection("data_sources")
	dataSources, err := dataSourceColl.CountDocuments(ctx, bson.M{"created_by": userID})
	if err != nil {
		log.Printf("Error counting data sources: %v", err)
		dataSources = 0
	}
	stats.TotalDataSources = dataSources

	// 3. 获取图表总数
	chartsColl := client.Database("bi_platform").Collection("charts")
	charts, err := chartsColl.CountDocuments(ctx, bson.M{"created_by": userID})
	if err != nil {
		log.Printf("Error counting charts: %v", err)
		charts = 0
	}
	stats.TotalCharts = charts

	// 4. 获取最近的仪表盘
	dashboardOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(5)

	dashboardCursor, err := dashboardColl.Find(ctx,
		bson.M{"created_by": userID},
		dashboardOpts,
	)

	if err == nil {
		var recentDashboards []models.Dashboard
		if err = dashboardCursor.All(ctx, &recentDashboards); err == nil {
			stats.RecentDashboards = recentDashboards
		}
		dashboardCursor.Close(ctx)
	}
	// 先获取数据库
	mlModelColl := client.Database("bi_platform").Collection("ml_models")

	// 4. 获取最近的机器学习模型
	mlModelOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(5)

	mlModelCursor, err := mlModelColl.Find(ctx,
		bson.M{"created_by": userID},
		mlModelOpts,
	)

	if err == nil {
		var recentMLModels []models.MLModel
		if err = mlModelCursor.All(ctx, &recentMLModels); err == nil {
			// 将结果添加到 stats 中
			stats.RecentMLModels = recentMLModels
		}
		mlModelCursor.Close(ctx)
	}

	// 5. 获取机器学习模型总数
	mlModels, err := mlModelColl.CountDocuments(ctx, bson.M{"created_by": userID})
	if err != nil {
		log.Printf("Error counting ML models: %v", err)
		mlModels = 0
	}
	stats.TotalMLModels = mlModels

	// 6. 获取机器学习模型统计信息
	if mlModels > 0 {
		// 获取模型类型统计...
		trainedModels, _ := mlModelColl.CountDocuments(ctx, bson.M{
			"created_by":      userID,
			"training_result": bson.M{"$exists": true, "$ne": nil},
		})
		stats.MLModelStats.TrainedModels = trainedModels
		stats.MLModelStats.PendingModels = mlModels - trainedModels
	}

	// 生成使用统计
	currentTime := time.Now()
	for i := 5; i >= 0; i-- {
		monthStart := time.Date(currentTime.Year(), currentTime.Month()-time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := monthStart.AddDate(0, 1, 0)

		filter := bson.M{
			"created_by": userID,
			"created_at": bson.M{
				"$gte": monthStart,
				"$lt":  monthEnd,
			},
		}

		dashboardCount, _ := dashboardColl.CountDocuments(ctx, filter)
		chartCount, _ := chartsColl.CountDocuments(ctx, filter)
		mlModelCount, _ := mlModelColl.CountDocuments(ctx, filter)

		stats.UsageStats = append(stats.UsageStats, models.UsageStat{
			Date:       monthStart.Format("2006-01"),
			Dashboards: dashboardCount,
			Charts:     chartCount,
			MLModels:   mlModelCount,
			Queries:    chartCount * 2, // 假设每个图表平均查询两次
		})
	}
	// 收集最近活动信息
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)

	// 1. 获取仪表盘活动
	dashboardPipeline := []bson.M{
		{
			"$match": bson.M{
				"created_by": userID,
				"created_at": bson.M{"$gte": sixMonthsAgo},
			},
		},
		{"$sort": bson.M{"created_at": -1}},
		{"$limit": 5},
		{
			"$project": bson.M{
				"_id":           1,
				"name":          1,
				"type":          1,
				"created_at":    1,
				"activity_type": bson.M{"$literal": "dashboard"},
			},
		},
	}

	var activities []map[string]interface{}
	cursor, err := dashboardColl.Aggregate(ctx, dashboardPipeline)
	if err == nil {
		cursor.All(ctx, &activities)
		for _, activity := range activities {
			stats.RecentActivity = append(stats.RecentActivity, activity)
		}
	}

	// 2. 获取图表活动
	chartPipeline := []bson.M{
		{
			"$match": bson.M{
				"created_by": userID,
				"created_at": bson.M{"$gte": sixMonthsAgo},
			},
		},
		{"$sort": bson.M{"created_at": -1}},
		{"$limit": 5},
		{
			"$project": bson.M{
				"_id":           1,
				"name":          1,
				"type":          1,
				"created_at":    1,
				"activity_type": bson.M{"$literal": "chart"},
			},
		},
	}

	cursor, err = chartsColl.Aggregate(ctx, chartPipeline)
	if err == nil {
		var chartActivities []map[string]interface{}
		cursor.All(ctx, &chartActivities)
		stats.RecentActivity = append(stats.RecentActivity, chartActivities...)
	}

	// 3. 获取机器学习模型活动
	mlModelPipeline := []bson.M{
		{
			"$match": bson.M{
				"created_by": userID,
				"created_at": bson.M{"$gte": sixMonthsAgo},
			},
		},
		{"$sort": bson.M{"created_at": -1}},
		{"$limit": 5},
		{
			"$project": bson.M{
				"_id":           1,
				"name":          1,
				"type":          1,
				"created_at":    1,
				"activity_type": bson.M{"$literal": "mlmodel"},
			},
		},
	}

	cursor, err = mlModelColl.Aggregate(ctx, mlModelPipeline)
	if err == nil {
		var mlActivities []map[string]interface{}
		cursor.All(ctx, &mlActivities)
		stats.RecentActivity = append(stats.RecentActivity, mlActivities...)
	}

	// 4. 将所有活动按时间排序
	if len(stats.RecentActivity) > 0 {
		sort.Slice(stats.RecentActivity, func(i, j int) bool {
			timeI, _ := stats.RecentActivity[i]["created_at"].(primitive.DateTime)
			timeJ, _ := stats.RecentActivity[j]["created_at"].(primitive.DateTime)
			return timeI > timeJ
		})

		// 只保留最新的5条
		if len(stats.RecentActivity) > 5 {
			stats.RecentActivity = stats.RecentActivity[:5]
		}
	}

	// 添加调试日志
	log.Printf("Recent activities count: %d", len(stats.RecentActivity))
	log.Printf("Recent activities: %+v", stats.RecentActivity)

	utils.Success(c, stats)
}
