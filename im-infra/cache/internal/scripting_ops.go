package internal

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

// scriptingOperations 实现了 ScriptingOperations 接口
type scriptingOperations struct {
	client *redis.Client
	logger clog.Logger
}

// newScriptingOperations 创建一个新的 scriptingOperations 实例
func newScriptingOperations(client *redis.Client, logger clog.Logger) *scriptingOperations {
	return &scriptingOperations{
		client: client,
		logger: logger,
	}
}

// EvalSha 执行已加载的 Lua 脚本
func (s *scriptingOperations) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	result, err := s.client.EvalSha(ctx, sha1, keys, args...).Result()
	if err != nil {
		s.logger.Error("执行 Lua 脚本失败",
			clog.String("sha1", sha1),
			clog.Any("keys", keys),
			clog.Err(err))
		return nil, fmt.Errorf("failed to eval script: %w", err)
	}
	return result, nil
}

// ScriptLoad 将 Lua 脚本加载到 Redis 中并返回其 SHA1 哈希值
func (s *scriptingOperations) ScriptLoad(ctx context.Context, script string) (string, error) {
	sha1, err := s.client.ScriptLoad(ctx, script).Result()
	if err != nil {
		s.logger.Error("加载 Lua 脚本失败", clog.Err(err))
		return "", fmt.Errorf("failed to load script: %w", err)
	}
	return sha1, nil
}

// ScriptExists 检查指定的 SHA1 哈希值对应的脚本是否已加载
func (s *scriptingOperations) ScriptExists(ctx context.Context, sha1 ...string) ([]bool, error) {
	result, err := s.client.ScriptExists(ctx, sha1...).Result()
	if err != nil {
		s.logger.Error("检查脚本存在性失败", clog.Any("sha1", sha1), clog.Err(err))
		return nil, fmt.Errorf("failed to check script existence: %w", err)
	}
	return result, nil
}
