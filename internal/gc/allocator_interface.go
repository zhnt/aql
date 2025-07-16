package gc

// AQLAllocator AQL分配器主接口 - 借鉴Lua的简洁性
type AQLAllocator interface {
	// 分配GC对象
	Allocate(size uint32, objType ObjectType) *GCObject

	// 分配独立对象，避免内存复用（专门用于避免循环引用）
	AllocateIsolated(size uint32, objType ObjectType) *GCObject

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
