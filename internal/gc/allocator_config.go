package gc

// AllocatorConfig 分配器配置
type AllocatorConfig struct {
	// Size Class配置
	EnableSmallAllocator bool   // 启用小对象分配器
	SmallObjectThreshold uint32 // 小对象阈值 (默认256字节)

	// Slab配置
	EnableSlabAllocator   bool   // 启用slab分配器
	SlabChunkSize         uint32 // slab chunk大小 (默认64KB)
	MediumObjectThreshold uint32 // 中等对象阈值 (默认4KB)

	// 大对象配置
	EnableDirectAllocator bool   // 启用直接分配器
	LargeObjectThreshold  uint32 // 大对象阈值 (默认4KB)

	// 性能优化
	EnableFastPath        bool // 启用快速分配路径
	EnableBatchAllocation bool // 启用批量分配
	BatchSize             int  // 批量分配大小

	// 调试选项
	EnableTracking      bool // 启用分配追踪
	EnableLeakDetection bool // 启用内存泄漏检测
	VerboseLogging      bool // 详细日志
}

// DefaultAllocatorConfig 默认分配器配置
var DefaultAllocatorConfig = AllocatorConfig{
	EnableSmallAllocator: true,
	SmallObjectThreshold: 256,

	EnableSlabAllocator:   true,
	SlabChunkSize:         64 * 1024, // 64KB
	MediumObjectThreshold: 4 * 1024,  // 4KB

	EnableDirectAllocator: true,
	LargeObjectThreshold:  4 * 1024, // 4KB

	EnableFastPath:        true,
	EnableBatchAllocation: false, // MVP阶段暂时关闭
	BatchSize:             16,

	EnableTracking:      false,
	EnableLeakDetection: false,
	VerboseLogging:      false,
}

// SizeClassInfo Size Class信息
type SizeClassInfo struct {
	Size           uint32  // 对象大小
	ObjectsPerPage int     // 每页对象数量
	WasteRatio     float64 // 内存浪费比例
}

// AQL Size Classes - 8个优化的size class
const (
	SizeClass16  = 0 // 16字节  - GCObjectHeader only
	SizeClass32  = 1 // 32字节  - 小数据 + header
	SizeClass48  = 2 // 48字节  - StringObject, ArrayObject (缓存友好)
	SizeClass64  = 3 // 64字节  - SmallObject (正好1个缓存行)
	SizeClass96  = 4 // 96字节  - 中等对象
	SizeClass128 = 5 // 128字节 - 大一点的对象 (2个缓存行)
	SizeClass192 = 6 // 192字节 - 更大对象 (3个缓存行)
	SizeClass256 = 7 // 256字节 - 小对象上限 (4个缓存行)

	NumSizeClasses = 8 // Size Class总数
)

// SizeClassTable Size Class配置表
var SizeClassTable = [NumSizeClasses]SizeClassInfo{
	{Size: 16, ObjectsPerPage: 256, WasteRatio: 0.00}, // 完美匹配
	{Size: 32, ObjectsPerPage: 128, WasteRatio: 0.00}, // 完美匹配
	{Size: 48, ObjectsPerPage: 85, WasteRatio: 0.02},  // 很少浪费
	{Size: 64, ObjectsPerPage: 64, WasteRatio: 0.00},  // 缓存行对齐
	{Size: 96, ObjectsPerPage: 42, WasteRatio: 0.03},  // 可接受
	{Size: 128, ObjectsPerPage: 32, WasteRatio: 0.00}, // 2缓存行
	{Size: 192, ObjectsPerPage: 21, WasteRatio: 0.05}, // 可接受
	{Size: 256, ObjectsPerPage: 16, WasteRatio: 0.00}, // 4缓存行
}

// GetSizeClass 根据大小获取对应的size class
func GetSizeClass(size uint32) int {
	for i, info := range SizeClassTable {
		if size <= info.Size {
			return i
		}
	}
	return -1 // 超出小对象范围
}

// AlignPage 页对齐 (4KB边界)
func AlignPage(size uint32) uint32 {
	const pageSize = 4096
	return (size + pageSize - 1) &^ (pageSize - 1)
}

// AlignCacheLine 缓存行对齐 (64字节边界)
func AlignCacheLine(size uint32) uint32 {
	const cacheLineSize = 64
	return (size + cacheLineSize - 1) &^ (cacheLineSize - 1)
}
