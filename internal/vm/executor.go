package vm

import (
	"fmt"
	"unsafe"
)

// Executor AQL虚拟机执行器（GC优化版）
type Executor struct {
	CurrentFrame *StackFrame
	MaxCallDepth int
	CallDepth    int
	Globals      []ValueGC // 全局变量存储

	// GC 优化组件
	gcOptimizer *GCOptimizer // GC优化器
	enableGCOpt bool         // 是否启用GC优化
}

// NewExecutor 创建新的执行器
func NewExecutor() *Executor {
	executor := &Executor{
		CurrentFrame: nil,
		MaxCallDepth: 1000,
		CallDepth:    0,
		Globals:      make([]ValueGC, 0, 256), // 初始化全局变量存储，预分配256个位置
		enableGCOpt:  true,                    // 默认启用GC优化
	}

	// 初始化GC优化器
	executor.gcOptimizer = NewGCOptimizer(executor, &DefaultGCOptimizerConfig)

	return executor
}

// NewExecutorWithGCConfig 创建带自定义GC配置的执行器
func NewExecutorWithGCConfig(gcConfig *GCOptimizerConfig) *Executor {
	executor := &Executor{
		CurrentFrame: nil,
		MaxCallDepth: 1000,
		CallDepth:    0,
		Globals:      make([]ValueGC, 0, 256),
		enableGCOpt:  true,
	}

	executor.gcOptimizer = NewGCOptimizer(executor, gcConfig)
	return executor
}

// DisableGCOptimization 禁用GC优化
func (e *Executor) DisableGCOptimization() {
	e.enableGCOpt = false
}

// EnableGCOptimization 启用GC优化
func (e *Executor) EnableGCOptimization() {
	e.enableGCOpt = true
}

// GetGCOptimizer 获取GC优化器
func (e *Executor) GetGCOptimizer() *GCOptimizer {
	return e.gcOptimizer
}

// Execute 执行函数
func (e *Executor) Execute(function *Function, args []ValueGC) ([]ValueGC, error) {
	// 创建主函数栈帧
	mainFrame := NewStackFrame(function, nil, -1)
	mainFrame.SetParameters(args)

	e.CurrentFrame = mainFrame
	e.CallDepth = 1

	// 执行主循环
	for e.CurrentFrame != nil {
		err := e.executeStep()
		if err != nil {
			return nil, err
		}
	}

	// 返回主函数的结果
	if mainFrame.Registers != nil && len(mainFrame.Registers) > 0 {
		return []ValueGC{mainFrame.Registers[0]}, nil
	}

	return []ValueGC{NewNilValueGC()}, nil
}

// executeStep 执行单步指令
func (e *Executor) executeStep() error {
	frame := e.CurrentFrame
	if frame == nil {
		return fmt.Errorf("no current frame")
	}

	// GC优化：定期检查并触发GC
	if e.enableGCOpt && e.gcOptimizer != nil {
		e.gcOptimizer.CheckAndTriggerGC()
	}

	instruction := frame.GetInstruction()

	switch instruction.OpCode {
	case OP_MOVE:
		return e.executeMove(instruction)
	case OP_LOADK:
		return e.executeLoadK(instruction)
	case OP_ADD:
		return e.executeAdd(instruction)
	case OP_SUB:
		return e.executeSub(instruction)
	case OP_MUL:
		return e.executeMul(instruction)
	case OP_DIV:
		return e.executeDIV(instruction)
	case OP_MOD:
		return e.executeMOD(instruction)
	case OP_POP:
		return e.executePop(instruction)
	case OP_JUMP:
		return e.executeJump(instruction)
	case OP_JUMP_IF_FALSE:
		return e.executeJumpIfFalse(instruction)
	case OP_JUMP_IF_TRUE:
		return e.executeJumpIfTrue(instruction)
	case OP_GET_GLOBAL:
		return e.executeGetGlobal(instruction)
	case OP_SET_GLOBAL:
		return e.executeSetGlobal(instruction)
	case OP_GET_LOCAL:
		return e.executeGetLocal(instruction)
	case OP_SET_LOCAL:
		return e.executeSetLocal(instruction)
	case OP_EQ:
		return e.executeEQ(instruction)
	case OP_NEQ:
		return e.executeNEQ(instruction)
	case OP_LT:
		return e.executeLT(instruction)
	case OP_GT:
		return e.executeGT(instruction)
	case OP_LTE:
		return e.executeLTE(instruction)
	case OP_GTE:
		return e.executeGTE(instruction)
	case OP_NOT:
		return e.executeNOT(instruction)
	case OP_NEG:
		return e.executeNEG(instruction)
	case OP_CALL:
		return e.executeCall(instruction)
	case OP_RETURN:
		return e.executeReturn(instruction)
	case OP_MAKE_CLOSURE:
		return e.executeMakeClosureNew(instruction) // 使用新版本的实现
	case OP_GET_UPVALUE:
		return e.executeGetUpvalue(instruction)
	case OP_SET_UPVALUE:
		return e.executeSetUpvalue(instruction)
	case OP_CLOSE_UPVALUE:
		return e.executeCloseUpvalue(instruction)
	case OP_NEW_ARRAY:
		return e.executeNewArray(instruction)
	case OP_NEW_ARRAY_WITH_CAPACITY:
		return e.executeNewArrayWithCapacity(instruction)
	case OP_ARRAY_GET:
		return e.executeArrayGet(instruction)
	case OP_ARRAY_SET:
		return e.executeArraySet(instruction)
	case OP_ARRAY_LEN:
		return e.executeArrayLen(instruction)
	case OP_GC_WRITE_BARRIER:
		return e.executeGCWriteBarrier(instruction)
	case OP_GC_INC_REF:
		return e.executeGCIncRef(instruction)
	case OP_GC_DEC_REF:
		return e.executeGCDecRef(instruction)
	case OP_GC_ALLOC:
		return e.executeGCAlloc(instruction)
	case OP_GC_COLLECT:
		return e.executeGCCollect(instruction)
	case OP_GC_CHECK:
		return e.executeGCCheck(instruction)
	case OP_GC_PIN:
		return e.executeGCPin(instruction)
	case OP_GC_UNPIN:
		return e.executeGCUnpin(instruction)
	case OP_WEAK_REF:
		return e.executeWeakRef(instruction)
	case OP_WEAK_GET:
		return e.executeWeakGet(instruction)
	case OP_HALT:
		e.CurrentFrame = nil
		return nil
	default:
		return fmt.Errorf("unknown opcode: %d", instruction.OpCode)
	}
}

// executeMove 执行MOVE指令: R(A) := R(B)
func (e *Executor) executeMove(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [MOVE] A=%d, B=%d\n", inst.A, inst.B)
	fmt.Printf("DEBUG [MOVE] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	srcValue := frame.GetRegister(inst.B)
	fmt.Printf("DEBUG [MOVE] 从寄存器[%d]获取值，类型: %s\n", inst.B, srcValue.Type())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, srcValue)
	}

	err := frame.SetRegister(inst.A, srcValue)
	if err != nil {
		fmt.Printf("DEBUG [MOVE] 设置寄存器[%d]失败: %v\n", inst.A, err)
		return err
	}

	fmt.Printf("DEBUG [MOVE] 成功移动到寄存器[%d]，类型: %s\n", inst.A, srcValue.Type())

	frame.PC++
	return nil
}

// executeLoadK 执行LOADK指令: R(A) := K(Bx)
func (e *Executor) executeLoadK(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [LOADK] A=%d, Bx=%d\n", inst.A, inst.Bx)
	fmt.Printf("DEBUG [LOADK] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	konstValue := frame.GetConstant(inst.Bx)
	fmt.Printf("DEBUG [LOADK] 加载常量[%d]，类型: %s\n", inst.Bx, konstValue.Type())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, konstValue)
	}

	err := frame.SetRegister(inst.A, konstValue)
	if err != nil {
		fmt.Printf("DEBUG [LOADK] 设置寄存器[%d]失败: %v\n", inst.A, err)
		return err
	}

	fmt.Printf("DEBUG [LOADK] 成功加载到寄存器[%d]，类型: %s\n", inst.A, konstValue.Type())

	frame.PC++
	return nil
}

// executePop 执行POP指令: 丢弃栈顶值（在寄存器架构中只需前进PC）
func (e *Executor) executePop(inst Instruction) error {
	frame := e.CurrentFrame

	// 在寄存器架构中，POP指令只需要前进程序计数器
	// 不需要实际操作，因为值已经在寄存器中
	frame.PC++
	return nil
}

// executeGetGlobal 执行GET_GLOBAL指令: R(A) := G(Bx)
func (e *Executor) executeGetGlobal(inst Instruction) error {
	frame := e.CurrentFrame

	// 确保全局变量索引有效
	if inst.Bx >= len(e.Globals) {
		return fmt.Errorf("undefined global variable at index %d", inst.Bx)
	}

	globalValue := e.Globals[inst.Bx]

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, globalValue)
	}

	err := frame.SetRegister(inst.A, globalValue)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeSetGlobal 执行SET_GLOBAL指令: G(Bx) := R(A)
func (e *Executor) executeSetGlobal(inst Instruction) error {
	frame := e.CurrentFrame

	registerValue := frame.GetRegister(inst.A)

	// 扩展全局变量数组（如果需要）
	for len(e.Globals) <= inst.Bx {
		e.Globals = append(e.Globals, NewNilValueGC())
	}

	// GC优化：管理全局变量引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := e.Globals[inst.Bx]
		e.gcOptimizer.OnRegisterSet(oldValue, registerValue)
	}

	e.Globals[inst.Bx] = registerValue

	frame.PC++
	return nil
}

// executeGetLocal 执行GET_LOCAL指令: R(A) := L(B)
func (e *Executor) executeGetLocal(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [GET_LOCAL] A=%d, B=%d\n", inst.A, inst.B)
	fmt.Printf("DEBUG [GET_LOCAL] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	// 在当前实现中，局部变量也存储在寄存器中
	// 这是一个简化的实现，实际上可能需要专门的局部变量存储
	localValue := frame.GetRegister(inst.B)
	fmt.Printf("DEBUG [GET_LOCAL] 从寄存器[%d]获取值，类型: %s\n", inst.B, localValue.Type())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, localValue)
	}

	err := frame.SetRegister(inst.A, localValue)
	if err != nil {
		fmt.Printf("DEBUG [GET_LOCAL] 设置寄存器[%d]失败: %v\n", inst.A, err)
		return err
	}

	fmt.Printf("DEBUG [GET_LOCAL] 成功设置寄存器[%d]，类型: %s\n", inst.A, localValue.Type())

	frame.PC++
	return nil
}

// executeSetLocal 执行SET_LOCAL指令: L(B) := R(A)
func (e *Executor) executeSetLocal(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [SET_LOCAL] A=%d, B=%d\n", inst.A, inst.B)
	fmt.Printf("DEBUG [SET_LOCAL] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	registerValue := frame.GetRegister(inst.A)
	fmt.Printf("DEBUG [SET_LOCAL] 从寄存器[%d]获取值，类型: %s\n", inst.A, registerValue.Type())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.B)
		e.gcOptimizer.OnRegisterSet(oldValue, registerValue)
	}

	err := frame.SetRegister(inst.B, registerValue)
	if err != nil {
		fmt.Printf("DEBUG [SET_LOCAL] 设置寄存器[%d]失败: %v\n", inst.B, err)
		return err
	}

	fmt.Printf("DEBUG [SET_LOCAL] 成功设置寄存器[%d]，类型: %s\n", inst.B, registerValue.Type())

	frame.PC++
	return nil
}

// executeJump 执行JUMP指令: PC := PC + Bx
func (e *Executor) executeJump(inst Instruction) error {
	frame := e.CurrentFrame
	frame.PC += inst.Bx
	return nil
}

// executeJumpIfFalse 执行JUMP_IF_FALSE指令: if !R(A) then PC := PC + Bx
func (e *Executor) executeJumpIfFalse(inst Instruction) error {
	frame := e.CurrentFrame

	conditionValue := frame.GetRegister(inst.A)

	// 如果条件为假，则跳转
	if !conditionValue.IsTruthy() {
		frame.PC += inst.Bx
	} else {
		frame.PC++
	}

	return nil
}

// executeJumpIfTrue 执行JUMP_IF_TRUE指令: if R(A) then PC := PC + Bx
func (e *Executor) executeJumpIfTrue(inst Instruction) error {
	frame := e.CurrentFrame

	conditionValue := frame.GetRegister(inst.A)

	// 如果条件为真，则跳转
	if conditionValue.IsTruthy() {
		frame.PC += inst.Bx
	} else {
		frame.PC++
	}

	return nil
}

// executeAdd 执行ADD指令: R(A) := R(B) + R(C)（优化版）
func (e *Executor) executeAdd(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 使用GC安全的加法运算
	result, err := AddValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeSub 执行SUB指令: R(A) := R(B) - R(C)（优化版）
func (e *Executor) executeSub(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 使用GC安全的减法运算
	result, err := SubtractValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeMul 执行MUL指令: R(A) := R(B) * R(C)（优化版）
func (e *Executor) executeMul(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 使用GC安全的乘法运算
	result, err := MultiplyValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeDIV 执行DIV指令: R(A) := R(B) / R(C)（优化版）
func (e *Executor) executeDIV(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 使用GC安全的除法运算
	result, err := DivideValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeMOD 执行MOD指令: R(A) := R(B) % R(C)（优化版）
func (e *Executor) executeMOD(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 使用GC安全的取模运算
	result, err := ModuloValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeEQ 执行EQ指令: R(A) := R(B) == R(C)
func (e *Executor) executeEQ(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	result := NewBoolValueGC(valueB.Equal(valueC))

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err := frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeNEQ 执行NEQ指令: R(A) := R(B) != R(C)
func (e *Executor) executeNEQ(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	result := NewBoolValueGC(!valueB.Equal(valueC))

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err := frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeLT 执行LT指令: R(A) := R(B) < R(C)
func (e *Executor) executeLT(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 比较两个值
	result, err := LessThanValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeGT 执行GT指令: R(A) := R(B) > R(C)
func (e *Executor) executeGT(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 比较两个值
	result, err := GreaterThanValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeLTE 执行LTE指令: R(A) := R(B) <= R(C)
func (e *Executor) executeLTE(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 比较两个值：<= 等价于 !(>)
	gtResult, err := GreaterThanValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	result := NewBoolValueGC(!gtResult.AsBool())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeGTE 执行GTE指令: R(A) := R(B) >= R(C)
func (e *Executor) executeGTE(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	valueC := frame.GetRegister(inst.C)

	// 比较两个值：>= 等价于 !(<)
	ltResult, err := LessThanValuesGC(valueB, valueC)
	if err != nil {
		return err
	}

	result := NewBoolValueGC(!ltResult.AsBool())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeNOT 执行NOT指令: R(A) := !R(B)
func (e *Executor) executeNOT(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	result := NewBoolValueGC(!valueB.IsTruthy())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err := frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeNEG 执行NEG指令: R(A) := -R(B)
func (e *Executor) executeNEG(inst Instruction) error {
	frame := e.CurrentFrame

	valueB := frame.GetRegister(inst.B)
	result, err := NegateValueGC(valueB)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, result)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeCall 执行CALL指令: R(A) := R(A)(R(A+1), ..., R(A+B-1))
func (e *Executor) executeCall(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [CALL] A=%d, B=%d, C=%d\n", inst.A, inst.B, inst.C)
	fmt.Printf("DEBUG [CALL] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	// 检查调用深度
	if e.CallDepth >= e.MaxCallDepth {
		return fmt.Errorf("stack overflow: max call depth %d exceeded", e.MaxCallDepth)
	}

	// 获取函数或闭包
	funcValue := frame.GetRegister(inst.A)
	fmt.Printf("DEBUG [CALL] 函数值类型: %s\n", funcValue.Type())

	var targetFunc *Function
	var callable *Callable
	var closure *Closure

	if funcValue.IsFunction() {
		// 普通函数
		fmt.Printf("DEBUG [CALL] 调用普通函数\n")
		targetFuncInterface := funcValue.AsFunction()
		var ok bool
		targetFunc, ok = targetFuncInterface.(*Function)
		if !ok {
			fmt.Printf("DEBUG [CALL] 错误: 无法转换为Function类型\n")
			return fmt.Errorf("invalid function type")
		}
		fmt.Printf("DEBUG [CALL] 目标函数: %s\n", targetFunc.Name)
	} else if funcValue.IsCallable() {
		// 新的Callable类型（统一的可调用对象）
		fmt.Printf("DEBUG [CALL] 调用Callable对象\n")

		callable = funcValue.AsCallable()
		if callable == nil {
			fmt.Printf("DEBUG [CALL] 错误: AsCallable返回nil\n")
			return fmt.Errorf("invalid callable")
		}

		targetFunc = callable.Function
		if targetFunc == nil {
			fmt.Printf("DEBUG [CALL] 错误: Callable中的函数为nil\n")
			return fmt.Errorf("callable function is nil")
		}

		// 安全地访问函数名，避免可能的内存问题
		funcName := "<unknown>"
		if targetFunc.Name != "" {
			funcName = targetFunc.Name
		}
		fmt.Printf("DEBUG [CALL] Callable函数: %s\n", funcName)
		fmt.Printf("DEBUG [CALL] Callable upvalue数量: %d\n", len(callable.Upvalues))

		// 只打印upvalue的名称，避免调用String()方法
		for i, upvalue := range callable.Upvalues {
			if upvalue != nil {
				fmt.Printf("DEBUG [CALL] upvalue[%d]: %s\n", i, upvalue.Name)
			}
		}
	} else if funcValue.IsClosure() {
		// 旧的闭包类型（即将废弃）
		fmt.Printf("DEBUG [CALL] 调用闭包（旧版本）\n")

		closure = funcValue.AsClosure()
		if closure == nil {
			fmt.Printf("DEBUG [CALL] 错误: AsClosure返回nil\n")
			return fmt.Errorf("invalid closure")
		}

		targetFunc = closure.Function
		if targetFunc == nil {
			fmt.Printf("DEBUG [CALL] 错误: 闭包中的函数为nil\n")
			return fmt.Errorf("closure function is nil")
		}

		// 安全地访问函数名，避免可能的内存问题
		funcName := "<unknown>"
		if targetFunc.Name != "" {
			funcName = targetFunc.Name
		}
		fmt.Printf("DEBUG [CALL] 闭包函数: %s\n", funcName)
		fmt.Printf("DEBUG [CALL] 闭包捕获变量数量: %d\n", len(closure.Captures))

		// 只打印捕获变量的名称，避免调用String()方法
		for name := range closure.Captures {
			fmt.Printf("DEBUG [CALL] 捕获变量名: %s\n", name)
		}
	} else {
		fmt.Printf("DEBUG [CALL] 错误: 尝试调用非函数值: %s\n", funcValue.Type())
		return fmt.Errorf("attempted to call non-function value")
	}

	// 获取参数
	argCount := inst.B - 1
	fmt.Printf("DEBUG [CALL] 参数数量: %d\n", argCount)
	args := make([]ValueGC, argCount)
	for i := 0; i < argCount; i++ {
		args[i] = frame.GetRegister(inst.A + 1 + i)
		// 只打印参数类型，避免String()方法可能的递归
		fmt.Printf("DEBUG [CALL] 参数[%d] 类型: %s\n", i, args[i].Type())
	}

	// 创建新栈帧
	newFrame := NewStackFrame(targetFunc, frame, frame.PC+1)
	newFrame.SetParameters(args)
	newFrame.ExpectedRets = inst.C

	fmt.Printf("DEBUG [CALL] 创建新栈帧: %s\n", newFrame.Function.Name)
	fmt.Printf("DEBUG [CALL] 新栈帧寄存器数量: %d\n", len(newFrame.Registers))

	// 为了支持递归调用，将函数对象自身设置到函数名对应的寄存器位置
	// 函数名的寄存器索引是参数数量（因为参数从0开始，函数名在参数之后）
	// 修复：使用更高的寄存器索引，避免与参数和临时寄存器冲突
	recursiveRefIndex := targetFunc.ParamCount + 8 // 在参数后留出足够的临时寄存器空间
	if recursiveRefIndex < len(newFrame.Registers) {
		newFrame.SetRegister(recursiveRefIndex, funcValue)
		fmt.Printf("DEBUG [CALL] 设置递归函数引用到寄存器[%d]\n", recursiveRefIndex)
	}

	// 如果是Callable或闭包调用，设置捕获的变量
	if callable != nil {
		fmt.Printf("DEBUG [CALL] 处理Callable upvalue...\n")

		// 直接设置upvalue到新栈帧
		if len(callable.Upvalues) > 0 {
			fmt.Printf("DEBUG [CALL] 设置upvalue到新栈帧...\n")

			newFrame.Upvalues = callable.Upvalues
			fmt.Printf("DEBUG [CALL] 成功设置%d个upvalue到新栈帧\n", len(callable.Upvalues))
		}
	} else if closure != nil {
		fmt.Printf("DEBUG [CALL] 处理闭包upvalue（旧版本）...\n")

		// 尝试设置upvalue（实验性）
		if len(closure.Captures) > 0 {
			fmt.Printf("DEBUG [CALL] 尝试设置upvalue到新栈帧...\n")

			upvalues := make([]*Upvalue, len(closure.Captures))
			index := 0
			for name, value := range closure.Captures {
				upvalue := &Upvalue{
					Stack:    nil,   // 关闭状态
					Value:    value, // 直接存储值
					IsClosed: true,  // 已关闭
					Name:     name,  // 变量名
				}
				upvalues[index] = upvalue

				fmt.Printf("DEBUG [CALL] 设置upvalue[%d]: %s (类型: %s)\n",
					index, name, value.Type())
				index++
			}

			newFrame.Upvalues = upvalues
			fmt.Printf("DEBUG [CALL] 成功设置%d个upvalue到新栈帧\n", len(upvalues))
		}
	}

	// GC优化：管理栈帧生命周期
	if e.enableGCOpt && e.gcOptimizer != nil {
		e.gcOptimizer.OnStackFrameCreate(newFrame)
	}

	// 切换到新栈帧
	e.CurrentFrame = newFrame
	e.CallDepth++

	fmt.Printf("DEBUG [CALL] 切换到新栈帧，调用深度: %d\n", e.CallDepth)

	return nil
}

// executeMakeClosure 执行MAKE_CLOSURE指令: R(A) := Closure(function=R(B), capture_count=C, captures=R(B+1)...R(B+C))
func (e *Executor) executeMakeClosure(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [MAKE_CLOSURE] A=%d, B=%d, C=%d\n", inst.A, inst.B, inst.C)

	// 获取函数对象
	funcValue := frame.GetRegister(inst.B)
	fmt.Printf("DEBUG [MAKE_CLOSURE] 函数值类型: %s\n", funcValue.Type())

	if !funcValue.IsFunction() {
		fmt.Printf("DEBUG [MAKE_CLOSURE] 错误: 寄存器[%d]不是函数\n", inst.B)
		return fmt.Errorf("expected function in MAKE_CLOSURE")
	}

	// 获取Function对象
	var targetFunc *Function
	funcInterface := funcValue.AsFunction()
	fmt.Printf("DEBUG [MAKE_CLOSURE] AsFunction返回: %p\n", funcInterface)

	if targetFunc = funcInterface.(*Function); targetFunc == nil {
		fmt.Printf("DEBUG [MAKE_CLOSURE] 错误: 无法转换为Function类型\n")
		return fmt.Errorf("invalid function in MAKE_CLOSURE")
	}

	fmt.Printf("DEBUG [MAKE_CLOSURE] 目标函数: %s (地址: %p)\n", targetFunc.Name, targetFunc)

	// 获取捕获变量数量
	captureCount := inst.C
	fmt.Printf("DEBUG [MAKE_CLOSURE] 捕获变量数量: %d\n", captureCount)

	// 获取捕获变量值
	captures := make(map[string]ValueGC)

	// TODO: 需要知道捕获变量的名称，当前先用数字索引
	// 实际实现中需要从编译器传递变量名信息
	for i := 0; i < captureCount; i++ {
		captureValue := frame.GetRegister(inst.B + 1 + i)
		captureName := fmt.Sprintf("capture_%d", i) // 临时的变量名
		captures[captureName] = captureValue
		fmt.Printf("DEBUG [MAKE_CLOSURE] 捕获变量[%d] 名称: %s, 类型: %s\n", i, captureName, captureValue.Type())
	}

	// 创建闭包ValueGC（堆分配，安全）
	fmt.Printf("DEBUG [MAKE_CLOSURE] 创建闭包，函数: %p\n", targetFunc)
	closureValue := NewClosureValueGC(targetFunc, captures)
	fmt.Printf("DEBUG [MAKE_CLOSURE] 闭包创建成功，类型: %s\n", closureValue.Type())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, closureValue)
	}

	err := frame.SetRegister(inst.A, closureValue)
	if err != nil {
		fmt.Printf("DEBUG [MAKE_CLOSURE] 存储到寄存器失败: %v\n", err)
		return err
	}

	fmt.Printf("DEBUG [MAKE_CLOSURE] 闭包存储到寄存器[%d]成功\n", inst.A)

	frame.PC++
	return nil
}

// executeReturn 执行RETURN指令: return R(A), ..., R(A+B-2)
func (e *Executor) executeReturn(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG [RETURN] A=%d, B=%d\n", inst.A, inst.B)
	fmt.Printf("DEBUG [RETURN] 当前栈帧: %s (PC: %d)\n", frame.Function.Name, frame.PC)

	// 获取返回值数量
	// 根据Lua规范: RETURN A B 返回 R(A), ..., R(A+B-2)
	// 如果B=1，返回1个值 R(A)
	// 如果B=2，返回2个值 R(A), R(A+1)
	// 如果B=0，返回从R(A)到栈顶的所有值（变参返回）
	var retCount int
	if inst.B == 0 {
		// 变参返回：返回从R(A)到栈顶的所有值
		retCount = len(frame.Registers) - inst.A
	} else {
		// 固定数量返回
		retCount = inst.B
	}

	fmt.Printf("DEBUG [RETURN] 返回值数量: %d\n", retCount)

	returnValues := make([]ValueGC, retCount)
	for i := 0; i < retCount; i++ {
		returnValues[i] = frame.GetRegister(inst.A + i)
		// 只打印返回值类型，避免String()方法的递归问题
		fmt.Printf("DEBUG [RETURN] 返回值[%d] 类型: %s\n", i, returnValues[i].Type())
	}

	// GC优化：管理栈帧销毁
	if e.enableGCOpt && e.gcOptimizer != nil {
		e.gcOptimizer.OnStackFrameDestroy(frame)
	}

	// 关闭upvalue（栈帧销毁时）
	fmt.Printf("DEBUG [RETURN] 关闭当前栈帧的upvalue...\n")
	frame.CloseUpvalues()

	// 恢复调用者栈帧
	caller := frame.Caller

	if caller == nil {
		fmt.Printf("DEBUG [RETURN] 主函数返回，程序结束\n")
		// 主函数返回，设置返回值到寄存器0以便Execute方法获取
		if len(returnValues) > 0 {
			err := frame.SetRegister(0, returnValues[0])
			if err != nil {
				return err
			}
		}
		e.CurrentFrame = nil
		return nil
	}

	fmt.Printf("DEBUG [RETURN] 恢复调用者栈帧: %s\n", caller.Function.Name)

	// 设置返回值到调用者的寄存器
	// CALL指令的A寄存器位置存储返回值
	// 使用当前frame的ReturnAddr，而不是caller的ReturnAddr
	if frame.ReturnAddr > 0 {
		callInst := caller.Function.Instructions[frame.ReturnAddr-1]
		fmt.Printf("DEBUG [RETURN] 设置返回值到调用者寄存器[%d]\n", callInst.A)

		for i, retVal := range returnValues {
			if i < caller.ExpectedRets {
				// GC优化：管理返回值引用计数
				if e.enableGCOpt && e.gcOptimizer != nil {
					oldValue := caller.GetRegister(callInst.A + i)
					e.gcOptimizer.OnRegisterSet(oldValue, retVal)
				}
				caller.SetRegister(callInst.A+i, retVal)
				fmt.Printf("DEBUG [RETURN] 设置返回值[%d]到寄存器[%d] (类型: %s)\n",
					i, callInst.A+i, retVal.Type())
			}
		}
		// 恢复调用者上下文
		caller.PC = frame.ReturnAddr
		fmt.Printf("DEBUG [RETURN] 恢复调用者PC: %d\n", caller.PC)
	} else {
		// 从主函数返回的情况
		for i, retVal := range returnValues {
			if i < len(returnValues) {
				// GC优化：管理返回值引用计数
				if e.enableGCOpt && e.gcOptimizer != nil {
					oldValue := caller.GetRegister(i)
					e.gcOptimizer.OnRegisterSet(oldValue, retVal)
				}
				caller.SetRegister(i, retVal)
			}
		}
		// 主函数返回，不需要修改PC
	}
	e.CurrentFrame = caller
	e.CallDepth--

	fmt.Printf("DEBUG [RETURN] 返回完成，调用深度: %d\n", e.CallDepth)

	return nil
}

// GetCallStack 获取调用栈信息（调试用）
func (e *Executor) GetCallStack() []string {
	var stack []string
	frame := e.CurrentFrame

	for frame != nil {
		info := fmt.Sprintf("%s (PC: %d)", frame.Function.Name, frame.PC)
		stack = append(stack, info)
		frame = frame.Caller
	}

	return stack
}

// updateVariableReferences 更新所有对旧数组对象的引用，指向新的数组对象
func (e *Executor) updateVariableReferences(oldArray, newArray ValueGC) error {
	if !oldArray.IsGCManaged() || !newArray.IsGCManaged() {
		return nil // 只处理GC管理的对象
	}

	oldObjPtr := uintptr(oldArray.data)
	fmt.Printf("DEBUG [updateVariableReferences] 更新引用: 旧对象=%p -> 新对象=%p\n",
		unsafe.Pointer(oldObjPtr), unsafe.Pointer(uintptr(newArray.data)))

	// 更新全局变量
	for i, globalVar := range e.Globals {
		if globalVar.IsGCManaged() && uintptr(globalVar.data) == oldObjPtr {
			fmt.Printf("DEBUG [updateVariableReferences] 更新全局变量[%d]\n", i)
			e.Globals[i] = newArray
		}
	}

	// 更新当前栈帧的寄存器
	if e.CurrentFrame != nil {
		for i, regValue := range e.CurrentFrame.Registers {
			if regValue.IsGCManaged() && uintptr(regValue.data) == oldObjPtr {
				fmt.Printf("DEBUG [updateVariableReferences] 更新寄存器[%d]\n", i)
				e.CurrentFrame.Registers[i] = newArray
			}
		}
	}

	// 更新调用栈中的所有栈帧
	frame := e.CurrentFrame
	for frame != nil {
		for i, regValue := range frame.Registers {
			if regValue.IsGCManaged() && uintptr(regValue.data) == oldObjPtr {
				fmt.Printf("DEBUG [updateVariableReferences] 更新栈帧寄存器[%d]\n", i)
				frame.Registers[i] = newArray
			}
		}
		frame = frame.Caller
	}

	return nil
}

// 优化方法：批量运算支持

// ExecuteArithmeticSequence 执行算术运算序列（专门优化）
func (e *Executor) ExecuteArithmeticSequence(ops []OpCode, operands [][]int) error {
	frame := e.CurrentFrame
	if frame == nil {
		return fmt.Errorf("no current frame")
	}

	// 批量执行算术运算，减少指令解码开销
	for i, op := range ops {
		if i >= len(operands) {
			break
		}

		inst := Instruction{
			OpCode: op,
			A:      operands[i][0],
			B:      operands[i][1],
			C:      operands[i][2],
		}

		switch op {
		case OP_ADD:
			err := e.executeAdd(inst)
			if err != nil {
				return err
			}
		case OP_SUB:
			err := e.executeSub(inst)
			if err != nil {
				return err
			}
		case OP_MUL:
			err := e.executeMul(inst)
			if err != nil {
				return err
			}
		case OP_DIV:
			err := e.executeDIV(inst)
			if err != nil {
				return err
			}
		case OP_MOD:
			err := e.executeMOD(inst)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported batch operation: %d", op)
		}

		frame.PC-- // 抵消每个操作中的PC++
	}

	frame.PC += len(ops) // 最终PC前进
	return nil
}

// 数组指令执行方法

// executeNewArray 执行NEW_ARRAY指令: R(A) := array(length=B)
func (e *Executor) executeNewArray(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG executeNewArray: A=%d, B=%d (length=%d)\n", inst.A, inst.B, inst.B)

	length := inst.B
	if length < 0 {
		return fmt.Errorf("invalid array length: %d", length)
	}

	// 创建指定长度的数组，初始化为nil值
	// 对于空数组，分配默认容量以支持后续的动态设置
	capacity := length
	if capacity == 0 {
		capacity = 4 // 默认容量，支持动态扩容
	}

	elements := make([]ValueGC, length, capacity)
	nilValue := NewNilValueGC()
	for i := range elements {
		elements[i] = nilValue
	}

	arrayValue := NewArrayValueGC(elements)
	fmt.Printf("DEBUG 创建的数组类型: %s\n", arrayValue.Type())
	fmt.Printf("DEBUG 创建的数组IsArray(): %v\n", arrayValue.IsArray())

	if _, arrayElements, err := arrayValue.AsArrayData(); err == nil {
		fmt.Printf("DEBUG 创建数组成功: Length=%d\n", len(arrayElements))
		fmt.Printf("DEBUG Elements长度: %d\n", len(arrayElements))
	} else {
		fmt.Printf("DEBUG 创建数组失败: %v\n", err)
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, arrayValue)
	}

	fmt.Printf("DEBUG 设置到寄存器前，数组类型: %s\n", arrayValue.Type())

	err := frame.SetRegister(inst.A, arrayValue)
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG 设置到寄存器后，开始验证...\n")

	// 验证设置后的寄存器
	verifyValue := frame.GetRegister(inst.A)
	fmt.Printf("DEBUG 验证值类型: %s\n", verifyValue.Type())
	fmt.Printf("DEBUG 验证值IsArray(): %v\n", verifyValue.IsArray())

	if verifyValue.IsArray() {
		if _, verifyElements, err := verifyValue.AsArrayData(); err == nil {
			fmt.Printf("DEBUG 寄存器验证: Length=%d\n", len(verifyElements))
		}
	} else {
		fmt.Printf("DEBUG 寄存器验证失败: 不是数组类型\n")
	}

	frame.PC++
	return nil
}

// executeNewArrayWithCapacity 执行NEW_ARRAY_WITH_CAPACITY指令: R(A) := array(capacity=R(B), default=R(C))
func (e *Executor) executeNewArrayWithCapacity(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG executeNewArrayWithCapacity: A=%d, B=%d, C=%d\n", inst.A, inst.B, inst.C)

	capacityValue := frame.GetRegister(inst.B)
	if !capacityValue.IsNumber() {
		return fmt.Errorf("array capacity must be a number")
	}
	capacityNum, err := capacityValue.ToNumber()
	if err != nil {
		return fmt.Errorf("invalid array capacity: %v", err)
	}
	capacity := int(capacityNum)

	var defaultValue ValueGC
	if inst.C != 0 {
		defaultValue = frame.GetRegister(inst.C)
	} else {
		defaultValue = NewNilValueGC()
	}

	if capacity < 0 {
		return fmt.Errorf("invalid array capacity: %d", capacity)
	}

	// 为了避免过大的预分配，设置合理的最大容量
	maxCapacity := 1000000 // 1M元素
	if capacity > maxCapacity {
		return fmt.Errorf("array capacity too large: %d (max: %d)", capacity, maxCapacity)
	}

	// 创建指定容量的数组，初始化为默认值
	elements := make([]ValueGC, 0, capacity) // 长度为0，容量为capacity

	// 只有当明确指定默认值时才预填充数组
	if inst.C != 0 {
		// 预填充数组到指定容量
		for i := 0; i < capacity; i++ {
			elements = append(elements, SafeCopyValueGC(defaultValue))
		}
	}

	arrayValue := NewArrayValueGC(elements)
	fmt.Printf("DEBUG 创建的数组类型: %s, 容量: %d\n", arrayValue.Type(), capacity)

	if arrData, _, err := arrayValue.AsArrayData(); err == nil {
		fmt.Printf("DEBUG 创建数组成功: Length=%d, Capacity=%d\n", arrData.Length, arrData.Capacity)
	} else {
		fmt.Printf("DEBUG 创建数组失败: %v\n", err)
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, arrayValue)
	}

	fmt.Printf("DEBUG 设置到寄存器前，数组类型: %s\n", arrayValue.Type())

	err = frame.SetRegister(inst.A, arrayValue)
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG 设置到寄存器后，开始验证...\n")

	// 验证设置后的寄存器
	verifyValue := frame.GetRegister(inst.A)
	fmt.Printf("DEBUG 验证值类型: %s\n", verifyValue.Type())
	fmt.Printf("DEBUG 验证值IsArray(): %v\n", verifyValue.IsArray())

	if verifyValue.IsArray() {
		if verifyArrData, _, err := verifyValue.AsArrayData(); err == nil {
			fmt.Printf("DEBUG 寄存器验证: Length=%d, Capacity=%d\n", verifyArrData.Length, verifyArrData.Capacity)
		}
	} else {
		fmt.Printf("DEBUG 寄存器验证失败: 不是数组类型\n")
	}

	frame.PC++
	return nil
}

// executeArrayGet 执行ARRAY_GET指令: R(A) := R(B)[R(C)]
func (e *Executor) executeArrayGet(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG executeArrayGet: A=%d, B=%d, C=%d\n", inst.A, inst.B, inst.C)

	arrayValue := frame.GetRegister(inst.B)
	indexValue := frame.GetRegister(inst.C)

	fmt.Printf("DEBUG 获取到的值类型:\n")
	fmt.Printf("  arrayValue类型: %s\n", arrayValue.Type())
	fmt.Printf("  indexValue类型: %s\n", indexValue.Type())

	// 检查array类型
	if !arrayValue.IsArray() {
		return fmt.Errorf("not an array")
	}

	// 检查index类型
	if !indexValue.IsNumber() {
		return fmt.Errorf("array index must be a number")
	}

	indexNum, err := indexValue.ToNumber()
	if err != nil {
		return fmt.Errorf("invalid array index: %v", err)
	}

	index := int(indexNum)
	fmt.Printf("DEBUG 即将调用ArrayGetValueGC: index=%d\n", index)

	element, err := ArrayGetValueGC(arrayValue, index)
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG ArrayGetValueGC 返回的元素类型: %s\n", element.Type())

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, element)
	}

	err = frame.SetRegister(inst.A, element)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeArraySet 执行ARRAY_SET指令: R(A)[R(B)] := R(C) - 支持动态扩容
func (e *Executor) executeArraySet(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG executeArraySet: A=%d, B=%d, C=%d\n", inst.A, inst.B, inst.C)

	arrayValue := frame.GetRegister(inst.A)
	indexValue := frame.GetRegister(inst.B)
	value := frame.GetRegister(inst.C)

	fmt.Printf("DEBUG 获取到的值类型:\n")
	fmt.Printf("  arrayValue类型: %s\n", arrayValue.Type())
	fmt.Printf("  indexValue类型: %s\n", indexValue.Type())
	fmt.Printf("  value类型: %s\n", value.Type())

	// 检查array类型
	if !arrayValue.IsArray() {
		return fmt.Errorf("not an array")
	}

	// 检查index类型
	if !indexValue.IsNumber() {
		return fmt.Errorf("array index must be a number")
	}

	indexNum, err := indexValue.ToNumber()
	if err != nil {
		return fmt.Errorf("invalid array index: %v", err)
	}

	index := int(indexNum)
	fmt.Printf("DEBUG 即将调用ArraySetValueGCWithExpansion: index=%d\n", index)

	// 使用新的支持扩容的设置方法
	newArrayValue, err := ArraySetValueGCWithExpansion(arrayValue, index, value)
	if err != nil {
		return fmt.Errorf("failed to set array element: %v", err)
	}

	// 检查是否发生了扩容（通过容量变化检测）
	if newArrayValue.data != arrayValue.data {
		fmt.Printf("DEBUG 数组扩容发生，更新寄存器引用\n")
		// 更新寄存器中的数组引用
		err = frame.SetRegister(inst.A, newArrayValue)
		if err != nil {
			return fmt.Errorf("failed to update array register: %v", err)
		}

		// 同时更新可能关联的变量存储位置
		err = e.updateVariableReferences(arrayValue, newArrayValue)
		if err != nil {
			fmt.Printf("DEBUG 警告: 更新变量引用失败: %v\n", err)
		}
	}

	fmt.Printf("DEBUG executeArraySet完成\n")
	frame.PC++
	return nil
}

// executeArrayLen 执行ARRAY_LEN指令: R(A) := len(R(B))
func (e *Executor) executeArrayLen(inst Instruction) error {
	frame := e.CurrentFrame

	arrayValue := frame.GetRegister(inst.B)

	// 检查array类型
	if !arrayValue.IsArray() {
		return fmt.Errorf("not an array")
	}

	lengthValue, err := ArrayLengthValueGC(arrayValue)
	if err != nil {
		return err
	}

	// GC优化：管理引用计数
	if e.enableGCOpt && e.gcOptimizer != nil {
		oldValue := frame.GetRegister(inst.A)
		e.gcOptimizer.OnRegisterSet(oldValue, lengthValue)
	}

	err = frame.SetRegister(inst.A, lengthValue)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// =============================================================================
// GC相关指令执行方法
// =============================================================================

// executeGCWriteBarrier 执行GC_WRITE_BARRIER指令: WriteBarrier(R(A), R(B))
func (e *Executor) executeGCWriteBarrier(inst Instruction) error {
	frame := e.CurrentFrame

	objValue := frame.GetRegister(inst.A)
	fieldValue := frame.GetRegister(inst.B)

	// 如果对象需要GC管理，执行写屏障
	if objValue.IsGCManaged() {
		err := WriteBarrierGC(objValue, fieldValue)
		if err != nil {
			return err
		}
	}

	frame.PC++
	return nil
}

// executeGCIncRef 执行GC_INC_REF指令: IncRef(R(A))
func (e *Executor) executeGCIncRef(inst Instruction) error {
	frame := e.CurrentFrame

	value := frame.GetRegister(inst.A)

	// 如果值需要GC管理，增加引用计数
	if value.IsGCManaged() {
		err := IncRefGC(value)
		if err != nil {
			return err
		}
	}

	frame.PC++
	return nil
}

// executeGCDecRef 执行GC_DEC_REF指令: DecRef(R(A))
func (e *Executor) executeGCDecRef(inst Instruction) error {
	frame := e.CurrentFrame

	value := frame.GetRegister(inst.A)

	// 如果值需要GC管理，减少引用计数
	if value.IsGCManaged() {
		err := DecRefGC(value)
		if err != nil {
			return err
		}
	}

	frame.PC++
	return nil
}

// executeGCAlloc 执行GC_ALLOC指令: R(A) := GCAlloc(type=B)
func (e *Executor) executeGCAlloc(inst Instruction) error {
	frame := e.CurrentFrame

	objectType := ValueTypeGC(inst.B)

	// 根据类型分配GC管理的对象
	var result ValueGC
	var err error

	switch objectType {
	case ValueGCTypeString:
		// 分配空字符串
		result = NewStringValueGC("")
	case ValueGCTypeArray:
		// 分配空数组
		result = NewArrayValueGC(make([]ValueGC, 0))
	default:
		return fmt.Errorf("unsupported GC allocation type: %v", objectType)
	}

	err = frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeGCCollect 执行GC_COLLECT指令: TriggerGC()
func (e *Executor) executeGCCollect(inst Instruction) error {
	frame := e.CurrentFrame

	// 触发垃圾回收
	err := TriggerGCCollection()
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeGCCheck 执行GC_CHECK指令: R(A) := IsValidPtr(R(A))
func (e *Executor) executeGCCheck(inst Instruction) error {
	frame := e.CurrentFrame

	value := frame.GetRegister(inst.A)

	// 检查值是否为有效的GC管理对象
	isValid := IsValidGCPointer(value)
	result := NewBoolValueGC(isValid)

	err := frame.SetRegister(inst.A, result)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeGCPin 执行GC_PIN指令: Pin(R(A))
func (e *Executor) executeGCPin(inst Instruction) error {
	frame := e.CurrentFrame

	value := frame.GetRegister(inst.A)

	// 如果值需要GC管理，固定对象防止移动
	if value.IsGCManaged() {
		err := PinGCObject(value)
		if err != nil {
			return err
		}
	}

	frame.PC++
	return nil
}

// executeGCUnpin 执行GC_UNPIN指令: Unpin(R(A))
func (e *Executor) executeGCUnpin(inst Instruction) error {
	frame := e.CurrentFrame

	value := frame.GetRegister(inst.A)

	// 如果值需要GC管理，取消固定对象
	if value.IsGCManaged() {
		err := UnpinGCObject(value)
		if err != nil {
			return err
		}
	}

	frame.PC++
	return nil
}

// executeWeakRef 执行WEAK_REF指令: R(A) := WeakRef(R(B))
func (e *Executor) executeWeakRef(inst Instruction) error {
	frame := e.CurrentFrame

	targetValue := frame.GetRegister(inst.B)

	// 创建弱引用
	weakRef, err := CreateWeakRefGC(targetValue)
	if err != nil {
		return err
	}

	err = frame.SetRegister(inst.A, weakRef)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeWeakGet 执行WEAK_GET指令: R(A) := WeakGet(R(B))
func (e *Executor) executeWeakGet(inst Instruction) error {
	frame := e.CurrentFrame

	weakRefValue := frame.GetRegister(inst.B)

	// 获取弱引用的目标值
	targetValue, err := GetWeakRefTargetGC(weakRefValue)
	if err != nil {
		return err
	}

	err = frame.SetRegister(inst.A, targetValue)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}
