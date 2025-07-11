package vm

import (
	"testing"
)

// 逐步执行调试测试
func TestStepByStepExecution(t *testing.T) {
	t.Log("=== 逐步执行调试测试 ===")

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

	// 使用优化版执行器，但手动控制执行步骤
	executor := NewOptimizedExecutor()
	executor.CurrentFrame = NewStackFrameFromPool(mainFn, nil, -1)
	executor.CallDepth = 1

	stepCount := 0
	maxSteps := 20 // 防止无限循环

	for executor.CurrentFrame != nil && stepCount < maxSteps {
		frame := executor.CurrentFrame
		inst := frame.GetInstruction()

		t.Logf("\n=== 步骤 %d ===", stepCount+1)
		t.Logf("当前栈帧: %s (PC: %d)", frame.Function.Name, frame.PC)
		t.Logf("指令: op=%d, A=%d, B=%d, C=%d, Bx=%d",
			inst.OpCode, inst.A, inst.B, inst.C, inst.Bx)

		// 在执行前检查相关寄存器状态
		if inst.OpCode == OP_LOADK {
			t.Logf("LOADK: 加载常量[%d]到R%d", inst.Bx, inst.A)
			const_val := frame.GetConstant(inst.Bx)
			t.Logf("常量值: type=%s, IsFunction=%v", const_val.Type(), const_val.IsFunction())
		} else if inst.OpCode == OP_CALL {
			t.Logf("CALL: 调用R%d(R%d, ..., R%d)", inst.A, inst.A+1, inst.A+inst.B-1)
			funcVal := frame.GetRegister(inst.A)
			t.Logf("函数值: type=%s, IsFunction=%v", funcVal.Type(), funcVal.IsFunction())

			for i := 0; i < inst.B; i++ {
				argVal := frame.GetRegister(inst.A + i)
				t.Logf("  arg[%d] R%d: type=%s, value=%s",
					i, inst.A+i, argVal.Type(), argVal.ToString())
			}
		}

		// 执行这一步
		err := executor.executeStepOptimized()
		if err != nil {
			t.Fatalf("执行错误在步骤%d: %v", stepCount+1, err)
		}

		// 检查执行后的状态
		if inst.OpCode == OP_LOADK {
			afterVal := frame.GetRegister(inst.A)
			t.Logf("执行后R%d: type=%s, IsFunction=%v",
				inst.A, afterVal.Type(), afterVal.IsFunction())
		}

		stepCount++

		// 如果切换了栈帧，报告新栈帧
		if executor.CurrentFrame != frame {
			if executor.CurrentFrame != nil {
				t.Logf("切换到新栈帧: %s", executor.CurrentFrame.Function.Name)
			} else {
				t.Logf("执行完成，栈帧为空")
			}
		}
	}

	if stepCount >= maxSteps {
		t.Errorf("执行步数超过限制，可能存在无限循环")
	} else {
		t.Logf("✅ 执行完成，总步数: %d", stepCount)
	}
}

// 测试原版执行器作为对比
func TestOriginalExecutorComparison(t *testing.T) {
	t.Log("=== 原版执行器对比测试 ===")

	// 创建相同的函数
	add10Fn := NewFunction("add10")
	add10Fn.ParamCount = 1
	add10Fn.MaxStackSize = 8

	k1 := add10Fn.AddNumberConstant(10)
	add10Fn.AddInstructionBx(OP_LOADK, 1, k1)
	add10Fn.AddInstruction(OP_ADD, 2, 0, 1)
	add10Fn.AddInstruction(OP_RETURN, 2, 2, 0)

	mainFn := NewFunction("main")
	mainFn.MaxStackSize = 16

	add10Value := NewFunctionValue(add10Fn)
	mainFn.AddConstant(add10Value)
	k2 := mainFn.AddNumberConstant(5)

	mainFn.AddInstructionBx(OP_LOADK, 0, 0)
	mainFn.AddInstructionBx(OP_LOADK, 1, k2)
	mainFn.AddInstruction(OP_CALL, 0, 2, 2)
	mainFn.AddInstruction(OP_RETURN, 0, 2, 0)

	// 测试原版执行器
	executor := NewExecutor()
	result, err := executor.Execute(mainFn, nil)
	if err != nil {
		t.Fatalf("原版执行器错误: %v", err)
	}

	expected := int32(5 + 10) // 15
	if !result[0].IsSmallInt() || result[0].AsSmallInt() != expected {
		t.Errorf("原版执行器结果错误: 期望%d，得到%s", expected, result[0].ToString())
	} else {
		t.Logf("✅ 原版执行器工作正常: %s", result[0].ToString())
	}
}

// 测试栈帧池的重置行为
func TestFramePoolReset(t *testing.T) {
	t.Log("=== 栈帧池重置测试 ===")

	// 创建一个简单函数
	testFn := NewFunction("test")
	testFn.MaxStackSize = 8
	testFn.AddNumberConstant(42)
	testFn.AddInstructionBx(OP_LOADK, 0, 0)
	testFn.AddInstruction(OP_RETURN, 0, 2, 0)

	// 从池中获取栈帧
	frame1 := NewStackFrameFromPool(testFn, nil, -1)
	t.Logf("第一次获取栈帧: %p", frame1)

	// 设置一些值
	frame1.Registers[0] = NewNumberValue(123)
	frame1.Registers[1] = NewStringValue("test")
	frame1.PC = 5

	t.Logf("设置值后: R0=%s, R1=%s, PC=%d",
		frame1.Registers[0].ToString(), frame1.Registers[1].ToString(), frame1.PC)

	// 归还到池中
	GlobalFramePool.Put(frame1)

	// 再次获取栈帧
	frame2 := NewStackFrameFromPool(testFn, nil, -1)
	t.Logf("第二次获取栈帧: %p", frame2)

	// 检查是否被正确重置
	t.Logf("重置后: R0=%s, R1=%s, PC=%d",
		frame2.Registers[0].ToString(), frame2.Registers[1].ToString(), frame2.PC)

	if frame2.PC != 0 {
		t.Errorf("PC未被重置: 期望0，得到%d", frame2.PC)
	}

	if !frame2.Registers[0].IsNil() || !frame2.Registers[1].IsNil() {
		t.Errorf("寄存器未被重置为nil")
	}

	// 测试常量获取
	const0 := frame2.GetConstant(0)
	t.Logf("常量获取测试: type=%s, value=%s", const0.Type(), const0.ToString())

	GlobalFramePool.Put(frame2)
}
