package vm

import (
	"testing"
)

// PC调试测试
func TestPCDebug(t *testing.T) {
	t.Log("=== PC调试测试 ===")

	// 创建被调函数：add10(x) = x + 10
	add10Fn := NewFunction("add10")
	add10Fn.ParamCount = 1
	add10Fn.MaxStackSize = 8

	k1 := add10Fn.AddNumberConstant(10)
	add10Fn.AddInstructionBx(OP_LOADK, 1, k1)  // R1 = 10
	add10Fn.AddInstruction(OP_ADD, 2, 0, 1)    // R2 = R0 + 10
	add10Fn.AddInstruction(OP_RETURN, 2, 2, 0) // return R2

	// 创建主函数
	mainFn := NewFunction("main")
	mainFn.MaxStackSize = 16

	add10Value := NewFunctionValue(add10Fn)
	mainFn.AddConstant(add10Value)
	k2 := mainFn.AddNumberConstant(5)

	mainFn.AddInstructionBx(OP_LOADK, 0, 0)   // PC=0: R0 = add10函数
	mainFn.AddInstructionBx(OP_LOADK, 1, k2)  // PC=1: R1 = 5
	mainFn.AddInstruction(OP_CALL, 0, 2, 2)   // PC=2: R0 = add10(5)
	mainFn.AddInstruction(OP_RETURN, 0, 2, 0) // PC=3: return R0

	// 手动模拟executeCallOptimized和executeReturnOptimized
	executor := NewOptimizedExecutor()
	executor.CurrentFrame = NewStackFrameFromPool(mainFn, nil, -1)
	executor.CallDepth = 1

	// 执行前两条LOADK指令
	t.Log("执行前两条LOADK指令...")
	executor.executeStepOptimized() // LOADK R0
	executor.executeStepOptimized() // LOADK R1

	t.Logf("执行LOADK后，PC=%d", executor.CurrentFrame.PC)
	t.Logf("R0: %s (IsFunction: %v)",
		executor.CurrentFrame.Registers[0].ToString(),
		executor.CurrentFrame.Registers[0].IsFunction())

	// 手动执行CALL指令
	t.Log("\n=== 手动执行CALL指令 ===")
	callFrame := executor.CurrentFrame
	callInst := callFrame.GetInstruction() // 应该是CALL指令

	t.Logf("CALL指令执行前: PC=%d", callFrame.PC)
	t.Logf("ReturnAddr应该设置为: %d", callFrame.PC+1)

	// 模拟executeCallOptimized
	funcValue := callFrame.GetRegister(callInst.A)
	targetFunc := funcValue.AsFunction()

	// 创建新栈帧，ReturnAddr = PC+1
	newFrame := NewStackFrameFromPool(targetFunc, callFrame, callFrame.PC+1)
	newFrame.ExpectedRets = callInst.C

	t.Logf("新栈帧ReturnAddr: %d", newFrame.ReturnAddr)

	// 设置参数
	argCount := callInst.B - 1
	for i := 0; i < argCount; i++ {
		newFrame.Registers[i] = callFrame.GetRegister(callInst.A + 1 + i)
	}

	// 切换栈帧
	executor.CurrentFrame = newFrame
	executor.CallDepth++

	t.Logf("切换后当前栈帧: %s (PC: %d)", newFrame.Function.Name, newFrame.PC)

	// 执行add10函数的所有指令
	t.Log("\n=== 执行add10函数 ===")
	for executor.CurrentFrame.Function.Name == "add10" {
		inst := executor.CurrentFrame.GetInstruction()
		t.Logf("add10执行: PC=%d, op=%d", executor.CurrentFrame.PC, inst.OpCode)

		if inst.OpCode == OP_RETURN {
			t.Log("即将执行RETURN指令，手动分析...")

			retFrame := executor.CurrentFrame
			caller := retFrame.Caller

			t.Logf("返回前: caller.PC=%d, caller.ReturnAddr=%d", caller.PC, caller.ReturnAddr)

			// 手动模拟executeReturnOptimized
			retCount := inst.B - 1
			if retCount < 0 {
				retCount = 1
			}

			// 设置返回值 - 使用正确的frame.ReturnAddr
			if retFrame.ReturnAddr > 0 {
				callInst := caller.Function.Instructions[retFrame.ReturnAddr-1]
				t.Logf("原CALL指令: A=%d, 将返回值设置到R%d", callInst.A, callInst.A)

				for i := 0; i < retCount && i < caller.ExpectedRets; i++ {
					returnVal := retFrame.GetRegister(inst.A + i)
					t.Logf("设置返回值[%d]: %s", i, returnVal.ToString())
					caller.SetRegister(callInst.A+i, returnVal)
				}

				// 恢复PC - 使用正确的frame.ReturnAddr
				t.Logf("设置caller.PC = retFrame.ReturnAddr = %d", retFrame.ReturnAddr)
				caller.PC = retFrame.ReturnAddr
			} else {
				// 从主函数返回的情况
				t.Log("从主函数返回，不修改PC")
			}

			// 切换栈帧
			executor.CurrentFrame = caller
			executor.CallDepth--

			// 归还栈帧
			GlobalFramePool.Put(retFrame)

			t.Logf("返回后: main.PC=%d", executor.CurrentFrame.PC)
			t.Logf("R0现在应该是返回值: %s (IsFunction: %v)",
				executor.CurrentFrame.Registers[0].ToString(),
				executor.CurrentFrame.Registers[0].IsFunction())

			break
		}

		executor.executeStepOptimized()
	}

	// 检查下一条指令
	t.Log("\n=== 检查下一条指令 ===")
	nextInst := executor.CurrentFrame.GetInstruction()
	t.Logf("下一条指令: PC=%d, op=%d (应该是OP_RETURN=6)",
		executor.CurrentFrame.PC, nextInst.OpCode)

	if nextInst.OpCode == OP_RETURN {
		t.Log("✅ PC正确指向RETURN指令")
	} else {
		t.Errorf("❌ PC错误，应该指向RETURN指令")
	}
}
