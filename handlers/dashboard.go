// handlers/dashboard.go
package handlers

import (
	"bi-backend/db"
	"bi-backend/models"
	"bi-backend/utils"
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func incrementDashboardEditCount(dashboardID primitive.ObjectID) error {
	collection := db.GetClient().Database("bi_platform").Collection("dashboards")

	// 查找当前仪表盘
	var dashboard models.Dashboard
	err := collection.FindOne(context.TODO(), bson.M{"_id": dashboardID}).Decode(&dashboard)
	if err != nil {
		return err
	}

	// 如果没有找到仪表盘，返回错误
	if err == mongo.ErrNoDocuments {
		return mongo.ErrNoDocuments
	}

	// 增加编辑次数
	dashboard.EditCount++

	// 更新仪表盘
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": dashboardID},
		bson.M{
			"$set": bson.M{
				"edit_count": dashboard.EditCount,
				"updated_at": time.Now(),
			},
		},
	)
	return err
}
func CreateDashboard(c *gin.Context) {
	var dashboard models.Dashboard
	if err := c.ShouldBindJSON(&dashboard); err != nil {
		utils.Error(c, 400, "Invalid request data")
		return
	}

	dashboard.CreatedBy = c.MustGet("user_id").(primitive.ObjectID)
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()
	dashboard.EditCount = 1 // 创建时初始化编辑次数为1

	collection := db.GetClient().Database("bi_platform").Collection("dashboards")
	result, err := collection.InsertOne(context.TODO(), dashboard)
	if err != nil {
		utils.Error(c, 500, "Failed to create dashboard")
		return
	}

	dashboard.ID = result.InsertedID.(primitive.ObjectID)
	utils.Success(c, dashboard)
}

func GetDashboards(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)

	collection := db.GetClient().Database("bi_platform").Collection("dashboards")
	cursor, err := collection.Find(context.TODO(), bson.M{"created_by": userID})
	if err != nil {
		utils.Error(c, 500, "Failed to get dashboards")
		return
	}
	defer cursor.Close(context.TODO())

	var dashboards []models.Dashboard
	if err = cursor.All(context.TODO(), &dashboards); err != nil {
		utils.Error(c, 500, "Failed to parse dashboards")
		return
	}

	utils.Success(c, dashboards)
}

// handlers/dashboard.go
func GetDashboard(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "Invalid dashboard ID")
		return
	}

	// 获取仪表盘基本信息
	collection := db.GetClient().Database("bi_platform").Collection("dashboards")
	var dashboard models.Dashboard
	err = collection.FindOne(context.TODO(), bson.M{
		"_id":        id,
		"created_by": c.MustGet("user_id").(primitive.ObjectID),
	}).Decode(&dashboard)

	if err != nil {
		utils.Error(c, 404, "Dashboard not found")
		return
	}

	// 如果有布局，获取所有相关图表的信息
	if len(dashboard.Layout) > 0 {
		chartCollection := db.GetClient().Database("bi_platform").Collection("charts")
		var chartIDs []primitive.ObjectID
		for _, item := range dashboard.Layout {
			chartIDs = append(chartIDs, item.ChartID)
		}

		// 获取所有图表信息
		cursor, err := chartCollection.Find(context.TODO(), bson.M{
			"_id": bson.M{"$in": chartIDs},
		})
		if err != nil {
			utils.Error(c, 500, "Failed to fetch charts")
			return
		}
		defer cursor.Close(context.TODO())

		var charts []models.Chart
		if err = cursor.All(context.TODO(), &charts); err != nil {
			utils.Error(c, 500, "Failed to decode charts")
			return
		}

		// 创建图表映射
		chartMap := make(map[primitive.ObjectID]models.Chart)
		for _, chart := range charts {
			chartMap[chart.ID] = chart
		}

		var fullLayout []gin.H
		for _, item := range dashboard.Layout {
			if chart, ok := chartMap[item.ChartID]; ok {
				// 确保所有字段名称与数据库一致
				fullLayout = append(fullLayout, gin.H{
					"chart_id":     item.ChartID.Hex(),
					"x":            item.X,
					"y":            item.Y,
					"width":        item.Width,  // 改用 width 而不是 w
					"height":       item.Height, // 改用 height 而不是 h
					"type":         chart.Type,
					"dataSourceId": chart.DataSourceID.Hex(),
					"config":       chart.Config,
					"name":         chart.Name,
				})
			} else {
				// 添加日志，帮助调试找不到的图表
				log.Printf("Chart not found for ID: %s", item.ChartID.Hex())
			}
		}

		// 返回完整的仪表盘数据
		utils.Success(c, gin.H{
			"id":          dashboard.ID.Hex(),
			"name":        dashboard.Name,
			"description": dashboard.Description,
			"layout":      fullLayout,
			"created_by":  dashboard.CreatedBy,
			"created_at":  dashboard.CreatedAt,
			"updated_at":  dashboard.UpdatedAt,
		})
		return
	}

	utils.Success(c, dashboard)
}

func UpdateDashboard(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "Invalid dashboard ID")
		return
	}
	// 增加编辑次数
	if err := incrementDashboardEditCount(id); err != nil {
		utils.Error(c, 500, "Failed to increment edit count")
		return
	}

	var updateData struct {
		Name        string               `json:"name"`
		Description string               `json:"description"`
		Layout      []models.ChartLayout `json:"layout"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		log.Printf("Binding error: %v", err)
		log.Printf("Raw data: %v", c.Request.Body)
		utils.Error(c, 400, "Invalid request data")
		return
	}

	// 打印接收到的数据
	log.Printf("Received update data: %+v", updateData)
	log.Printf("Layout data: %+v", updateData.Layout)

	collection := db.GetClient().Database("bi_platform").Collection("dashboards")
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":        id,
			"created_by": c.MustGet("user_id").(primitive.ObjectID),
		},
		bson.M{
			"$set": bson.M{
				"name":        updateData.Name,
				"description": updateData.Description,
				"layout":      updateData.Layout,
				"updated_at":  time.Now(),
			},
		},
	)

	if err != nil {
		utils.Error(c, 500, "Failed to update dashboard")
		return
	}

	if result.ModifiedCount == 0 {
		utils.Error(c, 404, "Dashboard not found")
		return
	}

	utils.Success(c, gin.H{"message": "Dashboard updated successfully"})
}

func DeleteDashboard(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "Invalid dashboard ID")
		return
	}

	// 直接执行删除操作，不使用事务
	dashboardColl := db.GetClient().Database("bi_platform").Collection("dashboards")

	// 1. 先获取仪表盘信息
	var dashboard models.Dashboard
	err = dashboardColl.FindOne(context.TODO(), bson.M{
		"_id":        id,
		"created_by": c.MustGet("user_id").(primitive.ObjectID),
	}).Decode(&dashboard)

	if err == mongo.ErrNoDocuments {
		utils.Error(c, 404, "Dashboard not found")
		return
	}
	if err != nil {
		utils.Error(c, 500, "Failed to find dashboard")
		return
	}

	// 2. 删除相关的图表
	if len(dashboard.Layout) > 0 {
		chartColl := db.GetClient().Database("bi_platform").Collection("charts")
		var chartIDs []primitive.ObjectID
		for _, layout := range dashboard.Layout {
			chartIDs = append(chartIDs, layout.ChartID)
		}

		_, err = chartColl.DeleteMany(context.TODO(), bson.M{
			"_id": bson.M{"$in": chartIDs},
		})
		if err != nil {
			utils.Error(c, 500, "Failed to delete charts")
			return
		}
	}

	// 3. 删除仪表盘
	result, err := dashboardColl.DeleteOne(context.TODO(), bson.M{
		"_id":        id,
		"created_by": c.MustGet("user_id").(primitive.ObjectID),
	})

	if err != nil {
		utils.Error(c, 500, "Failed to delete dashboard")
		return
	}

	if result.DeletedCount == 0 {
		utils.Error(c, 404, "Dashboard not found")
		return
	}

	utils.Success(c, gin.H{"message": "Dashboard and related charts deleted successfully"})
}
