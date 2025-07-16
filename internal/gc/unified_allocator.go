package gc

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// =============================================================================
// 统一分配器 - 简化的内存分配系统
// =============================================================================

// UnifiedAllocator 统一分配器 - 替代复杂的多层架构
type UnifiedAllocator struct {
	// 内存区域管理
	regions    []*MemoryRegion
	freeBlocks map[uint32][]*FreeBlock // size -> 空闲块列表

	// Size Class优化（借鉴原设计）
	sizeClasses [NumSizeClasses]*SizeClassAllocator

	// 统计信息（增强版）
	stats UnifiedAllocatorStats

	// 性能优化
	fastPath       map[uint32]*FreeBlock // 快速分配路径
	enableFastPath bool
	batchSize      int

	// 同步控制
	mutex sync.RWMutex

	// 调试选项
	enableDebug bool
}

// MemoryRegion 内存区域 - 大块连续内存
type MemoryRegion struct {
	memory    unsafe.Pointer // 起始地址
	size      uint32         // 总大小
	used      uint32         // 已使用大小
	allocID   uint64         // 分配ID（用于调试）
	alignment uint32         // 对齐方式（缓存行、页等）
}

// FreeBlock 空闲块
type FreeBlock struct {
	ptr       unsafe.Pointer // 块地址
	size      uint32         // 块大小
	next      *FreeBlock     // 下一个空闲块
	allocID   uint64         // 原始分配ID
	sizeClass int            // Size Class（-1表示非Size Class分配）
}

// SizeClassAllocator Size Class专用分配器
type SizeClassAllocator struct {
	sizeClass  int        // Size Class编号
	objectSize uint32     // 对象大小
	freeList   *FreeBlock // 空闲链表
	allocCount uint64     // 分配计数
	wasteBytes uint64     // 浪费字节数
	mutex      sync.Mutex
}

// UnifiedAllocatorStats 增强的统计信息
type UnifiedAllocatorStats struct {
	// 基础统计
	TotalAllocations   uint64
	TotalDeallocations uint64
	BytesAllocated     uint64
	BytesFreed         uint64
	ActiveObjects      uint64
	MemoryRegions      uint64
	LastAllocID        uint64

	// 性能统计（借鉴原设计）
	TotalAllocTime   uint64 // 总分配时间(纳秒)
	AverageAllocTime uint64 // 平均分配时间
	FastPathHits     uint64 // 快速路径命中
	SlowPathCalls    uint64 // 慢速路径调用

	// Size Class统计
	SizeClassStats [NumSizeClasses]EnhancedSizeClassStats

	// 内存对齐统计
	CacheLineAlignedAllocs uint64
	PageAlignedAllocs      uint64
	TotalWasteBytes        uint64

	// 错误统计
	AllocationFailures   uint64
	InvalidDeallocations uint64
}

// EnhancedSizeClassStats 增强的Size Class统计
type EnhancedSizeClassStats struct {
	Allocations    uint64 // 分配次数
	Deallocations  uint64 // 释放次数
	BytesAllocated uint64 // 分配字节数
	BytesFreed     uint64 // 释放字节数
	PagesAllocated uint64 // 分配页数
	WasteBytes     uint64 // 浪费字节数
	FastPathHits   uint64 // 快速路径命中
}

// Size Class配置（借鉴原优化设计，移除重复定义）
// 注意：Size Class常量在 allocator_config.go 中已定义

// SizeClassInfo Size Class信息（借鉴原设计）
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

// SizeClassTable 优化的Size Class配置（借鉴原设计）
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

// NewUnifiedAllocator 创建统一分配器
func NewUnifiedAllocator(enableDebug bool) *UnifiedAllocator {
	ua := &UnifiedAllocator{
		regions:        make([]*MemoryRegion, 0),
		freeBlocks:     make(map[uint32][]*FreeBlock),
		fastPath:       make(map[uint32]*FreeBlock),
		enableFastPath: true,
		batchSize:      16,
		enableDebug:    enableDebug,
	}

	// 初始化Size Class分配器
	for i := 0; i < NumSizeClasses; i++ {
		ua.sizeClasses[i] = &SizeClassAllocator{
			sizeClass:  i,
			objectSize: SizeClassTable[i].Size,
		}
	}

	return ua
}

// =============================================================================
// 核心分配方法（增强版）
// =============================================================================

// Allocate 统一分配入口 - 增强的分配逻辑
func (ua *UnifiedAllocator) Allocate(size uint32, objType ObjectType) *GCObject {
	startTime := time.Now()

	ua.mutex.Lock()
	defer ua.mutex.Unlock()

	// 1. 计算实际需要的大小
	headerSize := uint32(unsafe.Sizeof(GCObjectHeader{}))
	totalSize := headerSize + size

	// 2. 生成唯一的分配ID
	allocID := atomic.AddUint64(&ua.stats.LastAllocID, 1)

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 分配请求: ID=%d, size=%d, objType=%v\n",
			allocID, size, objType)
	}

	var ptr unsafe.Pointer
	var alignedSize uint32
	var usedSizeClass = -1

	// 3. 尝试Size Class分配（优化小对象分配）
	if sizeClass := ua.getSizeClass(totalSize); sizeClass >= 0 {
		usedSizeClass = sizeClass
		alignedSize = SizeClassTable[sizeClass].Size

		if ua.enableDebug {
			fmt.Printf("DEBUG [UnifiedAllocator] 使用Size Class %d: size=%d->%d\n",
				sizeClass, totalSize, alignedSize)
		}

		// 尝试快速路径
		if ua.enableFastPath {
			if fastPtr := ua.tryFastPath(alignedSize, allocID); fastPtr != nil {
				ptr = fastPtr
				atomic.AddUint64(&ua.stats.FastPathHits, 1)
				atomic.AddUint64(&ua.stats.SizeClassStats[sizeClass].FastPathHits, 1)
			}
		}

		// 回退到Size Class慢速路径
		if ptr == nil {
			ptr = ua.allocateSizeClass(sizeClass, allocID)
			atomic.AddUint64(&ua.stats.SlowPathCalls, 1)
		}
	} else {
		// 4. 大对象分配
		alignedSize = ua.alignSizeOptimal(totalSize, objType)

		if ua.enableDebug {
			fmt.Printf("DEBUG [UnifiedAllocator] 大对象分配: size=%d->%d\n",
				totalSize, alignedSize)
		}

		ptr = ua.allocateLargeObject(alignedSize, allocID, objType)
	}

	if ptr == nil {
		atomic.AddUint64(&ua.stats.AllocationFailures, 1)
		if ua.enableDebug {
			fmt.Printf("DEBUG [UnifiedAllocator] 分配失败: ID=%d\n", allocID)
		}
		return nil
	}

	// 5. 初始化GC对象
	obj := ua.initializeGCObject(ptr, objType, size, allocID)

	// 6. 更新统计
	ua.updateStatsEnhanced(alignedSize, usedSizeClass, time.Since(startTime), true)

	return obj
}

// AllocateIsolated 分配独立对象 - 增强版
func (ua *UnifiedAllocator) AllocateIsolated(size uint32, objType ObjectType) *GCObject {
	startTime := time.Now()

	ua.mutex.Lock()
	defer ua.mutex.Unlock()

	// 1. 计算大小
	headerSize := uint32(unsafe.Sizeof(GCObjectHeader{}))
	totalSize := headerSize + size
	alignedSize := ua.alignSizeOptimal(totalSize, objType)

	// 2. 生成分配ID
	allocID := atomic.AddUint64(&ua.stats.LastAllocID, 1)

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 独立分配: ID=%d, size=%d\n", allocID, size)
	}

	// 3. 强制分配新内存区域，完全避免复用
	ptr := ua.allocateNewRegionForced(alignedSize, allocID, objType)
	if ptr == nil {
		atomic.AddUint64(&ua.stats.AllocationFailures, 1)
		if ua.enableDebug {
			fmt.Printf("DEBUG [UnifiedAllocator] 独立分配失败: ID=%d\n", allocID)
		}
		return nil
	}

	obj := ua.initializeGCObject(ptr, objType, size, allocID)
	ua.updateStatsEnhanced(alignedSize, -1, time.Since(startTime), true)

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 独立分配成功: ID=%d, ptr=%p\n", allocID, ptr)
	}

	return obj
}

// =============================================================================
// Size Class优化方法（借鉴原设计）
// =============================================================================

// getSizeClass 获取Size Class（借鉴原算法）
func (ua *UnifiedAllocator) getSizeClass(size uint32) int {
	for i, info := range SizeClassTable {
		if size <= info.Size {
			return i
		}
	}
	return -1 // 超出Size Class范围
}

// tryFastPath 尝试快速路径分配
func (ua *UnifiedAllocator) tryFastPath(size uint32, allocID uint64) unsafe.Pointer {
	if block, exists := ua.fastPath[size]; exists && block != nil {
		// 从快速路径获取
		ua.fastPath[size] = block.next

		if ua.enableDebug {
			fmt.Printf("DEBUG [UnifiedAllocator] 快速路径命中: ID=%d, ptr=%p\n",
				allocID, block.ptr)
		}

		return block.ptr
	}
	return nil
}

// allocateSizeClass 从Size Class分配
func (ua *UnifiedAllocator) allocateSizeClass(sizeClass int, allocID uint64) unsafe.Pointer {
	sca := ua.sizeClasses[sizeClass]
	sca.mutex.Lock()
	defer sca.mutex.Unlock()

	// 从Size Class的空闲链表获取
	if sca.freeList != nil {
		block := sca.freeList
		sca.freeList = block.next
		atomic.AddUint64(&sca.allocCount, 1)

		if ua.enableDebug {
			fmt.Printf("DEBUG [UnifiedAllocator] Size Class %d分配: ptr=%p\n",
				sizeClass, block.ptr)
		}

		return block.ptr
	}

	// 分配新的Size Class内存页
	return ua.allocateNewSizeClassPage(sizeClass, allocID)
}

// allocateNewSizeClassPage 为Size Class分配新页面
func (ua *UnifiedAllocator) allocateNewSizeClassPage(sizeClass int, allocID uint64) unsafe.Pointer {
	info := SizeClassTable[sizeClass]
	pageSize := uint32(4096) // 4KB页面

	memory := allocateAlignedMemory(pageSize)
	if memory == nil {
		return nil
	}

	// 将页面划分为Size Class对象
	objectSize := info.Size
	objectCount := pageSize / objectSize

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 新Size Class页: class=%d, objects=%d\n",
			sizeClass, objectCount)
	}

	// 创建空闲链表
	sca := ua.sizeClasses[sizeClass]
	for i := uint32(1); i < objectCount; i++ {
		blockPtr := unsafe.Pointer(uintptr(memory) + uintptr(i*objectSize))
		block := &FreeBlock{
			ptr:       blockPtr,
			size:      objectSize,
			sizeClass: sizeClass,
			allocID:   allocID,
			next:      sca.freeList,
		}
		sca.freeList = block
	}

	// 返回第一个对象
	return memory
}

// =============================================================================
// 内存对齐优化（借鉴原设计）
// =============================================================================

// alignSizeOptimal 智能内存对齐
func (ua *UnifiedAllocator) alignSizeOptimal(size uint32, objType ObjectType) uint32 {
	// 根据对象类型选择最优对齐策略
	switch objType {
	case ObjectTypeArray, ObjectTypeStruct:
		// 数组和结构体：缓存行对齐，提高访问性能
		aligned := ua.alignCacheLine(size)
		if ua.enableDebug {
			fmt.Printf("DEBUG [UnifiedAllocator] 缓存行对齐: %d->%d\n", size, aligned)
		}
		atomic.AddUint64(&ua.stats.CacheLineAlignedAllocs, 1)
		return aligned

	case ObjectTypeString:
		// 字符串：16字节对齐即可
		return ua.alignSize(size)

	default:
		// 大对象：页对齐，便于内存管理
		if size > 4096 {
			aligned := ua.alignPage(size)
			atomic.AddUint64(&ua.stats.PageAlignedAllocs, 1)
			return aligned
		}
		return ua.alignSize(size)
	}
}

// alignCacheLine 缓存行对齐（64字节）
func (ua *UnifiedAllocator) alignCacheLine(size uint32) uint32 {
	const cacheLineSize = uint32(64)
	return (size + cacheLineSize - 1) & ^(cacheLineSize - 1)
}

// alignPage 页对齐（4KB）
func (ua *UnifiedAllocator) alignPage(size uint32) uint32 {
	const pageSize = uint32(4096)
	return (size + pageSize - 1) & ^(pageSize - 1)
}

// alignSize 基础对齐（16字节）
func (ua *UnifiedAllocator) alignSize(size uint32) uint32 {
	return (size + 15) & ^uint32(15)
}

// =============================================================================
// 大对象分配
// =============================================================================

// allocateLargeObject 分配大对象
func (ua *UnifiedAllocator) allocateLargeObject(size uint32, allocID uint64, objType ObjectType) unsafe.Pointer {
	// 大对象直接分配新区域
	return ua.allocateNewRegionForced(size, allocID, objType)
}

// allocateNewRegionForced 强制分配新区域
func (ua *UnifiedAllocator) allocateNewRegionForced(minSize uint32, allocID uint64, objType ObjectType) unsafe.Pointer {
	regionSize := ua.calculateRegionSize(minSize)

	memory := allocateAlignedMemory(regionSize)
	if memory == nil {
		return nil
	}

	// 根据对象类型选择对齐方式
	var alignment uint32 = 16 // 默认16字节对齐
	if objType == ObjectTypeArray || objType == ObjectTypeStruct {
		alignment = 64 // 缓存行对齐
	}

	region := &MemoryRegion{
		memory:    memory,
		size:      regionSize,
		used:      minSize,
		allocID:   allocID,
		alignment: alignment,
	}

	ua.regions = append(ua.regions, region)
	atomic.AddUint64(&ua.stats.MemoryRegions, 1)

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 强制新区域: ID=%d, ptr=%p, size=%d, align=%d\n",
			allocID, memory, regionSize, alignment)
	}

	return memory
}

// =============================================================================
// 释放方法（增强版）
// =============================================================================

// Deallocate 释放对象
func (ua *UnifiedAllocator) Deallocate(obj *GCObject) {
	if obj == nil {
		return
	}

	startTime := time.Now()
	ua.mutex.Lock()
	defer ua.mutex.Unlock()

	ptr := unsafe.Pointer(obj)
	objSize := obj.Size()
	totalSize := objSize + uint32(unsafe.Sizeof(GCObjectHeader{}))

	// 智能确定Size Class
	var sizeClass = -1
	if sc := ua.getSizeClass(totalSize); sc >= 0 {
		sizeClass = sc
		totalSize = SizeClassTable[sc].Size // 使用实际分配的大小
	} else {
		totalSize = ua.alignSizeOptimal(totalSize, obj.Header.GetObjectType())
	}

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 释放对象: ptr=%p, size=%d, sizeClass=%d\n",
			ptr, totalSize, sizeClass)
	}

	// 添加到合适的空闲列表
	if sizeClass >= 0 {
		// Size Class释放：添加到快速路径
		block := &FreeBlock{
			ptr:       ptr,
			size:      totalSize,
			sizeClass: sizeClass,
			allocID:   atomic.LoadUint64(&ua.stats.LastAllocID),
			next:      ua.fastPath[totalSize],
		}
		ua.fastPath[totalSize] = block

		// 同时添加到Size Class的空闲链表
		sca := ua.sizeClasses[sizeClass]
		sca.mutex.Lock()
		block.next = sca.freeList
		sca.freeList = block
		sca.mutex.Unlock()
	} else {
		// 大对象释放：添加到普通空闲块列表
		block := &FreeBlock{
			ptr:       ptr,
			size:      totalSize,
			sizeClass: -1,
			allocID:   atomic.LoadUint64(&ua.stats.LastAllocID),
		}
		ua.freeBlocks[totalSize] = append(ua.freeBlocks[totalSize], block)
	}

	// 更新统计
	ua.updateStatsEnhanced(totalSize, sizeClass, time.Since(startTime), false)
}

// =============================================================================
// 辅助方法（增强版）
// =============================================================================

// initializeGCObject 初始化GC对象
func (ua *UnifiedAllocator) initializeGCObject(ptr unsafe.Pointer, objType ObjectType, size uint32, allocID uint64) *GCObject {
	// 清零内存
	totalSize := size + uint32(unsafe.Sizeof(GCObjectHeader{}))
	memset(ptr, 0, int(totalSize))

	obj := (*GCObject)(ptr)
	obj.Header = NewGCObjectHeader(objType, size)

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 初始化GC对象: ID=%d, ptr=%p, type=%v, size=%d\n",
			allocID, ptr, objType, size)
	}

	return obj
}

// calculateRegionSize 计算内存区域大小
func (ua *UnifiedAllocator) calculateRegionSize(minSize uint32) uint32 {
	// 默认64KB，或者请求大小的2倍，取较大者
	const defaultRegionSize = 64 * 1024

	if minSize*2 > defaultRegionSize {
		// 对于大对象，分配页对齐的大小
		return ua.alignPage(minSize * 2)
	}

	return defaultRegionSize
}

// =============================================================================
// 增强的统计方法
// =============================================================================

// updateStatsEnhanced 更新增强统计信息
func (ua *UnifiedAllocator) updateStatsEnhanced(size uint32, sizeClass int, duration time.Duration, isAlloc bool) {
	if isAlloc {
		atomic.AddUint64(&ua.stats.TotalAllocations, 1)
		atomic.AddUint64(&ua.stats.BytesAllocated, uint64(size))
		atomic.AddUint64(&ua.stats.ActiveObjects, 1)
		atomic.AddUint64(&ua.stats.TotalAllocTime, uint64(duration.Nanoseconds()))

		// 计算平均分配时间
		totalAllocs := atomic.LoadUint64(&ua.stats.TotalAllocations)
		totalTime := atomic.LoadUint64(&ua.stats.TotalAllocTime)
		if totalAllocs > 0 {
			atomic.StoreUint64(&ua.stats.AverageAllocTime, totalTime/totalAllocs)
		}

		// Size Class统计
		if sizeClass >= 0 && sizeClass < NumSizeClasses {
			atomic.AddUint64(&ua.stats.SizeClassStats[sizeClass].Allocations, 1)
			atomic.AddUint64(&ua.stats.SizeClassStats[sizeClass].BytesAllocated, uint64(size))

			// 计算浪费字节数
			expectedSize := SizeClassTable[sizeClass].Size
			if size < expectedSize {
				wasteBytes := uint64(expectedSize - size)
				atomic.AddUint64(&ua.stats.SizeClassStats[sizeClass].WasteBytes, wasteBytes)
				atomic.AddUint64(&ua.stats.TotalWasteBytes, wasteBytes)
			}
		}
	} else {
		atomic.AddUint64(&ua.stats.TotalDeallocations, 1)
		atomic.AddUint64(&ua.stats.BytesFreed, uint64(size))
		atomic.AddUint64(&ua.stats.ActiveObjects, ^uint64(0)) // 原子减1

		// Size Class统计
		if sizeClass >= 0 && sizeClass < NumSizeClasses {
			atomic.AddUint64(&ua.stats.SizeClassStats[sizeClass].Deallocations, 1)
			atomic.AddUint64(&ua.stats.SizeClassStats[sizeClass].BytesFreed, uint64(size))
		}
	}
}

// GetStatsEnhanced 获取增强统计信息
func (ua *UnifiedAllocator) GetStatsEnhanced() UnifiedAllocatorStats {
	ua.mutex.RLock()
	defer ua.mutex.RUnlock()

	return ua.stats
}

// PrintStatsEnhanced 打印增强统计信息
func (ua *UnifiedAllocator) PrintStatsEnhanced() {
	stats := ua.GetStatsEnhanced()

	fmt.Println("=== 统一分配器增强统计 ===")
	fmt.Printf("总分配次数: %d\n", stats.TotalAllocations)
	fmt.Printf("总释放次数: %d\n", stats.TotalDeallocations)
	fmt.Printf("分配字节数: %d\n", stats.BytesAllocated)
	fmt.Printf("释放字节数: %d\n", stats.BytesFreed)
	fmt.Printf("活跃对象: %d\n", stats.ActiveObjects)
	fmt.Printf("内存区域: %d\n", stats.MemoryRegions)

	// 性能指标
	fmt.Printf("\n=== 性能指标 ===\n")
	if stats.TotalAllocations > 0 {
		avgTime := float64(stats.AverageAllocTime) / 1e6 // 转换为毫秒
		fmt.Printf("平均分配时间: %.3fms\n", avgTime)
	}
	fmt.Printf("快速路径命中: %d\n", stats.FastPathHits)
	fmt.Printf("慢速路径调用: %d\n", stats.SlowPathCalls)
	if stats.FastPathHits+stats.SlowPathCalls > 0 {
		hitRate := float64(stats.FastPathHits) / float64(stats.FastPathHits+stats.SlowPathCalls) * 100
		fmt.Printf("快速路径命中率: %.1f%%\n", hitRate)
	}

	// 内存对齐统计
	fmt.Printf("\n=== 内存对齐统计 ===\n")
	fmt.Printf("缓存行对齐分配: %d\n", stats.CacheLineAlignedAllocs)
	fmt.Printf("页对齐分配: %d\n", stats.PageAlignedAllocs)
	fmt.Printf("总浪费字节数: %d\n", stats.TotalWasteBytes)
	if stats.BytesAllocated > 0 {
		wasteRate := float64(stats.TotalWasteBytes) / float64(stats.BytesAllocated) * 100
		fmt.Printf("内存浪费率: %.2f%%\n", wasteRate)
	}

	// Size Class统计
	fmt.Printf("\n=== Size Class统计 ===\n")
	for i, sc := range stats.SizeClassStats {
		if sc.Allocations > 0 {
			fmt.Printf("Size Class %d (%d字节):\n", i, SizeClassTable[i].Size)
			fmt.Printf("  分配次数: %d\n", sc.Allocations)
			fmt.Printf("  分配字节: %d\n", sc.BytesAllocated)
			fmt.Printf("  浪费字节: %d\n", sc.WasteBytes)
			fmt.Printf("  快速路径: %d\n", sc.FastPathHits)
			if sc.BytesAllocated > 0 {
				wasteRate := float64(sc.WasteBytes) / float64(sc.BytesAllocated) * 100
				fmt.Printf("  浪费率: %.2f%%\n", wasteRate)
			}
		}
	}

	// 错误统计
	if stats.AllocationFailures > 0 || stats.InvalidDeallocations > 0 {
		fmt.Printf("\n=== 错误统计 ===\n")
		fmt.Printf("分配失败: %d\n", stats.AllocationFailures)
		fmt.Printf("无效释放: %d\n", stats.InvalidDeallocations)
	}
}

// =============================================================================
// 统计和管理方法（保留原接口）
// =============================================================================

// GetStats 获取基础统计信息（保持向后兼容）
func (ua *UnifiedAllocator) GetStats() AllocatorStats {
	enhanced := ua.GetStatsEnhanced()

	return AllocatorStats{
		TotalAllocations:   enhanced.TotalAllocations,
		TotalDeallocations: enhanced.TotalDeallocations,
		BytesAllocated:     enhanced.BytesAllocated,
		BytesFreed:         enhanced.BytesFreed,
		ActiveObjects:      enhanced.ActiveObjects,
		MemoryRegions:      enhanced.MemoryRegions,
		LastAllocID:        enhanced.LastAllocID,
	}
}

// PrintStats 打印基础统计信息（保持向后兼容）
func (ua *UnifiedAllocator) PrintStats() {
	stats := ua.GetStats()

	fmt.Println("=== 统一分配器统计 ===")
	fmt.Printf("总分配次数: %d\n", stats.TotalAllocations)
	fmt.Printf("总释放次数: %d\n", stats.TotalDeallocations)
	fmt.Printf("分配字节数: %d\n", stats.BytesAllocated)
	fmt.Printf("释放字节数: %d\n", stats.BytesFreed)
	fmt.Printf("活跃对象: %d\n", stats.ActiveObjects)
	fmt.Printf("内存区域: %d\n", stats.MemoryRegions)
	fmt.Printf("最后分配ID: %d\n", stats.LastAllocID)

	if stats.TotalAllocations > 0 {
		avgSize := stats.BytesAllocated / stats.TotalAllocations
		fmt.Printf("平均分配大小: %d字节\n", avgSize)
	}
}

// CompactMemory 内存压缩（增强实现）
func (ua *UnifiedAllocator) CompactMemory() int {
	ua.mutex.Lock()
	defer ua.mutex.Unlock()

	compacted := 0

	// 清理空的空闲块列表
	for size, blocks := range ua.freeBlocks {
		if len(blocks) == 0 {
			delete(ua.freeBlocks, size)
			compacted++
		}
	}

	// 清理空的快速路径
	for size, block := range ua.fastPath {
		if block == nil {
			delete(ua.fastPath, size)
			compacted++
		}
	}

	// 合并相邻的空闲块（简化实现）
	for size, blocks := range ua.freeBlocks {
		if len(blocks) > 1 {
			// 简单去重：移除重复的地址
			seen := make(map[uintptr]bool)
			unique := make([]*FreeBlock, 0, len(blocks))

			for _, block := range blocks {
				addr := uintptr(block.ptr)
				if !seen[addr] {
					seen[addr] = true
					unique = append(unique, block)
				} else {
					compacted++
				}
			}

			ua.freeBlocks[size] = unique
		}
	}

	return compacted
}

// Destroy 销毁分配器（增强版）
func (ua *UnifiedAllocator) Destroy() {
	ua.mutex.Lock()
	defer ua.mutex.Unlock()

	// 打印最终统计（直接访问stats，避免死锁）
	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 销毁前统计:\n")
		ua.printStatsEnhancedNoLock()
	}

	// 释放所有内存区域
	for _, region := range ua.regions {
		if region.memory != nil {
			freeAlignedMemory(region.memory, region.size)
		}
	}

	// 清理所有数据结构
	ua.regions = nil
	ua.freeBlocks = make(map[uint32][]*FreeBlock)
	ua.fastPath = make(map[uint32]*FreeBlock)

	// 清理Size Class分配器
	for i := range ua.sizeClasses {
		if ua.sizeClasses[i] != nil {
			ua.sizeClasses[i].freeList = nil
			ua.sizeClasses[i].allocCount = 0
		}
	}

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 分配器已销毁\n")
	}
}

// printStatsEnhancedNoLock 打印增强统计信息（不加锁版本，用于已持有锁的情况）
func (ua *UnifiedAllocator) printStatsEnhancedNoLock() {
	stats := ua.stats // 直接访问，不加锁

	fmt.Println("=== 统一分配器增强统计 ===")
	fmt.Printf("总分配次数: %d\n", stats.TotalAllocations)
	fmt.Printf("总释放次数: %d\n", stats.TotalDeallocations)
	fmt.Printf("分配字节数: %d\n", stats.BytesAllocated)
	fmt.Printf("释放字节数: %d\n", stats.BytesFreed)
	fmt.Printf("活跃对象: %d\n", stats.ActiveObjects)
	fmt.Printf("内存区域: %d\n", stats.MemoryRegions)

	// 性能指标
	fmt.Printf("\n=== 性能指标 ===\n")
	if stats.TotalAllocations > 0 {
		avgTime := float64(stats.AverageAllocTime) / 1e6 // 转换为毫秒
		fmt.Printf("平均分配时间: %.3fms\n", avgTime)
	}
	fmt.Printf("快速路径命中: %d\n", stats.FastPathHits)
	fmt.Printf("慢速路径调用: %d\n", stats.SlowPathCalls)
	if stats.FastPathHits+stats.SlowPathCalls > 0 {
		hitRate := float64(stats.FastPathHits) / float64(stats.FastPathHits+stats.SlowPathCalls) * 100
		fmt.Printf("快速路径命中率: %.1f%%\n", hitRate)
	}

	// 内存对齐统计
	fmt.Printf("\n=== 内存对齐统计 ===\n")
	fmt.Printf("缓存行对齐分配: %d\n", stats.CacheLineAlignedAllocs)
	fmt.Printf("页对齐分配: %d\n", stats.PageAlignedAllocs)
	fmt.Printf("总浪费字节数: %d\n", stats.TotalWasteBytes)
	if stats.BytesAllocated > 0 {
		wasteRate := float64(stats.TotalWasteBytes) / float64(stats.BytesAllocated) * 100
		fmt.Printf("内存浪费率: %.2f%%\n", wasteRate)
	}

	// Size Class统计
	fmt.Printf("\n=== Size Class统计 ===\n")
	for i, sc := range stats.SizeClassStats {
		if sc.Allocations > 0 {
			fmt.Printf("Size Class %d (%d字节):\n", i, SizeClassTable[i].Size)
			fmt.Printf("  分配次数: %d\n", sc.Allocations)
			fmt.Printf("  分配字节: %d\n", sc.BytesAllocated)
			fmt.Printf("  浪费字节: %d\n", sc.WasteBytes)
			fmt.Printf("  快速路径: %d\n", sc.FastPathHits)
			if sc.BytesAllocated > 0 {
				wasteRate := float64(sc.WasteBytes) / float64(sc.BytesAllocated) * 100
				fmt.Printf("  浪费率: %.2f%%\n", wasteRate)
			}
		}
	}

	// 错误统计
	if stats.AllocationFailures > 0 || stats.InvalidDeallocations > 0 {
		fmt.Printf("\n=== 错误统计 ===\n")
		fmt.Printf("分配失败: %d\n", stats.AllocationFailures)
		fmt.Printf("无效释放: %d\n", stats.InvalidDeallocations)
	}
}

// =============================================================================
// 兼容性接口 - 保持与现有代码兼容
// =============================================================================

// AllocatorStats 基础统计类型（保持向后兼容）
type AllocatorStats struct {
	TotalAllocations   uint64
	TotalDeallocations uint64
	BytesAllocated     uint64
	BytesFreed         uint64
	ActiveObjects      uint64
	MemoryRegions      uint64
	LastAllocID        uint64
}

// AQLUnifiedAllocator 实现AQLAllocator接口的包装器
type AQLUnifiedAllocator struct {
	*UnifiedAllocator
}

// NewAQLUnifiedAllocator 创建兼容的AQL分配器
func NewAQLUnifiedAllocator(enableDebug bool) *AQLUnifiedAllocator {
	return &AQLUnifiedAllocator{
		UnifiedAllocator: NewUnifiedAllocator(enableDebug),
	}
}

// 实现AQLAllocator接口的所有方法
func (aua *AQLUnifiedAllocator) Allocate(size uint32, objType ObjectType) *GCObject {
	return aua.UnifiedAllocator.Allocate(size, objType)
}

func (aua *AQLUnifiedAllocator) AllocateIsolated(size uint32, objType ObjectType) *GCObject {
	return aua.UnifiedAllocator.AllocateIsolated(size, objType)
}

func (aua *AQLUnifiedAllocator) Deallocate(obj *GCObject) {
	aua.UnifiedAllocator.Deallocate(obj)
}

func (aua *AQLUnifiedAllocator) DeallocateBatch(objects []*GCObject) {
	startTime := time.Now()

	for _, obj := range objects {
		if obj != nil {
			aua.Deallocate(obj)
		}
	}

	if aua.enableDebug {
		fmt.Printf("DEBUG [AQLUnifiedAllocator] 批量释放 %d 个对象，耗时: %v\n",
			len(objects), time.Since(startTime))
	}
}

func (aua *AQLUnifiedAllocator) Stats() *AllocationStats {
	// 转换为旧的统计格式（保持兼容性）
	stats := aua.GetStats()
	return &AllocationStats{
		TotalAllocations:    stats.TotalAllocations,
		TotalDeallocations:  stats.TotalDeallocations,
		TotalBytesAllocated: stats.BytesAllocated,
		TotalBytesFreed:     stats.BytesFreed,
		// 其他字段使用默认值，保持兼容性
	}
}

func (aua *AQLUnifiedAllocator) Configure(config *AllocatorConfig) {
	// 简化实现：从配置中提取有用的设置
	if config != nil {
		if config.EnableFastPath {
			aua.enableFastPath = true
		}
		if config.BatchSize > 0 {
			aua.batchSize = config.BatchSize
		}
		if config.VerboseLogging {
			aua.enableDebug = true
		}

		if aua.enableDebug {
			fmt.Printf("DEBUG [AQLUnifiedAllocator] 配置更新: FastPath=%v, BatchSize=%d, Debug=%v\n",
				aua.enableFastPath, aua.batchSize, aua.enableDebug)
		}
	}
}

func (aua *AQLUnifiedAllocator) CompactMemory() int {
	return aua.UnifiedAllocator.CompactMemory()
}

func (aua *AQLUnifiedAllocator) Destroy() {
	aua.UnifiedAllocator.Destroy()
}

// =============================================================================
// 高级功能扩展
// =============================================================================

// EnableFastPath 启用/禁用快速路径
func (ua *UnifiedAllocator) EnableFastPath(enable bool) {
	ua.mutex.Lock()
	defer ua.mutex.Unlock()

	ua.enableFastPath = enable

	if ua.enableDebug {
		fmt.Printf("DEBUG [UnifiedAllocator] 快速路径: %v\n", enable)
	}
}

// SetBatchSize 设置批量操作大小
func (ua *UnifiedAllocator) SetBatchSize(size int) {
	if size > 0 && size <= 1024 {
		ua.batchSize = size

		if ua.enableDebug {
			fmt.Printf("DEBUG [UnifiedAllocator] 批量大小设置为: %d\n", size)
		}
	}
}

// GetMemoryUsage 获取内存使用情况
func (ua *UnifiedAllocator) GetMemoryUsage() (allocated, freed, live uint64) {
	stats := ua.GetStatsEnhanced()
	return stats.BytesAllocated, stats.BytesFreed, stats.BytesAllocated - stats.BytesFreed
}

// GetPerformanceMetrics 获取性能指标
func (ua *UnifiedAllocator) GetPerformanceMetrics() (avgAllocTime float64, fastPathHitRate float64, wasteRate float64) {
	stats := ua.GetStatsEnhanced()

	// 平均分配时间（毫秒）
	if stats.TotalAllocations > 0 {
		avgAllocTime = float64(stats.AverageAllocTime) / 1e6
	}

	// 快速路径命中率
	totalRequests := stats.FastPathHits + stats.SlowPathCalls
	if totalRequests > 0 {
		fastPathHitRate = float64(stats.FastPathHits) / float64(totalRequests) * 100
	}

	// 内存浪费率
	if stats.BytesAllocated > 0 {
		wasteRate = float64(stats.TotalWasteBytes) / float64(stats.BytesAllocated) * 100
	}

	return
}
