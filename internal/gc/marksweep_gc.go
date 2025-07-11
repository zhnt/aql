package gc

import (
	"sync"
	"sync/atomic"
	"time"
)

// MarkSweepGC 标记清除垃圾收集器
// 处理5%循环引用对象，采用双色标记算法（简化版）
type MarkSweepGC struct {
	// 配置参数
	config *MarkSweepGCConfig

	// 对象跟踪
	trackedObjects map[*GCObject]*ObjectInfo // 被跟踪的对象
	rootObjects    []*GCObject               // 根对象集合

	// GC状态
	isRunning    bool      // GC是否正在运行
	lastRunTime  time.Time // 上次运行时间
	gcGeneration uint64    // GC代数

	// 统计信息
	stats MarkSweepGCStats

	// 同步控制
	mutex sync.RWMutex

	// 分配器引用
	allocator AQLAllocator
}

// ObjectInfo 对象跟踪信息
type ObjectInfo struct {
	LastAccess time.Time // 最后访问时间
	Generation uint64    // 对象所属GC代数
	IsRoot     bool      // 是否为根对象
}

// MarkSweepGCConfig 标记清除GC配置
type MarkSweepGCConfig struct {
	// 触发条件
	MaxObjects       int           // 最大跟踪对象数
	GCInterval       time.Duration // GC间隔时间
	ForceGCThreshold int           // 强制GC阈值

	// 性能调优
	EnableIncrementalGC bool          // 启用增量GC
	MaxMarkTime         time.Duration // 最大标记时间
	MaxSweepTime        time.Duration // 最大清除时间

	// 调试选项
	EnableGCLogging bool // 启用GC日志
	VerboseLogging  bool // 详细日志
}

// MarkSweepGCStats 标记清除GC统计
type MarkSweepGCStats struct {
	// 基本统计
	GCCycles         uint64 // GC周期数
	ObjectsCollected uint64 // 回收对象总数
	CyclesDetected   uint64 // 检测到的循环数

	// 性能指标
	TotalGCTime    uint64 // 总GC时间(纳秒)
	AverageGCTime  uint64 // 平均GC时间
	MarkPhaseTime  uint64 // 标记阶段时间
	SweepPhaseTime uint64 // 清除阶段时间

	// 对象统计
	TotalTrackedObjects uint64 // 总跟踪对象数
	LiveObjects         uint64 // 存活对象数
	RootObjects         uint64 // 根对象数

	// 错误统计
	GCErrors uint64 // GC错误次数
}

// DefaultMarkSweepGCConfig 默认标记清除GC配置
var DefaultMarkSweepGCConfig = MarkSweepGCConfig{
	MaxObjects:       10000,
	GCInterval:       100 * time.Millisecond,
	ForceGCThreshold: 5000,

	EnableIncrementalGC: false, // MVP阶段暂时禁用增量GC
	MaxMarkTime:         50 * time.Millisecond,
	MaxSweepTime:        50 * time.Millisecond,

	EnableGCLogging: false,
	VerboseLogging:  false,
}

// NewMarkSweepGC 创建标记清除垃圾收集器
func NewMarkSweepGC(allocator AQLAllocator, config *MarkSweepGCConfig) *MarkSweepGC {
	if config == nil {
		config = &DefaultMarkSweepGCConfig
	}

	gc := &MarkSweepGC{
		config:         config,
		trackedObjects: make(map[*GCObject]*ObjectInfo),
		rootObjects:    make([]*GCObject, 0),
		isRunning:      false,
		lastRunTime:    time.Now(),
		gcGeneration:   1,
		allocator:      allocator,
	}

	return gc
}

// TrackObject 开始跟踪对象
func (gc *MarkSweepGC) TrackObject(obj *GCObject) {
	if obj == nil {
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	// 检查是否已经跟踪
	if _, exists := gc.trackedObjects[obj]; exists {
		return
	}

	// 添加到跟踪列表
	gc.trackedObjects[obj] = &ObjectInfo{
		LastAccess: time.Now(),
		Generation: gc.gcGeneration,
		IsRoot:     false,
	}

	// 更新统计
	atomic.AddUint64(&gc.stats.TotalTrackedObjects, 1)
}

// UntrackObject 停止跟踪对象
func (gc *MarkSweepGC) UntrackObject(obj *GCObject) {
	if obj == nil {
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if _, exists := gc.trackedObjects[obj]; exists {
		delete(gc.trackedObjects, obj)
	}
}

// AddRootObject 添加根对象
func (gc *MarkSweepGC) AddRootObject(obj *GCObject) {
	if obj == nil {
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	// 确保对象被跟踪
	if info, exists := gc.trackedObjects[obj]; exists {
		info.IsRoot = true
	} else {
		gc.trackedObjects[obj] = &ObjectInfo{
			LastAccess: time.Now(),
			Generation: gc.gcGeneration,
			IsRoot:     true,
		}
		atomic.AddUint64(&gc.stats.TotalTrackedObjects, 1)
	}

	// 添加到根对象列表
	gc.rootObjects = append(gc.rootObjects, obj)
	atomic.AddUint64(&gc.stats.RootObjects, 1)
}

// RemoveRootObject 移除根对象
func (gc *MarkSweepGC) RemoveRootObject(obj *GCObject) {
	if obj == nil {
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	// 从根对象列表移除
	for i, root := range gc.rootObjects {
		if root == obj {
			gc.rootObjects = append(gc.rootObjects[:i], gc.rootObjects[i+1:]...)
			atomic.AddUint64(&gc.stats.RootObjects, ^uint64(0)) // 减1
			break
		}
	}

	// 更新对象信息
	if info, exists := gc.trackedObjects[obj]; exists {
		info.IsRoot = false
	}
}

// ShouldRunGC 检查是否应该运行GC
func (gc *MarkSweepGC) ShouldRunGC() bool {
	// 检查时间间隔
	if time.Since(gc.lastRunTime) < gc.config.GCInterval {
		return false
	}

	// 检查对象数量阈值
	trackedCount := len(gc.trackedObjects)
	return trackedCount >= gc.config.ForceGCThreshold
}

// RunGC 执行标记清除GC
func (gc *MarkSweepGC) RunGC() {
	if gc.isRunning {
		return // 已经在运行
	}

	gc.mutex.Lock()
	gc.isRunning = true
	gc.mutex.Unlock()

	defer func() {
		gc.mutex.Lock()
		gc.isRunning = false
		gc.lastRunTime = time.Now()
		gc.gcGeneration++
		gc.mutex.Unlock()
	}()

	startTime := time.Now()

	// 执行标记阶段
	markStartTime := time.Now()
	gc.markPhase()
	markDuration := time.Since(markStartTime)

	// 执行清除阶段
	sweepStartTime := time.Now()
	collected := gc.sweepPhase()
	sweepDuration := time.Since(sweepStartTime)

	totalDuration := time.Since(startTime)

	// 更新统计
	atomic.AddUint64(&gc.stats.GCCycles, 1)
	atomic.AddUint64(&gc.stats.ObjectsCollected, uint64(collected))
	atomic.AddUint64(&gc.stats.TotalGCTime, uint64(totalDuration.Nanoseconds()))
	atomic.AddUint64(&gc.stats.MarkPhaseTime, uint64(markDuration.Nanoseconds()))
	atomic.AddUint64(&gc.stats.SweepPhaseTime, uint64(sweepDuration.Nanoseconds()))
}

// markPhase 标记阶段 - 标记所有可达对象
func (gc *MarkSweepGC) markPhase() {
	// 清除所有标记
	for obj := range gc.trackedObjects {
		obj.Header.ClearMarked()
	}

	// 从根对象开始标记
	for _, root := range gc.rootObjects {
		if root != nil {
			gc.markObject(root)
		}
	}
}

// markObject 递归标记对象及其引用的对象
func (gc *MarkSweepGC) markObject(obj *GCObject) {
	if obj == nil || obj.Header.IsMarked() {
		return // 已标记或空对象
	}

	// 标记当前对象
	obj.Header.SetMarked()

	// 递归标记子对象
	gc.markChildren(obj)
}

// markChildren 标记子对象
func (gc *MarkSweepGC) markChildren(obj *GCObject) {
	if obj == nil {
		return
	}

	objType := obj.Type()

	// 根据对象类型处理子对象引用
	switch objType {
	case ObjectTypeArray:
		gc.markArrayChildren(obj)
	case ObjectTypeStruct:
		gc.markStructChildren(obj)
	case ObjectTypeClosure:
		gc.markClosureChildren(obj)
	case ObjectTypeString, ObjectTypeFunction:
		// 这些类型没有子对象引用
	default:
		// 未知类型，保守处理
		gc.markGenericChildren(obj)
	}
}

// markArrayChildren 标记数组子对象
func (gc *MarkSweepGC) markArrayChildren(obj *GCObject) {
	// TODO: 根据ArrayObject的具体结构来遍历元素
	// 现在先实现框架，具体的对象布局在后续完善
}

// markStructChildren 标记结构体子对象
func (gc *MarkSweepGC) markStructChildren(obj *GCObject) {
	// TODO: 根据StructObject的字段布局标记子对象引用
}

// markClosureChildren 标记闭包子对象
func (gc *MarkSweepGC) markClosureChildren(obj *GCObject) {
	// TODO: 标记闭包捕获的变量引用
}

// markGenericChildren 通用子对象标记
func (gc *MarkSweepGC) markGenericChildren(obj *GCObject) {
	// 保守策略：扫描对象内存区域，查找可能的GC对象指针
	// 这是一个昂贵操作，只在必要时使用
}

// sweepPhase 清除阶段 - 回收未标记的对象
func (gc *MarkSweepGC) sweepPhase() int {
	collected := 0

	// 遍历所有跟踪的对象
	for obj, info := range gc.trackedObjects {
		if !obj.Header.IsMarked() {
			// 未标记的对象，可以回收
			gc.collectObject(obj)
			delete(gc.trackedObjects, obj)
			collected++
		} else {
			// 标记的对象，更新访问时间
			info.LastAccess = time.Now()
		}
	}

	return collected
}

// collectObject 回收单个对象
func (gc *MarkSweepGC) collectObject(obj *GCObject) {
	if obj == nil {
		return
	}

	// 释放对象内存
	gc.allocator.Deallocate(obj)
}

// ForceGC 强制执行GC
func (gc *MarkSweepGC) ForceGC() {
	gc.RunGC()
}

// GetStats 获取统计信息
func (gc *MarkSweepGC) GetStats() MarkSweepGCStats {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	stats := MarkSweepGCStats{
		GCCycles:            atomic.LoadUint64(&gc.stats.GCCycles),
		ObjectsCollected:    atomic.LoadUint64(&gc.stats.ObjectsCollected),
		CyclesDetected:      atomic.LoadUint64(&gc.stats.CyclesDetected),
		TotalGCTime:         atomic.LoadUint64(&gc.stats.TotalGCTime),
		MarkPhaseTime:       atomic.LoadUint64(&gc.stats.MarkPhaseTime),
		SweepPhaseTime:      atomic.LoadUint64(&gc.stats.SweepPhaseTime),
		TotalTrackedObjects: atomic.LoadUint64(&gc.stats.TotalTrackedObjects),
		LiveObjects:         atomic.LoadUint64(&gc.stats.LiveObjects),
		RootObjects:         atomic.LoadUint64(&gc.stats.RootObjects),
		GCErrors:            atomic.LoadUint64(&gc.stats.GCErrors),
	}

	// 计算平均GC时间
	if stats.GCCycles > 0 {
		stats.AverageGCTime = stats.TotalGCTime / stats.GCCycles
	}

	// 更新存活对象数
	stats.LiveObjects = uint64(len(gc.trackedObjects))

	return stats
}

// IsRunning 检查GC是否正在运行
func (gc *MarkSweepGC) IsRunning() bool {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()
	return gc.isRunning
}

// GetTrackedObjectCount 获取跟踪对象数量
func (gc *MarkSweepGC) GetTrackedObjectCount() int {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()
	return len(gc.trackedObjects)
}

// GetRootObjectCount 获取根对象数量
func (gc *MarkSweepGC) GetRootObjectCount() int {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()
	return len(gc.rootObjects)
}

// Configure 配置标记清除GC
func (gc *MarkSweepGC) Configure(config *MarkSweepGCConfig) {
	if config == nil {
		return
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.config = config
}

// Shutdown 关闭标记清除GC
func (gc *MarkSweepGC) Shutdown() {
	// 执行最后一次GC
	gc.ForceGC()

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	// 清空跟踪对象
	gc.trackedObjects = make(map[*GCObject]*ObjectInfo)
	gc.rootObjects = make([]*GCObject, 0)
}

// GetEfficiency 获取标记清除GC效率
func (gc *MarkSweepGC) GetEfficiency() float64 {
	stats := gc.GetStats()

	if stats.TotalTrackedObjects == 0 {
		return 0.0
	}

	// 计算回收效率
	return float64(stats.ObjectsCollected) / float64(stats.TotalTrackedObjects)
}
