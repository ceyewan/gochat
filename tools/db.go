package tools

import (
	"fmt"
	"sync"
	"time"

	"gochat/clog"
	"gochat/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// User 定义用户模型结构体，对应数据库中的 user 表
type User struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`           // 用户ID，主键，自增
	UserName   string    `gorm:"size:20;not null;unique"`            // 用户名，不能为空，唯一
	Password   string    `gorm:"type:char(40);not null"`             // 密码，不能为空
	CreateTime time.Time `gorm:"not null;default:current_timestamp"` // 创建时间，默认为当前时间戳
}

// TableName 指定用户模型对应的数据库表名
func (User) TableName() string {
	return "user"
}

// dbMap 存储数据库连接的映射，键为数据库名称，值为对应的数据库连接
var dbMap = map[string]*gorm.DB{}

// syncLock 用于在操作 dbMap 时提供线程安全
var syncLock sync.Mutex

// init 在包被导入时自动执行，负责初始化数据库连接。
// 该函数调用 initDB 函数，并使用 "gochat" 作为数据库名称参数。
func init() {
	initDB("gochat")
}

// initDB 初始化指定名称的数据库连接，并执行自动迁移
// 参数 dbName: 数据库名称
func initDB(dbName string) {
	var err error

	// 获取数据库配置
	dbConfig := config.Conf.MySQL

	// 构建 DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbConfig.Username,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbName,
	)

	syncLock.Lock()
	defer syncLock.Unlock()

	// 配置 GORM 日志
	gormLogLevel := logger.Silent
	if config.GetMode() == "dev" {
		gormLogLevel = logger.Info
	}

	// 使用 GORM 打开数据库连接
	dbMap[dbName], err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})

	if err != nil {
		clog.Error("open database error: %v", err)
		return
	}

	// 获取通用数据库对象，配置连接池
	sqlDB, err := dbMap[dbName].DB()
	if err != nil {
		clog.Error("get database object error: %v", err)
		return
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(10)                  // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)                 // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大存活时间
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // 连接最大空闲时间

	// 自动迁移用户模型，创建或更新表结构
	err = dbMap[dbName].AutoMigrate(&User{})
	if err != nil {
		clog.Error("Auto migrate table failed: %v", err)
		return
	}
	clog.Info("Database initialized successfully with auto migration")
}

// GetDB 获取指定名称的数据库连接
// 参数 dbName: 数据库名称
// 返回: 对应的数据库连接，如果不存在则返回nil
func GetDB(dbName string) *gorm.DB {
	if db, ok := dbMap[dbName]; ok {
		return db
	}
	return nil
}

// DBGoChat 提供访问gochat数据库的结构体
type DBGoChat struct{}

// GetDBName 返回gochat数据库的名称
// 返回: 数据库名称字符串
func (*DBGoChat) GetDBName() string {
	return "gochat"
}
