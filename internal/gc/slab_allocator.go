package gc

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// SlabChunk Slab块 - 实际的内存块
type SlabChunk struct {
	memory      unsafe.Pointer // 内存起始地址
	freeList    *FreeList      // 空闲对象链表
	objectCount int            // 已分配对象数量
	maxObjects  int            // 最大对象数量
	next        *SlabChunk     // 链表指针
	prev        *SlabChunk
}

// Slab结构 - 管理相同大小的对象
type Slab struct {
	objectSize  uint32       // 对象大小
	chunkSize   uint32       // chunk大小
	chunks      []*SlabChunk // chunk列表
	partialList *SlabChunk   // 部分使用的chunk
	emptyList   *SlabChunk   // 空的chunk
	fullList    *SlabChunk   // 满的chunk

	mutex sync.Mutex
}

// SlabAllocator Slab分配器 - 处理256B-4KB的中等对象
type SlabAllocator struct {
	slabs     map[uint32]*Slab // size -> slab映射
	slabList  []*Slab          // 所有slab列表
	chunkSize uint32           // chunk大小 (通常64KB)

	stats *SlabStats
	mutex sync.RWMutex
}

// NewSlabAllocator 创建Slab分配器
func NewSlabAllocator(chunkSize uint32, stats *SlabStats) *SlabAllocator {
	return &SlabAllocator{
		slabs:     make(map[uint32]*Slab),
		slabList:  make([]*Slab, 0),
		chunkSize: chunkSize,
		stats:     stats,
	}
}

// Allocate 分配对象
func (sa *SlabAllocator) Allocate(size uint32) unsafe.Pointer {
	// 对齐到16字节边界
	alignedSize := Align16(size)

	sa.mutex.RLock()
	slab, exists := sa.slabs[alignedSize]
	sa.mutex.RUnlock()

	if !exists {
		// 创建新的slab
		slab = sa.createSlab(alignedSize)
	}

	return slab.Allocate()
}

// createSlab 创建新slab
func (sa *SlabAllocator) createSlab(objectSize uint32) *Slab {
	sa.mutex.Lock()
	defer sa.mutex.Unlock()

	// 双重检查
	if slab, exists := sa.slabs[objectSize]; exists {
		return slab
	}

	slab := &Slab{
		objectSize: objectSize,
		chunkSize:  sa.chunkSize,
		chunks:     make([]*SlabChunk, 0),
	}

	sa.slabs[objectSize] = slab
	sa.slabList = append(sa.slabList, slab)

	// 更新统计
	atomic.AddUint64(&sa.stats.TotalSlabs, 1)

	return slab
}

// Allocate Slab分配实现
func (slab *Slab) Allocate() unsafe.Pointer {
	slab.mutex.Lock()
	defer slab.mutex.Unlock()

	// 优先从部分使用的chunk分配
	if slab.partialList != nil {
		chunk := slab.partialList
		if chunk.freeList != nil {
			// 从空闲链表分配
			obj := chunk.freeList
			chunk.freeList = obj.next
			chunk.objectCount++

			// 如果chunk满了，移到full list
			if chunk.objectCount >= chunk.maxObjects {
				slab.removeFromPartial(chunk)
				slab.addToFull(chunk)
			}

			return unsafe.Pointer(obj)
		} else {
			// 从chunk的线性区域分配
			return slab.allocateFromChunk(chunk)
		}
	}

	// 从空chunk分配
	if slab.emptyList != nil {
		chunk := slab.emptyList
		slab.removeFromEmpty(chunk)
		slab.addToPartial(chunk)
		return slab.allocateFromChunk(chunk)
	}

	// 分配新chunk
	chunk := slab.allocateNewChunk()
	if chunk == nil {
		return nil // 内存不足
	}

	slab.addToPartial(chunk)
	return slab.allocateFromChunk(chunk)
}

// allocateFromChunk 从chunk分配对象
func (slab *Slab) allocateFromChunk(chunk *SlabChunk) unsafe.Pointer {
	if chunk.freeList != nil {
		// 从空闲链表获取
		obj := chunk.freeList
		chunk.freeList = obj.next
		chunk.objectCount++
		return unsafe.Pointer(obj)
	}

	if chunk.objectCount < chunk.maxObjects {
		// 从chunk的线性区域分配
		offset := uintptr(chunk.objectCount) * uintptr(slab.objectSize)
		ptr := unsafe.Pointer(uintptr(chunk.memory) + offset)
		chunk.objectCount++
		return ptr
	}

	return nil // chunk已满
}

// allocateNewChunk 分配新chunk
func (slab *Slab) allocateNewChunk() *SlabChunk {
	// 分配chunk内存
	memory := allocateAlignedMemory(slab.chunkSize)
	if memory == nil {
		return nil
	}

	maxObjects := int(slab.chunkSize) / int(slab.objectSize)
	chunk := &SlabChunk{
		memory:      memory,
		freeList:    nil,
		objectCount: 0,
		maxObjects:  maxObjects,
		next:        nil,
		prev:        nil,
	}

	slab.chunks = append(slab.chunks, chunk)
	return chunk
}

// Deallocate 释放对象
func (sa *SlabAllocator) Deallocate(ptr unsafe.Pointer, size uint32) {
	if ptr == nil {
		return
	}

	alignedSize := Align16(size)

	sa.mutex.RLock()
	slab, exists := sa.slabs[alignedSize]
	sa.mutex.RUnlock()

	if !exists {
		// 错误：试图释放未知大小的对象
		return
	}

	slab.Deallocate(ptr)
}

// Deallocate Slab释放实现
func (slab *Slab) Deallocate(ptr unsafe.Pointer) {
	slab.mutex.Lock()
	defer slab.mutex.Unlock()

	// 找到对象所属的chunk
	chunk := slab.findChunk(ptr)
	if chunk == nil {
		return // 找不到对应的chunk
	}

	// 将对象添加到chunk的空闲链表
	obj := (*FreeList)(ptr)
	obj.next = chunk.freeList
	chunk.freeList = obj
	chunk.objectCount--

	// 检查chunk状态变化
	if chunk.objectCount == 0 {
		// chunk变空，移到empty list
		slab.removeFromCurrent(chunk)
		slab.addToEmpty(chunk)
	} else if chunk.objectCount == chunk.maxObjects-1 {
		// chunk从满变为部分，移到partial list
		slab.removeFromFull(chunk)
		slab.addToPartial(chunk)
	}
}

// findChunk 查找对象所属的chunk
func (slab *Slab) findChunk(ptr unsafe.Pointer) *SlabChunk {
	ptrAddr := uintptr(ptr)

	for _, chunk := range slab.chunks {
		if chunk == nil {
			continue
		}

		chunkStart := uintptr(chunk.memory)
		chunkEnd := chunkStart + uintptr(slab.chunkSize)

		if ptrAddr >= chunkStart && ptrAddr < chunkEnd {
			return chunk
		}
	}

	return nil
}

// 链表操作函数
func (slab *Slab) removeFromPartial(chunk *SlabChunk) {
	if chunk.prev != nil {
		chunk.prev.next = chunk.next
	} else {
		slab.partialList = chunk.next
	}

	if chunk.next != nil {
		chunk.next.prev = chunk.prev
	}

	chunk.next = nil
	chunk.prev = nil
}

func (slab *Slab) addToPartial(chunk *SlabChunk) {
	chunk.next = slab.partialList
	chunk.prev = nil

	if slab.partialList != nil {
		slab.partialList.prev = chunk
	}

	slab.partialList = chunk
}

func (slab *Slab) removeFromEmpty(chunk *SlabChunk) {
	if chunk.prev != nil {
		chunk.prev.next = chunk.next
	} else {
		slab.emptyList = chunk.next
	}

	if chunk.next != nil {
		chunk.next.prev = chunk.prev
	}

	chunk.next = nil
	chunk.prev = nil
}

func (slab *Slab) addToEmpty(chunk *SlabChunk) {
	chunk.next = slab.emptyList
	chunk.prev = nil

	if slab.emptyList != nil {
		slab.emptyList.prev = chunk
	}

	slab.emptyList = chunk
}

func (slab *Slab) removeFromFull(chunk *SlabChunk) {
	if chunk.prev != nil {
		chunk.prev.next = chunk.next
	} else {
		slab.fullList = chunk.next
	}

	if chunk.next != nil {
		chunk.next.prev = chunk.prev
	}

	chunk.next = nil
	chunk.prev = nil
}

func (slab *Slab) addToFull(chunk *SlabChunk) {
	chunk.next = slab.fullList
	chunk.prev = nil

	if slab.fullList != nil {
		slab.fullList.prev = chunk
	}

	slab.fullList = chunk
}

func (slab *Slab) removeFromCurrent(chunk *SlabChunk) {
	// 从当前所在链表移除
	if chunk.prev != nil {
		chunk.prev.next = chunk.next
	}
	if chunk.next != nil {
		chunk.next.prev = chunk.prev
	}

	// 检查是否是链表头
	if slab.partialList == chunk {
		slab.partialList = chunk.next
	} else if slab.emptyList == chunk {
		slab.emptyList = chunk.next
	} else if slab.fullList == chunk {
		slab.fullList = chunk.next
	}

	chunk.next = nil
	chunk.prev = nil
}

// GetStats 获取统计信息
func (sa *SlabAllocator) GetStats() SlabStats {
	sa.mutex.RLock()
	defer sa.mutex.RUnlock()

	return SlabStats{
		TotalSlabs:     atomic.LoadUint64(&sa.stats.TotalSlabs),
		TotalChunks:    atomic.LoadUint64(&sa.stats.TotalChunks),
		ActiveChunks:   atomic.LoadUint64(&sa.stats.ActiveChunks),
		Allocations:    atomic.LoadUint64(&sa.stats.Allocations),
		Deallocations:  atomic.LoadUint64(&sa.stats.Deallocations),
		BytesAllocated: atomic.LoadUint64(&sa.stats.BytesAllocated),
		BytesFreed:     atomic.LoadUint64(&sa.stats.BytesFreed),
	}
}

// CompactMemory 内存压缩 - 释放完全空闲的chunk
func (sa *SlabAllocator) CompactMemory() int {
	sa.mutex.Lock()
	defer sa.mutex.Unlock()

	compactedChunks := 0

	for _, slab := range sa.slabList {
		if slab == nil {
			continue
		}

		slab.mutex.Lock()

		// 创建新的chunks数组，排除要释放的chunk
		var newChunks []*SlabChunk

		// 释放空闲的chunk
		current := slab.emptyList
		emptyChunks := make(map[*SlabChunk]bool)

		for current != nil {
			next := current.next
			emptyChunks[current] = true

			// 释放chunk内存
			freeAlignedMemory(current.memory, slab.chunkSize)
			compactedChunks++

			current = next
		}

		// 重建chunks数组，排除已释放的chunk
		for _, chunk := range slab.chunks {
			if chunk != nil && !emptyChunks[chunk] {
				newChunks = append(newChunks, chunk)
			}
		}

		slab.chunks = newChunks
		slab.emptyList = nil

		slab.mutex.Unlock()
	}

	return compactedChunks
}

// Destroy 销毁Slab分配器，释放所有内存
func (sa *SlabAllocator) Destroy() {
	sa.mutex.Lock()
	defer sa.mutex.Unlock()

	for _, slab := range sa.slabList {
		if slab == nil {
			continue
		}

		slab.mutex.Lock()

		// 释放所有chunk
		for _, chunk := range slab.chunks {
			if chunk != nil && chunk.memory != nil {
				freeAlignedMemory(chunk.memory, slab.chunkSize)
			}
		}

		slab.chunks = nil
		slab.partialList = nil
		slab.emptyList = nil
		slab.fullList = nil

		slab.mutex.Unlock()
	}

	sa.slabs = nil
	sa.slabList = nil
}
