package gc

import (
	"fmt"
	"time"
)

// =============================================================================
// GC配置系统
// =============================================================================

// GCConfig GC配置参数结构
type GCConfig struct {
	// ========== 引用计数GC配置 ==========
	RefCountEnabled   bool // 是否启用引用计数GC
	RefCountBatchSize int  // 批量处理大小
	RefCountQueueSize int  // 零引用对象队列大小

	// ========== 标记清除GC配置 ==========
	MarkSweepEnabled   bool   // 是否启用标记清除GC
	MarkSweepWorkers   int    // 标记清除工作线程数
	MarkSweepThreshold uint64 // 触发标记清除的阈值（字节）

	// ========== 性能调优参数 ==========
	MaxPauseTime    time.Duration // 最大暂停时间目标
	GCPercentage    int           // 堆增长百分比触发GC
	AllocatorType   string        // 分配器类型
	InitialHeapSize uint64        // 初始堆大小
	MaxHeapSize     uint64        // 最大堆大小

	// ========== 内存分配配置 ==========
	SmallObjectThreshold  uint32 // 小对象阈值
	MediumObjectThreshold uint32 // 中等对象阈值
	LargeObjectThreshold  uint32 // 大对象阈值

	// ========== 缓存和池配置 ==========
	ThreadLocalCacheSize int  // 线程本地缓存大小
	ObjectPoolEnabled    bool // 是否启用对象池
	ObjectPoolMaxSize    int  // 对象池最大大小

	// ========== 调试和监控选项 ==========
	EnableLeakDetection  bool // 启用内存泄漏检测
	EnableGCTracing      bool // 启用GC追踪
	EnablePerformanceLog bool // 启用性能日志
	EnableStats          bool // 启用统计信息收集
	VerboseLogging       bool // 详细日志记录

	// ========== 增量GC配置 ==========
	IncrementalMarkingEnabled bool          // 启用增量标记
	IncrementalTimeSlice      time.Duration // 增量执行时间片
	IncrementalStepSize       int           // 增量步长

	// ========== 并发控制 ==========
	ConcurrentMarking  bool // 并发标记
	ConcurrentSweeping bool // 并发清扫
	MaxConcurrentGCs   int  // 最大并发GC数量

	// ========== 写屏障配置 ==========
	WriteBarrierEnabled bool   // 启用写屏障
	WriteBarrierType    string // 写屏障类型

	// ========== 内存压缩配置 ==========
	CompactionEnabled   bool    // 启用内存压缩
	CompactionThreshold float64 // 压缩触发阈值
}

// =============================================================================
// 预定义配置
// =============================================================================

// DefaultGCConfig 默认GC配置
var DefaultGCConfig = GCConfig{
	// 引用计数GC
	RefCountEnabled:   true,
	RefCountBatchSize: 100,
	RefCountQueueSize: 1000,

	// 标记清除GC
	MarkSweepEnabled:   true,
	MarkSweepWorkers:   2,
	MarkSweepThreshold: 64 * 1024 * 1024, // 64MB

	// 性能调优
	MaxPauseTime:    1 * time.Millisecond,
	GCPercentage:    200, // 堆增长200%时触发GC
	AllocatorType:   "slab",
	InitialHeapSize: 16 * 1024 * 1024,   // 16MB
	MaxHeapSize:     1024 * 1024 * 1024, // 1GB

	// 内存分配
	SmallObjectThreshold:  256,
	MediumObjectThreshold: 4096,
	LargeObjectThreshold:  65536,

	// 缓存和池
	ThreadLocalCacheSize: 1024 * 1024, // 1MB
	ObjectPoolEnabled:    true,
	ObjectPoolMaxSize:    10000,

	// 调试和监控
	EnableLeakDetection:  false,
	EnableGCTracing:      false,
	EnablePerformanceLog: false,
	EnableStats:          true,
	VerboseLogging:       false,

	// 增量GC
	IncrementalMarkingEnabled: true,
	IncrementalTimeSlice:      100 * time.Microsecond,
	IncrementalStepSize:       1000,

	// 并发控制
	ConcurrentMarking:  true,
	ConcurrentSweeping: true,
	MaxConcurrentGCs:   1,

	// 写屏障
	WriteBarrierEnabled: true,
	WriteBarrierType:    "dijkstra",

	// 内存压缩
	CompactionEnabled:   false,
	CompactionThreshold: 0.3,
}

// DevelopmentGCConfig 开发环境GC配置（更多调试信息）
var DevelopmentGCConfig = GCConfig{
	// 继承默认配置
	RefCountEnabled:   true,
	RefCountBatchSize: 50, // 更小的批次，便于调试
	RefCountQueueSize: 500,

	MarkSweepEnabled:   true,
	MarkSweepWorkers:   1,                // 单线程便于调试
	MarkSweepThreshold: 32 * 1024 * 1024, // 32MB，更频繁的GC

	MaxPauseTime:    2 * time.Millisecond, // 允许更长的暂停
	GCPercentage:    100,                  // 更频繁的GC
	AllocatorType:   "slab",
	InitialHeapSize: 8 * 1024 * 1024,   // 8MB
	MaxHeapSize:     256 * 1024 * 1024, // 256MB

	SmallObjectThreshold:  256,
	MediumObjectThreshold: 4096,
	LargeObjectThreshold:  32768,

	ThreadLocalCacheSize: 512 * 1024, // 512KB
	ObjectPoolEnabled:    true,
	ObjectPoolMaxSize:    5000,

	// 开启所有调试功能
	EnableLeakDetection:  true,
	EnableGCTracing:      true,
	EnablePerformanceLog: true,
	EnableStats:          true,
	VerboseLogging:       true,

	IncrementalMarkingEnabled: true,
	IncrementalTimeSlice:      50 * time.Microsecond,
	IncrementalStepSize:       500,

	ConcurrentMarking:  false, // 关闭并发，便于调试
	ConcurrentSweeping: false,
	MaxConcurrentGCs:   1,

	WriteBarrierEnabled: true,
	WriteBarrierType:    "dijkstra",

	CompactionEnabled:   false,
	CompactionThreshold: 0.5,
}

// ProductionGCConfig 生产环境GC配置（性能优化）
var ProductionGCConfig = GCConfig{
	RefCountEnabled:   true,
	RefCountBatchSize: 200, // 更大的批次，提高性能
	RefCountQueueSize: 2000,

	MarkSweepEnabled:   true,
	MarkSweepWorkers:   4,                 // 更多工作线程
	MarkSweepThreshold: 128 * 1024 * 1024, // 128MB，减少GC频率

	MaxPauseTime:    500 * time.Microsecond, // 更严格的暂停时间
	GCPercentage:    300,                    // 减少GC频率
	AllocatorType:   "slab",
	InitialHeapSize: 64 * 1024 * 1024,   // 64MB
	MaxHeapSize:     4096 * 1024 * 1024, // 4GB

	SmallObjectThreshold:  256,
	MediumObjectThreshold: 4096,
	LargeObjectThreshold:  131072, // 128KB

	ThreadLocalCacheSize: 2 * 1024 * 1024, // 2MB
	ObjectPoolEnabled:    true,
	ObjectPoolMaxSize:    20000,

	// 关闭调试功能，只保留基本统计
	EnableLeakDetection:  false,
	EnableGCTracing:      false,
	EnablePerformanceLog: false,
	EnableStats:          true,
	VerboseLogging:       false,

	IncrementalMarkingEnabled: true,
	IncrementalTimeSlice:      200 * time.Microsecond,
	IncrementalStepSize:       2000,

	ConcurrentMarking:  true,
	ConcurrentSweeping: true,
	MaxConcurrentGCs:   2,

	WriteBarrierEnabled: true,
	WriteBarrierType:    "dijkstra",

	CompactionEnabled:   true,
	CompactionThreshold: 0.2, // 更激进的压缩
}

// =============================================================================
// 配置操作方法
// =============================================================================

// NewGCConfig 创建新的GC配置
func NewGCConfig() *GCConfig {
	config := DefaultGCConfig
	return &config
}

// NewDevelopmentGCConfig 创建开发环境GC配置
func NewDevelopmentGCConfig() *GCConfig {
	config := DevelopmentGCConfig
	return &config
}

// NewProductionGCConfig 创建生产环境GC配置
func NewProductionGCConfig() *GCConfig {
	config := ProductionGCConfig
	return &config
}

// Validate 验证配置参数的有效性
func (c *GCConfig) Validate() error {
	if c.RefCountBatchSize <= 0 {
		return fmt.Errorf("RefCountBatchSize must be positive, got %d", c.RefCountBatchSize)
	}

	if c.RefCountQueueSize <= 0 {
		return fmt.Errorf("RefCountQueueSize must be positive, got %d", c.RefCountQueueSize)
	}

	if c.MarkSweepWorkers <= 0 {
		return fmt.Errorf("MarkSweepWorkers must be positive, got %d", c.MarkSweepWorkers)
	}

	if c.MarkSweepThreshold == 0 {
		return fmt.Errorf("MarkSweepThreshold must be positive, got %d", c.MarkSweepThreshold)
	}

	if c.GCPercentage <= 0 {
		return fmt.Errorf("GCPercentage must be positive, got %d", c.GCPercentage)
	}

	if c.InitialHeapSize == 0 {
		return fmt.Errorf("InitialHeapSize must be positive, got %d", c.InitialHeapSize)
	}

	if c.MaxHeapSize < c.InitialHeapSize {
		return fmt.Errorf("MaxHeapSize (%d) must be >= InitialHeapSize (%d)",
			c.MaxHeapSize, c.InitialHeapSize)
	}

	if c.SmallObjectThreshold >= c.MediumObjectThreshold {
		return fmt.Errorf("SmallObjectThreshold (%d) must be < MediumObjectThreshold (%d)",
			c.SmallObjectThreshold, c.MediumObjectThreshold)
	}

	if c.MediumObjectThreshold >= c.LargeObjectThreshold {
		return fmt.Errorf("MediumObjectThreshold (%d) must be < LargeObjectThreshold (%d)",
			c.MediumObjectThreshold, c.LargeObjectThreshold)
	}

	if c.ThreadLocalCacheSize < 0 {
		return fmt.Errorf("ThreadLocalCacheSize must be non-negative, got %d", c.ThreadLocalCacheSize)
	}

	if c.ObjectPoolMaxSize < 0 {
		return fmt.Errorf("ObjectPoolMaxSize must be non-negative, got %d", c.ObjectPoolMaxSize)
	}

	if c.IncrementalStepSize <= 0 {
		return fmt.Errorf("IncrementalStepSize must be positive, got %d", c.IncrementalStepSize)
	}

	if c.MaxConcurrentGCs <= 0 {
		return fmt.Errorf("MaxConcurrentGCs must be positive, got %d", c.MaxConcurrentGCs)
	}

	if c.CompactionThreshold < 0 || c.CompactionThreshold > 1 {
		return fmt.Errorf("CompactionThreshold must be between 0 and 1, got %f", c.CompactionThreshold)
	}

	// 验证分配器类型
	switch c.AllocatorType {
	case "slab", "buddy", "tcmalloc", "simple":
		// 有效的分配器类型
	default:
		return fmt.Errorf("invalid AllocatorType: %s", c.AllocatorType)
	}

	// 验证写屏障类型
	switch c.WriteBarrierType {
	case "dijkstra", "yuasa", "hybrid", "none":
		// 有效的写屏障类型
	default:
		return fmt.Errorf("invalid WriteBarrierType: %s", c.WriteBarrierType)
	}

	return nil
}

// Clone 创建配置的深拷贝
func (c *GCConfig) Clone() *GCConfig {
	clone := *c
	return &clone
}

// IsDebugMode 检查是否处于调试模式
func (c *GCConfig) IsDebugMode() bool {
	return c.EnableLeakDetection || c.EnableGCTracing || c.VerboseLogging
}

// IsPerformanceMode 检查是否处于性能模式
func (c *GCConfig) IsPerformanceMode() bool {
	return !c.IsDebugMode() && c.ConcurrentMarking && c.ConcurrentSweeping
}

// GetEffectiveWorkerCount 获取有效的工作线程数
func (c *GCConfig) GetEffectiveWorkerCount() int {
	if c.IsDebugMode() {
		// 调试模式使用单线程
		return 1
	}
	return c.MarkSweepWorkers
}

// =============================================================================
// 配置调整方法
// =============================================================================

// AdjustForMemoryPressure 根据内存压力调整配置
func (c *GCConfig) AdjustForMemoryPressure(pressure float64) {
	if pressure > 0.8 {
		// 高内存压力：更激进的GC
		c.GCPercentage = 100
		c.MarkSweepThreshold = c.MarkSweepThreshold / 2
		c.IncrementalTimeSlice = c.IncrementalTimeSlice / 2
	} else if pressure < 0.2 {
		// 低内存压力：放松GC
		c.GCPercentage = 400
		c.MarkSweepThreshold = c.MarkSweepThreshold * 2
		c.IncrementalTimeSlice = c.IncrementalTimeSlice * 2
	}
}

// AdjustForLatencySensitivity 根据延迟敏感性调整配置
func (c *GCConfig) AdjustForLatencySensitivity(sensitive bool) {
	if sensitive {
		// 延迟敏感：更小的时间片，更多并发
		c.MaxPauseTime = 100 * time.Microsecond
		c.IncrementalTimeSlice = 50 * time.Microsecond
		c.IncrementalStepSize = 500
		c.ConcurrentMarking = true
		c.ConcurrentSweeping = true
	} else {
		// 延迟不敏感：可以有更长的暂停
		c.MaxPauseTime = 5 * time.Millisecond
		c.IncrementalTimeSlice = 500 * time.Microsecond
		c.IncrementalStepSize = 5000
	}
}

// AdjustForThroughput 根据吞吐量要求调整配置
func (c *GCConfig) AdjustForThroughput(highThroughput bool) {
	if highThroughput {
		// 高吞吐量：减少GC频率，增大批次
		c.GCPercentage = 400
		c.RefCountBatchSize = 500
		c.MarkSweepThreshold = c.MarkSweepThreshold * 2
		c.ThreadLocalCacheSize = c.ThreadLocalCacheSize * 2
	} else {
		// 普通吞吐量：平衡设置
		c.GCPercentage = 200
		c.RefCountBatchSize = 100
	}
}
