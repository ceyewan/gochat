package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// HealthChecker 健康检查器接口
type HealthChecker interface {
	// CheckHealth 检查服务健康状态
	CheckHealth(ctx context.Context) error
}

// healthChecker 健康检查器实现
type healthChecker struct {
	server *server
	logger clog.Logger
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(srv *server) HealthChecker {
	return &healthChecker{
		server: srv,
		logger: clog.Namespace("health-checker"),
	}
}

// CheckHealth 检查服务健康状态
func (h *healthChecker) CheckHealth(ctx context.Context) error {
	h.logger.Debug("开始健康检查")

	// 检查数据库连接
	if err := h.checkDatabase(ctx); err != nil {
		h.logger.Error("数据库健康检查失败", clog.Err(err))
		return err
	}

	// 检查缓存连接
	if err := h.checkCache(ctx); err != nil {
		h.logger.Error("缓存健康检查失败", clog.Err(err))
		return err
	}

	h.logger.Debug("健康检查通过")
	return nil
}

// checkDatabase 检查数据库连接
func (h *healthChecker) checkDatabase(ctx context.Context) error {
	if h.server.database == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 设置超时上下文
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 执行简单的数据库查询
	db := h.server.database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库实例为空")
	}

	var result int
	err := db.WithContext(checkCtx).Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		return fmt.Errorf("数据库查询失败: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("数据库查询结果异常: %d", result)
	}

	return nil
}

// checkCache 检查缓存连接
func (h *healthChecker) checkCache(ctx context.Context) error {
	if h.server.cache == nil {
		return fmt.Errorf("缓存连接未初始化")
	}

	// 设置超时上下文
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 执行 Ping 操作
	err := h.server.cache.Ping(checkCtx)
	if err != nil {
		return fmt.Errorf("缓存 Ping 失败: %w", err)
	}

	return nil
}
