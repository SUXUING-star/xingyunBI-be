// storage/oss.go
package storage

import (
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type CloudStorage struct {
	client *oss.Client
	bucket *oss.Bucket
}

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

// GetCloudStorage 获取云存储实例
func GetCloudStorage() *CloudStorage {
	return cloudStorage
}

// UploadFile 上传文件到 OSS
func UploadFile(file *multipart.FileHeader) (string, error) {
	if cloudStorage == nil || cloudStorage.bucket == nil {
		return "", fmt.Errorf("storage not initialized")
	}

	// 生成唯一文件名
	filename := time.Now().Format("20060102150405") + "_" + file.Filename
	objectKey := "uploads/" + filename

	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer src.Close()

	// 上传到 OSS
	err = cloudStorage.bucket.PutObject(objectKey, src)
	if err != nil {
		return "", fmt.Errorf("failed to upload to OSS: %v", err)
	}

	// 构造访问 URL
	bucketName := os.Getenv("OSS_BUCKET")
	cloudURL := fmt.Sprintf("https://%s.%s/%s",
		bucketName,
		"oss-cn-beijing.aliyuncs.com",
		objectKey)

	return cloudURL, nil
}
