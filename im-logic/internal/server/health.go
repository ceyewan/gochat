package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	logger     clog.Logger
	components map[string]HealthCheckable
	mu         sync.RWMutex
}

// HealthCheckable 健康检查接口
type HealthCheckable interface {
	// IsHealthy 检查组件是否健康
	IsHealthy() bool

	// GetName 获取组件名称
	GetName() string
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		logger:     clog.Namespace("health-checker"),
		components: make(map[string]HealthCheckable),
	}
}

// RegisterComponent 注册健康检查组件
func (h *HealthChecker) RegisterComponent(name string, component HealthCheckable) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.components[name] = component
	h.logger.Info("注册健康检查组件", clog.String("name", name))
}

// UnregisterComponent 注销健康检查组件
func (h *HealthChecker) UnregisterComponent(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.components, name)
	h.logger.Info("注销健康检查组件", clog.String("name", name))
}

// IsHealthy 检查所有组件是否健康
func (h *HealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for name, component := range h.components {
		if !component.IsHealthy() {
			h.logger.Warn("组件不健康", clog.String("name", name))
			return false
		}
	}

	return true
}

// CheckHealth 检查健康状态
func (h *HealthChecker) CheckHealth(ctx context.Context) error {
	if !h.IsHealthy() {
		return fmt.Errorf("部分组件不健康")
	}
	return nil
}

// GetComponentStatus 获取组件状态
func (h *HealthChecker) GetComponentStatus(name string) (bool, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	component, exists := h.components[name]
	if !exists {
		return false, false
	}

	return component.IsHealthy(), true
}

// GetAllStatuses 获取所有组件状态
func (h *HealthChecker) GetAllStatuses() map[string]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	statuses := make(map[string]bool)
	for name, component := range h.components {
		statuses[name] = component.IsHealthy()
	}

	return statuses
}

// GetUnhealthyComponents 获取不健康的组件
func (h *HealthChecker) GetUnhealthyComponents() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var unhealthy []string
	for name, component := range h.components {
		if !component.IsHealthy() {
			unhealthy = append(unhealthy, name)
		}
	}

	return unhealthy
}

// StartPeriodicCheck 启动定期健康检查
func (h *HealthChecker) StartPeriodicCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.logger.Debug("执行定期健康检查...")
			if h.IsHealthy() {
				h.logger.Debug("所有组件健康")
			} else {
				h.logger.Warn("发现不健康的组件")
			}
		}
	}
}
