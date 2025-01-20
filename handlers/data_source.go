// handlers/data_source.go
package handlers

import (
	"bi-backend/db"
	"bi-backend/models"
	"bi-backend/storage"
	"bi-backend/utils"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CloudStorage 结构体定义
type CloudStorage struct {
	client *oss.Client
	bucket *oss.Bucket
}

// ParseExcelFile 解析Excel文件
func ParseExcelFile(file multipart.File) ([][]string, []string, error) {
	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, err
	}
	defer xlsx.Close()

	// 获取第一个工作表
	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, errors.New("excel文件没有工作表")
	}

	// 获取所有行
	rows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		return nil, nil, err
	}

	if len(rows) == 0 {
		return nil, nil, errors.New("excel文件为空")
	}

	// 第一行作为表头
	headers := rows[0]
	content := rows[1:]

	return content, headers, nil
}

// ParseJSONFile 解析JSON文件
func ParseJSONFile(file multipart.File) ([][]string, []string, error) {
	var jsonData []map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&jsonData); err != nil {
		return nil, nil, err
	}

	if len(jsonData) == 0 {
		return nil, nil, errors.New("JSON数据为空")
	}

	// 提取表头
	headers := make([]string, 0)
	for key := range jsonData[0] {
		headers = append(headers, key)
	}

	// 将数据转换为二维字符串数组
	var content [][]string
	for _, item := range jsonData {
		var row []string
		for _, header := range headers {
			value := fmt.Sprint(item[header])
			row = append(row, value)
		}
		content = append(content, row)
	}

	return content, headers, nil
}

// 全局变量
var cloudStorage *CloudStorage

// InitCloudStorage 初始化云存储
func InitCloudStorage() error {
	required := []string{"OSS_ACCESS_KEY_ID", "OSS_ACCESS_KEY_SECRET", "OSS_ENDPOINT", "OSS_BUCKET"}
	for _, env := range required {
		if os.Getenv(env) == "" {
			return fmt.Errorf("missing required environment variable: %s", env)
		}
	}

	log.Printf("Initializing OSS client with endpoint: %s, bucket: %s",
		os.Getenv("OSS_ENDPOINT"),
		os.Getenv("OSS_BUCKET"))

	client, err := oss.New(
		os.Getenv("OSS_ENDPOINT"),
		os.Getenv("OSS_ACCESS_KEY_ID"),
		os.Getenv("OSS_ACCESS_KEY_SECRET"),
	)
	if err != nil {
		return fmt.Errorf("failed to create OSS client: %v", err)
	}

	bucket, err := client.Bucket(os.Getenv("OSS_BUCKET"))
	if err != nil {
		return fmt.Errorf("failed to get bucket: %v", err)
	}

	cloudStorage = &CloudStorage{
		client: client,
		bucket: bucket,
	}

	log.Println("Cloud storage initialized successfully")
	return nil
}

// 解析 Excel 文件
func parseExcel(file io.Reader) ([][]string, []string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open excel file: %v", err)
	}
	defer f.Close()

	// 获取第一个 Sheet
	firstSheet := f.GetSheetList()[0]

	// 读取所有行
	rows, err := f.GetRows(firstSheet)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read excel rows: %v", err)
	}

	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("empty excel file")
	}

	// 第一行作为表头
	headers := rows[0]
	// 其余行作为数据
	data := rows[1:]

	return data, headers, nil
}

// 解析 CSV 文件
func parseCSV(file io.Reader) ([][]string, []string, error) {
	reader := csv.NewReader(file)

	// 读取所有记录
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	if len(records) == 0 {
		return nil, nil, fmt.Errorf("empty CSV file")
	}

	// 第一行作为表头
	headers := records[0]
	// 其余行作为数据
	data := records[1:]

	return data, headers, nil
}

// 解析 JSON 文件
func parseJSON(file io.Reader) ([][]string, []string, error) {
	var jsonData []map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&jsonData); err != nil {
		return nil, nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	if len(jsonData) == 0 {
		return nil, nil, fmt.Errorf("empty JSON file")
	}

	// 从第一条记录获取表头
	headers := make([]string, 0)
	for key := range jsonData[0] {
		headers = append(headers, key)
	}

	// 转换数据
	var data [][]string
	for _, item := range jsonData {
		row := make([]string, len(headers))
		for i, header := range headers {
			if val, ok := item[header]; ok {
				row[i] = fmt.Sprint(val)
			}
		}
		data = append(data, row)
	}

	return data, headers, nil
}

func UploadDataSource(c *gin.Context) {
	log.Println("Starting file upload...")

	// 1. 获取文件
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("Error getting form file: %v", err)
		utils.Error(c, 400, "No file uploaded")
		return
	}

	// 2. 获取文件类型
	fileType := c.PostForm("type")
	log.Printf("File type: %s, filename: %s", fileType, file.Filename)

	// 3. 生成带时间戳的唯一文件名
	timestamp := time.Now().Format("20060102150405")
	fileName := fmt.Sprintf("%s_%s", timestamp, file.Filename)

	// 4. 上传到 OSS
	cloudURL, err := storage.UploadFile(file)
	if err != nil {
		log.Printf("Error uploading to OSS: %v", err)
		utils.Error(c, 500, "Failed to upload file")
		return
	}

	log.Printf("File uploaded successfully. URL: %s", cloudURL)

	// 5. 解析文件内容
	src, err := file.Open()
	if err != nil {
		log.Printf("Error opening file for parsing: %v", err)
		utils.Error(c, 500, "Failed to process file")
		return
	}
	defer src.Close()

	var data [][]string
	var headers []string

	switch fileType {
	case "excel":
		data, headers, err = parseExcel(src)
	case "csv":
		data, headers, err = parseCSV(src)
	case "json":
		data, headers, err = parseJSON(src)
	default:
		utils.Error(c, 400, "Unsupported file type")
		return
	}

	if err != nil {
		log.Printf("Error parsing file: %v", err)
		utils.Error(c, 400, "Failed to parse file")
		return
	}

	// 6. 创建数据源记录
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "Unauthorized")
		return
	}

	dataSource := models.DataSource{
		Name:      fileName, // 使用带时间戳的文件名
		Type:      fileType,
		FileURL:   cloudURL,
		Content:   data,
		Headers:   headers,
		CreatedBy: userID.(primitive.ObjectID),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 7. 保存到数据库
	collection := db.GetClient().Database("bi_platform").Collection("data_sources")
	result, err := collection.InsertOne(context.TODO(), dataSource)
	if err != nil {
		log.Printf("Error saving to database: %v", err)
		utils.Error(c, 500, "Failed to save data source")
		return
	}

	dataSource.ID = result.InsertedID.(primitive.ObjectID)
	log.Printf("Data source created successfully: %v", dataSource.ID)

	utils.Success(c, dataSource)
}

// 创建数据源
func CreateDataSource(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.Error(c, 400, "无法读取文件")
		return
	}
	defer file.Close()

	var content [][]string
	var headers []string

	// 根据文件类型选择解析方法
	switch filepath.Ext(header.Filename) {
	case ".csv":
		content, headers, err = parseCSV(file)
	case ".xlsx", ".xls":
		content, headers, err = utils.ParseExcelFile(file)
	case ".json":
		content, headers, err = utils.ParseJSONFile(file)
	default:
		utils.Error(c, 400, "不支持的文件类型")
		return
	}

	if err != nil {
		utils.Error(c, 400, "文件解析失败: "+err.Error())
		return
	}

	dataSource := models.DataSource{
		Name:      header.Filename,
		Type:      filepath.Ext(header.Filename)[1:],
		Content:   content,
		Headers:   headers,
		CreatedBy: c.MustGet("user_id").(primitive.ObjectID),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	collection := db.GetClient().Database("bi_platform").Collection("data_sources")
	result, err := collection.InsertOne(context.TODO(), dataSource)
	if err != nil {
		utils.Error(c, 500, "保存数据失败")
		return
	}

	dataSource.ID = result.InsertedID.(primitive.ObjectID)
	utils.Success(c, dataSource)
}

// 获取数据源列表
func GetDataSources(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)

	collection := db.GetClient().Database("bi_platform").Collection("data_sources")
	cursor, err := collection.Find(context.TODO(), bson.M{"created_by": userID})
	if err != nil {
		utils.Error(c, 500, "获取数据源失败")
		return
	}
	defer cursor.Close(context.TODO())

	var dataSources []models.DataSource
	if err = cursor.All(context.TODO(), &dataSources); err != nil {
		utils.Error(c, 500, "解析数据失败")
		return
	}

	utils.Success(c, dataSources)
}

// 获取单个数据源
func GetDataSource(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "无效的数据源ID")
		return
	}

	collection := db.GetClient().Database("bi_platform").Collection("data_sources")
	var dataSource models.DataSource
	err = collection.FindOne(context.TODO(), bson.M{
		"_id":        id,
		"created_by": c.MustGet("user_id").(primitive.ObjectID),
	}).Decode(&dataSource)

	if err != nil {
		utils.Error(c, 404, "数据源不存在")
		return
	}

	utils.Success(c, dataSource)
}

// 更新数据源
func UpdateDataSource(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "无效的数据源ID")
		return
	}

	var input struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, 400, "无效的请求数据")
		return
	}

	collection := db.GetClient().Database("bi_platform").Collection("data_sources")
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":        id,
			"created_by": c.MustGet("user_id").(primitive.ObjectID),
		},
		bson.M{
			"$set": bson.M{
				"name":       input.Name,
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		utils.Error(c, 500, "更新失败")
		return
	}

	if result.ModifiedCount == 0 {
		utils.Error(c, 404, "数据源不存在")
		return
	}

	utils.Success(c, gin.H{"message": "更新成功"})
}

// 删除数据源
func DeleteDataSource(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "无效的数据源ID")
		return
	}

	// 获取数据源信息
	collection := db.GetClient().Database("bi_platform").Collection("data_sources")
	var dataSource models.DataSource
	err = collection.FindOne(context.TODO(), bson.M{
		"_id":        id,
		"created_by": c.MustGet("user_id").(primitive.ObjectID),
	}).Decode(&dataSource)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			utils.Error(c, 404, "数据源不存在")
			return
		}
		utils.Error(c, 500, "获取数据源信息失败")
		return
	}

	// 删除关联的图表
	if len(dataSource.LinkedCharts) > 0 {
		chartCollection := db.GetClient().Database("bi_platform").Collection("charts")
		_, err = chartCollection.DeleteMany(context.TODO(), bson.M{
			"_id": bson.M{"$in": dataSource.LinkedCharts},
		})
		if err != nil {
			log.Printf("Failed to delete linked charts: %v", err)
		}

		// 更新包含这些图表的仪表盘
		dashboardCollection := db.GetClient().Database("bi_platform").Collection("dashboards")
		_, err = dashboardCollection.UpdateMany(
			context.TODO(),
			bson.M{"created_by": c.MustGet("user_id").(primitive.ObjectID)},
			bson.M{
				"$pull": bson.M{
					"layout": bson.M{
						"chart_id": bson.M{"$in": dataSource.LinkedCharts},
					},
				},
			},
		)
		if err != nil {
			log.Printf("Failed to update dashboards: %v", err)
		}
	}

	// 删除数据源
	result, err := collection.DeleteOne(context.TODO(), bson.M{
		"_id":        id,
		"created_by": c.MustGet("user_id").(primitive.ObjectID),
	})

	if err != nil {
		utils.Error(c, 500, "删除失败")
		return
	}

	if result.DeletedCount == 0 {
		utils.Error(c, 404, "数据源不存在")
		return
	}

	utils.Success(c, gin.H{
		"message": "删除成功",
		"details": fmt.Sprintf("已删除数据源及其关联的 %d 个图表", len(dataSource.LinkedCharts)),
	})
}

// handlers/data_source.go
func UpdatePreprocessing(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Error(c, 400, "无效的数据源ID")
		return
	}

	var input struct {
		Preprocessing []models.PreprocessingConfig `json:"preprocessing"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, 400, "无效的请求数据")
		return
	}

	collection := db.GetClient().Database("bi_platform").Collection("data_sources")
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":        id,
			"created_by": c.MustGet("user_id").(primitive.ObjectID),
		},
		bson.M{
			"$set": bson.M{
				"preprocessing": input.Preprocessing,
				"updated_at":    time.Now(),
			},
		},
	)

	if err != nil {
		utils.Error(c, 500, "更新失败")
		return
	}
	// 修改这里的判断逻辑
	if result.ModifiedCount == 0 {
		utils.Error(c, 404, "数据源不存在")
		return
	}
	utils.Success(c, gin.H{"message": "更新成功"})
}
