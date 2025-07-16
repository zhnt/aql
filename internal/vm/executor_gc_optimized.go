package vm

import (
	"fmt"
	"sync/atomic"
	"time"
)

// =============================================================================
// GC 优化的 Executor 扩展
// =============================================================================

// ExecutorGCStats GC相关的执行器统计
type ExecutorGCStats struct {
	// GC操作统计
	AutoGCTriggers     uint64 // 自动触发的GC次数
	WriteBarrierCalls  uint64 // 写屏障调用次数
	RefCountOperations uint64 // 引用计数操作次数
	WeakRefOperations  uint64 // 弱引用操作次数

	// 性能统计
	TotalGCTime   uint64 // 总GC时间(纳秒)
	AverageGCTime uint64 // 平均GC时间
	MaxGCPause    uint64 // 最大GC暂停时间

	// 内存统计
	AllocatedObjects uint64 // 分配的对象数量
	CollectedObjects uint64 // 回收的对象数量
	LiveObjects      uint64 // 存活对象数量

	// 错误统计
	GCErrors uint64 // GC错误次数
}

// GCOptimizer GC优化器
type GCOptimizer struct {
	executor *Executor
	stats    ExecutorGCStats
	config   *GCOptimizerConfig

	// 自适应调优
	lastGCTime        time.Time
	gcInterval        time.Duration
	adaptiveThreshold int

	// 批量操作缓冲区
	refCountBatch     []ValueGC
	writeBarrierBatch []struct {
		obj   ValueGC
		field ValueGC
	}
}

// GCOptimizerConfig GC优化器配置
type GCOptimizerConfig struct {
	// 自动GC触发
	EnableAutoGC         bool          // 启用自动GC
	GCInterval           time.Duration // GC间隔
	MemoryPressureLimit  int           // 内存压力阈值
	ObjectCountThreshold int           // 对象数量阈值

	// 批量操作
	EnableBatchOperations bool          // 启用批量操作
	BatchSize             int           // 批量大小
	FlushInterval         time.Duration // 批量刷新间隔

	// 写屏障优化
	EnableWriteBarrierOpt bool // 启用写屏障优化
	CoalesceWrites        bool // 合并写操作

	// 监控和调试
	EnableGCProfiling bool // 启用GC性能分析
	VerboseGCLogging  bool // 详细GC日志
}

// DefaultGCOptimizerConfig 默认GC优化器配置
var DefaultGCOptimizerConfig = GCOptimizerConfig{
	EnableAutoGC:         true,
	GCInterval:           10 * time.Millisecond,
	MemoryPressureLimit:  1000,
	ObjectCountThreshold: 10000,

	EnableBatchOperations: true,
	BatchSize:             16,
	FlushInterval:         1 * time.Millisecond,

	EnableWriteBarrierOpt: true,
	CoalesceWrites:        true,

	EnableGCProfiling: false,
	VerboseGCLogging:  false,
}

// NewGCOptimizer 创建GC优化器
func NewGCOptimizer(executor *Executor, config *GCOptimizerConfig) *GCOptimizer {
	if config == nil {
		config = &DefaultGCOptimizerConfig
	}

	optimizer := &GCOptimizer{
		executor:          executor,
		config:            config,
		lastGCTime:        time.Now(),
		gcInterval:        config.GCInterval,
		adaptiveThreshold: config.ObjectCountThreshold,
		refCountBatch:     make([]ValueGC, 0, config.BatchSize),
		writeBarrierBatch: make([]struct {
			obj   ValueGC
			field ValueGC
		}, 0, config.BatchSize),
	}

	return optimizer
}

// =============================================================================
// GC 自动管理方法
// =============================================================================

// CheckAndTriggerGC 检查并触发GC
func (opt *GCOptimizer) CheckAndTriggerGC() {
	if !opt.config.EnableAutoGC {
		return
	}

	// 时间间隔检查
	if time.Since(opt.lastGCTime) > opt.gcInterval {
		opt.triggerGC("interval")
		return
	}

	// 内存压力检查
	if opt.shouldTriggerByMemoryPressure() {
		opt.triggerGC("memory_pressure")
		return
	}

	// 对象数量检查
	if opt.shouldTriggerByObjectCount() {
		opt.triggerGC("object_count")
		return
	}
}

// shouldTriggerByMemoryPressure 检查是否因内存压力需要触发GC
func (opt *GCOptimizer) shouldTriggerByMemoryPressure() bool {
	if GlobalValueGCManager == nil || GlobalValueGCManager.gcManager == nil {
		return false
	}

	// 获取内存使用情况
	allocated, _, live := GlobalValueGCManager.gcManager.GetMemoryUsage()

	// 如果存活内存超过阈值，触发GC
	pressureRatio := float64(live) / float64(allocated)
	return pressureRatio > 0.8 // 80%内存压力阈值
}

// shouldTriggerByObjectCount 检查是否因对象数量需要触发GC
func (opt *GCOptimizer) shouldTriggerByObjectCount() bool {
	liveObjects := atomic.LoadUint64(&opt.stats.LiveObjects)
	return int(liveObjects) > opt.adaptiveThreshold
}

// triggerGC 触发GC
func (opt *GCOptimizer) triggerGC(reason string) {
	if opt.config.VerboseGCLogging {
		fmt.Printf("触发GC: 原因=%s, 时间=%v\n", reason, time.Now())
	}

	startTime := time.Now()

	// 刷新批量操作
	opt.flushBatchOperations()

	// 触发GC
	err := TriggerGCCollection()
	if err != nil {
		atomic.AddUint64(&opt.stats.GCErrors, 1)
		if opt.config.VerboseGCLogging {
			fmt.Printf("GC触发失败: %v\n", err)
		}
		return
	}

	// 更新统计
	duration := time.Since(startTime)
	atomic.AddUint64(&opt.stats.AutoGCTriggers, 1)
	atomic.AddUint64(&opt.stats.TotalGCTime, uint64(duration.Nanoseconds()))

	// 更新最大暂停时间
	pauseTime := uint64(duration.Nanoseconds())
	for {
		oldMax := atomic.LoadUint64(&opt.stats.MaxGCPause)
		if pauseTime <= oldMax || atomic.CompareAndSwapUint64(&opt.stats.MaxGCPause, oldMax, pauseTime) {
			break
		}
	}

	opt.lastGCTime = time.Now()

	// 自适应调整
	opt.adaptGCStrategy(duration)
}

// adaptGCStrategy 自适应调整GC策略
func (opt *GCOptimizer) adaptGCStrategy(lastGCDuration time.Duration) {
	// 如果GC暂停时间过长，增加触发阈值
	if lastGCDuration > 5*time.Millisecond {
		opt.adaptiveThreshold = int(float64(opt.adaptiveThreshold) * 1.2)
		opt.gcInterval = time.Duration(float64(opt.gcInterval) * 1.1)
	} else if lastGCDuration < 1*time.Millisecond {
		// 如果GC暂停时间很短，可以更频繁地触发
		opt.adaptiveThreshold = int(float64(opt.adaptiveThreshold) * 0.9)
		opt.gcInterval = time.Duration(float64(opt.gcInterval) * 0.9)
	}

	// 限制阈值范围
	if opt.adaptiveThreshold < 1000 {
		opt.adaptiveThreshold = 1000
	}
	if opt.adaptiveThreshold > 50000 {
		opt.adaptiveThreshold = 50000
	}

	// 限制间隔范围
	if opt.gcInterval < 1*time.Millisecond {
		opt.gcInterval = 1 * time.Millisecond
	}
	if opt.gcInterval > 100*time.Millisecond {
		opt.gcInterval = 100 * time.Millisecond
	}
}

// =============================================================================
// 批量GC操作方法
// =============================================================================

// BatchIncRef 批量增加引用计数
func (opt *GCOptimizer) BatchIncRef(value ValueGC) {
	// 无论是否启用批量操作，都要更新统计
	atomic.AddUint64(&opt.stats.RefCountOperations, 1)

	if !opt.config.EnableBatchOperations {
		// 直接执行
		if value.IsGCManaged() {
			IncRefGC(value)
		}
		return
	}

	// 添加到批量队列
	opt.refCountBatch = append(opt.refCountBatch, value)

	// 检查是否需要刷新
	if len(opt.refCountBatch) >= opt.config.BatchSize {
		opt.flushRefCountBatch()
	}
}

// BatchDecRef 批量减少引用计数
func (opt *GCOptimizer) BatchDecRef(value ValueGC) {
	// 无论是否启用批量操作，都要更新统计
	atomic.AddUint64(&opt.stats.RefCountOperations, 1)

	if !opt.config.EnableBatchOperations {
		// 直接执行
		if value.IsGCManaged() {
			DecRefGC(value)
		}
		return
	}

	// 对于减少引用计数，我们需要立即处理以避免悬挂指针
	// 但可以优化连续的减少操作
	if value.IsGCManaged() {
		DecRefGC(value)
	}
}

// BatchWriteBarrier 批量写屏障
func (opt *GCOptimizer) BatchWriteBarrier(objValue, fieldValue ValueGC) {
	// 无论是否启用批量操作，都要更新统计
	atomic.AddUint64(&opt.stats.WriteBarrierCalls, 1)

	if !opt.config.EnableBatchOperations || !opt.config.EnableWriteBarrierOpt {
		// 直接执行
		if objValue.IsGCManaged() {
			WriteBarrierGC(objValue, fieldValue)
		}
		return
	}

	// 添加到批量队列
	opt.writeBarrierBatch = append(opt.writeBarrierBatch, struct {
		obj   ValueGC
		field ValueGC
	}{objValue, fieldValue})

	// 检查是否需要刷新
	if len(opt.writeBarrierBatch) >= opt.config.BatchSize {
		opt.flushWriteBarrierBatch()
	}
}

// flushBatchOperations 刷新所有批量操作
func (opt *GCOptimizer) flushBatchOperations() {
	opt.flushRefCountBatch()
	opt.flushWriteBarrierBatch()
}

// flushRefCountBatch 刷新引用计数批量操作
func (opt *GCOptimizer) flushRefCountBatch() {
	if len(opt.refCountBatch) == 0 {
		return
	}

	// 批量处理引用计数增加
	for _, value := range opt.refCountBatch {
		if value.IsGCManaged() {
			IncRefGC(value)
		}
	}

	// 注意：统计已经在BatchIncRef中更新，这里不再重复更新

	// 清空批量队列
	opt.refCountBatch = opt.refCountBatch[:0]
}

// flushWriteBarrierBatch 刷新写屏障批量操作
func (opt *GCOptimizer) flushWriteBarrierBatch() {
	if len(opt.writeBarrierBatch) == 0 {
		return
	}

	if opt.config.CoalesceWrites {
		// 合并重复的写操作
		opt.coalesceWriteBarriers()
	}

	// 批量处理写屏障
	for _, wb := range opt.writeBarrierBatch {
		if wb.obj.IsGCManaged() {
			WriteBarrierGC(wb.obj, wb.field)
		}
	}

	// 注意：统计已经在BatchWriteBarrier中更新，这里不再重复更新

	// 清空批量队列
	opt.writeBarrierBatch = opt.writeBarrierBatch[:0]
}

// coalesceWriteBarriers 合并写屏障操作
func (opt *GCOptimizer) coalesceWriteBarriers() {
	if len(opt.writeBarrierBatch) <= 1 {
		return
	}

	// 简单的去重策略：移除重复的(obj, field)对
	seen := make(map[uintptr]map[uintptr]bool)
	unique := make([]struct {
		obj   ValueGC
		field ValueGC
	}, 0, len(opt.writeBarrierBatch))

	for _, wb := range opt.writeBarrierBatch {
		objPtr := uintptr(wb.obj.data)
		fieldPtr := uintptr(wb.field.data)

		if seen[objPtr] == nil {
			seen[objPtr] = make(map[uintptr]bool)
		}

		if !seen[objPtr][fieldPtr] {
			seen[objPtr][fieldPtr] = true
			unique = append(unique, wb)
		}
	}

	opt.writeBarrierBatch = unique
}

// =============================================================================
// 栈帧生命周期管理
// =============================================================================

// OnStackFrameCreate 栈帧创建时的GC管理
func (opt *GCOptimizer) OnStackFrameCreate(frame *StackFrame) {
	// 为栈帧中的GC对象增加引用计数
	for i := range frame.Registers {
		if frame.Registers[i].IsGCManaged() {
			opt.BatchIncRef(frame.Registers[i])
		}
	}

	// 更新统计
	atomic.AddUint64(&opt.stats.AllocatedObjects, uint64(len(frame.Registers)))
}

// OnStackFrameDestroy 栈帧销毁时的GC管理
func (opt *GCOptimizer) OnStackFrameDestroy(frame *StackFrame) {
	// 为栈帧中的GC对象减少引用计数
	for i := range frame.Registers {
		if frame.Registers[i].IsGCManaged() {
			opt.BatchDecRef(frame.Registers[i])
		}
	}

	// 刷新批量操作
	opt.flushBatchOperations()

	// 更新统计
	atomic.AddUint64(&opt.stats.CollectedObjects, uint64(len(frame.Registers)))
}

// OnRegisterSet 寄存器设置时的GC管理
func (opt *GCOptimizer) OnRegisterSet(oldValue, newValue ValueGC) {
	// 处理旧值
	if oldValue.IsGCManaged() {
		opt.BatchDecRef(oldValue)
	}

	// 处理新值
	if newValue.IsGCManaged() {
		opt.BatchIncRef(newValue)
	}
}

// =============================================================================
// 统计和监控方法
// =============================================================================

// GetGCStats 获取GC统计信息
func (opt *GCOptimizer) GetGCStats() ExecutorGCStats {
	stats := opt.stats

	// 计算平均GC时间
	totalGCTime := atomic.LoadUint64(&stats.TotalGCTime)
	gcTriggers := atomic.LoadUint64(&stats.AutoGCTriggers)
	if gcTriggers > 0 {
		stats.AverageGCTime = totalGCTime / gcTriggers
	}

	// 计算存活对象数量
	allocated := atomic.LoadUint64(&stats.AllocatedObjects)
	collected := atomic.LoadUint64(&stats.CollectedObjects)
	stats.LiveObjects = allocated - collected

	return stats
}

// ResetGCStats 重置GC统计信息
func (opt *GCOptimizer) ResetGCStats() {
	atomic.StoreUint64(&opt.stats.AutoGCTriggers, 0)
	atomic.StoreUint64(&opt.stats.WriteBarrierCalls, 0)
	atomic.StoreUint64(&opt.stats.RefCountOperations, 0)
	atomic.StoreUint64(&opt.stats.WeakRefOperations, 0)
	atomic.StoreUint64(&opt.stats.TotalGCTime, 0)
	atomic.StoreUint64(&opt.stats.AverageGCTime, 0)
	atomic.StoreUint64(&opt.stats.MaxGCPause, 0)
	atomic.StoreUint64(&opt.stats.AllocatedObjects, 0)
	atomic.StoreUint64(&opt.stats.CollectedObjects, 0)
	atomic.StoreUint64(&opt.stats.LiveObjects, 0)
	atomic.StoreUint64(&opt.stats.GCErrors, 0)
}

// PrintGCReport 打印GC报告
func (opt *GCOptimizer) PrintGCReport() {
	stats := opt.GetGCStats()

	fmt.Println("=== GC优化器报告 ===")
	fmt.Printf("自动GC触发次数: %d\n", stats.AutoGCTriggers)
	fmt.Printf("写屏障调用次数: %d\n", stats.WriteBarrierCalls)
	fmt.Printf("引用计数操作次数: %d\n", stats.RefCountOperations)
	fmt.Printf("弱引用操作次数: %d\n", stats.WeakRefOperations)
	fmt.Printf("总GC时间: %d纳秒\n", stats.TotalGCTime)
	fmt.Printf("平均GC时间: %d纳秒\n", stats.AverageGCTime)
	fmt.Printf("最大GC暂停: %d纳秒\n", stats.MaxGCPause)
	fmt.Printf("分配对象数: %d\n", stats.AllocatedObjects)
	fmt.Printf("回收对象数: %d\n", stats.CollectedObjects)
	fmt.Printf("存活对象数: %d\n", stats.LiveObjects)
	fmt.Printf("GC错误次数: %d\n", stats.GCErrors)

	// 计算性能指标
	if stats.AutoGCTriggers > 0 {
		avgPauseMs := float64(stats.AverageGCTime) / 1e6
		maxPauseMs := float64(stats.MaxGCPause) / 1e6
		fmt.Printf("平均GC暂停: %.2fms\n", avgPauseMs)
		fmt.Printf("最大GC暂停: %.2fms\n", maxPauseMs)
	}

	if stats.AllocatedObjects > 0 {
		collectionRate := float64(stats.CollectedObjects) / float64(stats.AllocatedObjects) * 100
		fmt.Printf("回收率: %.2f%%\n", collectionRate)
	}
}

// =============================================================================
// Executor 扩展方法
// =============================================================================

// AddGCOptimizer 为Executor添加GC优化器
func (e *Executor) AddGCOptimizer(config *GCOptimizerConfig) *GCOptimizer {
	return NewGCOptimizer(e, config)
}
