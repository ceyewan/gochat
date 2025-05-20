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

// User 数据库用户模型
type User struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`                         // 用户ID
	UserName   string    `gorm:"size:20;not null;unique"`                          // 用户名
	Password   string    `gorm:"type:char(128);not null"`                          // 密码(哈希值)
	CreateTime time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP"` // 创建时间
}

// TableName 指定表名
func (User) TableName() string {
	return "user"
}

var (
	// dbMap 数据库连接映射
	dbMap = map[string]*gorm.DB{}

	// dbLock 确保线程安全
	dbLock sync.Mutex

	// 确保数据库只初始化一次
	initOnce sync.Once
)

// initDB 初始化数据库连接和表结构
func initDB() {
	clog.Module("db").Debugf("Initializing database connection")

	// 获取配置
	dbConfig := config.Conf.MySQL
	dbName := dbConfig.DbName

	if dbName == "" {
		clog.Module("db").Errorf("Database name is empty")
		return
	}

	// 构建数据源名称
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		dbConfig.Username,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbName,
		dbConfig.Charset,
	)

	dbLock.Lock()
	defer dbLock.Unlock()

	// 设置日志级别
	gormLogLevel := logger.Silent
	if config.GetMode() == config.ModeDev {
		gormLogLevel = logger.Info
	}

	// 连接数据库
	var err error
	dbMap[dbName], err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})

	if err != nil {
		clog.Module("db").Errorf("Failed to connect to database: %v", err)
		return
	}

	// 配置连接池
	sqlDB, err := dbMap[dbName].DB()
	if err != nil {
		clog.Module("db").Errorf("Failed to get database instance: %v", err)
		return
	}

	sqlDB.SetMaxIdleConns(10)                  // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)                 // 最大连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大生命周期
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // 空闲连接超时时间

	// 自动迁移表结构
	if err = dbMap[dbName].AutoMigrate(&User{}); err != nil {
		clog.Module("db").Errorf("Failed to migrate database schema: %v", err)
		return
	}

	clog.Module("db").Infof("Database connection established and schema migrated successfully")
}

// GetDB 获取数据库连接
// 如果连接不存在，则初始化数据库
func GetDB() *gorm.DB {
	dbName := config.Conf.MySQL.DbName

	// 检查连接是否已存在
	dbLock.Lock()
	db, exists := dbMap[dbName]
	dbLock.Unlock()

	if exists {
		return db
	}

	// 惰性初始化
	initOnce.Do(initDB)

	dbLock.Lock()
	db = dbMap[dbName]
	dbLock.Unlock()

	if db == nil {
		clog.Module("db").Warnf("No database connection available for: %s", dbName)
	}

	return db
}

// CloseAllDBConnections 关闭所有数据库连接
func CloseAllDBConnections() {
	dbLock.Lock()
	defer dbLock.Unlock()

	for name, db := range dbMap {
		if sqlDB, err := db.DB(); err == nil {
			if err = sqlDB.Close(); err != nil {
				clog.Module("db").Errorf("Failed to close database connection %s: %v", name, err)
			} else {
				clog.Module("db").Infof("Database connection %s closed successfully", name)
			}
		}
	}

	// 清空连接映射
	dbMap = map[string]*gorm.DB{}
}
