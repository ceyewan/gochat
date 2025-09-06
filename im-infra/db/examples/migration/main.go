package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
)

// V1 模型 - 初始版本
type UserV1 struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex:idx_userv1_username;size:100;not null"`
	Email    string `gorm:"size:100"`
}

// TableName 指定 V1 模型的表名
func (UserV1) TableName() string {
	return "users_v1"
}

// V2 模型 - 添加字段
type UserV2 struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"uniqueIndex:idx_userv2_username;size:100;not null"`
	Email     string `gorm:"size:100"`
	Age       int    `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 指定 V2 模型的表名
func (UserV2) TableName() string {
	return "users_v2"
}

// V3 模型 - 添加更多字段
type UserV3 struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"uniqueIndex:idx_userv3_username;size:100;not null"`
	Email     string `gorm:"size:100"`
	Phone     string `gorm:"size:20"`
	Age       int    `gorm:"default:0"`
	Status    int    `gorm:"default:1;comment:用户状态 1=active 0=inactive"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// TableName 指定 V3 模型的表名
func (UserV3) TableName() string {
	return "users_v3"
}

// Profile 关联表
type ProfileV3 struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"not null;index:idx_profile_user_id"`
	Bio    string `gorm:"type:text"`
	Avatar string `gorm:"size:500"`
	User   UserV3 `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

// TableName 指定 Profile 表名
func (ProfileV3) TableName() string {
	return "profiles_v3"
}

func main() {
	ctx := context.Background()

	// 创建自定义日志器
	logger := clog.Module("db-migration-example")

	// 创建 MySQL 配置
	cfg := db.MySQLConfig("gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local")

	// 使用 New 函数创建数据库实例，并注入 Logger
	database, err := db.New(ctx, cfg, db.WithLogger(logger), db.WithComponentName("migration-example"))
	if err != nil {
		log.Fatalf("创建数据库实例失败: %v", err)
	}
	defer database.Close()

	logger.Info("开始数据库迁移示例")

	// === 第一次迁移：V1 模型 ===
	logger.Info("执行 V1 迁移：创建基础用户表")
	if err := database.AutoMigrate(&UserV1{}); err != nil {
		log.Fatalf("V1 迁移失败: %v", err)
	}

	// 插入一些 V1 数据
	gormDB := database.GetDB()
	users := []UserV1{
		{Username: "alice", Email: "alice@example.com"},
		{Username: "bob", Email: "bob@example.com"},
		{Username: "charlie", Email: "charlie@example.com"},
	}

	for _, user := range users {
		if err := gormDB.WithContext(ctx).Create(&user).Error; err != nil {
			logger.Error("创建 V1 用户失败", clog.Err(err))
		} else {
			logger.Info("创建 V1 用户成功", clog.String("username", user.Username))
		}
	}

	// === 第二次迁移：V2 模型 ===
	logger.Info("执行 V2 迁移：创建独立的 V2 表并迁移数据")
	if err := database.AutoMigrate(&UserV2{}); err != nil {
		log.Fatalf("V2 迁移失败: %v", err)
	}

	// 从 V1 表迁移数据到 V2 表
	var v1Users []UserV1
	if err := gormDB.WithContext(ctx).Find(&v1Users).Error; err != nil {
		logger.Error("查询 V1 用户失败", clog.Err(err))
	} else {
		for i, v1User := range v1Users {
			v2User := UserV2{
				ID:       v1User.ID,
				Username: v1User.Username,
				Email:    v1User.Email,
				Age:      20 + i*5, // 设置默认年龄
			}
			if err := gormDB.WithContext(ctx).Create(&v2User).Error; err != nil {
				logger.Error("迁移到 V2 用户失败", clog.Err(err))
			} else {
				logger.Info("V2 用户迁移成功",
					clog.String("username", v2User.Username),
					clog.Int("age", v2User.Age))
			}
		}
	}

	// === 第三次迁移：V3 模型 ===
	logger.Info("执行 V3 迁移：创建独立的 V3 表并迁移数据")
	if err := database.AutoMigrate(&UserV3{}); err != nil {
		log.Fatalf("V3 迁移失败: %v", err)
	}

	// 从 V2 表迁移数据到 V3 表
	var v2Users []UserV2
	if err := gormDB.WithContext(ctx).Find(&v2Users).Error; err != nil {
		logger.Error("查询 V2 用户失败", clog.Err(err))
	} else {
		for i, v2User := range v2Users {
			v3User := UserV3{
				ID:        v2User.ID,
				Username:  v2User.Username,
				Email:     v2User.Email,
				Age:       v2User.Age,
				Phone:     "1380000000" + string(rune('0'+i)),
				Status:    1,
				CreatedAt: v2User.CreatedAt,
				UpdatedAt: v2User.UpdatedAt,
			}
			if err := gormDB.WithContext(ctx).Create(&v3User).Error; err != nil {
				logger.Error("迁移到 V3 用户失败", clog.Err(err))
			} else {
				logger.Info("V3 用户迁移成功",
					clog.String("username", v3User.Username),
					clog.String("phone", v3User.Phone),
					clog.Int("status", v3User.Status))
			}
		}
	}

	// 查询迁移后的 V3 用户数据
	var v3Users []UserV3
	if err := gormDB.WithContext(ctx).Find(&v3Users).Error; err != nil {
		logger.Error("查询 V3 用户失败", clog.Err(err))
	}

	// === 第四次迁移：添加关联表 ===
	logger.Info("执行关联表迁移：创建 Profile 表")
	if err := database.AutoMigrate(&ProfileV3{}); err != nil {
		log.Fatalf("Profile 表迁移失败: %v", err)
	}

	// 为用户创建 Profile
	for _, user := range v3Users {
		profile := ProfileV3{
			UserID: user.ID,
			Bio:    "这是 " + user.Username + " 的个人简介",
			Avatar: "https://example.com/avatar/" + user.Username + ".jpg",
		}
		if err := gormDB.WithContext(ctx).Create(&profile).Error; err != nil {
			logger.Error("创建用户档案失败", clog.Err(err))
		} else {
			logger.Info("创建用户档案成功", clog.String("username", user.Username))
		}
	}

	// === 验证迁移结果 ===
	logger.Info("验证迁移结果")

	// 查询各个版本的用户数据
	logger.Info("=== 查询各版本数据 ===")

	// 查询 V1 用户
	var v1UsersCount int64
	gormDB.WithContext(ctx).Model(&UserV1{}).Count(&v1UsersCount)
	logger.Info("V1 用户统计", clog.Int64("count", v1UsersCount))

	// 查询 V2 用户
	var v2UsersCount int64
	gormDB.WithContext(ctx).Model(&UserV2{}).Count(&v2UsersCount)
	logger.Info("V2 用户统计", clog.Int64("count", v2UsersCount))

	// 查询 V3 用户
	var v3UsersCount int64
	gormDB.WithContext(ctx).Model(&UserV3{}).Count(&v3UsersCount)
	logger.Info("V3 用户统计", clog.Int64("count", v3UsersCount))

	// 查询带关联的用户数据
	var usersWithProfiles []UserV3
	if err := gormDB.WithContext(ctx).Preload("User").Table("profiles_v3").Find(&usersWithProfiles).Error; err != nil {
		logger.Error("查询用户及档案失败", clog.Err(err))
	} else {
		logger.Info("成功查询到用户及档案", clog.Int("count", len(usersWithProfiles)))
		for _, user := range v3Users {
			logger.Info("V3用户详细信息",
				clog.Uint("id", user.ID),
				clog.String("username", user.Username),
				clog.String("email", user.Email),
				clog.String("phone", user.Phone),
				clog.Int("age", user.Age),
				clog.Int("status", user.Status))
		}
	}

	// 查询档案数据
	var profiles []ProfileV3
	if err := gormDB.WithContext(ctx).Preload("User").Find(&profiles).Error; err != nil {
		logger.Error("查询档案失败", clog.Err(err))
	} else {
		logger.Info("成功查询到档案", clog.Int("count", len(profiles)))
		for _, profile := range profiles {
			logger.Info("档案信息",
				clog.Uint("profileID", profile.ID),
				clog.Uint("userID", profile.UserID),
				clog.String("username", profile.User.Username),
				clog.String("bio", profile.Bio))
		}
	}

	logger.Info("数据库迁移示例完成")
}
