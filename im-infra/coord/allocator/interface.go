package allocator

import "context"

// InstanceIDAllocator 为一类服务的实例分配唯一的、可自动回收的ID
type InstanceIDAllocator interface {
    // AcquireID 尝试获取一个未被使用的 ID
    // ctx 用于控制本次获取操作的超时
    // 返回的 AllocatedID 对象代表一个被成功占用的、会自动续租的 ID
    AcquireID(ctx context.Context) (AllocatedID, error)
}

// AllocatedID 代表一个被当前服务实例持有的、会自动续租的 ID
type AllocatedID interface {
    // ID 返回被分配的整数 ID
    ID() int
    // Close 主动释放当前持有的 ID。这是一个幂等操作
    // 如果不调用此方法，ID 将在服务实例关闭时通过 etcd 的租约机制自动释放
    // ctx 用于控制本次释放操作的超时
    Close(ctx context.Context) error
}