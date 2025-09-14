package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
	"gorm.io/gorm"
)

// V1 模型 - 初始版本
type UserV1 struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique;size:100;not null"`
	Email    string `gorm:"size:100"`
}

// TableName 指定 V1 模型的表名
func (UserV1) TableName() string {
	return "users_v1"
}

// V2 模型 - 添加字段
type UserV2 struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"unique;size:100;not null"`
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
	Username  string `gorm:"unique;size:100;not null"`
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
	UserID uint   `gorm:"index:idx_profile_user_id"`
	Bio    string `gorm:"type:text"`
	Avatar string `gorm:"size:500"`
	User   UserV3 `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName 指定 Profile 表名
func (ProfileV3) TableName() string {
	return "profiles_v3"
}

func main() {
	ctx := context.Background()

	// 创建自定义日志器
	logger := clog.Namespace("db-migration-example")

	// 创建 MySQL 配置
	cfg := db.MySQLConfig("gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local")

	// 使用 New 函数创建数据库实例，并注入 Logger
	database, err := db.New(ctx, cfg, db.WithLogger(logger), db.WithComponentName("migration-example"))
	if err != nil {
		log.Fatalf("创建数据库实例失败: %v", err)
	}
	defer database.Close()

	// 获取 GORM DB 实例用于调试
	var gormDB *gorm.DB
	gormDB = database.GetDB()

	logger.Info("开始数据库迁移示例")

	// === 清理遗留表（修复方案） ===
	logger.Info("=== 清理遗留表以修复迁移问题 ===")

	// 删除可能存在的遗留表
	tablesToDrop := []string{"profiles_v3", "users_v3", "users_v2", "users_v1"}
	for _, table := range tablesToDrop {
		if err := gormDB.WithContext(ctx).Exec("DROP TABLE IF EXISTS " + table).Error; err != nil {
			logger.Error("删除表失败", clog.String("table", table), clog.Err(err))
		} else {
			logger.Info("成功删除表", clog.String("table", table))
		}
	}

	logger.Info("遗留表清理完成")

	// === 调试：检查数据库初始状态 ===
	logger.Info("=== 调试：检查数据库初始状态 ===")

	// 检查是否已有相关表存在
	var existingTables []string
	if err := gormDB.WithContext(ctx).Raw("SHOW TABLES").Scan(&existingTables).Error; err != nil {
		logger.Error("查询现有表失败", clog.Err(err))
	} else {
		logger.Info("数据库现有表", clog.Int("tableCount", len(existingTables)))
		for _, table := range existingTables {
			logger.Info("现有表", clog.String("tableName", table))
		}
	}

	// 特别检查users_v1表
	var v1TableName string
	if err := gormDB.WithContext(ctx).Raw("SHOW TABLES LIKE 'users_v1'").Scan(&v1TableName).Error; err != nil {
		logger.Error("检查users_v1表存在性失败", clog.Err(err))
	} else if v1TableName != "" {
		logger.Info("users_v1表已存在，检查其约束")

		// 检查users_v1表的约束
		type V1ConstraintInfo struct {
			ConstraintName string `gorm:"column:CONSTRAINT_NAME"`
			TableName      string `gorm:"column:TABLE_NAME"`
			ColumnName     string `gorm:"column:COLUMN_NAME"`
			ConstraintType string `gorm:"column:CONSTRAINT_TYPE"`
		}

		var v1Constraints []V1ConstraintInfo
		if err := gormDB.WithContext(ctx).Raw(`
			SELECT CONSTRAINT_NAME, TABLE_NAME, COLUMN_NAME, CONSTRAINT_TYPE
			FROM information_schema.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users_v1'
		`).Scan(&v1Constraints).Error; err != nil {
			logger.Error("查询users_v1表约束失败", clog.Err(err))
		} else {
			logger.Info("users_v1表约束信息", clog.Int("constraintCount", len(v1Constraints)))
			for _, constraint := range v1Constraints {
				logger.Info("V1约束详情",
					clog.String("constraintName", constraint.ConstraintName),
					clog.String("tableName", constraint.TableName),
					clog.String("columnName", constraint.ColumnName),
					clog.String("constraintType", constraint.ConstraintType))
			}
		}
	} else {
		logger.Info("users_v1表不存在")
	}

	// === 第一次迁移：V1 模型 ===
	logger.Info("执行 V1 迁移：创建基础用户表")
	if err := database.AutoMigrate(&UserV1{}); err != nil {
		log.Fatalf("V1 迁移失败: %v", err)
	}

	// 插入一些 V1 数据
	gormDB = database.GetDB()
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

	// === 调试：检查 users_v3 表的约束 ===
	logger.Info("=== 调试：检查 users_v3 表结构 ===")

	// 检查表是否存在
	var tableName string
	if err := gormDB.WithContext(ctx).Raw("SHOW TABLES LIKE 'users_v3'").Scan(&tableName).Error; err != nil {
		logger.Error("检查表存在性失败", clog.Err(err))
	} else if tableName != "" {
		logger.Info("users_v3 表存在")
	} else {
		logger.Info("users_v3 表不存在")
	}

	// 检查表的约束
	type ConstraintInfo struct {
		ConstraintName      string `gorm:"column:CONSTRAINT_NAME"`
		TableName           string `gorm:"column:TABLE_NAME"`
		ColumnName          string `gorm:"column:COLUMN_NAME"`
		ReferencedTableName string `gorm:"column:REFERENCED_TABLE_NAME"`
	}

	var constraints []ConstraintInfo
	if err := gormDB.WithContext(ctx).Raw(`
		SELECT CONSTRAINT_NAME, TABLE_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users_v3'
	`).Scan(&constraints).Error; err != nil {
		logger.Error("查询表约束失败", clog.Err(err))
	} else {
		logger.Info("users_v3 表约束信息", clog.Int("constraintCount", len(constraints)))
		for _, constraint := range constraints {
			logger.Info("约束详情",
				clog.String("constraintName", constraint.ConstraintName),
				clog.String("tableName", constraint.TableName),
				clog.String("columnName", constraint.ColumnName),
				clog.String("referencedTable", constraint.ReferencedTableName))
		}
	}

	// 检查外键约束
	var foreignKeys []ConstraintInfo
	if err := gormDB.WithContext(ctx).Raw(`
		SELECT CONSTRAINT_NAME, TABLE_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users_v3' AND REFERENCED_TABLE_NAME IS NOT NULL
	`).Scan(&foreignKeys).Error; err != nil {
		logger.Error("查询外键约束失败", clog.Err(err))
	} else {
		logger.Info("users_v3 表外键约束信息", clog.Int("foreignKeyCount", len(foreignKeys)))
		for _, fk := range foreignKeys {
			logger.Info("外键详情",
				clog.String("constraintName", fk.ConstraintName),
				clog.String("referencedTable", fk.ReferencedTableName))
		}
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
	var profilesWithUsers []ProfileV3
	if err := gormDB.WithContext(ctx).Preload("User").Find(&profilesWithUsers).Error; err != nil {
		logger.Error("查询档案及用户失败", clog.Err(err))
	} else {
		logger.Info("成功查询到档案及用户", clog.Int("count", len(profilesWithUsers)))
		for _, profile := range profilesWithUsers {
			logger.Info("档案及用户信息",
				clog.Uint("profileID", profile.ID),
				clog.Uint("userID", profile.UserID),
				clog.String("username", profile.User.Username),
				clog.String("email", profile.User.Email),
				clog.String("phone", profile.User.Phone),
				clog.String("bio", profile.Bio))
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
