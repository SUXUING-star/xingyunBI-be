// db/mongodb.go
package db

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"bi-backend/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client *mongo.Client
	once   sync.Once
)

// 初始化MongoDB连接
func Init(ctx context.Context) error {
	var err error
	once.Do(func() {
		// 设置客户端选项
		clientOptions := options.Client().
			ApplyURI(config.GlobalConfig.Database.URI).
			SetMaxPoolSize(config.GlobalConfig.Database.PoolSize).
			SetMinPoolSize(5).
			SetMaxConnIdleTime(5 * time.Minute).
			SetTLSConfig(&tls.Config{
				InsecureSkipVerify: true, // 相当于 tlsAllowInvalidCertificates=true
			})

		// 连接MongoDB
		client, err = mongo.Connect(ctx, clientOptions)
		if err != nil {
			return
		}

		// 验证连接
		err = client.Ping(ctx, nil)
		if err != nil {
			return
		}

		// 创建索引
		err = createIndexes(ctx)
	})
	return err
}

// 获取MongoDB客户端实例
func GetClient() *mongo.Client {
	return client
}

// 获取数据库实例
func GetDB() *mongo.Database {
	return client.Database(config.GlobalConfig.Database.Database)
}

// 关闭数据库连接
func Close(ctx context.Context) error {
	if client != nil {
		return client.Disconnect(ctx)
	}
	return nil
}

// 创建索引
func createIndexes(ctx context.Context) error {
	db := GetDB()

	// 用户集合索引
	_, err := db.Collection("users").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{"username", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{"email", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{"verify_token", 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys:    bson.D{{"reset_token", 1}},
			Options: options.Index().SetSparse(true),
		},
	})
	if err != nil {
		return err
	}

	// 数据源集合索引
	_, err = db.Collection("data_sources").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{"created_by", 1}},
		},
		{
			Keys:    bson.D{{"name", 1}, {"created_by", 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return err
	}

	// 图表集合索引
	_, err = db.Collection("charts").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{"created_by", 1}},
		},
		{
			Keys: bson.D{{"data_source_id", 1}},
		},
	})
	if err != nil {
		return err
	}

	// ML模型集合索引
	_, err = db.Collection("ml_models").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{"created_by", 1}},
		},
		{
			Keys: bson.D{{"data_source_id", 1}},
		},
	})

	return err
}
func GetCollection(name string) *mongo.Collection {
	return GetDB().Collection(name)
}
