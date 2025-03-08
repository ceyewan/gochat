package clog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// 日志级别
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
)

// ============================
// 公开 API 接口
// ============================

// Debug 记录 Debug 级别日志
func Debug(format string, args ...interface{}) {
	defaultLogger.log(LevelDebug, format, args...)
}

// Info 记录 Info 级别日志
func Info(format string, args ...interface{}) {
	defaultLogger.log(LevelInfo, format, args...)
}

// Warning 记录 Warning 级别日志
func Warning(format string, args ...interface{}) {
	defaultLogger.log(LevelWarning, format, args...)
}

// Error 记录 Error 级别日志
func Error(format string, args ...interface{}) {
	defaultLogger.log(LevelError, format, args...)
}

// SetLogPath 设置日志输出路径
// 如果路径为空，则只输出到控制台
// 否则输出内容到指定文件
func SetLogPath(logDir string) error {
	if logDir == "" {
		defaultLogger.setTarget(LogToConsole)
		return nil
	}

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	defaultLogger.mu.Lock()
	defaultLogger.logDir = logDir
	defaultLogger.mu.Unlock()
	defaultLogger.setTarget(LogToFile)

	return nil
}

// SetLogLevel 设置日志级别
// 只输出大于等于该级别的日志
func SetLogLevel(level LogLevel) {
	defaultLogger.setLevel(level)
}

// Close 关闭日志系统，释放资源
func Close() error {
	return defaultLogger.close()
}

// ============================
// 内部实现
// ============================

// 日志输出目标
type logTarget int

const (
	LogToConsole logTarget = 1 << iota
	LogToFile
	LogToBoth = LogToConsole | LogToFile
)

// 日志级别对应的颜色
var levelColors = map[LogLevel]string{
	LevelDebug:   "\033[36m", // 青色
	LevelInfo:    "\033[32m", // 绿色
	LevelWarning: "\033[33m", // 黄色
	LevelError:   "\033[31m", // 红色
}

// 日志级别名称，使用固定宽度
var levelNames = map[LogLevel]string{
	LevelDebug:   "DEBUG  ",
	LevelInfo:    "INFO   ",
	LevelWarning: "WARNING",
	LevelError:   "ERROR  ",
}

// Logger 结构体
type logger struct {
	consoleLogger *log.Logger
	fileLogger    *log.Logger
	level         LogLevel
	target        logTarget
	mu            sync.Mutex
	fileWriter    io.WriteCloser

	// 日志按日期分割所需的字段
	logDir      string // 日志目录
	currentDate string // 当前日志对应的日期
}

// 全局默认日志实例
var defaultLogger *logger

func init() {
	// 初始化默认日志实例，默认输出到控制台
	defaultLogger = newLogger("", LevelInfo, LogToConsole)
}

// newLogger 创建一个新的 Logger 实例
func newLogger(logDir string, level LogLevel, target logTarget) *logger {
	logger := &logger{
		level:  level,
		target: target,
		logDir: logDir,
	}

	if target&LogToConsole != 0 {
		logger.consoleLogger = log.New(os.Stdout, "", 0)
	}

	if target&LogToFile != 0 && logDir != "" {
		// 确保日志目录存在
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("Failed to create log directory: %v", err)
			// 回退到只使用控制台
			logger.target = LogToConsole
		} else {
			// 初始化日志文件
			if err := logger.rotateLogFile(); err != nil {
				log.Printf("Failed to initialize log file: %v", err)
				// 回退到只使用控制台
				logger.target = LogToConsole
			}
		}
	}

	return logger
}

func (l *logger) rotateLogFile() error {
	// 如果未设置日志目录，则不输出到文件
	if l.logDir == "" {
		return nil
	}

	// 获取当前日期
	today := time.Now().Format("2006-01-02")
	// 如果日期没变且已有文件句柄，则不需要切换
	if today == l.currentDate && l.fileWriter != nil {
		return nil
	}
	// 关闭之前的文件
	if l.fileWriter != nil {
		l.fileWriter.Close()
		l.fileWriter = nil
		l.fileLogger = nil
	}
	// 生成当天的日志文件路径
	filename := fmt.Sprintf("log-%s.log", today)
	logPath := filepath.Join(l.logDir, filename)
	// 打开或创建日志文件
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %v", logPath, err)
	}
	// 更新Logger状态
	l.fileLogger = log.New(file, "", 0)
	l.fileWriter = file
	l.currentDate = today
	return nil
}

// close 关闭日志文件
func (l *logger) close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fileWriter != nil {
		return l.fileWriter.Close()
	}
	return nil
}

// setTarget 设置日志输出目标
func (l *logger) setTarget(target logTarget) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.target = target
}

// setLevel 设置日志级别
func (l *logger) setLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log 内部日志方法
func (l *logger) log(level LogLevel, format string, args ...interface{}) {
	// 快速检查日志级别
	if level < l.level {
		return
	}

	// 获取调用者的文件名和行号
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	} else {
		// 只保留文件名
		file = file[strings.LastIndex(file, "/")+1:]
	}

	// 对齐长度，方便查看
	fileInfo := fmt.Sprintf("%s:%d", file, line)
	if len(fileInfo) < 15 {
		fileInfo = fmt.Sprintf("%-15s", fileInfo)
	}

	// 格式化日志内容
	message := fmt.Sprintf(format, args...)
	timestamp := timeNow()

	l.mu.Lock()
	defer l.mu.Unlock()

	// 同步处理控制台输出
	if l.target&LogToConsole != 0 && l.consoleLogger != nil {
		// 控制台日志（带颜色）
		consoleLogEntry := fmt.Sprintf(
			"%s[%s]\033[0m \033[90m%s %s\033[0m %s",
			levelColors[level],
			levelNames[level],
			timestamp,
			fileInfo,
			message,
		)
		l.consoleLogger.Println(consoleLogEntry)
	}

	// 同步处理文件输出
	if l.target&LogToFile != 0 {
		// 检查是否需要切换日志文件
		if err := l.rotateLogFile(); err != nil {
			// 如果切换失败，记录错误到控制台
			if l.consoleLogger != nil {
				l.consoleLogger.Printf("切换日志文件失败: %v", err)
			}
		} else if l.fileLogger != nil {
			// 文件日志（不带颜色）
			fileLogEntry := fmt.Sprintf(
				"[%s] %s %s %s",
				levelNames[level],
				timestamp,
				fileInfo,
				message,
			)
			// 写入日志到文件
			l.fileLogger.Println(fileLogEntry)
		}
	}
}

// timeNow 返回当前时间的格式化字符串
func timeNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
