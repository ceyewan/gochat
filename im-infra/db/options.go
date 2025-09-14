package db

import (
	"github.com/ceyewan/gochat/im-infra/clog"
)

// Option 定义了用于定制 db Provider 的函数。
type Option func(*provider)

// provider 用于保存选项状态
type provider struct {
	logger       clog.Logger
	componentName string
}

// WithLogger 将一个 clog.Logger 实例注入 GORM，用于结构化记录 SQL 日志。
// 这是与 clog 组件联动的推荐做法。
func WithLogger(logger clog.Logger) Option {
	return func(p *provider) {
		p.logger = logger
	}
}

// WithComponentName 设置组件名称（向后兼容）
func WithComponentName(name string) Option {
	return func(p *provider) {
		p.componentName = name
	}
}
