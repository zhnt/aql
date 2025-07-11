package gc

import (
	"sync"
	"time"
	"unsafe"
)

// AQLAllocator AQL分配器主接口 - 借鉴Lua的简洁性
type AQLAllocator interface {
	// 分配GC对象
	Allocate(size uint32, objType ObjectType) *GCObject

	// 释放GC对象
	Deallocate(obj *GCObject)

	// 批量释放
	DeallocateBatch(objects []*GCObject)

	// 获取分配统计
	Stats() *AllocationStats

	// 配置调优
	Configure(config *AllocatorConfig)

	// 内存压缩
	CompactMemory() int

	// 销毁分配器
	Destroy()
}

// SimpleAllocator 简单分配器实现 - MVP版本
type SimpleAllocator struct {
	// === 三级分配策略 ===
	small  [NumSizeClasses]*FixedSizeAllocator // 小对象：8个size class
	medium *SlabAllocator                      // 中等对象：slab分配
	large  *DirectAllocator                    // 大对象：直接分配

	// === 配置与统计 ===
	config *AllocatorConfig // 分配器配置
	stats  *AllocationStats // 分配统计

	// === 同步控制 ===
	mutex sync.RWMutex // 保护分配器状态

	// === GC集成相关 (预留接口) ===
	gcManager interface{} // GC管理器接口，暂时用interface{}
}

// NewSimpleAllocator 创建简单分配器
func NewSimpleAllocator(config *AllocatorConfig) *SimpleAllocator {
	if config == nil {
		config = &DefaultAllocatorConfig
	}

	sa := &SimpleAllocator{
		config: config,
		stats:  &AllocationStats{},
	}

	// 初始化小对象分配器
	if config.EnableSmallAllocator {
		for i := 0; i < NumSizeClasses; i++ {
			sa.small[i] = NewFixedSizeAllocator(i, &sa.stats.SmallAllocStats[i])
		}
	}

	// 初始化Slab分配器
	if config.EnableSlabAllocator {
		sa.medium = NewSlabAllocator(config.SlabChunkSize, &sa.stats.MediumAllocStats)
	}

	// 初始化直接分配器
	if config.EnableDirectAllocator {
		sa.large = NewDirectAllocator(&sa.stats.LargeAllocStats)
	}

	return sa
}

// Allocate 统一分配入口 - 与GC深度集成
func (sa *SimpleAllocator) Allocate(size uint32, objType ObjectType) *GCObject {
	startTime := time.Now()

	// 1. 计算总大小 (对象大小 + GC头)
	headerSize := uint32(unsafe.Sizeof(GCObjectHeader{}))
	totalSize := headerSize + size
	alignedSize := Align16(totalSize)

	var ptr unsafe.Pointer

	// 2. 选择合适的分配策略
	switch {
	case alignedSize <= sa.config.SmallObjectThreshold:
		ptr = sa.allocateSmall(alignedSize)
	case alignedSize <= sa.config.MediumObjectThreshold:
		ptr = sa.allocateMedium(alignedSize)
	default:
		ptr = sa.allocateLarge(alignedSize, objType)
	}

	if ptr == nil {
		// 分配失败
		sa.stats.IncAllocationFailure()
		return nil
	}

	// 3. 初始化GC对象头
	obj := (*GCObject)(ptr)
	obj.Header = NewGCObjectHeader(objType, size)

	// 4. 设置GC相关标志
	if sa.mightHaveCycles(objType) {
		obj.Header.SetCyclic()
	}

	// 5. 更新统计信息
	sa.stats.IncAllocation(alignedSize)
	sa.stats.IncAllocationTime(time.Since(startTime))

	return obj
}

// allocateSmall 分配小对象
func (sa *SimpleAllocator) allocateSmall(size uint32) unsafe.Pointer {
	if !sa.config.EnableSmallAllocator {
		return nil
	}

	sizeClass := GetSizeClass(size)
	if sizeClass < 0 || sizeClass >= NumSizeClasses {
		return nil
	}

	if sa.config.EnableFastPath {
		// 快速路径
		if ptr := sa.small[sizeClass].Allocate(); ptr != nil {
			return ptr
		}
	}

	// 慢速路径
	return sa.small[sizeClass].Allocate()
}

// allocateMedium 分配中等对象
func (sa *SimpleAllocator) allocateMedium(size uint32) unsafe.Pointer {
	if !sa.config.EnableSlabAllocator || sa.medium == nil {
		return nil
	}

	return sa.medium.Allocate(size)
}

// allocateLarge 分配大对象
func (sa *SimpleAllocator) allocateLarge(size uint32, objType ObjectType) unsafe.Pointer {
	if !sa.config.EnableDirectAllocator || sa.large == nil {
		return nil
	}

	return sa.large.Allocate(size, objType)
}

// Deallocate 统一释放入口
func (sa *SimpleAllocator) Deallocate(obj *GCObject) {
	if obj == nil {
		return
	}

	// 1. 获取对象信息
	size := obj.Size() + uint32(unsafe.Sizeof(GCObjectHeader{}))
	ptr := unsafe.Pointer(obj)

	// 2. 更新统计信息
	sa.stats.IncDeallocation(size)

	// 3. 清理GC状态
	obj.Header.ClearMarked()

	// 4. 根据大小选择释放策略
	switch {
	case size <= sa.config.SmallObjectThreshold:
		sa.deallocateSmall(ptr, size)
	case size <= sa.config.MediumObjectThreshold:
		sa.deallocateMedium(ptr, size)
	default:
		sa.deallocateLarge(ptr)
	}
}

// deallocateSmall 释放小对象
func (sa *SimpleAllocator) deallocateSmall(ptr unsafe.Pointer, size uint32) {
	if !sa.config.EnableSmallAllocator {
		sa.stats.IncInvalidDeallocation()
		return
	}

	sizeClass := GetSizeClass(size)
	if sizeClass < 0 || sizeClass >= NumSizeClasses {
		sa.stats.IncInvalidDeallocation()
		return
	}

	sa.small[sizeClass].Deallocate(ptr)
}

// deallocateMedium 释放中等对象
func (sa *SimpleAllocator) deallocateMedium(ptr unsafe.Pointer, size uint32) {
	if !sa.config.EnableSlabAllocator || sa.medium == nil {
		sa.stats.IncInvalidDeallocation()
		return
	}

	sa.medium.Deallocate(ptr, size)
}

// deallocateLarge 释放大对象
func (sa *SimpleAllocator) deallocateLarge(ptr unsafe.Pointer) {
	if !sa.config.EnableDirectAllocator || sa.large == nil {
		sa.stats.IncInvalidDeallocation()
		return
	}

	sa.large.Deallocate(ptr)
}

// DeallocateBatch 批量释放对象
func (sa *SimpleAllocator) DeallocateBatch(objects []*GCObject) {
	if len(objects) == 0 {
		return
	}

	if !sa.config.EnableBatchAllocation {
		// 逐个释放
		for _, obj := range objects {
			sa.Deallocate(obj)
		}
		return
	}

	// 按size class分组
	smallGroups := make(map[int][]unsafe.Pointer)

	for _, obj := range objects {
		if obj == nil {
			continue
		}

		size := obj.Size() + uint32(unsafe.Sizeof(GCObjectHeader{}))
		ptr := unsafe.Pointer(obj)

		// 更新统计
		sa.stats.IncDeallocation(size)

		// 清理GC状态
		obj.Header.ClearMarked()

		if size <= sa.config.SmallObjectThreshold {
			sizeClass := GetSizeClass(size)
			if sizeClass >= 0 && sizeClass < NumSizeClasses {
				smallGroups[sizeClass] = append(smallGroups[sizeClass], ptr)
			}
		} else {
			// 中等和大对象单独处理
			if size <= sa.config.MediumObjectThreshold {
				sa.deallocateMedium(ptr, size)
			} else {
				sa.deallocateLarge(ptr)
			}
		}
	}

	// 批量释放小对象
	for sizeClass, ptrs := range smallGroups {
		if sa.config.EnableSmallAllocator && sizeClass < NumSizeClasses {
			sa.small[sizeClass].DeallocateBatch(ptrs)
		}
	}
}

// Stats 获取分配统计
func (sa *SimpleAllocator) Stats() *AllocationStats {
	sa.mutex.RLock()
	defer sa.mutex.RUnlock()

	// 返回统计信息的副本
	stats := *sa.stats

	// 更新实时计算的字段
	stats.AverageAllocSize = stats.GetAverageAllocSize()

	return &stats
}

// Configure 配置调优
func (sa *SimpleAllocator) Configure(config *AllocatorConfig) {
	if config == nil {
		return
	}

	sa.mutex.Lock()
	defer sa.mutex.Unlock()

	sa.config = config

	// 根据配置调整分配器行为
	// 这里可以添加动态调整逻辑
}

// CompactMemory 内存压缩
func (sa *SimpleAllocator) CompactMemory() int {
	sa.mutex.Lock()
	defer sa.mutex.Unlock()

	compactedCount := 0

	// 压缩小对象分配器
	if sa.config.EnableSmallAllocator {
		for i := 0; i < NumSizeClasses; i++ {
			if sa.small[i] != nil {
				compactedCount += sa.small[i].CompactMemory()
			}
		}
	}

	// 压缩Slab分配器
	if sa.config.EnableSlabAllocator && sa.medium != nil {
		compactedCount += sa.medium.CompactMemory()
	}

	// 压缩直接分配器 (通常不需要)
	if sa.config.EnableDirectAllocator && sa.large != nil {
		compactedCount += sa.large.CompactMemory()
	}

	return compactedCount
}

// Destroy 销毁分配器
func (sa *SimpleAllocator) Destroy() {
	sa.mutex.Lock()
	defer sa.mutex.Unlock()

	// 销毁小对象分配器
	for i := 0; i < NumSizeClasses; i++ {
		if sa.small[i] != nil {
			sa.small[i].Destroy()
			sa.small[i] = nil
		}
	}

	// 销毁Slab分配器
	if sa.medium != nil {
		sa.medium.Destroy()
		sa.medium = nil
	}

	// 销毁直接分配器
	if sa.large != nil {
		sa.large.Destroy()
		sa.large = nil
	}

	// 重置统计
	sa.stats.Reset()
}

// mightHaveCycles 判断对象类型是否可能有循环引用
func (sa *SimpleAllocator) mightHaveCycles(objType ObjectType) bool {
	switch objType {
	case ObjectTypeArray, ObjectTypeStruct, ObjectTypeClosure:
		return true // 这些类型可能包含对其他对象的引用
	case ObjectTypeString, ObjectTypeFunction:
		return false // 这些类型通常不含循环引用
	default:
		return true // 保守估计
	}
}

// GetMemoryUsage 获取内存使用情况
func (sa *SimpleAllocator) GetMemoryUsage() (allocated, freed, live uint64) {
	stats := sa.Stats()
	allocated = stats.TotalBytesAllocated
	freed = stats.TotalBytesFreed
	live = stats.GetLiveBytes()
	return
}

// IsValidPointer 检查指针是否是有效的分配器指针
func (sa *SimpleAllocator) IsValidPointer(ptr unsafe.Pointer) bool {
	// 检查大对象
	if sa.config.EnableDirectAllocator && sa.large != nil {
		if sa.large.IsValidPointer(ptr) {
			return true
		}
	}

	// 对于小对象和中等对象，需要更复杂的检查
	// 这里简化处理，实际实现中需要更精确的边界检查
	return false
}

// Report 生成分配器报告
func (sa *SimpleAllocator) Report() string {
	stats := sa.Stats()
	return stats.Report()
}

// SetGCManager 设置GC管理器 (预留接口)
func (sa *SimpleAllocator) SetGCManager(gcManager interface{}) {
	sa.mutex.Lock()
	defer sa.mutex.Unlock()
	sa.gcManager = gcManager
}
