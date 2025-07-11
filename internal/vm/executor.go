package vm

import (
	"fmt"
	"runtime"
)

// Executor AQL虚拟机执行器（优化版）
type Executor struct {
	CurrentFrame *StackFrame
	MaxCallDepth int
	CallDepth    int
	Globals      []Value // 全局变量存储
}

// NewExecutor 创建新的执行器
func NewExecutor() *Executor {
	return &Executor{
		CurrentFrame: nil,
		MaxCallDepth: 1000,
		CallDepth:    0,
		Globals:      make([]Value, 0, 256), // 初始化全局变量存储，预分配256个位置
	}
}

// Execute 执行函数
func (e *Executor) Execute(function *Function, args []Value) ([]Value, error) {
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
		return []Value{mainFrame.Registers[0]}, nil
	}

	return []Value{NewNilValue()}, nil
}

// executeStep 执行单步指令
func (e *Executor) executeStep() error {
	frame := e.CurrentFrame
	if frame == nil {
		return fmt.Errorf("no current frame")
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
	case OP_NEW_ARRAY:
		return e.executeNewArray(instruction)
	case OP_ARRAY_GET:
		return e.executeArrayGet(instruction)
	case OP_ARRAY_SET:
		return e.executeArraySet(instruction)
	case OP_ARRAY_LEN:
		return e.executeArrayLen(instruction)
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

	srcValue := frame.GetRegister(inst.B)
	err := frame.SetRegister(inst.A, srcValue)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeLoadK 执行LOADK指令: R(A) := K(Bx)
func (e *Executor) executeLoadK(inst Instruction) error {
	frame := e.CurrentFrame

	fmt.Printf("DEBUG executeLoadK: A=%d, Bx=%d\n", inst.A, inst.Bx)

	konstValue := frame.GetConstant(inst.Bx)
	fmt.Printf("DEBUG LOADK 常量值: %s (类型: %s)\n", konstValue.ToString(), konstValue.Type())

	// 检查寄存器0的数组是否仍然完好（如果存在）
	if inst.A != 0 {
		reg0 := frame.GetRegister(0)
		if reg0.IsArray() {
			array0, err := reg0.AsArray()
			if err == nil {
				fmt.Printf("DEBUG LOADK前 R[0]数组状态: Length=%d, 指针=%p\n", array0.Length, array0)
			}
		}
	}

	err := frame.SetRegister(inst.A, konstValue)
	if err != nil {
		return err
	}

	// 再次检查寄存器0的数组状态
	if inst.A != 0 {
		reg0 := frame.GetRegister(0)
		if reg0.IsArray() {
			array0, err := reg0.AsArray()
			if err == nil {
				fmt.Printf("DEBUG LOADK后 R[0]数组状态: Length=%d, 指针=%p\n", array0.Length, array0)
			}
		}
	}

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
		e.Globals = append(e.Globals, NewNilValue())
	}

	e.Globals[inst.Bx] = registerValue

	frame.PC++
	return nil
}

// executeGetLocal 执行GET_LOCAL指令: R(A) := L(B)
func (e *Executor) executeGetLocal(inst Instruction) error {
	frame := e.CurrentFrame

	// 在当前实现中，局部变量也存储在寄存器中
	// 这是一个简化的实现，实际上可能需要专门的局部变量存储
	localValue := frame.GetRegister(inst.B)
	err := frame.SetRegister(inst.A, localValue)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeSetLocal 执行SET_LOCAL指令: L(B) := R(A)
func (e *Executor) executeSetLocal(inst Instruction) error {
	frame := e.CurrentFrame

	registerValue := frame.GetRegister(inst.A)
	err := frame.SetRegister(inst.B, registerValue)
	if err != nil {
		return err
	}

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

	// 使用优化的加法运算
	result, err := AddValues(valueB, valueC)
	if err != nil {
		return err
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

	// 使用优化的减法运算
	result, err := SubtractValues(valueB, valueC)
	if err != nil {
		return err
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

	// 使用优化的乘法运算
	result, err := MultiplyValues(valueB, valueC)
	if err != nil {
		return err
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

	result := NewBoolValue(valueB.Equal(valueC))
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

	result := NewBoolValue(!valueB.Equal(valueC))
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
	result, err := LessThan(valueB, valueC)
	if err != nil {
		return err
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
	result, err := GreaterThan(valueB, valueC)
	if err != nil {
		return err
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
	gtResult, err := GreaterThan(valueB, valueC)
	if err != nil {
		return err
	}

	result := NewBoolValue(!gtResult.AsBool())
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
	ltResult, err := LessThan(valueB, valueC)
	if err != nil {
		return err
	}

	result := NewBoolValue(!ltResult.AsBool())
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
	result := NewBoolValue(!valueB.IsTruthy())

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
	result, err := NegateValue(valueB)
	if err != nil {
		return err
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

	// 检查调用深度
	if e.CallDepth >= e.MaxCallDepth {
		return fmt.Errorf("stack overflow: max call depth %d exceeded", e.MaxCallDepth)
	}

	// 获取函数
	funcValue := frame.GetRegister(inst.A)
	if !funcValue.IsFunction() {
		return fmt.Errorf("attempted to call non-function value")
	}

	targetFunc := funcValue.AsFunction()

	// 获取参数
	argCount := inst.B - 1
	args := make([]Value, argCount)
	for i := 0; i < argCount; i++ {
		args[i] = frame.GetRegister(inst.A + 1 + i)
	}

	// 创建新栈帧
	newFrame := NewStackFrame(targetFunc, frame, frame.PC+1)
	newFrame.SetParameters(args)
	newFrame.ExpectedRets = inst.C

	// 切换到新栈帧
	e.CurrentFrame = newFrame
	e.CallDepth++

	return nil
}

// executeReturn 执行RETURN指令: return R(A), ..., R(A+B-2)
func (e *Executor) executeReturn(inst Instruction) error {
	frame := e.CurrentFrame

	// 获取返回值
	retCount := inst.B - 1
	if retCount < 0 {
		retCount = 1 // 默认返回一个值
	}

	returnValues := make([]Value, retCount)
	for i := 0; i < retCount; i++ {
		returnValues[i] = frame.GetRegister(inst.A + i)
	}

	// 恢复调用者栈帧
	caller := frame.Caller
	if caller == nil {
		// 主函数返回，设置返回值到寄存器0
		if len(returnValues) > 0 {
			frame.SetRegister(0, returnValues[0])
		}
		e.CurrentFrame = nil
		return nil
	}

	// 设置返回值到调用者的寄存器
	// CALL指令的A寄存器位置存储返回值
	// 使用当前frame的ReturnAddr，而不是caller的ReturnAddr
	if frame.ReturnAddr > 0 {
		callInst := caller.Function.Instructions[frame.ReturnAddr-1]
		for i, retVal := range returnValues {
			if i < caller.ExpectedRets {
				caller.SetRegister(callInst.A+i, retVal)
			}
		}
		// 恢复调用者上下文
		caller.PC = frame.ReturnAddr
	} else {
		// 从主函数返回的情况
		for i, retVal := range returnValues {
			if i < len(returnValues) {
				caller.SetRegister(i, retVal)
			}
		}
		// 主函数返回，不需要修改PC
	}
	e.CurrentFrame = caller
	e.CallDepth--

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
	elements := make([]Value, length)
	nilValue := NewNilValue()
	for i := range elements {
		elements[i] = nilValue
	}

	arrayValue := NewArrayValue(elements)
	if array, err := arrayValue.AsArray(); err == nil {
		fmt.Printf("DEBUG 创建数组成功: Length=%d\n", array.Length)
		fmt.Printf("DEBUG 数组指针: %p\n", array)
		fmt.Printf("DEBUG Elements指针: %p\n", array.Elements)
	}

	err := frame.SetRegister(inst.A, arrayValue)
	if err != nil {
		return err
	}

	// 验证设置后的寄存器
	verifyValue := frame.GetRegister(inst.A)
	if verifyValue.IsArray() {
		if verifyArray, err := verifyValue.AsArray(); err == nil {
			fmt.Printf("DEBUG 寄存器验证: Length=%d, 指针=%p\n", verifyArray.Length, verifyArray)
		}
	} else {
		fmt.Printf("DEBUG 寄存器验证失败: 不是数组类型\n")
	}

	// 强制垃圾回收测试
	fmt.Printf("DEBUG 强制垃圾回收前...\n")
	runtime.GC()
	runtime.GC() // 两次确保完整回收
	fmt.Printf("DEBUG 强制垃圾回收后\n")

	// 再次验证数组
	verifyValue2 := frame.GetRegister(inst.A)
	if verifyValue2.IsArray() {
		if verifyArray2, err := verifyValue2.AsArray(); err == nil {
			fmt.Printf("DEBUG GC后验证: Length=%d, 指针=%p\n", verifyArray2.Length, verifyArray2)
		}
	}

	frame.PC++
	return nil
}

// executeArrayGet 执行ARRAY_GET指令: R(A) := R(B)[R(C)]
func (e *Executor) executeArrayGet(inst Instruction) error {
	frame := e.CurrentFrame

	arrayValue := frame.GetRegister(inst.B)
	indexValue := frame.GetRegister(inst.C)

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
	element, err := ArrayGetValue(arrayValue, index)
	if err != nil {
		return err
	}

	err = frame.SetRegister(inst.A, element)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}

// executeArraySet 执行ARRAY_SET指令: R(A)[R(B)] := R(C)
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

	// 在调用ArraySetValue前检查数组指针
	if arrayPtr, err := arrayValue.AsArray(); err == nil {
		fmt.Printf("DEBUG Array指针: %p\n", arrayPtr)
		fmt.Printf("DEBUG Array.Length: %d\n", arrayPtr.Length)
		fmt.Printf("DEBUG Array.Elements: %p\n", arrayPtr.Elements)
		if arrayPtr.Elements != nil {
			fmt.Printf("DEBUG Elements切片: len=%d, cap=%d\n", len(arrayPtr.Elements), cap(arrayPtr.Elements))
		} else {
			fmt.Printf("DEBUG Elements切片为nil!\n")
		}
	} else {
		fmt.Printf("DEBUG Array指针错误: %v\n", err)
		return fmt.Errorf("array pointer error: %v", err)
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
	fmt.Printf("DEBUG 即将调用ArraySetValue: index=%d\n", index)

	err = ArraySetValue(arrayValue, index, value)
	if err != nil {
		return err
	}

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

	lengthValue, err := ArrayLengthValue(arrayValue)
	if err != nil {
		return err
	}
	err = frame.SetRegister(inst.A, lengthValue)
	if err != nil {
		return err
	}

	frame.PC++
	return nil
}
