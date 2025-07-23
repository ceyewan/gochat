package slogx

import (
	"fmt"
	"io"
	"os"
)

// NewWriter 根据指定类型创建 io.Writer。
// 支持的 writerType：
//   - "stdout"：写入 os.Stdout
//   - "stderr"：写入 os.Stderr
func NewWriter(writerType string) (io.Writer, error) {
	switch writerType {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		return nil, fmt.Errorf("unsupported writer type: %s, supported types: stdout, stderr", writerType)
	}
}
