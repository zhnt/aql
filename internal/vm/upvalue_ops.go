package vm

import "fmt"

// executeGetUpvalue 执行GET_UPVALUE指令: R(A) := Upvalue[B].Get()
func (e *Executor) executeGetUpvalue(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [GET_UPVALUE] A=%d, B=%d\n", inst.A, inst.B)
	fmt.Printf("DEBUG [GET_UPVALUE] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	// 检查栈帧upvalue状态
	if frame.Upvalues == nil {
		fmt.Printf("DEBUG [GET_UPVALUE] 错误: 栈帧upvalue为nil\n")
		return fmt.Errorf("no upvalues in current frame")
	}

	fmt.Printf("DEBUG [GET_UPVALUE] 栈帧upvalue数量: %d\n", len(frame.Upvalues))

	// 获取upvalue
	upvalue := frame.GetUpvalue(inst.B)
	if upvalue == nil {
		fmt.Printf("DEBUG [GET_UPVALUE] 错误: upvalue[%d]为nil\n", inst.B)
		return fmt.Errorf("invalid upvalue index: %d", inst.B)
	}

	fmt.Printf("DEBUG [GET_UPVALUE] upvalue[%d] 状态: IsClosed=%v, Name=%s\n",
		inst.B, upvalue.IsClosed, upvalue.Name)

	// 获取值
	value := upvalue.Get()
	// 只打印类型，避免String()方法的递归
	fmt.Printf("DEBUG [GET_UPVALUE] 获取到的值类型: %s\n", value.Type())

	// 存储到寄存器
	err := frame.SetRegister(inst.A, value)
	if err != nil {
		fmt.Printf("DEBUG [GET_UPVALUE] 存储寄存器错误: %v\n", err)
		return err
	}

	fmt.Printf("DEBUG [GET_UPVALUE] 成功存储到寄存器[%d]\n", inst.A)

	frame.PC++
	return nil
}

// executeSetUpvalue 执行SET_UPVALUE指令: Upvalue[B].Set(R(A))
func (e *Executor) executeSetUpvalue(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [SET_UPVALUE] A=%d, B=%d\n", inst.A, inst.B)
	fmt.Printf("DEBUG [SET_UPVALUE] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	// 检查栈帧upvalue状态
	if frame.Upvalues == nil {
		fmt.Printf("DEBUG [SET_UPVALUE] 错误: 栈帧upvalue为nil\n")
		return fmt.Errorf("no upvalues in current frame")
	}

	// 获取upvalue和新值
	upvalue := frame.GetUpvalue(inst.B)
	if upvalue == nil {
		fmt.Printf("DEBUG [SET_UPVALUE] 错误: upvalue[%d]为nil\n", inst.B)
		return fmt.Errorf("invalid upvalue index: %d", inst.B)
	}

	newValue := frame.GetRegister(inst.A)
	// 只打印类型，避免String()方法的递归
	fmt.Printf("DEBUG [SET_UPVALUE] 设置新值类型: %s\n", newValue.Type())
	fmt.Printf("DEBUG [SET_UPVALUE] upvalue[%d] 状态: IsClosed=%v, Name=%s\n",
		inst.B, upvalue.IsClosed, upvalue.Name)

	// 设置值
	upvalue.Set(newValue)

	fmt.Printf("DEBUG [SET_UPVALUE] 成功设置upvalue[%d]\n", inst.B)

	frame.PC++
	return nil
}

// executeCloseUpvalue 执行CLOSE_UPVALUE指令: Close upvalues >= A
func (e *Executor) executeCloseUpvalue(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [CLOSE_UPVALUE] A=%d\n", inst.A)
	fmt.Printf("DEBUG [CLOSE_UPVALUE] 当前栈帧: %s\n", frame.Function.Name)

	// 关闭指定索引及以上的所有upvalue
	if frame.Upvalues != nil {
		fmt.Printf("DEBUG [CLOSE_UPVALUE] 栈帧有%d个upvalue\n", len(frame.Upvalues))
		for i := inst.A; i < len(frame.Upvalues); i++ {
			if frame.Upvalues[i] != nil && !frame.Upvalues[i].IsClosed {
				fmt.Printf("DEBUG [CLOSE_UPVALUE] 关闭upvalue[%d]: %s\n",
					i, frame.Upvalues[i].Name)
				frame.Upvalues[i].Close()
			}
		}
	} else {
		fmt.Printf("DEBUG [CLOSE_UPVALUE] 栈帧没有upvalue\n")
	}

	frame.PC++
	return nil
}

// executeMakeClosureNew 重新实现MAKE_CLOSURE指令，使用新的Callable结构
func (e *Executor) executeMakeClosureNew(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [MAKE_CLOSURE] A=%d, B=%d, C=%d\n", inst.A, inst.B, inst.C)
	fmt.Printf("DEBUG [MAKE_CLOSURE] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	// 获取函数对象
	funcValue := frame.GetRegister(inst.B)
	if !funcValue.IsFunction() {
		fmt.Printf("DEBUG [MAKE_CLOSURE] 错误: 寄存器[%d]不是函数: %s\n",
			inst.B, funcValue.Type())
		return fmt.Errorf("expected function in MAKE_CLOSURE")
	}

	// 获取Function对象
	var targetFunc *Function
	if targetFunc = funcValue.AsFunction().(*Function); targetFunc == nil {
		fmt.Printf("DEBUG [MAKE_CLOSURE] 错误: 无法转换为Function\n")
		return fmt.Errorf("invalid function in MAKE_CLOSURE")
	}

	fmt.Printf("DEBUG [MAKE_CLOSURE] 目标函数: %s\n", targetFunc.Name)

	// 获取捕获变量数量
	captureCount := inst.C
	fmt.Printf("DEBUG [MAKE_CLOSURE] 需要捕获%d个变量\n", captureCount)

	// 创建upvalue数组
	upvalues := make([]*Upvalue, captureCount)

	// 获取捕获变量值并创建upvalue
	for i := 0; i < captureCount; i++ {
		// 重要修复：不要从寄存器获取捕获变量的值，因为寄存器可能被覆盖
		// 而是从指定的寄存器位置获取（这些位置在编译时已经确定）
		captureValue := frame.GetRegister(inst.B + 1 + i)

		// 只打印类型，避免String()方法的递归
		fmt.Printf("DEBUG [MAKE_CLOSURE] 捕获变量[%d] 类型: %s\n",
			i, captureValue.Type())

		// 创建upvalue（暂时关闭状态，后续可优化为指向栈）
		upvalue := &Upvalue{
			Stack:    nil,                          // 暂时不指向栈
			Value:    captureValue,                 // 直接存储值
			IsClosed: true,                         // 暂时设为关闭状态
			Name:     fmt.Sprintf("capture_%d", i), // 临时名称
		}
		upvalues[i] = upvalue

		fmt.Printf("DEBUG [MAKE_CLOSURE] 创建upvalue[%d]: Name=%s, IsClosed=%v\n",
			i, upvalue.Name, upvalue.IsClosed)
	}

	// 直接创建Callable ValueGC（使用新的统一系统）
	callableValue := NewCallableValueGC(targetFunc, upvalues)

	fmt.Printf("DEBUG [MAKE_CLOSURE] 创建Callable ValueGC成功\n")

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, callableValue)
	}

	err := frame.SetRegister(inst.A, callableValue)
	if err != nil {
		fmt.Printf("DEBUG [MAKE_CLOSURE] 存储寄存器错误: %v\n", err)
		return err
	}

	fmt.Printf("DEBUG [MAKE_CLOSURE] 成功存储Callable到寄存器[%d]\n", inst.A)

	frame.PC++
	return nil
}
