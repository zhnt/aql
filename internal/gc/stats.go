package gc

import (
	"fmt"
	"sync/atomic"
	"time"
)

// =============================================================================
// GC统计信息
// =============================================================================

// GCStats GC统计信息，所有字段都是原子操作安全的
type GCStats struct {
	// ========== 总体统计 ==========
	TotalAllocations   uint64 // 总分配次数
	TotalDeallocations uint64 // 总释放次数
	BytesAllocated     uint64 // 已分配字节数
	BytesFreed         uint64 // 已释放字节数

	// ========== GC执行统计 ==========
	RefCountGCRuns  uint64 // 引用计数GC运行次数
	MarkSweepGCRuns uint64 // 标记清除GC运行次数
	TotalGCTime     int64  // 总GC时间（纳秒）
	MaxPauseTime    int64  // 最大暂停时间（纳秒）

	// ========== 性能指标 ==========
	AverageAllocSize uint64 // 平均分配大小
	HeapSize         uint64 // 当前堆大小

	// ========== 错误统计 ==========
	MemoryLeaks   uint64 // 内存泄漏次数
	DoubleFrees   uint64 // 重复释放次数
	UseAfterFrees uint64 // 释放后使用次数

	// ========== 对象类型统计 ==========
	StringObjects   uint64 // 字符串对象数量
	ArrayObjects    uint64 // 数组对象数量
	StructObjects   uint64 // 结构体对象数量
	FunctionObjects uint64 // 函数对象数量

	// ========== 引用计数统计 ==========
	RefCountIncs   uint64 // 引用计数递增次数
	RefCountDecs   uint64 // 引用计数递减次数
	ZeroRefObjects uint64 // 零引用对象数量

	// ========== 标记清除统计 ==========
	MarkPhaseTime  int64  // 标记阶段时间（纳秒）
	SweepPhaseTime int64  // 清扫阶段时间（纳秒）
	MarkedObjects  uint64 // 标记的对象数量
	SweptObjects   uint64 // 清扫的对象数量
	CyclicObjects  uint64 // 循环引用对象数量
}

// =============================================================================
// GC统计操作方法
// =============================================================================

// NewGCStats 创建新的GC统计实例
func NewGCStats() *GCStats {
	return &GCStats{}
}

// IncAllocation 增加分配统计
func (s *GCStats) IncAllocation(size uint32) {
	atomic.AddUint64(&s.TotalAllocations, 1)
	atomic.AddUint64(&s.BytesAllocated, uint64(size))
	atomic.AddUint64(&s.HeapSize, uint64(size))

	// 更新平均分配大小
	s.updateAverageAllocSize()
}

// IncDeallocation 增加释放统计
func (s *GCStats) IncDeallocation(size uint32) {
	atomic.AddUint64(&s.TotalDeallocations, 1)
	atomic.AddUint64(&s.BytesFreed, uint64(size))
	atomic.AddUint64(&s.HeapSize, ^uint64(size-1)) // 原子减法
}

// IncObjectType 增加对象类型统计
func (s *GCStats) IncObjectType(objType ObjectType) {
	switch objType {
	case ObjectTypeString:
		atomic.AddUint64(&s.StringObjects, 1)
	case ObjectTypeArray:
		atomic.AddUint64(&s.ArrayObjects, 1)
	case ObjectTypeStruct:
		atomic.AddUint64(&s.StructObjects, 1)
	case ObjectTypeFunction:
		atomic.AddUint64(&s.FunctionObjects, 1)
	}
}

// DecObjectType 减少对象类型统计
func (s *GCStats) DecObjectType(objType ObjectType) {
	switch objType {
	case ObjectTypeString:
		atomic.AddUint64(&s.StringObjects, ^uint64(0))
	case ObjectTypeArray:
		atomic.AddUint64(&s.ArrayObjects, ^uint64(0))
	case ObjectTypeStruct:
		atomic.AddUint64(&s.StructObjects, ^uint64(0))
	case ObjectTypeFunction:
		atomic.AddUint64(&s.FunctionObjects, ^uint64(0))
	}
}

// IncRefCountOp 增加引用计数操作统计
func (s *GCStats) IncRefCountOp(inc bool) {
	if inc {
		atomic.AddUint64(&s.RefCountIncs, 1)
	} else {
		atomic.AddUint64(&s.RefCountDecs, 1)
	}
}

// IncZeroRefObject 增加零引用对象统计
func (s *GCStats) IncZeroRefObject() {
	atomic.AddUint64(&s.ZeroRefObjects, 1)
}

// IncRefCountGC 增加引用计数GC运行统计
func (s *GCStats) IncRefCountGC() {
	atomic.AddUint64(&s.RefCountGCRuns, 1)
}

// IncMarkSweepGC 增加标记清除GC运行统计，记录执行时间
func (s *GCStats) IncMarkSweepGC(duration time.Duration) {
	atomic.AddUint64(&s.MarkSweepGCRuns, 1)
	atomic.AddInt64(&s.TotalGCTime, int64(duration))

	// 更新最大暂停时间
	s.updateMaxPauseTime(duration)
}

// RecordMarkPhase 记录标记阶段时间
func (s *GCStats) RecordMarkPhase(duration time.Duration, markedCount uint64) {
	atomic.AddInt64(&s.MarkPhaseTime, int64(duration))
	atomic.AddUint64(&s.MarkedObjects, markedCount)
}

// RecordSweepPhase 记录清扫阶段时间
func (s *GCStats) RecordSweepPhase(duration time.Duration, sweptCount uint64) {
	atomic.AddInt64(&s.SweepPhaseTime, int64(duration))
	atomic.AddUint64(&s.SweptObjects, sweptCount)
}

// IncCyclicObject 增加循环引用对象统计
func (s *GCStats) IncCyclicObject() {
	atomic.AddUint64(&s.CyclicObjects, 1)
}

// IncError 增加错误统计
func (s *GCStats) IncError(errorType string) {
	switch errorType {
	case "memory_leak":
		atomic.AddUint64(&s.MemoryLeaks, 1)
	case "double_free":
		atomic.AddUint64(&s.DoubleFrees, 1)
	case "use_after_free":
		atomic.AddUint64(&s.UseAfterFrees, 1)
	}
}

// =============================================================================
// 私有辅助方法
// =============================================================================

// updateAverageAllocSize 更新平均分配大小
func (s *GCStats) updateAverageAllocSize() {
	totalAllocs := atomic.LoadUint64(&s.TotalAllocations)
	totalBytes := atomic.LoadUint64(&s.BytesAllocated)

	if totalAllocs > 0 {
		avgSize := totalBytes / totalAllocs
		atomic.StoreUint64(&s.AverageAllocSize, avgSize)
	}
}

// updateMaxPauseTime 更新最大暂停时间
func (s *GCStats) updateMaxPauseTime(duration time.Duration) {
	newPause := int64(duration)
	for {
		oldMax := atomic.LoadInt64(&s.MaxPauseTime)
		if newPause <= oldMax {
			break
		}
		if atomic.CompareAndSwapInt64(&s.MaxPauseTime, oldMax, newPause) {
			break
		}
	}
}

// =============================================================================
// 获取统计数据的方法
// =============================================================================

// GetTotalAllocations 获取总分配次数
func (s *GCStats) GetTotalAllocations() uint64 {
	return atomic.LoadUint64(&s.TotalAllocations)
}

// GetTotalDeallocations 获取总释放次数
func (s *GCStats) GetTotalDeallocations() uint64 {
	return atomic.LoadUint64(&s.TotalDeallocations)
}

// GetBytesAllocated 获取已分配字节数
func (s *GCStats) GetBytesAllocated() uint64 {
	return atomic.LoadUint64(&s.BytesAllocated)
}

// GetBytesFreed 获取已释放字节数
func (s *GCStats) GetBytesFreed() uint64 {
	return atomic.LoadUint64(&s.BytesFreed)
}

// GetHeapSize 获取当前堆大小
func (s *GCStats) GetHeapSize() uint64 {
	return atomic.LoadUint64(&s.HeapSize)
}

// GetHeapUtilization 获取堆利用率
func (s *GCStats) GetHeapUtilization() float64 {
	allocated := float64(atomic.LoadUint64(&s.BytesAllocated))
	freed := float64(atomic.LoadUint64(&s.BytesFreed))
	heapSize := float64(atomic.LoadUint64(&s.HeapSize))

	if heapSize > 0 {
		return (allocated - freed) / heapSize
	}
	return 0.0
}

// GetGCOverhead 获取GC开销百分比
func (s *GCStats) GetGCOverhead() float64 {
	totalGCTime := time.Duration(atomic.LoadInt64(&s.TotalGCTime))
	// 这里需要与总运行时间比较，暂时返回0
	// 在实际使用中，需要跟踪程序总运行时间
	_ = totalGCTime
	return 0.0
}

// GetAverageAllocSize 获取平均分配大小
func (s *GCStats) GetAverageAllocSize() uint64 {
	return atomic.LoadUint64(&s.AverageAllocSize)
}

// GetMaxPauseTime 获取最大暂停时间
func (s *GCStats) GetMaxPauseTime() time.Duration {
	return time.Duration(atomic.LoadInt64(&s.MaxPauseTime))
}

// =============================================================================
// 统计报告生成
// =============================================================================

// Report 生成详细的GC性能报告
func (s *GCStats) Report() string {
	// 获取所有统计数据的快照
	totalAllocs := s.GetTotalAllocations()
	totalDeallocs := s.GetTotalDeallocations()
	bytesAllocated := s.GetBytesAllocated()
	bytesFreed := s.GetBytesFreed()
	heapSize := s.GetHeapSize()
	avgAllocSize := s.GetAverageAllocSize()
	maxPause := s.GetMaxPauseTime()
	heapUtil := s.GetHeapUtilization()

	refCountRuns := atomic.LoadUint64(&s.RefCountGCRuns)
	markSweepRuns := atomic.LoadUint64(&s.MarkSweepGCRuns)
	memoryLeaks := atomic.LoadUint64(&s.MemoryLeaks)

	// 对象类型统计
	stringObjs := atomic.LoadUint64(&s.StringObjects)
	arrayObjs := atomic.LoadUint64(&s.ArrayObjects)
	structObjs := atomic.LoadUint64(&s.StructObjects)
	funcObjs := atomic.LoadUint64(&s.FunctionObjects)

	// 引用计数统计
	refIncs := atomic.LoadUint64(&s.RefCountIncs)
	refDecs := atomic.LoadUint64(&s.RefCountDecs)
	zeroRefs := atomic.LoadUint64(&s.ZeroRefObjects)

	return fmt.Sprintf(`
AQL GC Performance Report
========================
Memory Statistics:
- Total Allocations:     %d
- Total Deallocations:   %d
- Bytes Allocated:       %d (%.2f MB)
- Bytes Freed:           %d (%.2f MB)
- Current Heap Size:     %d (%.2f MB)
- Heap Utilization:      %.2f%%
- Average Alloc Size:    %d bytes

GC Statistics:
- RefCount GC Runs:      %d
- MarkSweep GC Runs:     %d
- Max Pause Time:        %v
- Memory Leaks:          %d

Object Type Distribution:
- String Objects:        %d
- Array Objects:         %d
- Struct Objects:        %d
- Function Objects:      %d

Reference Counting:
- RefCount Increments:   %d
- RefCount Decrements:   %d
- Zero-Ref Objects:      %d
========================
`,
		totalAllocs, totalDeallocs,
		bytesAllocated, float64(bytesAllocated)/1024/1024,
		bytesFreed, float64(bytesFreed)/1024/1024,
		heapSize, float64(heapSize)/1024/1024,
		heapUtil*100, avgAllocSize,
		refCountRuns, markSweepRuns, maxPause, memoryLeaks,
		stringObjs, arrayObjs, structObjs, funcObjs,
		refIncs, refDecs, zeroRefs)
}

// Summary 生成简要的统计摘要
func (s *GCStats) Summary() string {
	totalAllocs := s.GetTotalAllocations()
	heapSize := s.GetHeapSize()
	maxPause := s.GetMaxPauseTime()
	heapUtil := s.GetHeapUtilization()

	return fmt.Sprintf("Allocs: %d, Heap: %.1fMB, Util: %.1f%%, MaxPause: %v",
		totalAllocs,
		float64(heapSize)/1024/1024,
		heapUtil*100,
		maxPause)
}

// Reset 重置所有统计数据
func (s *GCStats) Reset() {
	atomic.StoreUint64(&s.TotalAllocations, 0)
	atomic.StoreUint64(&s.TotalDeallocations, 0)
	atomic.StoreUint64(&s.BytesAllocated, 0)
	atomic.StoreUint64(&s.BytesFreed, 0)
	atomic.StoreUint64(&s.RefCountGCRuns, 0)
	atomic.StoreUint64(&s.MarkSweepGCRuns, 0)
	atomic.StoreInt64(&s.TotalGCTime, 0)
	atomic.StoreInt64(&s.MaxPauseTime, 0)
	atomic.StoreUint64(&s.AverageAllocSize, 0)
	atomic.StoreUint64(&s.HeapSize, 0)
	atomic.StoreUint64(&s.MemoryLeaks, 0)
	atomic.StoreUint64(&s.DoubleFrees, 0)
	atomic.StoreUint64(&s.UseAfterFrees, 0)
	atomic.StoreUint64(&s.StringObjects, 0)
	atomic.StoreUint64(&s.ArrayObjects, 0)
	atomic.StoreUint64(&s.StructObjects, 0)
	atomic.StoreUint64(&s.FunctionObjects, 0)
	atomic.StoreUint64(&s.RefCountIncs, 0)
	atomic.StoreUint64(&s.RefCountDecs, 0)
	atomic.StoreUint64(&s.ZeroRefObjects, 0)
	atomic.StoreInt64(&s.MarkPhaseTime, 0)
	atomic.StoreInt64(&s.SweepPhaseTime, 0)
	atomic.StoreUint64(&s.MarkedObjects, 0)
	atomic.StoreUint64(&s.SweptObjects, 0)
	atomic.StoreUint64(&s.CyclicObjects, 0)
}
