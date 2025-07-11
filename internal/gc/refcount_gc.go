package gc

import (
	"sync"
	"sync/atomic"
)

// RefCountGC 引用计数垃圾收集器
// 处理95%常规对象的即时回收，支持即时释放和延迟清理
type RefCountGC struct {
	// 配置参数
	config *RefCountGCConfig

	// 延迟清理队列 - 处理可能的循环引用
	deferredQueue chan *GCObject
	queueSize     int32 // 改为int32以支持原子操作
	queueWorker   bool

	// 统计信息
	stats RefCountGCStats

	// 同步控制
	mutex sync.RWMutex

	// 分配器引用
	allocator AQLAllocator
}

// RefCountGCConfig 引用计数GC配置
type RefCountGCConfig struct {
	// 延迟清理配置
	EnableDeferredCleanup bool // 启用延迟清理
	DeferredQueueSize     int  // 延迟队列大小
	MaxDeferredObjects    int  // 最大延迟对象数

	// 性能调优
	EnableZeroRefOptimization bool // 启用零引用优化
	EnableBatchCleanup        bool // 启用批量清理

	// 调试选项
	EnableRefCountLogging bool // 启用引用计数日志
	VerboseLogging        bool // 详细日志
}

// RefCountGCStats 引用计数GC统计
type RefCountGCStats struct {
	// 基本统计
	ObjectsCollected     uint64 // 回收对象总数
	ImmediateCollections uint64 // 即时回收次数
	DeferredCollections  uint64 // 延迟回收次数

	// 引用计数操作
	IncRefOperations uint64 // 增加引用计数操作
	DecRefOperations uint64 // 减少引用计数操作
	ZeroRefEvents    uint64 // 零引用事件

	// 性能指标
	TotalCleanupTime   uint64 // 总清理时间(纳秒)
	AverageCleanupTime uint64 // 平均清理时间

	// 错误统计
	CleanupErrors uint64 // 清理错误次数
}

// DefaultRefCountGCConfig 默认引用计数GC配置
var DefaultRefCountGCConfig = RefCountGCConfig{
	EnableDeferredCleanup:     true,
	DeferredQueueSize:         1000,
	MaxDeferredObjects:        10000,
	EnableZeroRefOptimization: true,
	EnableBatchCleanup:        true,
	EnableRefCountLogging:     false,
	VerboseLogging:            false,
}

// NewRefCountGC 创建引用计数垃圾收集器
func NewRefCountGC(allocator AQLAllocator, config *RefCountGCConfig) *RefCountGC {
	if config == nil {
		config = &DefaultRefCountGCConfig
	}

	gc := &RefCountGC{
		config:        config,
		deferredQueue: make(chan *GCObject, config.DeferredQueueSize),
		queueSize:     0,
		queueWorker:   false,
		allocator:     allocator,
	}

	// 启动延迟清理worker
	if config.EnableDeferredCleanup {
		go gc.deferredCleanupWorker()
		gc.queueWorker = true
	}

	return gc
}

// IncRef 增加对象引用计数
func (gc *RefCountGC) IncRef(obj *GCObject) {
	if obj == nil {
		return
	}

	// 原子增加引用计数
	newCount := obj.Header.IncRef()

	// 更新统计
	atomic.AddUint64(&gc.stats.IncRefOperations, 1)

	if gc.config.EnableRefCountLogging {
		// 这里可以添加日志记录
		_ = newCount
	}
}

// DecRef 减少对象引用计数，可能触发回收
func (gc *RefCountGC) DecRef(obj *GCObject) {
	if obj == nil {
		return
	}

	// 原子减少引用计数
	newCount := obj.Header.DecRef()

	// 更新统计
	atomic.AddUint64(&gc.stats.DecRefOperations, 1)

	// 检查是否需要回收
	if newCount == 0 {
		gc.handleZeroRef(obj)
	}
}

// handleZeroRef 处理零引用对象
func (gc *RefCountGC) handleZeroRef(obj *GCObject) {
	atomic.AddUint64(&gc.stats.ZeroRefEvents, 1)

	// 检查对象是否可能有循环引用
	if obj.Header.IsCyclic() {
		// 可能有循环引用，放入延迟清理队列
		gc.deferCleanup(obj)
	} else {
		// 没有循环引用，即时清理
		gc.immediateCleanup(obj)
	}
}

// immediateCleanup 即时清理对象
func (gc *RefCountGC) immediateCleanup(obj *GCObject) {
	// 先递归减少子对象引用
	gc.cleanupChildren(obj)

	// 释放对象内存
	gc.allocator.Deallocate(obj)

	// 更新统计
	atomic.AddUint64(&gc.stats.ObjectsCollected, 1)
	atomic.AddUint64(&gc.stats.ImmediateCollections, 1)
}

// deferCleanup 延迟清理对象
func (gc *RefCountGC) deferCleanup(obj *GCObject) {
	if !gc.config.EnableDeferredCleanup {
		// 禁用延迟清理，直接清理
		gc.immediateCleanup(obj)
		return
	}

	// 尝试放入延迟队列
	select {
	case gc.deferredQueue <- obj:
		atomic.AddInt32(&gc.queueSize, 1)
	default:
		// 队列满，直接清理
		gc.immediateCleanup(obj)
	}
}

// deferredCleanupWorker 延迟清理工作线程
func (gc *RefCountGC) deferredCleanupWorker() {
	for obj := range gc.deferredQueue {
		if obj != nil {
			gc.processDeferredObject(obj)
			atomic.AddInt32(&gc.queueSize, -1)
		}
	}
}

// processDeferredObject 处理延迟清理对象
func (gc *RefCountGC) processDeferredObject(obj *GCObject) {
	// 再次检查引用计数，可能在队列中时被重新引用
	if obj.Header.RefCount() > 0 {
		return // 对象已被重新引用，不需要清理
	}

	// 执行清理
	gc.immediateCleanup(obj)

	// 更新统计
	atomic.AddUint64(&gc.stats.DeferredCollections, 1)
}

// cleanupChildren 清理子对象的引用
func (gc *RefCountGC) cleanupChildren(obj *GCObject) {
	if obj == nil {
		return
	}

	objType := obj.Type()

	// 根据对象类型处理子对象引用
	switch objType {
	case ObjectTypeArray:
		gc.cleanupArrayChildren(obj)
	case ObjectTypeStruct:
		gc.cleanupStructChildren(obj)
	case ObjectTypeClosure:
		gc.cleanupClosureChildren(obj)
	case ObjectTypeString, ObjectTypeFunction:
		// 这些类型没有子对象引用
	default:
		// 未知类型，保守处理
		gc.cleanupGenericChildren(obj)
	}
}

// cleanupArrayChildren 清理数组子对象
func (gc *RefCountGC) cleanupArrayChildren(obj *GCObject) {
	// 获取数组数据（简化实现，实际需要根据具体的数组结构）
	// 这里假设数组元素紧跟在GCObject后面

	// TODO: 实际实现需要根据ArrayObject的具体结构来遍历元素
	// 现在先实现框架，具体的对象布局在后续完善
}

// cleanupStructChildren 清理结构体子对象
func (gc *RefCountGC) cleanupStructChildren(obj *GCObject) {
	// TODO: 根据StructObject的字段布局清理子对象引用
}

// cleanupClosureChildren 清理闭包子对象
func (gc *RefCountGC) cleanupClosureChildren(obj *GCObject) {
	// TODO: 清理闭包捕获的变量引用
}

// cleanupGenericChildren 通用子对象清理
func (gc *RefCountGC) cleanupGenericChildren(obj *GCObject) {
	// 保守策略：扫描对象内存区域，查找可能的GC对象指针
	// 这是一个昂贵操作，只在必要时使用
}

// ForceCollect 强制执行引用计数回收
func (gc *RefCountGC) ForceCollect() {
	// 处理延迟队列中的所有对象
	if gc.config.EnableDeferredCleanup {
		gc.flushDeferredQueue()
	}
}

// flushDeferredQueue 清空延迟队列
func (gc *RefCountGC) flushDeferredQueue() {
	for {
		select {
		case obj := <-gc.deferredQueue:
			if obj != nil {
				gc.processDeferredObject(obj)
				atomic.AddInt32(&gc.queueSize, -1)
			}
		default:
			return // 队列已空
		}
	}
}

// GetStats 获取统计信息
func (gc *RefCountGC) GetStats() RefCountGCStats {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	stats := RefCountGCStats{
		ObjectsCollected:     atomic.LoadUint64(&gc.stats.ObjectsCollected),
		ImmediateCollections: atomic.LoadUint64(&gc.stats.ImmediateCollections),
		DeferredCollections:  atomic.LoadUint64(&gc.stats.DeferredCollections),
		IncRefOperations:     atomic.LoadUint64(&gc.stats.IncRefOperations),
		DecRefOperations:     atomic.LoadUint64(&gc.stats.DecRefOperations),
		ZeroRefEvents:        atomic.LoadUint64(&gc.stats.ZeroRefEvents),
		TotalCleanupTime:     atomic.LoadUint64(&gc.stats.TotalCleanupTime),
		CleanupErrors:        atomic.LoadUint64(&gc.stats.CleanupErrors),
	}

	// 计算平均清理时间
	if stats.ObjectsCollected > 0 {
		stats.AverageCleanupTime = stats.TotalCleanupTime / stats.ObjectsCollected
	}

	return stats
}

// GetQueueSize 获取延迟队列大小
func (gc *RefCountGC) GetQueueSize() int {
	return int(atomic.LoadInt32(&gc.queueSize))
}

// Configure 配置引用计数GC
func (gc *RefCountGC) Configure(config *RefCountGCConfig) {
	if config == nil {
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.config = config
}

// Shutdown 关闭引用计数GC
func (gc *RefCountGC) Shutdown() {
	// 处理剩余的延迟对象
	gc.flushDeferredQueue()

	// 关闭延迟队列
	if gc.queueWorker {
		close(gc.deferredQueue)
		gc.queueWorker = false
	}
}

// IsObjectAlive 检查对象是否存活
func (gc *RefCountGC) IsObjectAlive(obj *GCObject) bool {
	if obj == nil {
		return false
	}

	return obj.Header.RefCount() > 0
}

// TrackObject 开始跟踪对象（用于调试）
func (gc *RefCountGC) TrackObject(obj *GCObject) {
	if obj == nil || !gc.config.EnableRefCountLogging {
		return
	}

	// TODO: 实现对象跟踪逻辑用于调试
}

// GetEfficiency 获取引用计数GC效率
func (gc *RefCountGC) GetEfficiency() float64 {
	stats := gc.GetStats()

	if stats.DecRefOperations == 0 {
		return 0.0
	}

	// 计算即时回收率
	return float64(stats.ImmediateCollections) / float64(stats.DecRefOperations)
}
