// handlers/user.go
package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"bi-backend/db"
	"bi-backend/models"
	"bi-backend/utils"
)

// 获取用户资料
func GetProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)

	collection := db.GetClient().Database("bi_platform").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		utils.Error(c, 404, "用户不存在")
		return
	}

	utils.Success(c, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"role":          user.Role,
		"is_verified":   user.IsVerified,
		"preferences":   user.Preferences,
		"created_at":    user.CreatedAt,
		"last_login_at": user.LastLoginAt,
	})
}

// 更新用户资料
func UpdateProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)

	var input struct {
		Email       string                 `json:"email"`
		Preferences models.UserPreferences `json:"preferences"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, 400, "无效的请求数据")
		return
	}

	update := bson.M{
		"$set": bson.M{
			"preferences": input.Preferences,
			"updated_at":  time.Now(),
		},
	}

	// 如果修改了邮箱,需要重新验证
	if input.Email != "" && input.Email != "null" {
		if !utils.ValidateEmail(input.Email) {
			utils.Error(c, 400, "无效的邮箱格式")
			return
		}

		// 检查邮箱是否已被其他用户使用
		collection := db.GetClient().Database("bi_platform").Collection("users")
		var existingUser models.User
		err := collection.FindOne(context.TODO(), bson.M{
			"email": input.Email,
			"_id":   bson.M{"$ne": userID},
		}).Decode(&existingUser)

		if err == nil {
			utils.Error(c, 400, "该邮箱已被使用")
			return
		}

		verifyToken := utils.GenerateRandomToken()
		update["$set"].(bson.M)["email"] = input.Email
		update["$set"].(bson.M)["is_verified"] = false
		update["$set"].(bson.M)["verify_token"] = verifyToken
		update["$set"].(bson.M)["token_expired_at"] = time.Now().Add(24 * time.Hour)

		// 发送新的验证邮件
		go utils.SendVerificationEmail(input.Email, verifyToken)
	}

	collection := db.GetClient().Database("bi_platform").Collection("users")
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": userID},
		update,
	)

	if err != nil {
		utils.Error(c, 500, "更新失败")
		return
	}

	if result.ModifiedCount == 0 {
		utils.Error(c, 404, "用户不存在")
		return
	}

	utils.Success(c, gin.H{
		"message":                     "更新成功",
		"email_verification_required": input.Email != "",
	})
}

// 修改密码
func UpdatePassword(c *gin.Context) {
	userID := c.MustGet("user_id").(primitive.ObjectID)

	var input struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, 400, "无效的请求数据")
		return
	}

	// 验证新密码格式
	if !utils.ValidatePassword(input.NewPassword) {
		utils.Error(c, 400, "新密码必须至少8位，包含字母和数字")
		return
	}

	collection := db.GetClient().Database("bi_platform").Collection("users")
	var user models.User
	err := collection.FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		utils.Error(c, 404, "用户不存在")
		return
	}

	// 验证旧密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword))
	if err != nil {
		utils.Error(c, 400, "旧密码错误")
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.Error(c, 500, "密码处理失败")
		return
	}

	// 更新密码
	_, err = collection.UpdateOne( // 删除未使用的result变量
		context.TODO(),
		bson.M{"_id": userID},
		bson.M{
			"$set": bson.M{
				"password":   string(hashedPassword),
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		utils.Error(c, 500, "修改密码失败")
		return
	}

	utils.Success(c, gin.H{"message": "密码修改成功"})
}

// 获取用户列表(管理员)
func GetUsers(c *gin.Context) {
	collection := db.GetClient().Database("bi_platform").Collection("users")

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	skip := (page - 1) * pageSize

	// 查询条件
	filter := bson.M{}
	if search := c.Query("search"); search != "" {
		filter["$or"] = []bson.M{
			{"username": bson.M{"$regex": search, "$options": "i"}},
			{"email": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	// 获取总数
	total, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		utils.Error(c, 500, "获取用户总数失败")
		return
	}

	// 获取用户列表
	cursor, err := collection.Find(context.TODO(), filter, options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{"created_at", -1}}))

	if err != nil {
		utils.Error(c, 500, "获取用户列表失败")
		return
	}
	defer cursor.Close(context.TODO())

	var users []models.User
	if err = cursor.All(context.TODO(), &users); err != nil {
		utils.Error(c, 500, "解析用户数据失败")
		return
	}

	utils.Success(c, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"users":     users,
	})
}
