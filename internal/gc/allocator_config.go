package gc

import (
	"fmt"
	"time"
)

// =============================================================================
// 分配器配置系统
// =============================================================================

// AllocatorConfig 分配器配置结构
type AllocatorConfig struct {
	// ========== 基础配置 ==========
	EnableDebug    bool   // 启用调试输出
	VerboseLogging bool   // 详细日志记录
	EnableStats    bool   // 启用统计信息收集
	AllocatorType  string // 分配器类型（"unified", "simple", "slab"）

	// ========== 性能优化配置 ==========
	EnableFastPath  bool // 启用快速路径分配
	BatchSize       int  // 批量操作大小
	EnableSizeClass bool // 启用Size Class优化
	CacheLineAlign  bool // 启用缓存行对齐

	// ========== 内存管理配置 ==========
	DefaultRegionSize     uint32 // 默认内存区域大小
	MaxRegionSize         uint32 // 最大内存区域大小
	SmallObjectThreshold  uint32 // 小对象阈值
	MediumObjectThreshold uint32 // 中等对象阈值
	LargeObjectThreshold  uint32 // 大对象阈值

	// ========== Size Class配置 ==========
	NumSizeClasses      int     // Size Class数量
	SizeClassAlignment  uint32  // Size Class对齐
	SizeClassWasteLimit float64 // Size Class浪费限制

	// ========== 内存对齐配置 ==========
	DefaultAlignment uint32 // 默认对齐
	CacheLineSize    uint32 // 缓存行大小
	PageSize         uint32 // 页面大小
	EnablePageAlign  bool   // 启用页面对齐

	// ========== 内存池配置 ==========
	EnableObjectPool  bool // 启用对象池
	ObjectPoolSize    int  // 对象池大小
	ObjectPoolMaxSize int  // 对象池最大大小
	PoolPrealloc      bool // 预分配池对象

	// ========== 垃圾回收集成 ==========
	EnableGC    bool          // 启用GC集成
	GCThreshold uint64        // GC触发阈值
	GCInterval  time.Duration // GC检查间隔
	AutoCompact bool          // 自动内存压缩

	// ========== 错误处理配置 ==========
	EnableLeakDetection bool // 启用内存泄漏检测
	PanicOnError        bool // 错误时panic
	StrictMode          bool // 严格模式
	MaxRetries          int  // 最大重试次数

	// ========== 性能监控配置 ==========
	EnableProfiling    bool          // 启用性能分析
	ProfilingInterval  time.Duration // 性能分析间隔
	MaxPerformanceLog  int           // 最大性能日志条目
	EnableLatencyStats bool          // 启用延迟统计
}

// DefaultAllocatorConfig 默认分配器配置
var DefaultAllocatorConfig = AllocatorConfig{
	// 基础配置
	EnableDebug:    false,
	VerboseLogging: false,
	EnableStats:    true,
	AllocatorType:  "unified",

	// 性能优化
	EnableFastPath:  true,
	BatchSize:       16,
	EnableSizeClass: true,
	CacheLineAlign:  true,

	// 内存管理
	DefaultRegionSize:     64 * 1024,        // 64KB
	MaxRegionSize:         16 * 1024 * 1024, // 16MB
	SmallObjectThreshold:  256,              // 256字节
	MediumObjectThreshold: 4096,             // 4KB
	LargeObjectThreshold:  65536,            // 64KB

	// Size Class配置
	NumSizeClasses:      8,
	SizeClassAlignment:  16,
	SizeClassWasteLimit: 0.1, // 10%

	// 内存对齐
	DefaultAlignment: 16,
	CacheLineSize:    64,
	PageSize:         4096,
	EnablePageAlign:  true,

	// 内存池
	EnableObjectPool:  true,
	ObjectPoolSize:    1000,
	ObjectPoolMaxSize: 10000,
	PoolPrealloc:      false,

	// GC集成
	EnableGC:    true,
	GCThreshold: 1024 * 1024, // 1MB
	GCInterval:  time.Second,
	AutoCompact: true,

	// 错误处理
	EnableLeakDetection: false,
	PanicOnError:        false,
	StrictMode:          false,
	MaxRetries:          3,

	// 性能监控
	EnableProfiling:    false,
	ProfilingInterval:  time.Minute,
	MaxPerformanceLog:  1000,
	EnableLatencyStats: false,
}

// DebugAllocatorConfig 调试模式分配器配置
var DebugAllocatorConfig = AllocatorConfig{
	// 基础配置
	EnableDebug:    true,
	VerboseLogging: true,
	EnableStats:    true,
	AllocatorType:  "unified",

	// 性能优化
	EnableFastPath:  true,
	BatchSize:       8, // 较小的批量大小用于调试
	EnableSizeClass: true,
	CacheLineAlign:  true,

	// 内存管理
	DefaultRegionSize:     32 * 1024,       // 32KB（较小用于调试）
	MaxRegionSize:         8 * 1024 * 1024, // 8MB
	SmallObjectThreshold:  256,
	MediumObjectThreshold: 4096,
	LargeObjectThreshold:  65536,

	// Size Class配置
	NumSizeClasses:      8,
	SizeClassAlignment:  16,
	SizeClassWasteLimit: 0.05, // 5%（更严格）

	// 内存对齐
	DefaultAlignment: 16,
	CacheLineSize:    64,
	PageSize:         4096,
	EnablePageAlign:  true,

	// 内存池
	EnableObjectPool:  true,
	ObjectPoolSize:    100, // 较小的池大小
	ObjectPoolMaxSize: 1000,
	PoolPrealloc:      false,

	// GC集成
	EnableGC:    true,
	GCThreshold: 512 * 1024, // 512KB（更频繁的GC）
	GCInterval:  time.Second,
	AutoCompact: true,

	// 错误处理
	EnableLeakDetection: true,
	PanicOnError:        false,
	StrictMode:          true,
	MaxRetries:          5,

	// 性能监控
	EnableProfiling:    true,
	ProfilingInterval:  10 * time.Second,
	MaxPerformanceLog:  10000,
	EnableLatencyStats: true,
}

// ProductionAllocatorConfig 生产环境分配器配置
var ProductionAllocatorConfig = AllocatorConfig{
	// 基础配置
	EnableDebug:    false,
	VerboseLogging: false,
	EnableStats:    false, // 生产环境关闭统计以提高性能
	AllocatorType:  "unified",

	// 性能优化
	EnableFastPath:  true,
	BatchSize:       32, // 较大的批量大小
	EnableSizeClass: true,
	CacheLineAlign:  true,

	// 内存管理
	DefaultRegionSize:     128 * 1024,       // 128KB
	MaxRegionSize:         32 * 1024 * 1024, // 32MB
	SmallObjectThreshold:  256,
	MediumObjectThreshold: 4096,
	LargeObjectThreshold:  65536,

	// Size Class配置
	NumSizeClasses:      8,
	SizeClassAlignment:  16,
	SizeClassWasteLimit: 0.15, // 15%（允许更多浪费换取性能）

	// 内存对齐
	DefaultAlignment: 16,
	CacheLineSize:    64,
	PageSize:         4096,
	EnablePageAlign:  true,

	// 内存池
	EnableObjectPool:  true,
	ObjectPoolSize:    5000, // 较大的池大小
	ObjectPoolMaxSize: 50000,
	PoolPrealloc:      true, // 预分配以提高性能

	// GC集成
	EnableGC:    true,
	GCThreshold: 4 * 1024 * 1024, // 4MB
	GCInterval:  5 * time.Second,
	AutoCompact: false, // 关闭自动压缩

	// 错误处理
	EnableLeakDetection: false,
	PanicOnError:        false,
	StrictMode:          false,
	MaxRetries:          1,

	// 性能监控
	EnableProfiling:    false,
	ProfilingInterval:  time.Hour,
	MaxPerformanceLog:  100,
	EnableLatencyStats: false,
}

// =============================================================================
// 配置辅助函数
// =============================================================================

// NewAllocatorConfig 创建新的分配器配置
func NewAllocatorConfig() *AllocatorConfig {
	config := DefaultAllocatorConfig
	return &config
}

// NewDebugAllocatorConfig 创建调试模式分配器配置
func NewDebugAllocatorConfig() *AllocatorConfig {
	config := DebugAllocatorConfig
	return &config
}

// NewProductionAllocatorConfig 创建生产环境分配器配置
func NewProductionAllocatorConfig() *AllocatorConfig {
	config := ProductionAllocatorConfig
	return &config
}

// Clone 克隆配置
func (config *AllocatorConfig) Clone() *AllocatorConfig {
	cloned := *config
	return &cloned
}

// Validate 验证配置有效性
func (config *AllocatorConfig) Validate() error {
	if config.BatchSize <= 0 {
		return fmt.Errorf("批量大小必须大于0")
	}
	if config.DefaultRegionSize < 1024 {
		return fmt.Errorf("默认区域大小必须至少为1024字节")
	}
	if config.MaxRegionSize < config.DefaultRegionSize {
		return fmt.Errorf("最大区域大小必须大于等于默认区域大小")
	}
	if config.SmallObjectThreshold >= config.MediumObjectThreshold {
		return fmt.Errorf("小对象阈值必须小于中等对象阈值")
	}
	if config.MediumObjectThreshold >= config.LargeObjectThreshold {
		return fmt.Errorf("中等对象阈值必须小于大对象阈值")
	}
	if config.NumSizeClasses < 1 || config.NumSizeClasses > 32 {
		return fmt.Errorf("Size Class数量必须在1-32之间")
	}
	if config.SizeClassWasteLimit < 0 || config.SizeClassWasteLimit > 1 {
		return fmt.Errorf("Size Class浪费限制必须在0-1之间")
	}
	if config.DefaultAlignment < 1 || (config.DefaultAlignment&(config.DefaultAlignment-1)) != 0 {
		return fmt.Errorf("默认对齐必须是2的幂")
	}
	return nil
}

// String 返回配置的字符串表示
func (config *AllocatorConfig) String() string {
	return fmt.Sprintf("AllocatorConfig{Type=%s, FastPath=%v, BatchSize=%d, Debug=%v}",
		config.AllocatorType, config.EnableFastPath, config.BatchSize, config.EnableDebug)
}

// SetDebugMode 设置调试模式
func (config *AllocatorConfig) SetDebugMode(enable bool) {
	config.EnableDebug = enable
	config.VerboseLogging = enable
	config.EnableStats = enable
	config.EnableLeakDetection = enable
	config.StrictMode = enable
	config.EnableProfiling = enable
	config.EnableLatencyStats = enable
}

// SetProductionMode 设置生产模式
func (config *AllocatorConfig) SetProductionMode() {
	config.EnableDebug = false
	config.VerboseLogging = false
	config.EnableStats = false
	config.EnableLeakDetection = false
	config.StrictMode = false
	config.EnableProfiling = false
	config.EnableLatencyStats = false
	config.BatchSize = 32
	config.AutoCompact = false
}

// OptimizeForPerformance 优化性能设置
func (config *AllocatorConfig) OptimizeForPerformance() {
	config.EnableFastPath = true
	config.BatchSize = 64
	config.EnableSizeClass = true
	config.CacheLineAlign = true
	config.EnableObjectPool = true
	config.PoolPrealloc = true
	config.AutoCompact = false
}

// OptimizeForMemory 优化内存使用设置
func (config *AllocatorConfig) OptimizeForMemory() {
	config.DefaultRegionSize = 32 * 1024
	config.BatchSize = 8
	config.SizeClassWasteLimit = 0.05
	config.EnableObjectPool = false
	config.AutoCompact = true
	config.GCThreshold = 512 * 1024
}
