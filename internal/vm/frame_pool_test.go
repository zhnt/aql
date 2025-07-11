package vm

import (
	"testing"
)

// 测试栈帧池化优化效果

func BenchmarkFramePool_vs_DirectCreate(b *testing.B) {
	// 测试直接创建栈帧
	b.Run("DirectCreate", func(b *testing.B) {
		fn := NewFunction("test")
		fn.MaxStackSize = 16

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			frame := NewStackFrame(fn, nil, -1)
			_ = frame
		}
	})

	// 测试从池中获取栈帧
	b.Run("PoolCreate", func(b *testing.B) {
		fn := NewFunction("test")
		fn.MaxStackSize = 16

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			frame := NewStackFrameFromPool(fn, nil, -1)
			GlobalFramePool.Put(frame)
		}
	})
}

func BenchmarkOptimizedExecutor_vs_Original(b *testing.B) {
	// 创建测试函数：return 10 + 20
	fn := NewFunction("simple_add")
	fn.MaxStackSize = 8

	k1 := fn.AddNumberConstant(10)
	k2 := fn.AddNumberConstant(20)

	fn.AddInstructionBx(OP_LOADK, 0, k1)
	fn.AddInstructionBx(OP_LOADK, 1, k2)
	fn.AddInstruction(OP_ADD, 2, 0, 1)
	fn.AddInstruction(OP_RETURN, 2, 2, 0)

	// 测试原版执行器
	b.Run("OriginalExecutor", func(b *testing.B) {
		executor := NewExecutor()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := executor.Execute(fn, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// 测试优化版执行器
	b.Run("OptimizedExecutor", func(b *testing.B) {
		executor := NewOptimizedExecutor()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := executor.ExecuteOptimized(fn, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkInlineOptimization(b *testing.B) {
	// 测试内联优化效果：大量MOVE和ADD指令
	fn := NewFunction("inline_test")
	fn.MaxStackSize = 32

	// 创建多个常量
	for i := 0; i < 5; i++ {
		fn.AddNumberConstant(float64(i + 1))
	}

	// 大量LOADK操作：将常量加载到寄存器
	for i := 0; i < 5; i++ {
		fn.AddInstructionBx(OP_LOADK, i, i)
	}

	// 连续加法：累加所有数字
	fn.AddInstruction(OP_ADD, 10, 0, 1)  // R10 = R0 + R1
	fn.AddInstruction(OP_ADD, 11, 10, 2) // R11 = R10 + R2
	fn.AddInstruction(OP_ADD, 12, 11, 3) // R12 = R11 + R3
	fn.AddInstruction(OP_ADD, 13, 12, 4) // R13 = R12 + R4

	fn.AddInstruction(OP_RETURN, 13, 2, 0)

	// 原版执行器
	b.Run("WithoutInline", func(b *testing.B) {
		executor := NewExecutor()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := executor.Execute(fn, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// 内联优化版
	b.Run("WithInline", func(b *testing.B) {
		executor := NewOptimizedExecutor()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := executor.ExecuteOptimized(fn, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 功能正确性测试
func TestOptimizedExecutorCorrectness(t *testing.T) {
	t.Log("=== 优化版执行器功能验证 ===")

	// 创建复杂测试函数：f(x) = (x + 5) * (x - 3)
	fn := NewFunction("complex")
	fn.ParamCount = 1
	fn.MaxStackSize = 16

	k1 := fn.AddNumberConstant(5)
	k2 := fn.AddNumberConstant(3)

	fn.AddInstructionBx(OP_LOADK, 1, k1)  // R1 = 5
	fn.AddInstructionBx(OP_LOADK, 2, k2)  // R2 = 3
	fn.AddInstruction(OP_ADD, 3, 0, 1)    // R3 = x + 5
	fn.AddInstruction(OP_SUB, 4, 0, 2)    // R4 = x - 3
	fn.AddInstruction(OP_MUL, 5, 3, 4)    // R5 = (x+5) * (x-3)
	fn.AddInstruction(OP_RETURN, 5, 2, 0) // return R5

	// 测试原版执行器
	originalExecutor := NewExecutor()
	args := []Value{NewSmallIntValue(10)}
	result1, err1 := originalExecutor.Execute(fn, args)
	if err1 != nil {
		t.Fatalf("原版执行器错误: %v", err1)
	}

	// 测试优化版执行器
	optimizedExecutor := NewOptimizedExecutor()
	result2, err2 := optimizedExecutor.ExecuteOptimized(fn, args)
	if err2 != nil {
		t.Fatalf("优化版执行器错误: %v", err2)
	}

	// 验证结果一致性
	if len(result1) != len(result2) {
		t.Fatalf("返回值数量不匹配: %d vs %d", len(result1), len(result2))
	}

	expected := int32((10 + 5) * (10 - 3)) // 105

	if !result1[0].IsSmallInt() || result1[0].AsSmallInt() != expected {
		t.Errorf("原版执行器结果错误: 期望%d，得到%s", expected, result1[0].ToString())
	}

	if !result2[0].IsSmallInt() || result2[0].AsSmallInt() != expected {
		t.Errorf("优化版执行器结果错误: 期望%d，得到%s", expected, result2[0].ToString())
	}

	t.Log("✅ 优化版执行器功能验证通过")
	t.Logf("   原版结果: %s", result1[0].ToString())
	t.Logf("   优化版结果: %s", result2[0].ToString())
}

// 内存分配验证
func TestMemoryAllocationReduction(t *testing.T) {
	t.Log("=== 内存分配验证 ===")

	// 创建简单函数
	fn := NewFunction("simple")
	fn.MaxStackSize = 8

	k1 := fn.AddNumberConstant(42)
	fn.AddInstructionBx(OP_LOADK, 0, k1)
	fn.AddInstruction(OP_RETURN, 0, 2, 0)

	// 测试原版执行器的内存分配
	t.Run("OriginalAllocations", func(t *testing.T) {
		executor := NewExecutor()

		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = executor.Execute(fn, nil)
			}
		})

		t.Logf("原版执行器: %v", result)
	})

	// 测试优化版执行器的内存分配
	t.Run("OptimizedAllocations", func(t *testing.T) {
		executor := NewOptimizedExecutor()

		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = executor.ExecuteOptimized(fn, nil)
			}
		})

		t.Logf("优化版执行器: %v", result)
	})
}
