package internal

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// snowflakeGenerator 雪花算法 ID 生成器的实现
type snowflakeGenerator struct {
	config  SnowflakeConfig
	node    *snowflake.Node
	logger  clog.Logger
	once    sync.Once
	initErr error
}

// NewSnowflakeGenerator 创建新的雪花算法 ID 生成器
func NewSnowflakeGenerator(config SnowflakeConfig) (SnowflakeGenerator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid snowflake configimpl: %w", err)
	}

	logger := clog.Module("idgen")
	logger.Info("创建雪花算法 ID 生成器",
		clog.Int64("node_id", config.NodeID),
		clog.Bool("auto_node_id", config.AutoNodeID),
		clog.Int64("epoch", config.Epoch),
	)

	generator := &snowflakeGenerator{
		config: config,
		logger: logger,
	}

	// 初始化雪花算法节点
	if err := generator.initNode(); err != nil {
		return nil, fmt.Errorf("failed to initialize snowflake node: %w", err)
	}

	logger.Info("雪花算法 ID 生成器创建成功")
	return generator, nil
}

// initNode 初始化雪花算法节点
func (g *snowflakeGenerator) initNode() error {
	var nodeID int64

	if g.config.AutoNodeID {
		// 自动生成节点 ID
		ip, err := getLocalIP()
		if err != nil {
			g.logger.Warn("获取本机 IP 失败，使用默认节点 ID", clog.Int64("default_node_id", 1), clog.Err(err))
			nodeID = 1
		} else {
			// 使用 IP 地址的最后一个字节作为节点 ID
			ipObj := net.ParseIP(ip)
			if ipObj == nil {
				g.logger.Warn("解析 IP 失败，使用默认节点 ID", clog.String("ip", ip), clog.Int64("default_node_id", 1))
				nodeID = 1
			} else {
				nodeID = int64(ipObj.To4()[3])
				g.logger.Debug("从 IP 地址生成节点 ID", clog.String("ip", ip), clog.Int64("node_id", nodeID))
			}
		}
	} else {
		nodeID = g.config.NodeID
		g.logger.Debug("使用配置的节点 ID", clog.Int64("node_id", nodeID))
	}

	// 设置自定义起始时间
	if g.config.Epoch > 0 {
		snowflake.Epoch = g.config.Epoch
		g.logger.Debug("设置自定义起始时间", clog.Int64("epoch", g.config.Epoch))
	}

	// 创建雪花算法节点
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		g.logger.Error("创建雪花算法节点失败", clog.Int64("node_id", nodeID), clog.Err(err))
		return fmt.Errorf("failed to create snowflake node with id %d: %w", nodeID, err)
	}

	g.node = node
	g.logger.Info("雪花算法节点初始化成功", clog.Int64("node_id", nodeID))
	return nil
}

// GenerateString 生成字符串类型的 ID
func (g *snowflakeGenerator) GenerateString(ctx context.Context) (string, error) {
	id, err := g.GenerateInt64(ctx)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

// GenerateInt64 生成 int64 类型的 ID
func (g *snowflakeGenerator) GenerateInt64(ctx context.Context) (int64, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		g.logger.Debug("生成雪花 ID",
			clog.Duration("duration", duration),
		)
	}()

	if g.node == nil {
		g.logger.Error("雪花算法节点未初始化")
		return 0, fmt.Errorf("snowflake node not initialized")
	}

	// 生成雪花 ID
	id := g.node.Generate().Int64()
	g.logger.Debug("生成雪花 ID 成功", clog.Int64("id", id))
	return id, nil
}

// GetNodeID 获取当前节点 ID
func (g *snowflakeGenerator) GetNodeID() int64 {
	if g.node == nil {
		return -1
	}
	// 从雪花 ID 中提取节点 ID（位 12-21）
	// 这里我们返回配置中的节点 ID，因为 bwmarrin/snowflake 库没有直接提供获取节点 ID 的方法
	if g.config.AutoNodeID {
		// 如果是自动生成的，尝试从 IP 获取
		ip, err := getLocalIP()
		if err != nil {
			return 1
		}
		ipObj := net.ParseIP(ip)
		if ipObj == nil {
			return 1
		}
		return int64(ipObj.To4()[3])
	}
	return g.config.NodeID
}

// ParseID 解析雪花 ID，返回时间戳、节点 ID 和序列号
func (g *snowflakeGenerator) ParseID(id int64) (timestamp int64, nodeID int64, sequence int64) {
	// 雪花算法 ID 结构：
	// 1 位符号位（始终为 0）+ 41 位时间戳 + 10 位节点 ID + 12 位序列号

	// 提取序列号（低 12 位）
	sequence = id & 0xFFF

	// 提取节点 ID（第 12-21 位）
	nodeID = (id >> 12) & 0x3FF

	// 提取时间戳（第 22-62 位）
	timestamp = (id >> 22) + snowflake.Epoch

	return timestamp, nodeID, sequence
}

// Type 返回生成器类型
func (g *snowflakeGenerator) Type() GeneratorType {
	return SnowflakeType
}

// Close 关闭生成器
func (g *snowflakeGenerator) Close() error {
	g.logger.Info("关闭雪花算法 ID 生成器")
	return nil
}

// getLocalIP 获取本机首个非环回IPv4地址
func getLocalIP() (string, error) {
	logger := clog.Module("idgen")
	logger.Debug("尝试获取本机 IP 地址")

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		logger.Error("获取网络接口失败", clog.Err(err))
		return "", err
	}

	// 遍历所有网络接口
	for _, iface := range interfaces {
		// 跳过禁用的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 获取该接口的所有地址
		addrs, err := iface.Addrs()
		if err != nil {
			logger.Debug("获取接口地址失败", clog.String("interface", iface.Name), clog.Err(err))
			continue
		}

		// 遍历所有地址
		for _, addr := range addrs {
			// 检查是否为IP地址
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// 检查是否为IPv4地址，且不是环回地址
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}

			// 返回找到的第一个有效IP地址
			ipAddress := ip.String()
			logger.Debug("找到有效的本机 IP", clog.String("ip", ipAddress))
			return ipAddress, nil
		}
	}

	logger.Warn("未找到有效的本机 IP 地址")
	return "", errors.New("no valid local IP address found")
}
