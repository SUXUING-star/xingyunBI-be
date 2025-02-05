// models/dashboard.go
package models

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Dashboard struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Layout      []ChartLayout      `bson:"layout" json:"layout"`
	EditCount   int                `bson:"edit_count" json:"edit_count"` // 添加编辑次数字段
	CreatedBy   primitive.ObjectID `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at,omitempty"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at,omitempty"`
}

type ChartLayout struct {
	ChartID primitive.ObjectID `bson:"chart_id" json:"chart_id"`
	X       int                `bson:"x" json:"x"`
	Y       int                `bson:"y" json:"y"`
	Width   int                `bson:"width" json:"width"`
	Height  int                `bson:"height" json:"height"`
}

type Chart struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Type         string             `bson:"type" json:"type"` // bar, line, pie etc.
	DataSourceID primitive.ObjectID `bson:"data_source_id" json:"data_source_id"`
	Config       ChartConfig        `bson:"config" json:"config"`
	CreatedBy    primitive.ObjectID `bson:"created_by" json:"created_by"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

type ChartConfig struct {
	Dimensions []ChartDimension `bson:"dimensions" json:"dimensions"`
	Metrics    []ChartMetric    `bson:"metrics" json:"metrics"`
	Settings   interface{}      `bson:"settings" json:"settings"`
	VisualMap  *VisualMapConfig `bson:"visual_map" json:"visualMap,omitempty"` // 添加 visualMap 配置
	DualAxis   *DualAxisConfig  `bson:"dual_axis" json:"dualAxis,omitempty"`   // 添加 dualAxis 配置
}

type VisualMapConfig struct {
	ColorField  string    `bson:"color_field" json:"colorField"`
	ColorRange  []string  `bson:"color_range" json:"colorRange"`
	SizeField   string    `bson:"size_field" json:"sizeField"`
	SizeRange   []float64 `bson:"size_range" json:"sizeRange"`
	LabelFields []string  `bson:"label_fields" json:"labelFields"` // 改为数组
}

// 新增 DualAxis 配置结构
type DualAxisConfig struct {
	Enabled bool     `bson:"enabled" json:"enabled"`
	Types   []string `bson:"types" json:"types"`
}

type ChartDimension struct {
	Field  string `bson:"field" json:"field"`
	Type   string `bson:"type" json:"type"` // date, category etc.
	Format string `bson:"format" json:"format,omitempty"`
}

type ChartMetric struct {
	Field      string `bson:"field" json:"field"`
	Aggregator string `bson:"aggregator" json:"aggregator"` // sum, avg, count etc.
	Alias      string `bson:"alias" json:"alias,omitempty"`
}

type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username       string             `bson:"username" json:"username" binding:"required"`
	Password       string             `bson:"password" json:"password" binding:"required"`
	Email          string             `bson:"email" json:"email" binding:"required,email"`
	Role           string             `bson:"role" json:"role"`
	IsVerified     bool               `bson:"is_verified" json:"is_verified"`
	VerifyToken    string             `bson:"verify_token,omitempty" json:"-"`
	TokenExpiredAt time.Time          `bson:"token_expired_at,omitempty" json:"-"`
	ResetToken     string             `bson:"reset_token,omitempty" json:"-"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	LastLoginAt    time.Time          `bson:"last_login_at" json:"last_login_at"`
	Preferences    UserPreferences    `bson:"preferences" json:"preferences"`
}

// UserStats 用户统计数据结构
type UserStats struct {
	TotalDashboards  int64                    `json:"total_dashboards"`
	TotalDataSources int64                    `json:"total_data_sources"`
	TotalCharts      int64                    `json:"total_charts"`
	TotalMLModels    int64                    `json:"total_ml_models"` // 添加机器学习模型总数
	RecentActivity   []map[string]interface{} `json:"recent_activity"`
	RecentDashboards []Dashboard              `json:"recent_dashboards"`
	RecentMLModels   []MLModel                `json:"recent_ml_models"` // 添加这一行
	UsageStats       []UsageStat              `json:"usage_stats"`
	MLModelStats     MLModelStats             `json:"ml_model_stats"` // 添加机器学习模型统计
}

// 修改 UsageStat 结构
type UsageStat struct {
	Date       string `json:"date"`
	Dashboards int64  `json:"dashboards"`
	Charts     int64  `json:"charts"`
	Queries    int64  `json:"queries"`
	MLModels   int64  `json:"ml_models"` // 添加机器学习模型数量
}

// Activity 定义活动结构
type Activity struct {
	ID           primitive.ObjectID `json:"_id"`
	Name         string             `json:"name"`
	Type         string             `json:"type"`
	CreatedAt    time.Time          `json:"created_at"`
	ActivityType string             `json:"activity_type"`
	DashboardID  primitive.ObjectID `json:"dashboard_id,omitempty"` // 添加这个字段
}

// 用户偏好设置
type UserPreferences struct {
	DefaultDashboard string   `bson:"default_dashboard" json:"default_dashboard"`
	Theme            string   `bson:"theme" json:"theme"`
	DataSources      []string `bson:"data_sources" json:"data_sources"`
}

// JWT Claims
type Claims struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// 数据源
type DataSource struct {
	ID            primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	Name          string                `bson:"name" json:"name"`
	Type          string                `bson:"type" json:"type"`
	Content       [][]string            `bson:"content" json:"content"`
	Headers       []string              `bson:"headers" json:"headers"`
	CreatedBy     primitive.ObjectID    `bson:"created_by" json:"created_by"`
	CreatedAt     time.Time             `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time             `bson:"updated_at" json:"updated_at"`
	FileURL       string                `bson:"file_url" json:"file_url"`
	Preprocessing []PreprocessingConfig `bson:"preprocessing" json:"preprocessing"`
	LinkedCharts  []primitive.ObjectID  `bson:"linked_charts,omitempty" json:"linked_charts,omitempty"`
}

type PreprocessingConfig struct {
	Field      string `bson:"field" json:"field"`           // 字段名称
	Type       string `bson:"type" json:"type"`             // 预处理类型：number/date/text
	Format     string `bson:"format" json:"format"`         // 数据格式
	Aggregator string `bson:"aggregator" json:"aggregator"` // 聚合方式
}

// MLModel 机器学习模型结构
type MLModel struct {
	ID             primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	Name           string                `bson:"name" json:"name"`
	Description    string                `bson:"description" json:"description"`
	Type           string                `bson:"type" json:"type"` // linear_regression, decision_tree, correlation
	DataSourceID   primitive.ObjectID    `bson:"data_source_id" json:"data_source_id"`
	Features       []string              `bson:"features" json:"features"` // 特征列
	Target         string                `bson:"target" json:"target"`     // 目标列
	Parameters     MLParameters          `bson:"parameters" json:"parameters"`
	Preprocessing  []PreprocessingConfig `bson:"preprocessing" json:"preprocessing"`
	TrainingResult MLResult              `bson:"training_result" json:"training_result"`
	CreatedBy      primitive.ObjectID    `bson:"created_by" json:"created_by"`
	CreatedAt      time.Time             `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time             `bson:"updated_at" json:"updated_at"`
}

// 添加机器学习模型统计结构
type MLModelStats struct {
	ModelTypes     map[string]int64   `json:"model_types"`     // 各类型模型数量
	TrainedModels  int64              `json:"trained_models"`  // 已训练模型数量
	PendingModels  int64              `json:"pending_models"`  // 未训练模型数量
	AverageMetrics map[string]float64 `json:"average_metrics"` // 平均评估指标
}

type MLParameters struct {
	// 通用参数
	TestSize float64 `bson:"test_size" json:"test_size"` // 测试集比例

	// 线性回归参数
	Regularization string  `bson:"regularization" json:"regularization"` // none, l1, l2
	Alpha          float64 `bson:"alpha" json:"alpha"`                   // 正则化强度

	// 决策树参数
	MaxDepth   int  `bson:"max_depth" json:"max_depth"`
	MinSamples int  `bson:"min_samples" json:"min_samples"`
	AutoEncode bool `bson:"auto_encode" json:"auto_encode"`

	// kmeans参数
	NClusters int `bson:"n_clusters" json:"n_clusters"`
	MaxIter   int `bson:"max_iter" json:"max_iter"`
}

type MLResult struct {
	Metrics           map[string]float64            `bson:"metrics" json:"metrics"`
	FeatureImportance map[string]float64            `bson:"feature_importance" json:"feature_importance"`
	PredictionSamples []PredictionSample            `bson:"prediction_samples" json:"prediction_samples"`
	CorrelationMatrix map[string]map[string]float64 `bson:"correlation_matrix,omitempty" json:"correlation_matrix,omitempty"`
	TreeStructure     map[string]interface{}        `bson:"tree_structure" json:"tree_structure"`
	ModelParams       []ModelParam                  `bson:"model_params" json:"model_params"`

	ClusterCenters [][]float64     `bson:"cluster_centers,omitempty" json:"cluster_centers,omitempty"`
	ClusterSizes   map[string]int  `bson:"cluster_sizes,omitempty" json:"cluster_sizes,omitempty"`
	SampleClusters []SampleCluster `bson:"sample_clusters,omitempty" json:"sample_clusters,omitempty"`
}

// 新增 K-means 样本聚类结果结构
type SampleCluster struct {
	Features []float64 `bson:"features" json:"features"`
	Cluster  int       `bson:"cluster" json:"cluster"`
}

type ModelParam struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type PredictionSample struct {
	Actual    float64 `bson:"actual" json:"actual"`
	Predicted float64 `bson:"predicted" json:"predicted"`
}
