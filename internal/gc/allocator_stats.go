package gc

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// AllocationStats 分配器统计信息
type AllocationStats struct {
	// 按size class统计
	SmallAllocStats  [NumSizeClasses]SizeClassStats // 8个小对象size class
	MediumAllocStats SlabStats                      // 中等对象统计
	LargeAllocStats  DirectAllocStats               // 大对象统计

	// 总体统计
	TotalAllocations    uint64 // 总分配次数
	TotalDeallocations  uint64 // 总释放次数
	TotalBytesAllocated uint64 // 总分配字节数
	TotalBytesFreed     uint64 // 总释放字节数

	// 性能指标
	TotalAllocTime   uint64 // 总分配时间 (纳秒)
	AverageAllocSize uint64 // 平均分配大小

	// 错误统计
	AllocationFailures   uint64 // 分配失败次数
	InvalidDeallocations uint64 // 无效释放次数
}

// SizeClassStats Size Class统计
type SizeClassStats struct {
	Allocations    uint64 // 分配次数
	Deallocations  uint64 // 释放次数
	BytesAllocated uint64 // 分配字节数
	BytesFreed     uint64 // 释放字节数
	PagesAllocated uint64 // 分配页数
	WasteBytes     uint64 // 浪费字节数
}

// SlabStats Slab分配器统计
type SlabStats struct {
	TotalSlabs     uint64 // 总slab数量
	TotalChunks    uint64 // 总chunk数量
	ActiveChunks   uint64 // 活跃chunk数量
	Allocations    uint64 // 分配次数
	Deallocations  uint64 // 释放次数
	BytesAllocated uint64 // 分配字节数
	BytesFreed     uint64 // 释放字节数
}

// DirectAllocStats 直接分配器统计
type DirectAllocStats struct {
	LargeObjects   uint64 // 大对象数量
	Allocations    uint64 // 分配次数
	Deallocations  uint64 // 释放次数
	BytesAllocated uint64 // 分配字节数
	BytesFreed     uint64 // 释放字节数
}

// IncAllocation 增加分配统计
func (stats *AllocationStats) IncAllocation(size uint32) {
	atomic.AddUint64(&stats.TotalAllocations, 1)
	atomic.AddUint64(&stats.TotalBytesAllocated, uint64(size))
}

// IncDeallocation 增加释放统计
func (stats *AllocationStats) IncDeallocation(size uint32) {
	atomic.AddUint64(&stats.TotalDeallocations, 1)
	atomic.AddUint64(&stats.TotalBytesFreed, uint64(size))
}

// IncAllocationTime 增加分配时间统计
func (stats *AllocationStats) IncAllocationTime(duration time.Duration) {
	atomic.AddUint64(&stats.TotalAllocTime, uint64(duration.Nanoseconds()))
}

// IncAllocationFailure 增加分配失败统计
func (stats *AllocationStats) IncAllocationFailure() {
	atomic.AddUint64(&stats.AllocationFailures, 1)
}

// IncInvalidDeallocation 增加无效释放统计
func (stats *AllocationStats) IncInvalidDeallocation() {
	atomic.AddUint64(&stats.InvalidDeallocations, 1)
}

// GetAverageAllocTime 获取平均分配时间
func (stats *AllocationStats) GetAverageAllocTime() time.Duration {
	allocations := atomic.LoadUint64(&stats.TotalAllocations)
	if allocations == 0 {
		return 0
	}
	totalTime := atomic.LoadUint64(&stats.TotalAllocTime)
	return time.Duration(totalTime / allocations)
}

// GetAverageAllocSize 获取平均分配大小
func (stats *AllocationStats) GetAverageAllocSize() uint64 {
	allocations := atomic.LoadUint64(&stats.TotalAllocations)
	if allocations == 0 {
		return 0
	}
	totalBytes := atomic.LoadUint64(&stats.TotalBytesAllocated)
	return totalBytes / allocations
}

// GetFragmentationRatio 获取碎片率
func (stats *AllocationStats) GetFragmentationRatio() float64 {
	totalAllocated := atomic.LoadUint64(&stats.TotalBytesAllocated)
	totalFreed := atomic.LoadUint64(&stats.TotalBytesFreed)

	if totalAllocated == 0 {
		return 0.0
	}

	// 计算总浪费字节数
	var totalWaste uint64
	for i := 0; i < NumSizeClasses; i++ {
		totalWaste += atomic.LoadUint64(&stats.SmallAllocStats[i].WasteBytes)
	}

	return float64(totalWaste) / float64(totalAllocated-totalFreed)
}

// GetLiveObjects 获取存活对象数量
func (stats *AllocationStats) GetLiveObjects() uint64 {
	allocations := atomic.LoadUint64(&stats.TotalAllocations)
	deallocations := atomic.LoadUint64(&stats.TotalDeallocations)
	return allocations - deallocations
}

// GetLiveBytes 获取存活字节数
func (stats *AllocationStats) GetLiveBytes() uint64 {
	allocated := atomic.LoadUint64(&stats.TotalBytesAllocated)
	freed := atomic.LoadUint64(&stats.TotalBytesFreed)
	return allocated - freed
}

// Report 生成分配器性能报告
func (stats *AllocationStats) Report() string {
	var report strings.Builder

	report.WriteString("AQL Allocator Performance Report\n")
	report.WriteString("=================================\n\n")

	// 总体统计
	report.WriteString("Overall Statistics:\n")
	report.WriteString(fmt.Sprintf("  Total Allocations:     %d\n", atomic.LoadUint64(&stats.TotalAllocations)))
	report.WriteString(fmt.Sprintf("  Total Deallocations:   %d\n", atomic.LoadUint64(&stats.TotalDeallocations)))
	report.WriteString(fmt.Sprintf("  Live Objects:          %d\n", stats.GetLiveObjects()))
	report.WriteString(fmt.Sprintf("  Bytes Allocated:       %d (%.2f MB)\n",
		atomic.LoadUint64(&stats.TotalBytesAllocated),
		float64(atomic.LoadUint64(&stats.TotalBytesAllocated))/1024/1024))
	report.WriteString(fmt.Sprintf("  Live Bytes:            %d (%.2f MB)\n",
		stats.GetLiveBytes(),
		float64(stats.GetLiveBytes())/1024/1024))
	report.WriteString(fmt.Sprintf("  Average Alloc Size:    %d bytes\n", stats.GetAverageAllocSize()))
	report.WriteString(fmt.Sprintf("  Average Alloc Time:    %v\n", stats.GetAverageAllocTime()))
	report.WriteString(fmt.Sprintf("  Fragmentation Ratio:   %.2f%%\n", stats.GetFragmentationRatio()*100))
	report.WriteString(fmt.Sprintf("  Allocation Failures:   %d\n", atomic.LoadUint64(&stats.AllocationFailures)))

	// Size Class详情
	report.WriteString("\nSize Class Distribution:\n")
	report.WriteString("Size | Allocs   | Deallocs | Live | Pages | Waste%\n")
	report.WriteString("-----|----------|----------|------|-------|-------\n")

	for i := 0; i < NumSizeClasses; i++ {
		scs := &stats.SmallAllocStats[i]
		allocations := atomic.LoadUint64(&scs.Allocations)
		deallocations := atomic.LoadUint64(&scs.Deallocations)
		live := allocations - deallocations
		pages := atomic.LoadUint64(&scs.PagesAllocated)

		var wasteRatio float64
		if atomic.LoadUint64(&scs.BytesAllocated) > 0 {
			wasteRatio = float64(atomic.LoadUint64(&scs.WasteBytes)) /
				float64(atomic.LoadUint64(&scs.BytesAllocated)) * 100
		}

		report.WriteString(fmt.Sprintf("%4d | %8d | %8d | %4d | %5d | %5.1f%%\n",
			SizeClassTable[i].Size, allocations, deallocations,
			live, pages, wasteRatio))
	}

	// 中等对象统计
	report.WriteString("\nMedium Objects (Slab Allocator):\n")
	report.WriteString(fmt.Sprintf("  Total Slabs:           %d\n", atomic.LoadUint64(&stats.MediumAllocStats.TotalSlabs)))
	report.WriteString(fmt.Sprintf("  Active Chunks:         %d\n", atomic.LoadUint64(&stats.MediumAllocStats.ActiveChunks)))
	report.WriteString(fmt.Sprintf("  Allocations:           %d\n", atomic.LoadUint64(&stats.MediumAllocStats.Allocations)))
	report.WriteString(fmt.Sprintf("  Bytes Allocated:       %d\n", atomic.LoadUint64(&stats.MediumAllocStats.BytesAllocated)))

	// 大对象统计
	report.WriteString("\nLarge Objects (Direct Allocator):\n")
	report.WriteString(fmt.Sprintf("  Large Objects:         %d\n", atomic.LoadUint64(&stats.LargeAllocStats.LargeObjects)))
	report.WriteString(fmt.Sprintf("  Allocations:           %d\n", atomic.LoadUint64(&stats.LargeAllocStats.Allocations)))
	report.WriteString(fmt.Sprintf("  Bytes Allocated:       %d (%.2f MB)\n",
		atomic.LoadUint64(&stats.LargeAllocStats.BytesAllocated),
		float64(atomic.LoadUint64(&stats.LargeAllocStats.BytesAllocated))/1024/1024))

	return report.String()
}

// Reset 重置统计信息
func (stats *AllocationStats) Reset() {
	// 重置总体统计
	atomic.StoreUint64(&stats.TotalAllocations, 0)
	atomic.StoreUint64(&stats.TotalDeallocations, 0)
	atomic.StoreUint64(&stats.TotalBytesAllocated, 0)
	atomic.StoreUint64(&stats.TotalBytesFreed, 0)
	atomic.StoreUint64(&stats.TotalAllocTime, 0)
	atomic.StoreUint64(&stats.AllocationFailures, 0)
	atomic.StoreUint64(&stats.InvalidDeallocations, 0)

	// 重置Size Class统计
	for i := 0; i < NumSizeClasses; i++ {
		scs := &stats.SmallAllocStats[i]
		atomic.StoreUint64(&scs.Allocations, 0)
		atomic.StoreUint64(&scs.Deallocations, 0)
		atomic.StoreUint64(&scs.BytesAllocated, 0)
		atomic.StoreUint64(&scs.BytesFreed, 0)
		atomic.StoreUint64(&scs.PagesAllocated, 0)
		atomic.StoreUint64(&scs.WasteBytes, 0)
	}

	// 重置Slab统计
	atomic.StoreUint64(&stats.MediumAllocStats.TotalSlabs, 0)
	atomic.StoreUint64(&stats.MediumAllocStats.TotalChunks, 0)
	atomic.StoreUint64(&stats.MediumAllocStats.ActiveChunks, 0)
	atomic.StoreUint64(&stats.MediumAllocStats.Allocations, 0)
	atomic.StoreUint64(&stats.MediumAllocStats.Deallocations, 0)
	atomic.StoreUint64(&stats.MediumAllocStats.BytesAllocated, 0)
	atomic.StoreUint64(&stats.MediumAllocStats.BytesFreed, 0)

	// 重置大对象统计
	atomic.StoreUint64(&stats.LargeAllocStats.LargeObjects, 0)
	atomic.StoreUint64(&stats.LargeAllocStats.Allocations, 0)
	atomic.StoreUint64(&stats.LargeAllocStats.Deallocations, 0)
	atomic.StoreUint64(&stats.LargeAllocStats.BytesAllocated, 0)
	atomic.StoreUint64(&stats.LargeAllocStats.BytesFreed, 0)
}
