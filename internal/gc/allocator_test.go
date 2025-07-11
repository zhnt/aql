package gc

import (
	"testing"
	"unsafe"
)

func TestSimpleAllocator_Basic(t *testing.T) {
	// 创建分配器
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 测试小对象分配 (16字节)
	obj1 := allocator.Allocate(16, ObjectTypeString)
	if obj1 == nil {
		t.Fatal("Failed to allocate small object")
	}

	// 验证对象头
	if obj1.Type() != ObjectTypeString {
		t.Errorf("Expected ObjectTypeString, got %v", obj1.Type())
	}

	if obj1.Size() != 16 {
		t.Errorf("Expected size 16, got %d", obj1.Size())
	}

	// 释放对象
	allocator.Deallocate(obj1)

	// 验证统计信息
	stats := allocator.Stats()
	if stats.TotalAllocations != 1 {
		t.Errorf("Expected 1 allocation, got %d", stats.TotalAllocations)
	}
	if stats.TotalDeallocations != 1 {
		t.Errorf("Expected 1 deallocation, got %d", stats.TotalDeallocations)
	}
}

func TestSimpleAllocator_SizeClasses(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 测试所有size class
	testSizes := []uint32{16, 32, 48, 64, 96, 128, 192, 256}

	for _, size := range testSizes {
		obj := allocator.Allocate(size, ObjectTypeArray)
		if obj == nil {
			t.Fatalf("Failed to allocate object of size %d", size)
		}

		if obj.Size() != size {
			t.Errorf("Size mismatch: expected %d, got %d", size, obj.Size())
		}

		allocator.Deallocate(obj)
	}
}

func TestSimpleAllocator_MediumObjects(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 测试中等对象 (512字节)
	obj := allocator.Allocate(512, ObjectTypeStruct)
	if obj == nil {
		t.Fatal("Failed to allocate medium object")
	}

	if obj.Size() != 512 {
		t.Errorf("Expected size 512, got %d", obj.Size())
	}

	allocator.Deallocate(obj)
}

func TestSimpleAllocator_LargeObjects(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 测试大对象 (8KB)
	largeSize := uint32(8 * 1024)
	obj := allocator.Allocate(largeSize, ObjectTypeClosure)
	if obj == nil {
		t.Fatal("Failed to allocate large object")
	}

	if obj.Size() != largeSize {
		t.Errorf("Expected size %d, got %d", largeSize, obj.Size())
	}

	allocator.Deallocate(obj)
}

func TestSimpleAllocator_BatchDeallocation(t *testing.T) {
	config := DefaultAllocatorConfig
	config.EnableBatchAllocation = true

	allocator := NewSimpleAllocator(&config)
	defer allocator.Destroy()

	// 分配多个对象
	objects := make([]*GCObject, 10)
	for i := 0; i < 10; i++ {
		objects[i] = allocator.Allocate(64, ObjectTypeArray)
		if objects[i] == nil {
			t.Fatalf("Failed to allocate object %d", i)
		}
	}

	// 批量释放
	allocator.DeallocateBatch(objects)

	// 验证统计
	stats := allocator.Stats()
	if stats.TotalAllocations != 10 {
		t.Errorf("Expected 10 allocations, got %d", stats.TotalAllocations)
	}
	if stats.TotalDeallocations != 10 {
		t.Errorf("Expected 10 deallocations, got %d", stats.TotalDeallocations)
	}
}

func TestSimpleAllocator_CompactMemory(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 分配一些对象然后释放
	for i := 0; i < 100; i++ {
		obj := allocator.Allocate(64, ObjectTypeString)
		if obj != nil {
			allocator.Deallocate(obj)
		}
	}

	// 执行内存压缩
	compacted := allocator.CompactMemory()
	t.Logf("Compacted %d memory units", compacted)
}

func TestSimpleAllocator_MemoryAlignment(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 测试内存对齐
	obj := allocator.Allocate(17, ObjectTypeString) // 非对齐大小
	if obj == nil {
		t.Fatal("Failed to allocate object")
	}

	// 检查对象地址是否16字节对齐
	addr := uintptr(unsafe.Pointer(obj))
	if addr%16 != 0 {
		t.Errorf("Object not 16-byte aligned: address 0x%x", addr)
	}

	allocator.Deallocate(obj)
}

func TestSimpleAllocator_Configuration(t *testing.T) {
	// 测试禁用小对象分配器
	config := DefaultAllocatorConfig
	config.EnableSmallAllocator = false

	allocator := NewSimpleAllocator(&config)
	defer allocator.Destroy()

	// 尝试分配小对象
	obj := allocator.Allocate(64, ObjectTypeString)
	// 应该失败或使用其他分配器
	if obj != nil {
		allocator.Deallocate(obj)
	}
}

func TestSimpleAllocator_Stats(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 分配一些对象
	objects := make([]*GCObject, 5)
	for i := 0; i < 5; i++ {
		objects[i] = allocator.Allocate(64, ObjectTypeArray)
	}

	stats := allocator.Stats()
	if stats.TotalAllocations != 5 {
		t.Errorf("Expected 5 allocations, got %d", stats.TotalAllocations)
	}

	// 释放部分对象
	allocator.Deallocate(objects[0])
	allocator.Deallocate(objects[1])

	stats = allocator.Stats()
	if stats.TotalDeallocations != 2 {
		t.Errorf("Expected 2 deallocations, got %d", stats.TotalDeallocations)
	}
	if stats.GetLiveObjects() != 3 {
		t.Errorf("Expected 3 live objects, got %d", stats.GetLiveObjects())
	}

	// 清理剩余对象
	for i := 2; i < 5; i++ {
		allocator.Deallocate(objects[i])
	}
}

func TestSimpleAllocator_Report(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 分配各种大小的对象
	obj1 := allocator.Allocate(32, ObjectTypeString)
	obj2 := allocator.Allocate(128, ObjectTypeArray)
	obj3 := allocator.Allocate(1024, ObjectTypeStruct)
	obj4 := allocator.Allocate(8192, ObjectTypeClosure)

	// 生成报告
	report := allocator.Report()
	if len(report) == 0 {
		t.Error("Empty report generated")
	}

	t.Logf("Allocator Report:\n%s", report)

	// 清理
	allocator.Deallocate(obj1)
	allocator.Deallocate(obj2)
	allocator.Deallocate(obj3)
	allocator.Deallocate(obj4)
}

func BenchmarkSimpleAllocator_SmallObjects(b *testing.B) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		obj := allocator.Allocate(64, ObjectTypeString)
		if obj != nil {
			allocator.Deallocate(obj)
		}
	}
}

func BenchmarkSimpleAllocator_MediumObjects(b *testing.B) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		obj := allocator.Allocate(1024, ObjectTypeArray)
		if obj != nil {
			allocator.Deallocate(obj)
		}
	}
}

func BenchmarkSimpleAllocator_LargeObjects(b *testing.B) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		obj := allocator.Allocate(8192, ObjectTypeStruct)
		if obj != nil {
			allocator.Deallocate(obj)
		}
	}
}

func BenchmarkSimpleAllocator_Mixed(b *testing.B) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	sizes := []uint32{32, 128, 512, 2048}
	types := []ObjectType{ObjectTypeString, ObjectTypeArray, ObjectTypeStruct, ObjectTypeClosure}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx := i % len(sizes)
		obj := allocator.Allocate(sizes[idx], types[idx])
		if obj != nil {
			allocator.Deallocate(obj)
		}
	}
}
