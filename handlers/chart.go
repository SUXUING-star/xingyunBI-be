package handlers

import (
	"bi-backend/db"     // 导入数据库操作相关的包
	"bi-backend/models" // 导入数据模型相关的包
	"bi-backend/utils"  // 导入工具函数相关的包
	"context"           // 导入上下文操作相关的包
	"log"               // 导入日志相关的包
	"strings"           // 导入字符串操作相关的包
	"time"              // 导入时间操作相关的包

	"github.com/gin-gonic/gin"                   // 导入 Gin 框架相关的包
	"go.mongodb.org/mongo-driver/bson"           // 导入 MongoDB BSON 相关的包
	"go.mongodb.org/mongo-driver/bson/primitive" // 导入 MongoDB BSON 原始类型相关的包
)

// CreateChart 创建图表
func CreateChart(c *gin.Context) {
	var chart models.Chart // 定义一个 Chart 结构体变量，用于接收请求的 JSON 数据
	// 将请求的 JSON 数据绑定到 chart 变量
	if err := c.ShouldBindJSON(&chart); err != nil {
		// 如果绑定失败，返回 400 错误，并提示 "Invalid request data"
		utils.Error(c, 400, "Invalid request data")
		return
	}

	// 获取数据源集合
	dsCollection := db.GetClient().Database("bi_platform").Collection("data_sources")
	var dataSource models.DataSource // 定义一个 DataSource 结构体变量
	// 查询数据源是否存在
	err := dsCollection.FindOne(context.TODO(), bson.M{
		"_id":        chart.DataSourceID,                        // 根据 chart 中的 DataSourceID 查找
		"created_by": c.MustGet("user_id").(primitive.ObjectID), // 并且确保创建者是当前用户
	}).Decode(&dataSource) // 将查询结果解码到 dataSource 变量
	if err != nil {
		// 如果数据源不存在，返回 404 错误，并提示 "Data source not found"
		utils.Error(c, 404, "Data source not found")
		return
	}

	// 设置图表的创建者和创建、更新时间
	chart.CreatedBy = c.MustGet("user_id").(primitive.ObjectID) // 从上下文中获取当前用户 ID，并设置为图表的创建者
	chart.CreatedAt = time.Now()                                // 设置图表的创建时间为当前时间
	chart.UpdatedAt = time.Now()                                // 设置图表的更新时间为当前时间

	// 获取图表集合
	collection := db.GetClient().Database("bi_platform").Collection("charts")
	result, err := collection.InsertOne(context.TODO(), chart) // 将图表数据插入到数据库
	if err != nil {
		// 如果插入失败，返回 500 错误，并提示 "Failed to create chart"
		utils.Error(c, 500, "Failed to create chart")
		return
	}

	chart.ID = result.InsertedID.(primitive.ObjectID) // 获取插入的图表的 ID，并赋值给 chart 变量

	// 更新数据源的 LinkedCharts 字段，将新创建的图表 ID 添加到数据源的 linked_charts 数组中
	_, err = dsCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": chart.DataSourceID}, // 根据数据源 ID 查找
		bson.M{
			"$addToSet": bson.M{ // 使用 $addToSet 操作符，向数组中添加元素，如果已存在则不会重复添加
				"linked_charts": chart.ID, // 将图表 ID 添加到 linked_charts 数组
			},
		},
	)
	if err != nil {
		// 如果更新数据源失败，记录错误日志
		log.Printf("Failed to update data source with linked chart: %v", err)
	}
	// 如果有关联的仪表盘，增加仪表盘的编辑次数
	if dashboardID := c.Query("dashboard_id"); dashboardID != "" { // 获取 dashboard_id 查询参数
		if objID, err := primitive.ObjectIDFromHex(dashboardID); err == nil { // 将仪表盘 ID 转换为 ObjectID
			incrementDashboardEditCount(objID) // 调用函数增加仪表盘的编辑次数
		}
	}

	utils.Success(c, chart) // 返回成功响应，并将创建的图表数据返回
}

// UpdateChartConfig 更新图表配置
func UpdateChartConfig(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id")) // 从 URL 参数中获取图表 ID，并转换为 ObjectID
	if err != nil {
		// 如果图表 ID 转换失败，返回 400 错误，并提示 "Invalid chart ID"
		utils.Error(c, 400, "Invalid chart ID")
		return
	}

	var config models.ChartConfig // 定义一个 ChartConfig 结构体变量，用于接收请求的 JSON 数据
	if err := c.ShouldBindJSON(&config); err != nil {
		// 如果 JSON 绑定失败，返回 400 错误，并提示 "Invalid request data"
		utils.Error(c, 400, "Invalid request data")
		return
	}

	// 获取图表集合
	collection := db.GetClient().Database("bi_platform").Collection("charts")
	// 更新图表配置
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":        id,                                        // 根据图表 ID 查找
			"created_by": c.MustGet("user_id").(primitive.ObjectID), // 并且确保创建者是当前用户
		},
		bson.M{
			"$set": bson.M{ // 使用 $set 操作符，更新指定的字段
				"config":     config,     // 更新图表配置
				"updated_at": time.Now(), // 更新图表更新时间
			},
		},
	)

	if err != nil {
		// 如果更新失败，返回 500 错误，并提示 "Failed to update chart"
		utils.Error(c, 500, "Failed to update chart")
		return
	}

	if result.ModifiedCount == 0 {
		// 如果更新的文档数为 0，表示图表不存在，返回 404 错误，并提示 "Chart not found"
		utils.Error(c, 404, "Chart not found")
		return
	}
	// 如果有关联的仪表盘，增加仪表盘的编辑次数
	if dashboardID := c.Query("dashboard_id"); dashboardID != "" { // 获取 dashboard_id 查询参数
		if objID, err := primitive.ObjectIDFromHex(dashboardID); err == nil { // 将仪表盘 ID 转换为 ObjectID
			incrementDashboardEditCount(objID) // 调用函数增加仪表盘的编辑次数
		}
	}

	utils.Success(c, gin.H{"message": "Chart updated successfully"}) // 返回成功响应，并提示 "Chart updated successfully"
}

// GetChart 获取图表
func GetChart(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id")) // 从 URL 参数中获取图表 ID，并转换为 ObjectID
	if err != nil {
		// 如果图表 ID 转换失败，返回 400 错误，并提示 "Invalid chart ID"
		utils.Error(c, 400, "Invalid chart ID")
		return
	}

	// 获取图表集合
	collection := db.GetClient().Database("bi_platform").Collection("charts")
	var chart models.Chart // 定义一个 Chart 结构体变量
	// 根据 ID 和创建者查找图表
	err = collection.FindOne(context.TODO(), bson.M{
		"_id":        id,                                        // 根据图表 ID 查找
		"created_by": c.MustGet("user_id").(primitive.ObjectID), // 并且确保创建者是当前用户
	}).Decode(&chart) // 将查询结果解码到 chart 变量

	if err != nil {
		// 如果图表不存在，返回 404 错误，并提示 "Chart not found"
		utils.Error(c, 404, "Chart not found")
		return
	}

	// 获取数据源信息
	dsCollection := db.GetClient().Database("bi_platform").Collection("data_sources")
	var dataSource models.DataSource // 定义一个 DataSource 结构体变量
	err = dsCollection.FindOne(context.TODO(), bson.M{
		"_id": chart.DataSourceID, // 根据图表的 DataSourceID 查找数据源
	}).Decode(&dataSource) // 将查询结果解码到 dataSource 变量

	if err == nil {
		// 如果找到了数据源，则返回图表和数据源的信息
		utils.Success(c, gin.H{
			"chart":      chart,      // 返回图表信息
			"dataSource": dataSource, // 返回数据源信息
		})
		return
	}

	utils.Success(c, chart) // 如果没有找到数据源，则只返回图表信息
}

// GetCharts 获取图表列表
func GetCharts(c *gin.Context) {
	// 获取图表ID列表，逗号分隔
	chartIDs := strings.Split(c.Query("ids"), ",") // 从 URL 参数中获取 ids 列表，并用逗号分割
	var objectIDs []primitive.ObjectID             // 定义一个 ObjectID 数组，用于存储转换后的 ID
	for _, id := range chartIDs {                  // 遍历 ID 列表
		objectID, err := primitive.ObjectIDFromHex(id) // 将 ID 转换为 ObjectID
		if err != nil {
			// 如果 ID 转换失败，返回 400 错误，并提示 "Invalid chart ID"
			utils.Error(c, 400, "Invalid chart ID")
			return
		}
		objectIDs = append(objectIDs, objectID) // 将转换后的 ID 添加到数组中
	}

	// 获取图表集合
	collection := db.GetClient().Database("bi_platform").Collection("charts")
	// 查询符合条件的图表
	cursor, err := collection.Find(context.TODO(), bson.M{
		"_id":        bson.M{"$in": objectIDs},                  // 根据 ID 列表查找
		"created_by": c.MustGet("user_id").(primitive.ObjectID), // 并且确保创建者是当前用户
	})

	if err != nil {
		// 如果查询失败，返回 500 错误，并提示 "Failed to fetch charts"
		utils.Error(c, 500, "Failed to fetch charts")
		return
	}
	defer cursor.Close(context.TODO()) // 确保游标被正确关闭

	var charts []models.Chart                                  // 定义一个 Chart 结构体数组
	if err = cursor.All(context.TODO(), &charts); err != nil { // 将查询结果解码到数组中
		// 如果解码失败，返回 500 错误，并提示 "Failed to decode charts"
		utils.Error(c, 500, "Failed to decode charts")
		return
	}

	utils.Success(c, charts) // 返回成功响应，并将图表列表返回
}

// UpdateChart 更新图表
func UpdateChart(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id")) // 从 URL 参数中获取图表 ID，并转换为 ObjectID
	if err != nil {
		// 如果图表 ID 转换失败，返回 400 错误，并提示 "Invalid chart ID"
		utils.Error(c, 400, "Invalid chart ID")
		return
	}

	// 定义一个匿名结构体，用于接收请求的 JSON 数据
	var updateData struct {
		Name   string             `json:"name"`   // 图表名称
		Type   string             `json:"type"`   // 图表类型
		Config models.ChartConfig `json:"config"` // 图表配置
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		// 如果 JSON 绑定失败，返回 400 错误，并提示 "Invalid request data"
		utils.Error(c, 400, "Invalid request data")
		return
	}

	// 获取图表集合
	collection := db.GetClient().Database("bi_platform").Collection("charts")
	// 更新图表
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":        id,                                        // 根据图表 ID 查找
			"created_by": c.MustGet("user_id").(primitive.ObjectID), // 并且确保创建者是当前用户
		},
		bson.M{
			"$set": bson.M{ // 使用 $set 操作符，更新指定的字段
				"name":       updateData.Name,   // 更新图表名称
				"type":       updateData.Type,   // 更新图表类型
				"config":     updateData.Config, // 更新图表配置
				"updated_at": time.Now(),        // 更新图表更新时间
			},
		},
	)

	if err != nil {
		// 如果更新失败，返回 500 错误，并提示 "Failed to update chart"
		utils.Error(c, 500, "Failed to update chart")
		return
	}

	if result.ModifiedCount == 0 {
		// 如果更新的文档数为 0，表示图表不存在，返回 404 错误，并提示 "Chart not found"
		utils.Error(c, 404, "Chart not found")
		return
	}

	utils.Success(c, gin.H{"message": "Chart updated successfully"}) // 返回成功响应，并提示 "Chart updated successfully"
}

// DeleteChart 删除图表
func DeleteChart(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id")) // 从 URL 参数中获取图表 ID，并转换为 ObjectID
	if err != nil {
		// 如果图表 ID 转换失败，返回 400 错误，并提示 "Invalid chart ID"
		utils.Error(c, 400, "Invalid chart ID")
		return
	}

	// 获取图表集合
	collection := db.GetClient().Database("bi_platform").Collection("charts")
	// 删除图表
	result, err := collection.DeleteOne(context.TODO(), bson.M{
		"_id":        id,                                        // 根据图表 ID 查找
		"created_by": c.MustGet("user_id").(primitive.ObjectID), // 并且确保创建者是当前用户
	})

	if err != nil {
		// 如果删除失败，返回 500 错误，并提示 "Failed to delete chart"
		utils.Error(c, 500, "Failed to delete chart")
		return
	}

	if result.DeletedCount == 0 {
		// 如果删除的文档数为 0，表示图表不存在，返回 404 错误，并提示 "Chart not found"
		utils.Error(c, 404, "Chart not found")
		return
	}

	// 更新相关仪表盘，移除已删除的图表
	dashboardColl := db.GetClient().Database("bi_platform").Collection("dashboards")
	// 从仪表盘布局中删除已删除的图表
	_, err = dashboardColl.UpdateMany(
		context.TODO(),
		bson.M{"created_by": c.MustGet("user_id").(primitive.ObjectID)}, // 根据用户 ID 查找仪表盘
		bson.M{
			"$pull": bson.M{ // 使用 $pull 操作符，从数组中删除元素
				"layout": bson.M{"chart_id": id}, // 从 layout 数组中删除 chart_id 为指定 ID 的元素
			},
		},
	)

	if err != nil {
		// 记录错误但不影响删除图表的结果
		log.Printf("Error updating dashboards after chart deletion: %v", err)
	}

	utils.Success(c, gin.H{"message": "Chart deleted successfully"}) // 返回成功响应，并提示 "Chart deleted successfully"
}
