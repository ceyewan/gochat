package internal

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// clogLogger 是集成 clog 的 GORM 日志器实现
type clogLogger struct {
	logger        clog.Logger
	logLevel      logger.LogLevel
	slowThreshold time.Duration
}

// NewClogLogger 创建一个新的 clog 集成日志器
func NewClogLogger(clogInstance clog.Logger, config Config) logger.Interface {
	var logLevel logger.LogLevel

	switch config.LogLevel {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Warn
	}

	return &clogLogger{
		logger:        clogInstance,
		logLevel:      logLevel,
		slowThreshold: config.SlowThreshold,
	}
}

// LogMode 设置日志级别
func (l *clogLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info 记录信息级别日志
func (l *clogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		clog.WithContext(ctx).Info(fmt.Sprintf(msg, data...))
	}
}

// Warn 记录警告级别日志
func (l *clogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		clog.WithContext(ctx).Warn(fmt.Sprintf(msg, data...))
	}
}

// Error 记录错误级别日志
func (l *clogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		clog.WithContext(ctx).Error(fmt.Sprintf(msg, data...))
	}
}

// Trace 记录 SQL 执行日志
func (l *clogLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)

	// 更强的防护性检查
	if fc == nil {
		clog.WithContext(ctx).Error("SQL执行函数为nil - 可能是分片库兼容性问题")
		return
	}

	// 使用 defer 和 recover 来防止 panic
	var sql string
	var rows int64
	func() {
		defer func() {
			if r := recover(); r != nil {
				clog.WithContext(ctx).Error("SQL执行函数调用时发生panic",
					clog.String("panic", fmt.Sprintf("%v", r)),
					clog.String("stack", string(debug.Stack())))
				sql = "PANIC_IN_SQL_EXECUTION"
				rows = 0
			}
		}()
		sql, rows = fc()
	}()

	// 获取调用位置信息
	fileWithLineNum := utils.FileWithLineNum()

	fields := []clog.Field{
		clog.Duration("elapsed", elapsed),
		clog.String("sql", sql),
		clog.Int64("rows", rows),
		clog.String("source", fileWithLineNum),
	}

	switch {
	case err != nil && l.logLevel >= logger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.isIgnoreRecordNotFoundError()):
		// 记录错误
		clog.WithContext(ctx).Error("SQL 执行错误",
			append(fields, clog.Err(err))...,
		)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.logLevel >= logger.Warn:
		// 记录慢查询
		clog.WithContext(ctx).Warn("检测到慢查询",
			append(fields,
				clog.Duration("threshold", l.slowThreshold),
				clog.String("level", "slow"),
			)...,
		)
	case l.logLevel == logger.Info:
		// 记录普通查询
		clog.WithContext(ctx).Info("SQL 执行",
			fields...,
		)
	}
}

// isIgnoreRecordNotFoundError 检查是否忽略记录未找到错误
func (l *clogLogger) isIgnoreRecordNotFoundError() bool {
	// 默认忽略记录未找到错误
	return true
}

// QueryLogger 查询日志记录器
type QueryLogger struct {
	logger clog.Logger
}

// NewQueryLogger 创建查询日志记录器
func NewQueryLogger(logger clog.Logger) *QueryLogger {
	return &QueryLogger{
		logger: logger,
	}
}

// LogQuery 记录查询操作
func (q *QueryLogger) LogQuery(ctx context.Context, operation string, table string, duration time.Duration, err error) {
	fields := []clog.Field{
		clog.String("operation", operation),
		clog.String("table", table),
		clog.Duration("duration", duration),
	}

	if err != nil {
		clog.WithContext(ctx).Error("数据库操作失败",
			append(fields, clog.Err(err))...,
		)
	} else {
		clog.WithContext(ctx).Debug("数据库操作成功",
			fields...,
		)
	}
}

// LogTransaction 记录事务操作
func (q *QueryLogger) LogTransaction(ctx context.Context, operation string, duration time.Duration, err error) {
	fields := []clog.Field{
		clog.String("operation", operation),
		clog.Duration("duration", duration),
	}

	if err != nil {
		clog.WithContext(ctx).Error("事务操作失败",
			append(fields, clog.Err(err))...,
		)
	} else {
		clog.WithContext(ctx).Info("事务操作成功",
			fields...,
		)
	}
}

// LogConnection 记录连接操作
func (q *QueryLogger) LogConnection(ctx context.Context, event string, details map[string]interface{}) {
	fields := []clog.Field{
		clog.String("event", event),
	}

	for key, value := range details {
		switch v := value.(type) {
		case string:
			fields = append(fields, clog.String(key, v))
		case int:
			fields = append(fields, clog.Int(key, v))
		case int64:
			fields = append(fields, clog.Int64(key, v))
		case time.Duration:
			fields = append(fields, clog.Duration(key, v))
		case bool:
			fields = append(fields, clog.Bool(key, v))
		default:
			fields = append(fields, clog.String(key, fmt.Sprintf("%v", v)))
		}
	}

	clog.WithContext(ctx).Info("数据库连接事件",
		fields...,
	)
}
