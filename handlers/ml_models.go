// handlers/ml_model.go
package handlers

import (
	"bi-backend/db"
	"bi-backend/models"
	"bi-backend/utils"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// handlers/ml_model.go 中的 CreateMLModel 函数
func CreateMLModel(c *gin.Context) {
	// 打印收到的请求数据，以便调试
	body, _ := io.ReadAll(c.Request.Body)
	log.Printf("Received request body: %s", string(body))
	// 记得重新设置 body
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var model models.MLModel
	if err := c.ShouldBindJSON(&model); err != nil {
		log.Printf("Binding JSON error: %v", err)
		utils.Error(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	// 打印绑定后的模型数据
	log.Printf("Bound model data: %+v", model)

	// 从 context 中获取用户 ID
	userID := c.MustGet("user_id").(primitive.ObjectID)

	// 验证数据源ID
	if !primitive.IsValidObjectID(model.DataSourceID.Hex()) {
		utils.Error(c, 400, "无效的数据源ID")
		return
	}

	// 设置创建信息
	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now
	model.CreatedBy = userID

	// 插入数据库
	result, err := db.GetClient().Database("bi_platform").Collection("ml_models").InsertOne(
		context.Background(),
		model,
	)
	if err != nil {
		log.Printf("Database error: %v", err)
		utils.Error(c, 500, "创建模型失败")
		return
	}

	// 设置返回的ID
	model.ID = result.InsertedID.(primitive.ObjectID)
	utils.Success(c, model)
}

// GetMLModels 获取机器学习模型列表
func GetMLModels(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)

	// 添加日志来检查 userID
	//log.Printf("Fetching models for userID: %v", userID)

	var models []models.MLModel
	cursor, err := db.GetClient().Database("bi_platform").Collection("ml_models").Find(
		context.Background(),
		bson.M{"created_by": userID},
	)
	if err != nil {
		log.Printf("Error fetching models: %v", err)
		utils.Error(c, 500, "获取模型列表失败")
		return
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &models); err != nil {
		log.Printf("Error decoding models: %v", err)
		utils.Error(c, 500, "解析模型数据失败")
		return
	}

	// 添加日志来检查查询结果
	//log.Printf("Found %d models", len(models))
	//log.Printf("Models: %+v", models)

	utils.Success(c, models)
}

// GetMLModel 获取单个机器学习模型
func GetMLModel(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "无效的模型ID")
		return
	}

	var model models.MLModel
	err = db.GetClient().Database("bi_platform").Collection("ml_models").FindOne(context.Background(), bson.M{
		"_id": id,
	}).Decode(&model)

	if err == mongo.ErrNoDocuments {
		utils.Error(c, 404, "模型不存在")
		return
	} else if err != nil {
		utils.Error(c, 500, "获取模型失败")
		return
	}

	utils.Success(c, model)
}

// UpdateMLModel 更新机器学习模型
func UpdateMLModel(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "无效的模型ID")
		return
	}

	var model models.MLModel
	if err := c.ShouldBindJSON(&model); err != nil {
		utils.Error(c, 400, "无效的请求数据")
		return
	}

	model.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"name":          model.Name,
			"description":   model.Description,
			"type":          model.Type,
			"features":      model.Features,
			"target":        model.Target,
			"parameters":    model.Parameters,
			"preprocessing": model.Preprocessing,
			"updated_at":    model.UpdatedAt,
		},
	}

	result, err := db.GetClient().Database("bi_platform").Collection("ml_models").UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		update,
	)

	if err != nil {
		utils.Error(c, 500, "更新模型失败")
		return
	}

	if result.MatchedCount == 0 {
		utils.Error(c, 404, "模型不存在")
		return
	}

	utils.Success(c, gin.H{"message": "更新成功"})
}

// handlers/ml_model.go 中的 UpdateMLModelResult 函数
func UpdateMLModelResult(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "无效的模型ID")
		return
	}

	// 记录请求体内容
	body, _ := io.ReadAll(c.Request.Body)
	log.Printf("Received training result: %s", string(body))
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var input struct {
		TrainingResult models.MLResult `json:"training_result"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("Binding JSON error: %v", err)
		utils.Error(c, 400, "无效的请求数据")
		return
	}

	update := bson.M{
		"$set": bson.M{
			"training_result": bson.M{
				"metrics":            input.TrainingResult.Metrics,
				"feature_importance": input.TrainingResult.FeatureImportance,
				"prediction_samples": input.TrainingResult.PredictionSamples,
				"tree_structure":     input.TrainingResult.TreeStructure,
				"correlation_matrix": input.TrainingResult.CorrelationMatrix,
				"model_params":       input.TrainingResult.ModelParams,
				"cluster_centers":    input.TrainingResult.ClusterCenters,
				"cluster_sizes":      input.TrainingResult.ClusterSizes,
				"sample_clusters":    input.TrainingResult.SampleClusters,
			},
			"updated_at": time.Now(),
		},
	}

	result, err := db.GetClient().Database("bi_platform").Collection("ml_models").UpdateOne(
		context.Background(),
		bson.M{"_id": id},
		update,
	)

	if err != nil {
		log.Printf("Database error: %v", err)
		utils.Error(c, 500, "更新训练结果失败")
		return
	}

	if result.MatchedCount == 0 {
		utils.Error(c, 404, "模型不存在")
		return
	}

	// 验证保存结果
	var savedModel models.MLModel
	err = db.GetClient().Database("bi_platform").Collection("ml_models").
		FindOne(context.Background(), bson.M{"_id": id}).
		Decode(&savedModel)

	if err == nil {
		log.Printf("Saved training result structure: %+v", savedModel.TrainingResult)
	}

	utils.Success(c, gin.H{"message": "训练结果更新成功"})
}

// DeleteMLModel 删除机器学习模型
func DeleteMLModel(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "无效的模型ID")
		return
	}

	result, err := db.GetClient().Database("bi_platform").Collection("ml_models").DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		utils.Error(c, 500, "删除模型失败")
		return
	}

	if result.DeletedCount == 0 {
		utils.Error(c, 404, "模型不存在")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}
