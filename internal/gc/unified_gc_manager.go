package gc

import (
	"sync"
	"sync/atomic"
	"time"
)

// UnifiedGCManager 统一GC管理器
// 协调95%引用计数 + 5%标记清除的双重GC策略
type UnifiedGCManager struct {
	// GC组件
	refCountGC  *RefCountGC
	markSweepGC *MarkSweepGC
	allocator   AQLAllocator

	// 配置参数
	config *UnifiedGCConfig

	// GC状态
	isEnabled    bool      // GC是否启用
	lastFullGC   time.Time // 上次完整GC时间
	gcGeneration uint64    // GC代数

	// 统计信息
	stats UnifiedGCStats

	// 同步控制
	mutex sync.RWMutex

	// 触发器和调度器
	triggerChan chan struct{} // GC触发信号
	stopChan    chan struct{} // 停止信号
	workerCount int           // 工作线程数
}

// UnifiedGCConfig 统一GC配置
type UnifiedGCConfig struct {
	// 双重策略配置
	RefCountConfig  *RefCountGCConfig  // 引用计数GC配置
	MarkSweepConfig *MarkSweepGCConfig // 标记清除GC配置

	// 触发策略
	FullGCInterval      time.Duration // 完整GC间隔
	MemoryPressureLimit uint64        // 内存压力阈值
	ObjectCountLimit    int           // 对象数量阈值

	// 性能调优
	EnableConcurrentGC bool          // 启用并发GC
	MaxGCPauseTime     time.Duration // 最大GC暂停时间
	GCWorkerCount      int           // GC工作线程数

	// 策略选择
	CyclicObjectThreshold float64 // 循环引用对象阈值比例

	// 调试选项
	EnableGCLogging bool // 启用GC日志
	VerboseLogging  bool // 详细日志
}

// UnifiedGCStats 统一GC统计
type UnifiedGCStats struct {
	// 基本统计
	TotalGCCycles    uint64 // 总GC周期数
	RefCountCycles   uint64 // 引用计数GC周期数
	MarkSweepCycles  uint64 // 标记清除GC周期数
	ObjectsCollected uint64 // 回收对象总数

	// 性能指标
	TotalGCTime   uint64 // 总GC时间
	AverageGCTime uint64 // 平均GC时间
	MaxPauseTime  uint64 // 最大暂停时间
	GCThroughput  uint64 // GC吞吐量

	// 内存统计
	HeapSize       uint64 // 堆大小
	LiveObjects    uint64 // 存活对象数
	AllocatedBytes uint64 // 已分配字节数
	FreedBytes     uint64 // 已释放字节数

	// 策略统计
	RefCountEfficiency  float64 // 引用计数效率
	MarkSweepEfficiency float64 // 标记清除效率
	CyclicObjectRatio   float64 // 循环引用对象比例

	// 错误统计
	GCErrors uint64 // GC错误次数
}

// DefaultUnifiedGCConfig 默认统一GC配置
var DefaultUnifiedGCConfig = UnifiedGCConfig{
	RefCountConfig:  &DefaultRefCountGCConfig,
	MarkSweepConfig: &DefaultMarkSweepGCConfig,

	FullGCInterval:      1 * time.Second,
	MemoryPressureLimit: 100 * 1024 * 1024, // 100MB
	ObjectCountLimit:    100000,

	EnableConcurrentGC: false, // MVP阶段暂时禁用并发GC
	MaxGCPauseTime:     10 * time.Millisecond,
	GCWorkerCount:      1,

	CyclicObjectThreshold: 0.05, // 5%循环引用阈值

	EnableGCLogging: false,
	VerboseLogging:  false,
}

// NewUnifiedGCManager 创建统一GC管理器
func NewUnifiedGCManager(allocator AQLAllocator, config *UnifiedGCConfig) *UnifiedGCManager {
	if config == nil {
		config = &DefaultUnifiedGCConfig
	}

	// 创建RefCountGC
	refCountGC := NewRefCountGC(allocator, config.RefCountConfig)

	// 创建MarkSweepGC
	markSweepGC := NewMarkSweepGC(allocator, config.MarkSweepConfig)

	mgr := &UnifiedGCManager{
		refCountGC:   refCountGC,
		markSweepGC:  markSweepGC,
		allocator:    allocator,
		config:       config,
		isEnabled:    true,
		lastFullGC:   time.Now(),
		gcGeneration: 1,
		triggerChan:  make(chan struct{}, 10),
		stopChan:     make(chan struct{}),
		workerCount:  config.GCWorkerCount,
	}

	// 启动GC工作线程
	mgr.startWorkers()

	return mgr
}

// startWorkers 启动GC工作线程
func (mgr *UnifiedGCManager) startWorkers() {
	for i := 0; i < mgr.workerCount; i++ {
		go mgr.gcWorker()
	}
}

// gcWorker GC工作线程
func (mgr *UnifiedGCManager) gcWorker() {
	for {
		select {
		case <-mgr.triggerChan:
			if mgr.isEnabled {
				mgr.runGCCycle()
			}
		case <-mgr.stopChan:
			return
		}
	}
}

// OnObjectAllocated 对象分配时的回调
func (mgr *UnifiedGCManager) OnObjectAllocated(obj *GCObject) {
	if obj == nil || !mgr.isEnabled {
		return
	}

	// 引用计数GC总是处理所有对象
	mgr.refCountGC.IncRef(obj)

	// 检查是否为可能的循环引用对象
	if obj.Header.IsCyclic() {
		mgr.markSweepGC.TrackObject(obj)
	}

	// 更新统计
	atomic.AddUint64(&mgr.stats.AllocatedBytes, uint64(obj.Size()))

	// 检查是否需要触发GC
	if mgr.shouldTriggerGC() {
		mgr.TriggerGC()
	}
}

// OnObjectReferenced 对象被引用时的回调
func (mgr *UnifiedGCManager) OnObjectReferenced(obj *GCObject) {
	if obj == nil || !mgr.isEnabled {
		return
	}

	mgr.refCountGC.IncRef(obj)
}

// OnObjectDereferenced 对象被解除引用时的回调
func (mgr *UnifiedGCManager) OnObjectDereferenced(obj *GCObject) {
	if obj == nil || !mgr.isEnabled {
		return
	}

	mgr.refCountGC.DecRef(obj)
}

// OnObjectFreed 对象被释放时的回调
func (mgr *UnifiedGCManager) OnObjectFreed(obj *GCObject) {
	if obj == nil {
		return
	}

	// 从标记清除GC中取消跟踪
	mgr.markSweepGC.UntrackObject(obj)

	// 更新统计
	atomic.AddUint64(&mgr.stats.FreedBytes, uint64(obj.Size()))
	atomic.AddUint64(&mgr.stats.ObjectsCollected, 1)
}

// AddRootObject 添加根对象
func (mgr *UnifiedGCManager) AddRootObject(obj *GCObject) {
	if obj == nil {
		return
	}

	mgr.markSweepGC.AddRootObject(obj)
}

// RemoveRootObject 移除根对象
func (mgr *UnifiedGCManager) RemoveRootObject(obj *GCObject) {
	if obj == nil {
		return
	}

	mgr.markSweepGC.RemoveRootObject(obj)
}

// shouldTriggerGC 检查是否应该触发GC
func (mgr *UnifiedGCManager) shouldTriggerGC() bool {
	// 检查内存压力
	allocatedBytes := atomic.LoadUint64(&mgr.stats.AllocatedBytes)
	freedBytes := atomic.LoadUint64(&mgr.stats.FreedBytes)
	liveBytes := allocatedBytes - freedBytes

	if liveBytes > mgr.config.MemoryPressureLimit {
		return true
	}

	// 检查对象数量
	trackedObjects := mgr.markSweepGC.GetTrackedObjectCount()
	if trackedObjects > mgr.config.ObjectCountLimit {
		return true
	}

	// 检查时间间隔
	if time.Since(mgr.lastFullGC) > mgr.config.FullGCInterval {
		return true
	}

	return false
}

// TriggerGC 触发GC
func (mgr *UnifiedGCManager) TriggerGC() {
	if !mgr.isEnabled {
		return
	}

	select {
	case mgr.triggerChan <- struct{}{}:
	default:
		// 信道满，GC请求被丢弃
	}
}

// runGCCycle 运行GC周期
func (mgr *UnifiedGCManager) runGCCycle() {
	startTime := time.Now()

	mgr.mutex.Lock()
	mgr.gcGeneration++
	mgr.mutex.Unlock()

	// 首先运行引用计数GC
	mgr.refCountGC.ForceCollect()
	atomic.AddUint64(&mgr.stats.RefCountCycles, 1)

	// 检查是否需要运行标记清除GC
	if mgr.markSweepGC.ShouldRunGC() {
		mgr.markSweepGC.RunGC()
		atomic.AddUint64(&mgr.stats.MarkSweepCycles, 1)
		mgr.lastFullGC = time.Now()
	}

	duration := time.Since(startTime)

	// 更新统计
	atomic.AddUint64(&mgr.stats.TotalGCCycles, 1)
	atomic.AddUint64(&mgr.stats.TotalGCTime, uint64(duration.Nanoseconds()))

	// 更新最大暂停时间
	pauseTime := uint64(duration.Nanoseconds())
	for {
		oldMax := atomic.LoadUint64(&mgr.stats.MaxPauseTime)
		if pauseTime <= oldMax || atomic.CompareAndSwapUint64(&mgr.stats.MaxPauseTime, oldMax, pauseTime) {
			break
		}
	}
}

// ForceGC 强制执行完整GC
func (mgr *UnifiedGCManager) ForceGC() {
	if !mgr.isEnabled {
		return
	}

	startTime := time.Now()

	mgr.mutex.Lock()
	mgr.gcGeneration++
	mgr.mutex.Unlock()

	// 强制运行两个GC组件
	mgr.refCountGC.ForceCollect()
	atomic.AddUint64(&mgr.stats.RefCountCycles, 1)

	mgr.markSweepGC.ForceGC()
	atomic.AddUint64(&mgr.stats.MarkSweepCycles, 1)

	mgr.lastFullGC = time.Now()

	duration := time.Since(startTime)

	// 更新统计
	atomic.AddUint64(&mgr.stats.TotalGCCycles, 1)
	atomic.AddUint64(&mgr.stats.TotalGCTime, uint64(duration.Nanoseconds()))

	// 更新最大暂停时间
	pauseTime := uint64(duration.Nanoseconds())
	for {
		oldMax := atomic.LoadUint64(&mgr.stats.MaxPauseTime)
		if pauseTime <= oldMax || atomic.CompareAndSwapUint64(&mgr.stats.MaxPauseTime, oldMax, pauseTime) {
			break
		}
	}
}

// Enable 启用GC
func (mgr *UnifiedGCManager) Enable() {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	mgr.isEnabled = true
}

// Disable 禁用GC
func (mgr *UnifiedGCManager) Disable() {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	mgr.isEnabled = false
}

// IsEnabled 检查GC是否启用
func (mgr *UnifiedGCManager) IsEnabled() bool {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	return mgr.isEnabled
}

// GetStats 获取统计信息
func (mgr *UnifiedGCManager) GetStats() UnifiedGCStats {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()

	// 获取子组件统计
	refCountStats := mgr.refCountGC.GetStats()
	markSweepStats := mgr.markSweepGC.GetStats()

	stats := UnifiedGCStats{
		TotalGCCycles:    atomic.LoadUint64(&mgr.stats.TotalGCCycles),
		RefCountCycles:   atomic.LoadUint64(&mgr.stats.RefCountCycles),
		MarkSweepCycles:  atomic.LoadUint64(&mgr.stats.MarkSweepCycles),
		ObjectsCollected: refCountStats.ObjectsCollected + markSweepStats.ObjectsCollected,

		TotalGCTime:  atomic.LoadUint64(&mgr.stats.TotalGCTime),
		MaxPauseTime: atomic.LoadUint64(&mgr.stats.MaxPauseTime),

		AllocatedBytes: atomic.LoadUint64(&mgr.stats.AllocatedBytes),
		FreedBytes:     atomic.LoadUint64(&mgr.stats.FreedBytes),
		LiveObjects:    markSweepStats.LiveObjects,

		RefCountEfficiency:  mgr.refCountGC.GetEfficiency(),
		MarkSweepEfficiency: mgr.markSweepGC.GetEfficiency(),

		GCErrors: refCountStats.CleanupErrors + markSweepStats.GCErrors,
	}

	// 计算派生统计
	if stats.TotalGCCycles > 0 {
		stats.AverageGCTime = stats.TotalGCTime / stats.TotalGCCycles
	}

	if stats.AllocatedBytes > 0 {
		stats.HeapSize = stats.AllocatedBytes - stats.FreedBytes
	}

	// 计算循环引用对象比例
	totalTracked := markSweepStats.TotalTrackedObjects
	if totalTracked > 0 {
		stats.CyclicObjectRatio = float64(markSweepStats.LiveObjects) / float64(totalTracked)
	}

	return stats
}

// GetRefCountGC 获取引用计数GC组件
func (mgr *UnifiedGCManager) GetRefCountGC() *RefCountGC {
	return mgr.refCountGC
}

// GetMarkSweepGC 获取标记清除GC组件
func (mgr *UnifiedGCManager) GetMarkSweepGC() *MarkSweepGC {
	return mgr.markSweepGC
}

// Configure 配置统一GC管理器
func (mgr *UnifiedGCManager) Configure(config *UnifiedGCConfig) {
	if config == nil {
		return
	}

	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	mgr.config = config

	// 配置子组件
	if config.RefCountConfig != nil {
		mgr.refCountGC.Configure(config.RefCountConfig)
	}
	if config.MarkSweepConfig != nil {
		mgr.markSweepGC.Configure(config.MarkSweepConfig)
	}
}

// Shutdown 关闭GC管理器
func (mgr *UnifiedGCManager) Shutdown() {
	mgr.Disable()

	// 停止工作线程
	close(mgr.stopChan)

	// 执行最后一次GC
	mgr.ForceGC()

	// 关闭子组件
	mgr.refCountGC.Shutdown()
	mgr.markSweepGC.Shutdown()
}

// GetMemoryUsage 获取内存使用情况
func (mgr *UnifiedGCManager) GetMemoryUsage() (allocated, freed, live uint64) {
	allocated = atomic.LoadUint64(&mgr.stats.AllocatedBytes)
	freed = atomic.LoadUint64(&mgr.stats.FreedBytes)
	live = allocated - freed
	return
}

// GetEfficiency 获取总体GC效率
func (mgr *UnifiedGCManager) GetEfficiency() float64 {
	stats := mgr.GetStats()

	// 加权平均效率：95%引用计数 + 5%标记清除
	refCountWeight := 0.95
	markSweepWeight := 0.05

	return refCountWeight*stats.RefCountEfficiency + markSweepWeight*stats.MarkSweepEfficiency
}

// IsGCRunning 检查GC是否正在运行
func (mgr *UnifiedGCManager) IsGCRunning() bool {
	return mgr.markSweepGC.IsRunning()
}

// GetGCGeneration 获取当前GC代数
func (mgr *UnifiedGCManager) GetGCGeneration() uint64 {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	return mgr.gcGeneration
}
