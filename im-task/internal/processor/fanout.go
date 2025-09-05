package processor

import (
	"context"
	"encoding/json"

	"github.com/ceyewan/gochat/api/kafka"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// FanoutProcessor 大群消息扩散处理器
type FanoutProcessor struct {
	logger clog.Logger

	// TODO: 添加依赖
	// repoClient repov1.GroupServiceClient
	// mqProducer mq.Producer
}

// NewFanoutProcessor 创建扩散处理器实例
func NewFanoutProcessor() *FanoutProcessor {
	return &FanoutProcessor{
		logger: clog.Module("fanout-processor"),
	}
}

// GetTaskType 返回任务类型
func (p *FanoutProcessor) GetTaskType() kafka.TaskType {
	return kafka.TaskTypeFanout
}

// Process 处理大群消息扩散任务
// TODO: 实现具体的扩散逻辑
func (p *FanoutProcessor) Process(ctx context.Context, task *kafka.TaskMessage) error {
	p.logger.Info("处理大群消息扩散任务",
		clog.String("task_id", task.TaskID),
		clog.String("trace_id", task.TraceID))

	// 解析任务数据
	var data kafka.FanoutTaskData
	if err := json.Unmarshal(task.Data, &data); err != nil {
		p.logger.Error("解析扩散任务数据失败", clog.Err(err))
		return err
	}

	// TODO: 实现具体的扩散逻辑
	// 1. 分批获取群组成员
	// 2. 批量查询在线状态
	// 3. 生产下行消息到对应网关

	p.logger.Info("大群消息扩散任务处理完成",
		clog.String("group_id", data.GroupID),
		clog.String("message_id", data.MessageID))

	return nil
}
