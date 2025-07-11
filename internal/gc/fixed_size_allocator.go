package gc

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// FreeList 空闲链表节点
type FreeList struct {
	next *FreeList
}

// AllocPage 分配页结构
type AllocPage struct {
	memory      unsafe.Pointer // 页内存起始地址
	freeList    *FreeList      // 页内空闲对象链表
	objectCount int            // 已分配对象数量
	maxObjects  int            // 最大对象数量
	next        *AllocPage     // 下一个页
}

// FixedSizeAllocator 固定大小分配器 - 管理特定size class的对象
type FixedSizeAllocator struct {
	sizeClass      int    // size class编号
	objectSize     uint32 // 对象大小
	pageSize       uint32 // 页大小 (通常4KB)
	objectsPerPage int    // 每页对象数量

	// 内存页管理
	pages       []*AllocPage // 分配页列表
	currentPage *AllocPage   // 当前分配页
	freePages   []*AllocPage // 空闲页列表

	// 性能优化
	fastPath *FreeList // 快速分配路径
	stats    *SizeClassStats
	mutex    sync.Mutex
}

// NewFixedSizeAllocator 创建固定大小分配器
func NewFixedSizeAllocator(sizeClass int, stats *SizeClassStats) *FixedSizeAllocator {
	info := SizeClassTable[sizeClass]

	return &FixedSizeAllocator{
		sizeClass:      sizeClass,
		objectSize:     info.Size,
		pageSize:       4096, // 4KB页
		objectsPerPage: info.ObjectsPerPage,
		pages:          make([]*AllocPage, 0),
		freePages:      make([]*AllocPage, 0),
		stats:          stats,
	}
}

// Allocate 分配对象
func (fsa *FixedSizeAllocator) Allocate() unsafe.Pointer {
	// 快速路径：从空闲链表获取
	if obj := fsa.fastPath; obj != nil {
		fsa.fastPath = obj.next

		// 更新统计
		atomic.AddUint64(&fsa.stats.Allocations, 1)
		atomic.AddUint64(&fsa.stats.BytesAllocated, uint64(fsa.objectSize))

		return unsafe.Pointer(obj)
	}

	// 慢速路径：需要新页或查找空闲页
	return fsa.allocateSlow()
}

// allocateSlow 慢速分配路径
func (fsa *FixedSizeAllocator) allocateSlow() unsafe.Pointer {
	fsa.mutex.Lock()
	defer fsa.mutex.Unlock()

	// 尝试当前页
	if fsa.currentPage != nil && fsa.currentPage.objectCount < fsa.currentPage.maxObjects {
		ptr := fsa.allocateFromPage(fsa.currentPage)
		if ptr != nil {
			return ptr
		}
	}

	// 尝试空闲页
	if len(fsa.freePages) > 0 {
		page := fsa.freePages[len(fsa.freePages)-1]
		fsa.freePages = fsa.freePages[:len(fsa.freePages)-1]
		fsa.currentPage = page
		ptr := fsa.allocateFromPage(page)
		if ptr != nil {
			return ptr
		}
	}

	// 分配新页
	page := fsa.allocateNewPage()
	if page == nil {
		return nil // 内存不足
	}

	fsa.currentPage = page
	fsa.pages = append(fsa.pages, page)

	return fsa.allocateFromPage(page)
}

// allocateFromPage 从页中分配对象
func (fsa *FixedSizeAllocator) allocateFromPage(page *AllocPage) unsafe.Pointer {
	if page.freeList != nil {
		// 从空闲链表获取
		obj := page.freeList
		page.freeList = obj.next
		page.objectCount++

		// 更新统计
		atomic.AddUint64(&fsa.stats.Allocations, 1)
		atomic.AddUint64(&fsa.stats.BytesAllocated, uint64(fsa.objectSize))

		return unsafe.Pointer(obj)
	}

	if page.objectCount < page.maxObjects {
		// 从页的线性区域分配
		offset := uintptr(page.objectCount) * uintptr(fsa.objectSize)
		ptr := unsafe.Pointer(uintptr(page.memory) + offset)
		page.objectCount++

		// 更新统计
		atomic.AddUint64(&fsa.stats.Allocations, 1)
		atomic.AddUint64(&fsa.stats.BytesAllocated, uint64(fsa.objectSize))

		return ptr
	}

	return nil // 页已满
}

// allocateNewPage 分配新页
func (fsa *FixedSizeAllocator) allocateNewPage() *AllocPage {
	// 分配页内存
	memory := allocateAlignedMemory(fsa.pageSize)
	if memory == nil {
		return nil
	}

	page := &AllocPage{
		memory:      memory,
		freeList:    nil,
		objectCount: 0,
		maxObjects:  fsa.objectsPerPage,
		next:        nil,
	}

	// 更新统计
	atomic.AddUint64(&fsa.stats.PagesAllocated, 1)

	return page
}

// Deallocate 释放对象
func (fsa *FixedSizeAllocator) Deallocate(ptr unsafe.Pointer) {
	if ptr == nil {
		return
	}

	// 更新统计
	atomic.AddUint64(&fsa.stats.Deallocations, 1)
	atomic.AddUint64(&fsa.stats.BytesFreed, uint64(fsa.objectSize))

	// 将对象添加到快速路径
	obj := (*FreeList)(ptr)
	obj.next = fsa.fastPath
	fsa.fastPath = obj
}

// DeallocateBatch 批量释放对象
func (fsa *FixedSizeAllocator) DeallocateBatch(objects []unsafe.Pointer) {
	if len(objects) == 0 {
		return
	}

	// 更新统计
	count := uint64(len(objects))
	bytes := count * uint64(fsa.objectSize)
	atomic.AddUint64(&fsa.stats.Deallocations, count)
	atomic.AddUint64(&fsa.stats.BytesFreed, bytes)

	// 批量添加到快速路径
	for _, ptr := range objects {
		if ptr != nil {
			obj := (*FreeList)(ptr)
			obj.next = fsa.fastPath
			fsa.fastPath = obj
		}
	}
}

// GetStats 获取统计信息
func (fsa *FixedSizeAllocator) GetStats() SizeClassStats {
	fsa.mutex.Lock()
	defer fsa.mutex.Unlock()

	return SizeClassStats{
		Allocations:    atomic.LoadUint64(&fsa.stats.Allocations),
		Deallocations:  atomic.LoadUint64(&fsa.stats.Deallocations),
		BytesAllocated: atomic.LoadUint64(&fsa.stats.BytesAllocated),
		BytesFreed:     atomic.LoadUint64(&fsa.stats.BytesFreed),
		PagesAllocated: atomic.LoadUint64(&fsa.stats.PagesAllocated),
		WasteBytes:     fsa.calculateWasteBytes(),
	}
}

// calculateWasteBytes 计算浪费的字节数
func (fsa *FixedSizeAllocator) calculateWasteBytes() uint64 {
	var totalWaste uint64

	for _, page := range fsa.pages {
		if page != nil {
			// 计算页内未使用的空间
			usedBytes := uint64(page.objectCount) * uint64(fsa.objectSize)
			pageBytes := uint64(fsa.pageSize)

			if pageBytes > usedBytes {
				totalWaste += pageBytes - usedBytes
			}
		}
	}

	return totalWaste
}

// Destroy 销毁分配器，释放所有内存
func (fsa *FixedSizeAllocator) Destroy() {
	fsa.mutex.Lock()
	defer fsa.mutex.Unlock()

	// 释放所有页
	for _, page := range fsa.pages {
		if page != nil && page.memory != nil {
			freeAlignedMemory(page.memory, fsa.pageSize)
		}
	}

	// 释放空闲页
	for _, page := range fsa.freePages {
		if page != nil && page.memory != nil {
			freeAlignedMemory(page.memory, fsa.pageSize)
		}
	}

	// 清空所有引用
	fsa.pages = nil
	fsa.freePages = nil
	fsa.currentPage = nil
	fsa.fastPath = nil
}

// CompactMemory 内存压缩 - 释放完全空闲的页
func (fsa *FixedSizeAllocator) CompactMemory() int {
	fsa.mutex.Lock()
	defer fsa.mutex.Unlock()

	compactedPages := 0
	newPages := make([]*AllocPage, 0, len(fsa.pages))

	for _, page := range fsa.pages {
		if page == nil {
			continue
		}

		// 检查页是否完全空闲
		if page.objectCount == 0 && page.freeList == nil {
			// 释放空闲页
			freeAlignedMemory(page.memory, fsa.pageSize)
			compactedPages++
		} else {
			// 保留有对象的页
			newPages = append(newPages, page)
		}
	}

	fsa.pages = newPages
	return compactedPages
}

// 内存分配函数在memory.go中实现
