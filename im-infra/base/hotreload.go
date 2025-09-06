package base

import "context"

// HotReloadable 定义了支持配置热更新的组件必须实现的接口。
type HotReloadable interface {
	// HotReload 在检测到配置变更时被触发。
	// 它接收新的配置对象，并应以平滑、无中断的方式应用新配置。
	HotReload(ctx context.Context, newConfig any) error
}
