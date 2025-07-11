package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zhnt/aql/internal/gc"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: gc-demo <模式>")
		fmt.Println("模式:")
		fmt.Println("  basic       - 基础GC演示")
		fmt.Println("  refcount    - 引用计数GC演示")
		fmt.Println("  marksweep   - 标记清除GC演示")
		fmt.Println("  unified     - 统一GC管理器演示")
		fmt.Println("  stress      - 压力测试")
		fmt.Println("  monitor     - 实时监控")
		fmt.Println("  all         - 运行所有演示")
		return
	}

	mode := os.Args[1]

	switch mode {
	case "basic":
		demoBasicGC()
	case "refcount":
		demoRefCountGC()
	case "marksweep":
		demoMarkSweepGC()
	case "unified":
		demoUnifiedGC()
	case "stress":
		demoStressTest()
	case "monitor":
		demoMonitoring()
	case "all":
		runAllDemos()
	default:
		fmt.Printf("未知模式: %s\n", mode)
		os.Exit(1)
	}
}

func demoBasicGC() {
	fmt.Println("=== 基础GC演示 ===")

	// 创建分配器
	allocator := gc.NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 创建GC管理器
	gcManager := gc.NewUnifiedGCManager(allocator, nil)
	defer gcManager.Shutdown()

	fmt.Println("1. 创建对象...")

	// 创建一些对象
	objects := make([]*gc.GCObject, 10)
	for i := 0; i < 10; i++ {
		objects[i] = allocator.Allocate(64, gc.ObjectTypeString)
		if objects[i] != nil {
			gcManager.OnObjectAllocated(objects[i])
			fmt.Printf("   对象 %d: 大小=%d, 引用计数=%d\n",
				i, objects[i].Size(), objects[i].Header.RefCount())
		}
	}

	fmt.Println("\n2. 引用操作...")

	// 增加引用
	for i := 0; i < 5; i++ {
		if objects[i] != nil {
			gcManager.OnObjectReferenced(objects[i])
			fmt.Printf("   对象 %d 引用计数: %d\n",
				i, objects[i].Header.RefCount())
		}
	}

	fmt.Println("\n3. 解除引用...")

	// 解除引用
	for i := 0; i < 10; i++ {
		if objects[i] != nil {
			gcManager.OnObjectDereferenced(objects[i])
			fmt.Printf("   对象 %d 引用计数: %d\n",
				i, objects[i].Header.RefCount())
		}
	}

	fmt.Println("\n4. GC统计:")
	stats := gcManager.GetStats()
	printUnifiedStats(stats)
}

func demoRefCountGC() {
	fmt.Println("=== 引用计数GC演示 ===")

	allocator := gc.NewSimpleAllocator(nil)
	defer allocator.Destroy()

	refCountGC := gc.NewRefCountGC(allocator, nil)
	defer refCountGC.Shutdown()

	fmt.Println("1. 测试即时回收...")

	// 创建对象
	obj := allocator.Allocate(128, gc.ObjectTypeString)
	if obj != nil {
		fmt.Printf("   创建对象: 引用计数=%d\n", obj.Header.RefCount())

		// 增加引用
		refCountGC.IncRef(obj)
		fmt.Printf("   增加引用: 引用计数=%d\n", obj.Header.RefCount())

		// 减少到0
		refCountGC.DecRef(obj)
		refCountGC.DecRef(obj)
		fmt.Printf("   减少到0: 对象应被回收\n")
	}

	fmt.Println("\n2. 测试循环引用检测...")

	// 创建可能循环的对象
	cyclicObj := allocator.Allocate(96, gc.ObjectTypeArray)
	if cyclicObj != nil {
		cyclicObj.Header.SetCyclic()
		fmt.Printf("   创建循环对象: 引用计数=%d, 循环标志=%v\n",
			cyclicObj.Header.RefCount(), cyclicObj.Header.IsCyclic())

		// 减少到0
		refCountGC.DecRef(cyclicObj)
		fmt.Printf("   减少到0: 进入延迟清理队列\n")
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n3. RefCount GC统计:")
	stats := refCountGC.GetStats()
	printRefCountStats(stats)
}

func demoMarkSweepGC() {
	fmt.Println("=== 标记清除GC演示 ===")

	allocator := gc.NewSimpleAllocator(nil)
	defer allocator.Destroy()

	markSweepGC := gc.NewMarkSweepGC(allocator, nil)
	defer markSweepGC.Shutdown()

	fmt.Println("1. 创建对象图...")

	// 创建根对象
	root := allocator.Allocate(64, gc.ObjectTypeStruct)
	if root != nil {
		markSweepGC.TrackObject(root)
		markSweepGC.AddRootObject(root)
		fmt.Printf("   根对象创建: ID=%d\n", root.ID())
	}

	// 创建可达对象
	reachable := allocator.Allocate(48, gc.ObjectTypeString)
	if reachable != nil {
		markSweepGC.TrackObject(reachable)
		fmt.Printf("   可达对象创建: ID=%d\n", reachable.ID())
	}

	// 创建不可达对象
	unreachable := allocator.Allocate(32, gc.ObjectTypeString)
	if unreachable != nil {
		markSweepGC.TrackObject(unreachable)
		fmt.Printf("   不可达对象创建: ID=%d\n", unreachable.ID())
	}

	fmt.Printf("   跟踪对象数: %d\n", markSweepGC.GetTrackedObjectCount())
	fmt.Printf("   根对象数: %d\n", markSweepGC.GetRootObjectCount())

	fmt.Println("\n2. 执行标记清除...")

	initialStats := markSweepGC.GetStats()
	fmt.Printf("   GC前回收数: %d\n", initialStats.ObjectsCollected)

	markSweepGC.RunGC()

	finalStats := markSweepGC.GetStats()
	fmt.Printf("   GC后回收数: %d\n", finalStats.ObjectsCollected)
	fmt.Printf("   存活对象数: %d\n", finalStats.LiveObjects)

	fmt.Println("\n3. MarkSweep GC统计:")
	printMarkSweepStats(finalStats)
}

func demoUnifiedGC() {
	fmt.Println("=== 统一GC管理器演示 ===")

	allocator := gc.NewSimpleAllocator(nil)
	defer allocator.Destroy()

	// 自定义配置
	config := gc.DefaultUnifiedGCConfig
	config.MemoryPressureLimit = 2048 // 2KB
	config.FullGCInterval = 500 * time.Millisecond

	gcManager := gc.NewUnifiedGCManager(allocator, &config)
	defer gcManager.Shutdown()

	fmt.Println("1. 模拟真实工作负载...")

	objects := make([]*gc.GCObject, 50)

	// 创建对象
	for i := 0; i < 50; i++ {
		size := uint32(32 + i*8) // 变化的大小
		objType := gc.ObjectType(i % 5)

		objects[i] = allocator.Allocate(size, objType)
		if objects[i] != nil {
			gcManager.OnObjectAllocated(objects[i])

			// 一些对象标记为循环引用
			if i%7 == 0 {
				objects[i].Header.SetCyclic()
			}

			// 添加根对象
			if i%10 == 0 {
				gcManager.AddRootObject(objects[i])
			}
		}
	}

	fmt.Printf("   创建了 %d 个对象\n", len(objects))

	fmt.Println("\n2. 模拟引用操作...")

	// 增加引用
	for i := 0; i < 25; i++ {
		if objects[i] != nil {
			gcManager.OnObjectReferenced(objects[i])
		}
	}

	// 解除引用
	for i := 25; i < 50; i++ {
		if objects[i] != nil {
			gcManager.OnObjectDereferenced(objects[i])
		}
	}

	fmt.Println("\n3. 触发GC...")

	gcManager.TriggerGC()
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n4. 强制完整GC...")
	gcManager.ForceGC()

	fmt.Println("\n5. 统一GC统计:")
	stats := gcManager.GetStats()
	printUnifiedStats(stats)

	fmt.Printf("\n6. 内存使用情况:\n")
	allocated, freed, live := gcManager.GetMemoryUsage()
	fmt.Printf("   已分配: %d 字节\n", allocated)
	fmt.Printf("   已释放: %d 字节\n", freed)
	fmt.Printf("   存活: %d 字节\n", live)
	fmt.Printf("   GC效率: %.2f%%\n", gcManager.GetEfficiency()*100)
}

func demoStressTest() {
	fmt.Println("=== GC压力测试 ===")

	allocator := gc.NewSimpleAllocator(nil)
	defer allocator.Destroy()

	config := gc.DefaultUnifiedGCConfig
	config.MemoryPressureLimit = 1024 * 1024 // 1MB
	config.FullGCInterval = 100 * time.Millisecond

	gcManager := gc.NewUnifiedGCManager(allocator, &config)
	defer gcManager.Shutdown()

	fmt.Println("1. 大量对象分配...")

	startTime := time.Now()

	for i := 0; i < 10000; i++ {
		obj := allocator.Allocate(64, gc.ObjectTypeString)
		if obj != nil {
			gcManager.OnObjectAllocated(obj)

			// 随机引用/解除引用操作
			if i%3 == 0 {
				gcManager.OnObjectReferenced(obj)
			}
			if i%5 == 0 {
				gcManager.OnObjectDereferenced(obj)
			}
		}

		// 定期输出进度
		if i%1000 == 0 {
			fmt.Printf("   已处理: %d 对象\n", i)
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("   完成时间: %v\n", duration)

	fmt.Println("\n2. 最终统计:")
	stats := gcManager.GetStats()
	printUnifiedStats(stats)

	fmt.Printf("\n3. 性能指标:")
	fmt.Printf("   处理速度: %.0f 对象/秒\n", 10000.0/duration.Seconds())
	fmt.Printf("   平均GC时间: %v\n", time.Duration(stats.AverageGCTime))
	fmt.Printf("   最大暂停时间: %v\n", time.Duration(stats.MaxPauseTime))
}

func demoMonitoring() {
	fmt.Println("=== GC实时监控 ===")

	allocator := gc.NewSimpleAllocator(nil)
	defer allocator.Destroy()

	gcManager := gc.NewUnifiedGCManager(allocator, nil)
	defer gcManager.Shutdown()

	fmt.Println("开始5秒监控，Ctrl+C停止...")

	done := make(chan bool)

	// 监控线程
	go func() {
		for i := 0; i < 10; i++ { // 5秒，每500ms一次
			time.Sleep(500 * time.Millisecond)

			stats := gcManager.GetStats()
			allocated, freed, live := gcManager.GetMemoryUsage()

			fmt.Printf("\n时刻 %d:\n", i+1)
			fmt.Printf("  内存: 分配=%dB, 存活=%dB\n", allocated, live)
			fmt.Printf("  GC: 周期=%d, 回收=%d对象\n",
				stats.TotalGCCycles, stats.ObjectsCollected)
			fmt.Printf("  效率: RefCount=%.1f%%, MarkSweep=%.1f%%\n",
				stats.RefCountEfficiency*100, stats.MarkSweepEfficiency*100)
		}
		done <- true
	}()

	// 工作负载线程
	go func() {
		for i := 0; i < 1000; i++ {
			obj := allocator.Allocate(64, gc.ObjectTypeString)
			if obj != nil {
				gcManager.OnObjectAllocated(obj)
				gcManager.OnObjectReferenced(obj)

				// 随机延时
				time.Sleep(time.Duration(i%10) * time.Millisecond)

				gcManager.OnObjectDereferenced(obj)
			}
		}
	}()

	<-done
	fmt.Println("\n监控完成。")
}

func runAllDemos() {
	fmt.Println("=== 运行所有GC演示 ===\n")

	demos := []struct {
		name string
		fn   func()
	}{
		{"基础GC", demoBasicGC},
		{"引用计数GC", demoRefCountGC},
		{"标记清除GC", demoMarkSweepGC},
		{"统一GC管理器", demoUnifiedGC},
		{"压力测试", demoStressTest},
	}

	for i, demo := range demos {
		fmt.Printf(">>> 演示 %d: %s\n", i+1, demo.name)
		demo.fn()
		fmt.Println("\n" + strings.Repeat("-", 50) + "\n")
		time.Sleep(1 * time.Second) // 演示间隔
	}

	fmt.Println("所有演示完成！")
}

func printRefCountStats(stats gc.RefCountGCStats) {
	fmt.Printf("  回收对象数: %d\n", stats.ObjectsCollected)
	fmt.Printf("  即时回收: %d\n", stats.ImmediateCollections)
	fmt.Printf("  延迟回收: %d\n", stats.DeferredCollections)
	fmt.Printf("  引用操作: +%d / -%d\n",
		stats.IncRefOperations, stats.DecRefOperations)
	fmt.Printf("  零引用事件: %d\n", stats.ZeroRefEvents)
	fmt.Printf("  平均清理时间: %v\n", time.Duration(stats.AverageCleanupTime))
	fmt.Printf("  清理错误: %d\n", stats.CleanupErrors)
}

func printMarkSweepStats(stats gc.MarkSweepGCStats) {
	fmt.Printf("  GC周期数: %d\n", stats.GCCycles)
	fmt.Printf("  回收对象数: %d\n", stats.ObjectsCollected)
	fmt.Printf("  检测循环数: %d\n", stats.CyclesDetected)
	fmt.Printf("  总GC时间: %v\n", time.Duration(stats.TotalGCTime))
	fmt.Printf("  平均GC时间: %v\n", time.Duration(stats.AverageGCTime))
	fmt.Printf("  标记时间: %v\n", time.Duration(stats.MarkPhaseTime))
	fmt.Printf("  清除时间: %v\n", time.Duration(stats.SweepPhaseTime))
	fmt.Printf("  跟踪对象数: %d\n", stats.TotalTrackedObjects)
	fmt.Printf("  存活对象数: %d\n", stats.LiveObjects)
	fmt.Printf("  根对象数: %d\n", stats.RootObjects)
	fmt.Printf("  GC错误: %d\n", stats.GCErrors)
}

func printUnifiedStats(stats gc.UnifiedGCStats) {
	fmt.Printf("  总GC周期: %d (RefCount: %d, MarkSweep: %d)\n",
		stats.TotalGCCycles, stats.RefCountCycles, stats.MarkSweepCycles)
	fmt.Printf("  回收对象数: %d\n", stats.ObjectsCollected)
	fmt.Printf("  总GC时间: %v\n", time.Duration(stats.TotalGCTime))
	fmt.Printf("  平均GC时间: %v\n", time.Duration(stats.AverageGCTime))
	fmt.Printf("  最大暂停时间: %v\n", time.Duration(stats.MaxPauseTime))
	fmt.Printf("  堆大小: %d字节\n", stats.HeapSize)
	fmt.Printf("  存活对象: %d\n", stats.LiveObjects)
	fmt.Printf("  已分配: %d字节\n", stats.AllocatedBytes)
	fmt.Printf("  已释放: %d字节\n", stats.FreedBytes)
	fmt.Printf("  RefCount效率: %.1f%%\n", stats.RefCountEfficiency*100)
	fmt.Printf("  MarkSweep效率: %.1f%%\n", stats.MarkSweepEfficiency*100)
	fmt.Printf("  循环对象比例: %.1f%%\n", stats.CyclicObjectRatio*100)
	fmt.Printf("  GC错误: %d\n", stats.GCErrors)
}
