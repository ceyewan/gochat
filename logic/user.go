package logic

import (
	"gochat/clog"
	"gochat/tools"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// dbIns 全局数据库实例
var (
	dbIns *gorm.DB
	once  sync.Once
)

// getDB 获取数据库实例（懒加载）
func getDB() *gorm.DB {
	once.Do(func() {
		dbIns = tools.GetDB()
	})
	return dbIns
}

// hashPassword 对密码进行哈希处理
// 使用bcrypt算法，默认成本因子为10
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		clog.Error("Failed to hash password: %v", err)
		return "", errors.Wrap(err, "password hashing failed")
	}
	return string(hashedBytes), nil
}

// checkPasswordHash 验证密码是否与哈希值匹配
func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Add 添加新用户
// 接受用户名和原始密码，将用户信息存储到数据库
func Add(userName, password string) error {
	clog.Debug("Attempting to add new user: %s", userName)

	// 对密码进行哈希处理
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return errors.Wrap(err, "password hashing failed")
	}

	// 创建用户记录
	user := tools.User{
		UserName:   userName,
		Password:   hashedPassword,
		CreateTime: time.Now(),
	}

	// 将用户记录插入数据库
	if err := getDB().Table(user.TableName()).Create(&user).Error; err != nil {
		clog.Error("Failed to create user %s: %v", userName, err)
		return errors.Wrap(err, "failed to create user")
	}

	clog.Info("User created successfully: %s (ID: %d)", userName, user.ID)
	return nil
}

// CheckHaveUserName 检查用户名是否已存在
// 存在则返回对应的用户信息
func CheckHaveUserName(userName string) (tools.User, error) {
	clog.Debug("Checking if username exists: %s", userName)

	var user tools.User
	if err := getDB().Table(user.TableName()).
		Where("user_name = ?", userName).
		First(&user).Error; err != nil {

		clog.Debug("User not found: %s (%v)", userName, err)
		return tools.User{}, errors.Wrap(err, "failed to query user")
	}

	clog.Debug("User found: %s (ID: %d)", userName, user.ID)
	return user, nil
}

// GetUserNameByUserID 根据用户ID获取用户名
func GetUserNameByUserID(userID uint) (string, error) {
	clog.Debug("Looking up username for user ID: %d", userID)

	var user tools.User
	if err := getDB().Table(user.TableName()).
		Where("id = ?", userID).
		First(&user).Error; err != nil {

		clog.Error("Failed to find user with ID %d: %v", userID, err)
		return "", errors.Wrap(err, "failed to query user")
	}

	clog.Debug("Found username %s for ID %d", user.UserName, userID)
	return user.UserName, nil
}

// ValidateCredentials 验证用户凭据
// 成功则返回用户信息，失败返回错误
func ValidateCredentials(userName, password string) (tools.User, error) {
	clog.Debug("Validating credentials for user: %s", userName)

	user, err := CheckHaveUserName(userName)
	if err != nil {
		return tools.User{}, errors.Wrap(err, "user not found")
	}

	if !checkPasswordHash(password, user.Password) {
		clog.Warning("Invalid password attempt for user: %s", userName)
		return tools.User{}, errors.New("invalid credentials")
	}

	clog.Info("User authenticated successfully: %s", userName)
	return user, nil
}
