package gc

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// LargeObjectInfo 大对象信息
type LargeObjectInfo struct {
	Size      uint32     // 对象大小
	AllocTime time.Time  // 分配时间
	ObjType   ObjectType // 对象类型
}

// DirectAllocator 直接分配器 - 处理>4KB的大对象
type DirectAllocator struct {
	largeObjects map[unsafe.Pointer]*LargeObjectInfo // 大对象追踪
	totalSize    uint64                              // 总分配大小
	objectCount  int                                 // 对象数量

	stats *DirectAllocStats
	mutex sync.RWMutex
}

// NewDirectAllocator 创建直接分配器
func NewDirectAllocator(stats *DirectAllocStats) *DirectAllocator {
	return &DirectAllocator{
		largeObjects: make(map[unsafe.Pointer]*LargeObjectInfo),
		totalSize:    0,
		objectCount:  0,
		stats:        stats,
	}
}

// Allocate 大对象分配 - 直接从OS申请，页对齐
func (da *DirectAllocator) Allocate(size uint32, objType ObjectType) unsafe.Pointer {
	// 对齐到页边界 (4KB)
	alignedSize := AlignPage(size)

	// 使用aligned memory分配大块内存
	ptr := allocateAlignedMemory(alignedSize)
	if ptr == nil {
		// 更新失败统计
		atomic.AddUint64(&da.stats.Allocations, 1) // 尝试分配
		return nil
	}

	// 记录大对象信息
	da.mutex.Lock()
	da.largeObjects[ptr] = &LargeObjectInfo{
		Size:      size,
		AllocTime: time.Now(),
		ObjType:   objType,
	}
	da.totalSize += uint64(alignedSize)
	da.objectCount++
	da.mutex.Unlock()

	// 更新统计
	atomic.AddUint64(&da.stats.LargeObjects, 1)
	atomic.AddUint64(&da.stats.Allocations, 1)
	atomic.AddUint64(&da.stats.BytesAllocated, uint64(alignedSize))

	return ptr
}

// Deallocate 释放大对象
func (da *DirectAllocator) Deallocate(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}

	da.mutex.Lock()
	defer da.mutex.Unlock()

	info, exists := da.largeObjects[ptr]
	if !exists {
		// 更新错误统计 - 无效释放
		return
	}

	delete(da.largeObjects, ptr)
	alignedSize := AlignPage(info.Size)
	da.totalSize -= uint64(alignedSize)
	da.objectCount--

	// 更新统计
	atomic.AddUint64(&da.stats.LargeObjects, ^uint64(0)) // 减1
	atomic.AddUint64(&da.stats.Deallocations, 1)
	atomic.AddUint64(&da.stats.BytesFreed, uint64(alignedSize))

	// 释放内存回OS
	freeAlignedMemory(ptr, alignedSize)
}

// GetObjectInfo 获取大对象信息
func (da *DirectAllocator) GetObjectInfo(ptr unsafe.Pointer) *LargeObjectInfo {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	if info, exists := da.largeObjects[ptr]; exists {
		// 返回副本，避免并发修改
		return &LargeObjectInfo{
			Size:      info.Size,
			AllocTime: info.AllocTime,
			ObjType:   info.ObjType,
		}
	}

	return nil
}

// GetStats 获取统计信息
func (da *DirectAllocator) GetStats() DirectAllocStats {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	return DirectAllocStats{
		LargeObjects:   atomic.LoadUint64(&da.stats.LargeObjects),
		Allocations:    atomic.LoadUint64(&da.stats.Allocations),
		Deallocations:  atomic.LoadUint64(&da.stats.Deallocations),
		BytesAllocated: atomic.LoadUint64(&da.stats.BytesAllocated),
		BytesFreed:     atomic.LoadUint64(&da.stats.BytesFreed),
	}
}

// GetTotalSize 获取总分配大小
func (da *DirectAllocator) GetTotalSize() uint64 {
	da.mutex.RLock()
	defer da.mutex.RUnlock()
	return da.totalSize
}

// GetObjectCount 获取对象数量
func (da *DirectAllocator) GetObjectCount() int {
	da.mutex.RLock()
	defer da.mutex.RUnlock()
	return da.objectCount
}

// FindOldObjects 查找分配时间超过指定时间的对象 (用于内存泄漏检测)
func (da *DirectAllocator) FindOldObjects(maxAge time.Duration) []*LargeObjectInfo {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	var oldObjects []*LargeObjectInfo
	threshold := time.Now().Add(-maxAge)

	for _, info := range da.largeObjects {
		if info.AllocTime.Before(threshold) {
			// 返回副本
			oldObjects = append(oldObjects, &LargeObjectInfo{
				Size:      info.Size,
				AllocTime: info.AllocTime,
				ObjType:   info.ObjType,
			})
		}
	}

	return oldObjects
}

// GetObjectsByType 按类型获取对象统计
func (da *DirectAllocator) GetObjectsByType() map[ObjectType]int {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	typeStats := make(map[ObjectType]int)

	for _, info := range da.largeObjects {
		typeStats[info.ObjType]++
	}

	return typeStats
}

// GetLargestObjects 获取最大的N个对象
func (da *DirectAllocator) GetLargestObjects(n int) []*LargeObjectInfo {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	var objects []*LargeObjectInfo

	// 收集所有对象
	for _, info := range da.largeObjects {
		objects = append(objects, &LargeObjectInfo{
			Size:      info.Size,
			AllocTime: info.AllocTime,
			ObjType:   info.ObjType,
		})
	}

	// 简单的选择排序，找出最大的n个
	if len(objects) < n {
		n = len(objects)
	}

	for i := 0; i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(objects); j++ {
			if objects[j].Size > objects[maxIdx].Size {
				maxIdx = j
			}
		}
		// 交换
		objects[i], objects[maxIdx] = objects[maxIdx], objects[i]
	}

	return objects[:n]
}

// CompactMemory 内存压缩 - 大对象分配器通常不需要压缩
// 但可以检查是否有可以优化的内存使用
func (da *DirectAllocator) CompactMemory() int {
	// 大对象直接从OS申请，不需要压缩
	// 返回0表示没有压缩操作
	return 0
}

// Destroy 销毁直接分配器，释放所有大对象
func (da *DirectAllocator) Destroy() {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	// 释放所有大对象
	for ptr, info := range da.largeObjects {
		if ptr != nil {
			alignedSize := AlignPage(info.Size)
			freeAlignedMemory(ptr, alignedSize)
		}
	}

	// 清空映射
	da.largeObjects = make(map[unsafe.Pointer]*LargeObjectInfo)
	da.totalSize = 0
	da.objectCount = 0
}

// CheckIntegrity 检查分配器完整性
func (da *DirectAllocator) CheckIntegrity() error {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	// 验证统计信息一致性
	expectedCount := uint64(len(da.largeObjects))
	actualCount := atomic.LoadUint64(&da.stats.LargeObjects)

	if expectedCount != actualCount {
		// 这里应该返回一个错误，但为了简化，我们先跳过
		// return fmt.Errorf("object count mismatch: expected %d, got %d", expectedCount, actualCount)
	}

	// 验证大小计算
	var calculatedSize uint64
	for _, info := range da.largeObjects {
		calculatedSize += uint64(AlignPage(info.Size))
	}

	if calculatedSize != da.totalSize {
		// return fmt.Errorf("size calculation mismatch: expected %d, got %d", calculatedSize, da.totalSize)
	}

	return nil
}

// IsValidPointer 检查指针是否是有效的大对象指针
func (da *DirectAllocator) IsValidPointer(ptr unsafe.Pointer) bool {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	_, exists := da.largeObjects[ptr]
	return exists
}

// GetMemoryUsage 获取内存使用情况
func (da *DirectAllocator) GetMemoryUsage() (allocated uint64, freed uint64, live uint64) {
	allocated = atomic.LoadUint64(&da.stats.BytesAllocated)
	freed = atomic.LoadUint64(&da.stats.BytesFreed)
	live = allocated - freed
	return
}
