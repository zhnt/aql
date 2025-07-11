package gc

import (
	"testing"
	"time"
)

func TestRefCountGC_Basic(t *testing.T) {
	// 创建分配器
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 创建引用计数GC
	refCountGC := NewRefCountGC(allocator, nil)
	defer refCountGC.Shutdown()

	// 创建测试对象
	obj := allocator.Allocate(64, ObjectTypeString)
	if obj == nil {
		t.Fatal("Failed to allocate object")
	}

	// 测试引用计数操作
	initialCount := obj.Header.RefCount()
	if initialCount != 1 {
		t.Errorf("Expected initial ref count 1, got %d", initialCount)
	}

	// 增加引用
	refCountGC.IncRef(obj)
	if obj.Header.RefCount() != 2 {
		t.Errorf("Expected ref count 2, got %d", obj.Header.RefCount())
	}

	// 减少引用
	refCountGC.DecRef(obj)
	if obj.Header.RefCount() != 1 {
		t.Errorf("Expected ref count 1, got %d", obj.Header.RefCount())
	}

	// 检查统计
	stats := refCountGC.GetStats()
	if stats.IncRefOperations == 0 {
		t.Error("Expected inc ref operations > 0")
	}
	if stats.DecRefOperations == 0 {
		t.Error("Expected dec ref operations > 0")
	}
}

func TestRefCountGC_ZeroRefCleanup(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	refCountGC := NewRefCountGC(allocator, nil)
	defer refCountGC.Shutdown()

	// 创建非循环对象
	obj := allocator.Allocate(32, ObjectTypeString)
	if obj == nil {
		t.Fatal("Failed to allocate object")
	}

	initialCollected := refCountGC.GetStats().ObjectsCollected

	// 减少引用计数到0，应该触发即时清理
	refCountGC.DecRef(obj)

	// 等待一小段时间确保异步清理完成
	time.Sleep(10 * time.Millisecond)

	stats := refCountGC.GetStats()
	if stats.ObjectsCollected <= initialCollected {
		t.Error("Expected object to be collected")
	}
	if stats.ImmediateCollections == 0 {
		t.Error("Expected immediate collection")
	}
}

func TestRefCountGC_CyclicObject(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	config := DefaultRefCountGCConfig
	config.EnableDeferredCleanup = true
	refCountGC := NewRefCountGC(allocator, &config)
	defer refCountGC.Shutdown()

	// 创建可能有循环引用的对象
	obj := allocator.Allocate(48, ObjectTypeArray)
	if obj == nil {
		t.Fatal("Failed to allocate object")
	}

	// 标记为循环引用
	obj.Header.SetCyclic()

	initialDeferred := refCountGC.GetStats().DeferredCollections

	// 减少引用计数到0，应该进入延迟清理队列
	refCountGC.DecRef(obj)

	// 等待延迟清理
	time.Sleep(50 * time.Millisecond)

	stats := refCountGC.GetStats()
	if stats.DeferredCollections <= initialDeferred {
		t.Error("Expected deferred collection")
	}
}

func TestMarkSweepGC_Basic(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	markSweepGC := NewMarkSweepGC(allocator, nil)
	defer markSweepGC.Shutdown()

	// 创建测试对象
	obj1 := allocator.Allocate(64, ObjectTypeStruct)
	obj2 := allocator.Allocate(96, ObjectTypeClosure)

	if obj1 == nil || obj2 == nil {
		t.Fatal("Failed to allocate objects")
	}

	// 添加到跟踪列表
	markSweepGC.TrackObject(obj1)
	markSweepGC.TrackObject(obj2)

	// 添加根对象
	markSweepGC.AddRootObject(obj1)

	// 检查跟踪状态
	if markSweepGC.GetTrackedObjectCount() != 2 {
		t.Errorf("Expected 2 tracked objects, got %d", markSweepGC.GetTrackedObjectCount())
	}
	if markSweepGC.GetRootObjectCount() != 1 {
		t.Errorf("Expected 1 root object, got %d", markSweepGC.GetRootObjectCount())
	}
}

func TestMarkSweepGC_Collection(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	markSweepGC := NewMarkSweepGC(allocator, nil)
	defer markSweepGC.Shutdown()

	// 创建对象
	root := allocator.Allocate(64, ObjectTypeStruct)
	orphan := allocator.Allocate(32, ObjectTypeString)

	if root == nil || orphan == nil {
		t.Fatal("Failed to allocate objects")
	}

	// 跟踪对象
	markSweepGC.TrackObject(root)
	markSweepGC.TrackObject(orphan)

	// 只有root是根对象
	markSweepGC.AddRootObject(root)

	initialCollected := markSweepGC.GetStats().ObjectsCollected

	// 运行GC
	markSweepGC.RunGC()

	stats := markSweepGC.GetStats()
	if stats.ObjectsCollected <= initialCollected {
		t.Error("Expected objects to be collected")
	}
	if stats.GCCycles == 0 {
		t.Error("Expected GC cycles > 0")
	}
}

func TestUnifiedGCManager_Basic(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 创建统一GC管理器
	gcManager := NewUnifiedGCManager(allocator, nil)
	defer gcManager.Shutdown()

	// 检查初始状态
	if !gcManager.IsEnabled() {
		t.Error("Expected GC to be enabled by default")
	}

	// 创建对象并通知GC
	obj := allocator.Allocate(64, ObjectTypeString)
	if obj == nil {
		t.Fatal("Failed to allocate object")
	}

	gcManager.OnObjectAllocated(obj)

	// 检查统计
	stats := gcManager.GetStats()
	if stats.AllocatedBytes == 0 {
		t.Error("Expected allocated bytes > 0")
	}

	// 测试引用操作
	gcManager.OnObjectReferenced(obj)
	gcManager.OnObjectDereferenced(obj)

	// 检查引用计数GC组件
	refCountGC := gcManager.GetRefCountGC()
	if refCountGC == nil {
		t.Error("Expected RefCountGC component")
	}

	// 检查标记清除GC组件
	markSweepGC := gcManager.GetMarkSweepGC()
	if markSweepGC == nil {
		t.Error("Expected MarkSweepGC component")
	}
}

func TestUnifiedGCManager_Integration(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	gcManager := NewUnifiedGCManager(allocator, nil)
	defer gcManager.Shutdown()

	// 创建多个对象
	objects := make([]*GCObject, 10)
	for i := 0; i < 10; i++ {
		objects[i] = allocator.Allocate(32, ObjectTypeString)
		if objects[i] != nil {
			gcManager.OnObjectAllocated(objects[i])
		}
	}

	// 添加根对象
	if objects[0] != nil {
		gcManager.AddRootObject(objects[0])
	}

	// 触发GC
	gcManager.TriggerGC()

	// 等待GC完成
	time.Sleep(100 * time.Millisecond)

	// 检查统计
	stats := gcManager.GetStats()
	if stats.TotalGCCycles == 0 {
		t.Error("Expected GC cycles > 0")
	}

	// 测试效率
	efficiency := gcManager.GetEfficiency()
	if efficiency < 0 || efficiency > 1 {
		t.Errorf("Expected efficiency between 0 and 1, got %f", efficiency)
	}
}

func TestUnifiedGCManager_Configuration(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 自定义配置
	config := DefaultUnifiedGCConfig
	config.MemoryPressureLimit = 1024
	config.ObjectCountLimit = 100

	gcManager := NewUnifiedGCManager(allocator, &config)
	defer gcManager.Shutdown()

	// 测试禁用/启用
	gcManager.Disable()
	if gcManager.IsEnabled() {
		t.Error("Expected GC to be disabled")
	}

	gcManager.Enable()
	if !gcManager.IsEnabled() {
		t.Error("Expected GC to be enabled")
	}

	// 测试强制GC
	gcManager.ForceGC()

	stats := gcManager.GetStats()
	if stats.TotalGCCycles == 0 {
		t.Error("Expected forced GC to increment cycles")
	}
}

func BenchmarkRefCountGC_IncRef(b *testing.B) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	refCountGC := NewRefCountGC(allocator, nil)
	defer refCountGC.Shutdown()

	obj := allocator.Allocate(64, ObjectTypeString)
	if obj == nil {
		b.Fatal("Failed to allocate object")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		refCountGC.IncRef(obj)
		refCountGC.DecRef(obj)
	}
}

func BenchmarkUnifiedGCManager_ObjectOperations(b *testing.B) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	gcManager := NewUnifiedGCManager(allocator, nil)
	defer gcManager.Shutdown()

	objects := make([]*GCObject, 1000)
	for i := 0; i < 1000; i++ {
		objects[i] = allocator.Allocate(64, ObjectTypeString)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		obj := objects[i%1000]
		if obj != nil {
			gcManager.OnObjectAllocated(obj)
			gcManager.OnObjectReferenced(obj)
			gcManager.OnObjectDereferenced(obj)
		}
	}
}

func TestGCIntegration_MemoryPressure(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 设置低内存阈值
	config := DefaultUnifiedGCConfig
	config.MemoryPressureLimit = 1024 // 1KB
	config.FullGCInterval = 10 * time.Millisecond

	gcManager := NewUnifiedGCManager(allocator, &config)
	defer gcManager.Shutdown()

	// 分配大量对象触发内存压力
	for i := 0; i < 100; i++ {
		obj := allocator.Allocate(32, ObjectTypeString)
		if obj != nil {
			gcManager.OnObjectAllocated(obj)
		}
	}

	// 等待GC触发
	time.Sleep(50 * time.Millisecond)

	stats := gcManager.GetStats()
	if stats.TotalGCCycles == 0 {
		t.Error("Expected memory pressure to trigger GC")
	}
}

func TestGC_ErrorHandling(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	gcManager := NewUnifiedGCManager(allocator, nil)
	defer gcManager.Shutdown()

	// 测试nil对象处理
	gcManager.OnObjectAllocated(nil)
	gcManager.OnObjectReferenced(nil)
	gcManager.OnObjectDereferenced(nil)
	gcManager.OnObjectFreed(nil)
	gcManager.AddRootObject(nil)
	gcManager.RemoveRootObject(nil)

	// 应该没有panic或错误
}

func TestGC_ConcurrentAccess(t *testing.T) {
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	gcManager := NewUnifiedGCManager(allocator, nil)
	defer gcManager.Shutdown()

	// 并发分配对象
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				obj := allocator.Allocate(32, ObjectTypeString)
				if obj != nil {
					gcManager.OnObjectAllocated(obj)
					gcManager.OnObjectReferenced(obj)
					gcManager.OnObjectDereferenced(obj)
				}
			}
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := gcManager.GetStats()
	if stats.AllocatedBytes == 0 {
		t.Error("Expected allocated bytes > 0")
	}
}
