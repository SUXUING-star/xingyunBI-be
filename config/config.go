// config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Email    EmailConfig
	Frontend FrontendConfig
}

type ServerConfig struct {
	Port string
	Mode string
}

type DatabaseConfig struct {
	URI      string
	Database string
	PoolSize uint64
}

type JWTConfig struct {
	Secret        string
	ExpireDays    int
	RefreshSecret string
}

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

var GlobalConfig Config

type FrontendConfig struct {
	URL string
}

func Init() error {
	// 设置默认值并从环境变量加载
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	poolSize, _ := strconv.ParseUint(os.Getenv("MONGODB_POOL_SIZE"), 10, 64)
	if poolSize == 0 {
		poolSize = 100
	}

	expireDays, _ := strconv.Atoi(os.Getenv("JWT_EXPIRE_DAYS"))
	if expireDays == 0 {
		expireDays = 7
	}

	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if smtpPort == 0 {
		smtpPort = 587
	}

	GlobalConfig = Config{
		Server: ServerConfig{
			Port: port,
			Mode: os.Getenv("GIN_MODE"),
		},
		Database: DatabaseConfig{
			URI:      os.Getenv("MONGODB_URI"),
			Database: "bi_platform",
			PoolSize: poolSize,
		},
		JWT: JWTConfig{
			Secret:        os.Getenv("JWT_SECRET"),
			ExpireDays:    expireDays,
			RefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		},
		Email: EmailConfig{
			Host:     os.Getenv("EMAIL_SMTP_HOST"),
			Port:     587,                          // QQ邮箱固定使用587端口
			Username: os.Getenv("EMAIL_FROM"),      // 使用EMAIL_FROM作为用户名
			Password: os.Getenv("EMAIL_AUTH_CODE"), // 使用授权码作为密码
			From:     os.Getenv("EMAIL_FROM"),      // 发件人邮箱也是EMAIL_FROM
		},
		Frontend: FrontendConfig{
			URL: os.Getenv("FRONTEND_URL"),
		},
	}

	// 验证必需的配置
	if GlobalConfig.Database.URI == "" {
		return fmt.Errorf("MONGODB_URI is required")
	}
	if GlobalConfig.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if GlobalConfig.Email.Host == "" || GlobalConfig.Email.Username == "" || GlobalConfig.Email.Password == "" {
		return fmt.Errorf("SMTP configuration is required")
	}

	return nil
}
