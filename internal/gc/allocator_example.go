package gc

import (
	"fmt"
	"time"
)

// ExampleAllocatorUsage 演示分配器使用
func ExampleAllocatorUsage() {
	fmt.Println("=== AQL Memory Allocator Demo ===")

	// 创建分配器
	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	fmt.Println("\n1. Small Objects (Size Classes):")
	demonstrateSmallObjects(allocator)

	fmt.Println("\n2. Medium Objects (Slab Allocator):")
	demonstrateMediumObjects(allocator)

	fmt.Println("\n3. Large Objects (Direct Allocator):")
	demonstrateLargeObjects(allocator)

	fmt.Println("\n4. Batch Operations:")
	demonstrateBatchOperations(allocator)

	fmt.Println("\n5. Performance Report:")
	fmt.Print(allocator.Report())

	fmt.Println("\n6. Memory Compaction:")
	compacted := allocator.CompactMemory()
	fmt.Printf("Compacted %d memory units\n", compacted)
}

func demonstrateSmallObjects(allocator *SimpleAllocator) {
	sizes := []uint32{16, 32, 48, 64, 96, 128, 192, 256}
	types := []ObjectType{
		ObjectTypeString, ObjectTypeFunction, ObjectTypeArray,
		ObjectTypeStruct, ObjectTypeClosure, ObjectTypeString,
		ObjectTypeArray, ObjectTypeStruct,
	}

	objects := make([]*GCObject, len(sizes))

	// 分配小对象
	for i, size := range sizes {
		start := time.Now()
		objects[i] = allocator.Allocate(size, types[i])
		duration := time.Since(start)

		if objects[i] != nil {
			fmt.Printf("  Size %3d bytes: allocated in %v (type: %v)\n",
				size, duration, types[i])
		}
	}

	// 释放小对象
	for _, obj := range objects {
		if obj != nil {
			allocator.Deallocate(obj)
		}
	}
}

func demonstrateMediumObjects(allocator *SimpleAllocator) {
	sizes := []uint32{512, 1024, 2048, 4096}

	for _, size := range sizes {
		start := time.Now()
		obj := allocator.Allocate(size, ObjectTypeArray)
		duration := time.Since(start)

		if obj != nil {
			fmt.Printf("  Size %4d bytes: allocated in %v\n", size, duration)
			allocator.Deallocate(obj)
		}
	}
}

func demonstrateLargeObjects(allocator *SimpleAllocator) {
	sizes := []uint32{8192, 16384, 32768, 65536}

	for _, size := range sizes {
		start := time.Now()
		obj := allocator.Allocate(size, ObjectTypeClosure)
		duration := time.Since(start)

		if obj != nil {
			fmt.Printf("  Size %5d bytes: allocated in %v\n", size, duration)
			allocator.Deallocate(obj)
		}
	}
}

func demonstrateBatchOperations(allocator *SimpleAllocator) {
	config := DefaultAllocatorConfig
	config.EnableBatchAllocation = true
	allocator.Configure(&config)

	// 分配100个对象
	const count = 100
	objects := make([]*GCObject, count)

	start := time.Now()
	for i := 0; i < count; i++ {
		objects[i] = allocator.Allocate(64, ObjectTypeString)
	}
	allocDuration := time.Since(start)

	// 批量释放
	start = time.Now()
	allocator.DeallocateBatch(objects)
	batchDuration := time.Since(start)

	fmt.Printf("  Allocated %d objects in %v\n", count, allocDuration)
	fmt.Printf("  Batch deallocated in %v\n", batchDuration)
	fmt.Printf("  Average alloc time: %v\n", allocDuration/count)
}

// ExampleConfigurationTuning 演示配置调优
func ExampleConfigurationTuning() {
	fmt.Println("=== Configuration Tuning Demo ===")

	// 测试不同配置 - 为每个配置创建副本
	configs := []AllocatorConfig{
		{
			EnableSmallAllocator:  true,
			EnableSlabAllocator:   true,
			EnableDirectAllocator: true,
			EnableFastPath:        true,
			EnableBatchAllocation: false,
			SmallObjectThreshold:  256,
			SlabChunkSize:         64 * 1024,
			MediumObjectThreshold: 4 * 1024,
			LargeObjectThreshold:  4 * 1024,
		},
		{
			EnableSmallAllocator:  true,
			EnableSlabAllocator:   false, // 禁用Slab分配器
			EnableDirectAllocator: true,
			EnableFastPath:        true,
			EnableBatchAllocation: true,
			SmallObjectThreshold:  256,
			SlabChunkSize:         64 * 1024,
			MediumObjectThreshold: 4 * 1024,
			LargeObjectThreshold:  4 * 1024,
		},
		{
			EnableSmallAllocator:  false, // 禁用小对象分配器
			EnableSlabAllocator:   true,
			EnableDirectAllocator: true,
			EnableFastPath:        false,
			EnableBatchAllocation: false,
			SmallObjectThreshold:  256,
			SlabChunkSize:         64 * 1024,
			MediumObjectThreshold: 4 * 1024,
			LargeObjectThreshold:  4 * 1024,
		},
	}

	for i, config := range configs {
		fmt.Printf("\nConfiguration %d:\n", i+1)
		fmt.Printf("  Small: %v, Slab: %v, Direct: %v\n",
			config.EnableSmallAllocator,
			config.EnableSlabAllocator,
			config.EnableDirectAllocator)
		fmt.Printf("  FastPath: %v, Batch: %v\n",
			config.EnableFastPath,
			config.EnableBatchAllocation)

		// 为每个配置创建独立的分配器
		allocator := NewSimpleAllocator(&config)

		// 测试分配性能
		start := time.Now()
		const testCount = 1000

		for j := 0; j < testCount; j++ {
			obj := allocator.Allocate(64, ObjectTypeString)
			if obj != nil {
				allocator.Deallocate(obj)
			}
		}

		duration := time.Since(start)
		fmt.Printf("  %d alloc/dealloc operations in %v\n", testCount, duration)
		fmt.Printf("  Average: %v per operation\n", duration/testCount)

		allocator.Destroy()
	}
}

// ExampleMemoryUsageMonitoring 演示内存使用监控
func ExampleMemoryUsageMonitoring() {
	fmt.Println("=== Memory Usage Monitoring Demo ===")

	allocator := NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 模拟不同工作负载
	workloads := []struct {
		name  string
		sizes []uint32
		count int
	}{
		{"Small objects", []uint32{32, 64, 96}, 50},
		{"Medium objects", []uint32{512, 1024}, 20},
		{"Large objects", []uint32{8192, 16384}, 5},
		{"Mixed workload", []uint32{32, 512, 8192}, 30},
	}

	for _, workload := range workloads {
		fmt.Printf("\n%s:\n", workload.name)

		var objects []*GCObject

		// 分配对象
		for i := 0; i < workload.count; i++ {
			size := workload.sizes[i%len(workload.sizes)]
			obj := allocator.Allocate(size, ObjectTypeArray)
			if obj != nil {
				objects = append(objects, obj)
			}
		}

		// 显示内存使用情况
		allocated, freed, live := allocator.GetMemoryUsage()
		fmt.Printf("  Allocated: %d bytes\n", allocated)
		fmt.Printf("  Freed: %d bytes\n", freed)
		fmt.Printf("  Live: %d bytes\n", live)

		stats := allocator.Stats()
		fmt.Printf("  Live objects: %d\n", stats.GetLiveObjects())
		fmt.Printf("  Fragmentation: %.2f%%\n", stats.GetFragmentationRatio()*100)

		// 释放一半对象
		half := len(objects) / 2
		for i := 0; i < half; i++ {
			allocator.Deallocate(objects[i])
		}

		allocated, freed, live = allocator.GetMemoryUsage()
		fmt.Printf("  After partial cleanup - Live: %d bytes\n", live)

		// 清理剩余对象
		for i := half; i < len(objects); i++ {
			allocator.Deallocate(objects[i])
		}
	}
}
