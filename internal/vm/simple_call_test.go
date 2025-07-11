package vm

import (
	"testing"
)

// 简单的函数调用测试，用于调试执行器问题
func TestSimpleCall(t *testing.T) {
	t.Log("=== 简单函数调用测试 ===")

	// 创建被调函数：add10(x) = x + 10
	add10Fn := NewFunction("add10")
	add10Fn.ParamCount = 1
	add10Fn.MaxStackSize = 8

	k1 := add10Fn.AddNumberConstant(10)
	add10Fn.AddInstructionBx(OP_LOADK, 1, k1)  // R1 = 10
	add10Fn.AddInstruction(OP_ADD, 2, 0, 1)    // R2 = R0 + 10
	add10Fn.AddInstruction(OP_RETURN, 2, 2, 0) // return R2

	// 创建主函数：main() = add10(5)
	mainFn := NewFunction("main")
	mainFn.MaxStackSize = 16

	add10Value := NewFunctionValue(add10Fn)
	mainFn.AddConstant(add10Value)
	k2 := mainFn.AddNumberConstant(5)

	mainFn.AddInstructionBx(OP_LOADK, 0, 0)   // R0 = add10函数
	mainFn.AddInstructionBx(OP_LOADK, 1, k2)  // R1 = 5
	mainFn.AddInstruction(OP_CALL, 0, 2, 2)   // R0 = add10(5)
	mainFn.AddInstruction(OP_RETURN, 0, 2, 0) // return R0

	// 测试优化版执行器
	executor := NewOptimizedExecutor()
	result, err := executor.ExecuteOptimized(mainFn, nil)
	if err != nil {
		t.Fatalf("执行错误: %v", err)
	}

	expected := int32(5 + 10) // 15
	if !result[0].IsSmallInt() || result[0].AsSmallInt() != expected {
		t.Errorf("结果错误: 期望%d，得到%s", expected, result[0].ToString())
	}

	t.Logf("✅ 简单函数调用测试通过")
	t.Logf("   结果: %s (期望: %d)", result[0].ToString(), expected)
}
