// handlers/auth.go
package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"bi-backend/db"
	"bi-backend/models"
	"bi-backend/utils"
)

// 注册
// handlers/auth.go
func Register(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Printf("Invalid registration data: %v", err)
		utils.Error(c, 400, fmt.Sprintf("无效的注册信息: %v", err))
		return
	}

	// 打印接收到的数据
	log.Printf("Received registration data: %+v", user)

	// 验证必填字段
	if user.Username == "" || user.Password == "" || user.Email == "" {
		log.Printf("Missing required fields: username=%v, password=%v, email=%v",
			user.Username != "", user.Password != "", user.Email != "")
		utils.Error(c, 400, "用户名、密码和邮箱都不能为空")
		return
	}

	// 验证邮箱格式
	if !utils.ValidateEmail(user.Email) {
		log.Printf("Invalid email format: %s", user.Email)
		utils.Error(c, 400, "无效的邮箱格式")
		return
	}

	// 验证密码格式
	if !utils.ValidatePassword(user.Password) {
		log.Printf("Invalid password format")
		utils.Error(c, 400, "密码必须至少6位，包含字母和数字")
		return
	}

	collection := db.GetClient().Database("bi_platform").Collection("users")

	// 检查用户名和邮箱是否已存在
	var existingUser models.User
	err := collection.FindOne(context.TODO(), bson.M{
		"$or": []bson.M{
			{"username": user.Username},
			{"email": user.Email},
		},
	}).Decode(&existingUser)

	if err == nil {
		log.Printf("User already exists: username=%s, email=%s", user.Username, user.Email)
		utils.Error(c, 400, "用户名或邮箱已存在")
		return
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error(c, 500, "密码处理失败")
		return
	}

	// 生成验证token
	verifyToken := utils.GenerateRandomToken()

	// 创建新用户
	newUser := models.User{
		Username:       user.Username,
		Password:       string(hashedPassword),
		Email:          user.Email,
		Role:           "user",
		IsVerified:     false,
		VerifyToken:    verifyToken,
		TokenExpiredAt: time.Now().Add(24 * time.Hour),
		CreatedAt:      time.Now(),
		Preferences: models.UserPreferences{
			Theme:       "light",
			DataSources: []string{},
		},
	}

	result, err := collection.InsertOne(context.TODO(), newUser)
	if err != nil {
		utils.Error(c, 500, "创建用户失败")
		return
	}

	// 发送验证邮件
	go utils.SendVerificationEmail(user.Email, verifyToken)

	utils.Success(c, gin.H{
		"message": "注册成功,请查收验证邮件",
		"user_id": result.InsertedID,
	})
}

// 登录
func Login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		utils.Error(c, 400, "无效的登录信息")
		return
	}

	collection := db.GetClient().Database("bi_platform").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{
		"username": credentials.Username,
	}).Decode(&user)

	if err != nil {
		utils.Error(c, 401, "用户名或密码错误")
		return
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err != nil {
		utils.Error(c, 401, "用户名或密码错误")
		return
	}

	if !user.IsVerified {
		utils.Error(c, 401, "请先验证邮箱")
		return
	}

	// 更新最后登录时间
	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"last_login_at": time.Now()}},
	)

	// 生成JWT token
	token, err := utils.GenerateToken(user.ID.Hex(), user.Username, user.Role)
	if err != nil {
		utils.Error(c, 500, "生成token失败")
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID.Hex(),
			"username":    user.Username,
			"email":       user.Email,
			"role":        user.Role,
			"is_verified": user.IsVerified,
			"preferences": user.Preferences,
			"created_at":  user.CreatedAt,
			"last_login":  user.LastLoginAt,
		},
	})
}

// 验证邮箱
// handlers/auth.go
func VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		log.Printf("Missing verification token")
		utils.Error(c, 400, "缺少验证token")
		return
	}

	log.Printf("Attempting to verify email with token: %s", token)

	collection := db.GetClient().Database("bi_platform").Collection("users")

	// 先查找用户，以便记录日志
	var user models.User
	err := collection.FindOne(
		context.TODO(),
		bson.M{
			"verify_token":     token,
			"token_expired_at": bson.M{"$gt": time.Now()},
			"is_verified":      false,
		},
	).Decode(&user)

	if err != nil {
		log.Printf("Failed to find user with token: %v", err)
		if err == mongo.ErrNoDocuments {
			utils.Error(c, 400, "无效或已过期的验证链接")
			return
		}
		utils.Error(c, 500, "服务器错误")
		return
	}

	// 更新用户验证状态
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{
			"verify_token":     token,
			"token_expired_at": bson.M{"$gt": time.Now()},
			"is_verified":      false,
		},
		bson.M{
			"$set": bson.M{
				"is_verified":      true,
				"verify_token":     "",
				"token_expired_at": time.Time{},
				"updated_at":       time.Now(),
			},
		},
	)

	if err != nil {
		log.Printf("Failed to update user verification status: %v", err)
		utils.Error(c, 500, "更新验证状态失败")
		return
	}

	if result.ModifiedCount == 0 {
		log.Printf("No user was updated with token: %s", token)
		utils.Error(c, 400, "无效或已过期的验证链接")
		return
	}

	log.Printf("Successfully verified email for user: %s", user.Username)
	utils.Success(c, gin.H{
		"message":  "邮箱验证成功",
		"username": user.Username,
	})
}

// 忘记密码
func ForgotPassword(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, 400, "请提供邮箱地址")
		return
	}

	if !utils.ValidateEmail(input.Email) {
		utils.Error(c, 400, "无效的邮箱格式")
		return
	}

	collection := db.GetClient().Database("bi_platform").Collection("users")

	resetToken := utils.GenerateRandomToken()
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"email": input.Email},
		bson.M{
			"$set": bson.M{
				"reset_token":      resetToken,
				"token_expired_at": time.Now().Add(1 * time.Hour),
			},
		},
	)

	if err != nil || result.ModifiedCount == 0 {
		utils.Error(c, 400, "邮箱不存在")
		return
	}

	// 发送重置密码邮件
	go utils.SendPasswordResetEmail(input.Email, resetToken)

	utils.Success(c, gin.H{"message": "重置密码邮件已发送"})
}

// 重置密码
func ResetPassword(c *gin.Context) {
	var input struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, 400, "无效的请求数据")
		return
	}

	if !utils.ValidatePassword(input.Password) {
		utils.Error(c, 400, "密码必须至少8位，包含字母和数字")
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error(c, 500, "密码处理失败")
		return
	}

	collection := db.GetClient().Database("bi_platform").Collection("users")
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{
			"reset_token":      input.Token,
			"token_expired_at": bson.M{"$gt": time.Now()},
		},
		bson.M{
			"$set": bson.M{
				"password":         string(hashedPassword),
				"reset_token":      "",
				"token_expired_at": time.Time{},
			},
		},
	)

	if err != nil || result.ModifiedCount == 0 {
		utils.Error(c, 400, "无效或已过期的重置链接")
		return
	}

	utils.Success(c, gin.H{"message": "密码重置成功"})
}
