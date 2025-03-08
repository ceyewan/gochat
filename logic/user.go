package logic

import (
	"gochat/clog"
	"gochat/tools"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// dbIns 是数据库实例的全局变量，用于执行所有数据库操作
var dbIns = tools.GetDB("gochat")

// 生成密码哈希值
func hashPassword(password string) (string, error) {
	// 使用默认成本因子(10)生成哈希值
	// 成本因子越高，计算哈希值所需时间越长，防暴力破解效果越好
	// 可以根据服务器性能适当调整(8-14之间)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.Wrap(err, "密码哈希生成失败")
	}
	return string(hashedBytes), nil
}

// 验证密码是否匹配
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Add 添加新用户
func Add(userName, password string) error {
	// 对密码进行哈希处理
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return errors.Wrap(err, "密码哈希失败")
	}

	user := tools.User{
		UserName:   userName,
		Password:   hashedPassword,
		CreateTime: time.Now(),
	}
	if err := dbIns.Table(user.TableName()).Create(&user).Error; err != nil {
		return errors.Wrap(err, "failed to create user")
	}
	clog.Info("add user success, user_name: %s", userName)
	return nil
}

// CheckHaveUserName 检查用户名是否已存在，存在返回用户信息
func CheckHaveUserName(userName string) (tools.User, error) {
	var user tools.User
	if err := dbIns.Table(user.TableName()).Where("user_name = ?", userName).First(&user).Error; err != nil {
		return tools.User{}, errors.Wrap(err, "failed to query user")
	}
	clog.Info("check user name success, user_name: %s", userName)
	return user, nil
}

// GetUserNameByUserID 根据用户ID获取用户名
func GetUserNameByUserID(userID uint) (string, error) {
	var user tools.User
	if err := dbIns.Table(user.TableName()).Where("id = ?", userID).First(&user).Error; err != nil {
		return "", errors.Wrap(err, "failed to query user")
	}
	clog.Info("get user name success, user_id: %d, user_name: %v", userID, user.UserName)
	return user.UserName, nil
}
