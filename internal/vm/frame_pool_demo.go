package vm

import (
	"fmt"
	"sync"
)

// FramePool 栈帧对象池，消除函数调用分配开销
type FramePool struct {
	pool sync.Pool
}

// GlobalFramePool 全局栈帧池
var GlobalFramePool = &FramePool{
	pool: sync.Pool{
		New: func() interface{} {
			return &StackFrame{
				Registers: make([]Value, 64), // 预分配足够大的寄存器
			}
		},
	},
}

// Get 从池中获取栈帧
func (fp *FramePool) Get() *StackFrame {
	frame := fp.pool.Get().(*StackFrame)
	return frame
}

// Put 将栈帧归还到池中
func (fp *FramePool) Put(frame *StackFrame) {
	// 重置栈帧状态
	frame.reset()
	fp.pool.Put(frame)
}

// reset 重置栈帧到初始状态
func (sf *StackFrame) reset() {
	sf.Function = nil
	sf.PC = 0
	sf.Base = 0
	sf.Caller = nil
	sf.ReturnAddr = 0
	sf.ExpectedRets = 1

	// 清空寄存器为nil值（复用slice避免重新分配）
	nilValue := NewNilValue()
	for i := range sf.Registers {
		sf.Registers[i] = nilValue
	}
}

// NewStackFrameFromPool 从池中创建栈帧（零分配版本）
func NewStackFrameFromPool(function *Function, caller *StackFrame, returnAddr int) *StackFrame {
	frame := GlobalFramePool.Get()

	// 设置栈帧信息
	frame.Function = function
	frame.PC = 0
	frame.Base = 0
	frame.Caller = caller
	frame.ReturnAddr = returnAddr
	frame.ExpectedRets = 1

	// 确保寄存器足够大
	if len(frame.Registers) < function.MaxStackSize {
		// 只在必要时扩展
		frame.Registers = make([]Value, function.MaxStackSize)
	}

	return frame
}

// OptimizedExecutor 优化版执行器，使用栈帧池
type OptimizedExecutor struct {
	CurrentFrame *StackFrame
	MaxCallDepth int
	CallDepth    int

	// 参数缓存，避免重复分配
	argBuffer    []Value
	returnBuffer []Value
}

// NewOptimizedExecutor 创建优化版执行器
func NewOptimizedExecutor() *OptimizedExecutor {
	return &OptimizedExecutor{
		CurrentFrame: nil,
		MaxCallDepth: 1000,
		CallDepth:    0,
		argBuffer:    make([]Value, 16), // 预分配参数缓冲区
		returnBuffer: make([]Value, 16), // 预分配返回值缓冲区
	}
}

// executeCallOptimized 优化版函数调用（零分配）
func (e *OptimizedExecutor) executeCallOptimized(inst Instruction) error {
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

	// 从池中获取新栈帧（零分配）
	newFrame := NewStackFrameFromPool(targetFunc, frame, frame.PC+1)
	newFrame.ExpectedRets = inst.C

	// 直接拷贝参数，避免中间数组分配
	argCount := inst.B - 1
	for i := 0; i < argCount; i++ {
		if i < len(newFrame.Registers) {
			newFrame.Registers[i] = frame.GetRegister(inst.A + 1 + i)
		}
	}

	// 切换到新栈帧
	e.CurrentFrame = newFrame
	e.CallDepth++

	return nil
}

// executeReturnOptimized 优化版函数返回（零分配）
func (e *OptimizedExecutor) executeReturnOptimized(inst Instruction) error {
	frame := e.CurrentFrame

	// 获取返回值数量
	retCount := inst.B - 1
	if retCount < 0 {
		retCount = 1
	}

	// 恢复调用者栈帧
	caller := frame.Caller
	if caller == nil {
		// 主函数返回，设置返回值到寄存器0
		if retCount > 0 {
			frame.SetRegister(0, frame.GetRegister(inst.A))
		}

		// 归还栈帧到池中
		GlobalFramePool.Put(frame)
		e.CurrentFrame = nil
		return nil
	}

	// 直接拷贝返回值，避免中间数组分配
	// 使用当前frame的ReturnAddr，而不是caller的ReturnAddr
	if frame.ReturnAddr > 0 {
		callInst := caller.Function.Instructions[frame.ReturnAddr-1]
		for i := 0; i < retCount && i < caller.ExpectedRets; i++ {
			caller.SetRegister(callInst.A+i, frame.GetRegister(inst.A+i))
		}
		// 恢复调用者PC到返回地址
		caller.PC = frame.ReturnAddr
	} else {
		// 从主函数返回，直接设置到寄存器0
		for i := 0; i < retCount; i++ {
			caller.SetRegister(i, frame.GetRegister(inst.A+i))
		}
		// 主函数返回，不需要修改PC
	}

	e.CurrentFrame = caller
	e.CallDepth--

	// 归还栈帧到池中
	GlobalFramePool.Put(frame)

	return nil
}

// ExecuteOptimized 优化版执行函数
func (e *OptimizedExecutor) ExecuteOptimized(function *Function, args []Value) ([]Value, error) {
	// 从池中获取主函数栈帧
	mainFrame := NewStackFrameFromPool(function, nil, -1)

	// 设置参数
	for i, arg := range args {
		if i < len(mainFrame.Registers) {
			mainFrame.Registers[i] = arg
		}
	}

	e.CurrentFrame = mainFrame
	e.CallDepth = 1

	// 保存返回值的变量
	var returnValue Value = NewNilValue()

	// 执行主循环
	for e.CurrentFrame != nil {
		// 如果当前帧是主函数且即将执行RETURN，先保存返回值
		if e.CurrentFrame == mainFrame && e.CurrentFrame.PC < len(e.CurrentFrame.Function.Instructions) {
			inst := e.CurrentFrame.Function.Instructions[e.CurrentFrame.PC]
			if inst.OpCode == OP_RETURN {
				// 保存返回值
				if inst.B > 1 && inst.A < len(e.CurrentFrame.Registers) {
					returnValue = e.CurrentFrame.Registers[inst.A]
				}
			}
		}

		err := e.executeStepOptimized()
		if err != nil {
			// 清理栈帧
			if e.CurrentFrame != nil {
				GlobalFramePool.Put(e.CurrentFrame)
			}
			return nil, err
		}
	}

	return []Value{returnValue}, nil
}

// executeStepOptimized 优化版单步执行
func (e *OptimizedExecutor) executeStepOptimized() error {
	frame := e.CurrentFrame
	if frame == nil {
		return fmt.Errorf("no current frame")
	}

	instruction := frame.GetInstruction()

	// 内联热点指令，避免函数调用开销
	switch instruction.OpCode {
	case OP_MOVE:
		// 内联MOVE指令
		frame.Registers[instruction.A] = frame.Registers[instruction.B]
		frame.PC++
		return nil

	case OP_LOADK:
		// 内联LOADK指令
		frame.Registers[instruction.A] = frame.GetConstant(instruction.Bx)
		frame.PC++
		return nil

	case OP_ADD:
		// 内联ADD指令的快速路径
		valueB := frame.Registers[instruction.B]
		valueC := frame.Registers[instruction.C]

		// 小整数快速路径
		if valueB.IsSmallInt() && valueC.IsSmallInt() {
			aInt := valueB.AsSmallInt()
			bInt := valueC.AsSmallInt()
			result64 := int64(aInt) + int64(bInt)

			if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
				frame.Registers[instruction.A] = NewSmallIntValue(int32(result64))
				frame.PC++
				return nil
			}
		}

		// 复杂情况fallback
		result, err := AddValues(valueB, valueC)
		if err != nil {
			return err
		}
		frame.Registers[instruction.A] = result
		frame.PC++
		return nil

	case OP_SUB:
		// 内联SUB指令的快速路径
		valueB := frame.Registers[instruction.B]
		valueC := frame.Registers[instruction.C]

		// 小整数快速路径
		if valueB.IsSmallInt() && valueC.IsSmallInt() {
			aInt := valueB.AsSmallInt()
			bInt := valueC.AsSmallInt()
			result64 := int64(aInt) - int64(bInt)

			if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
				frame.Registers[instruction.A] = NewSmallIntValue(int32(result64))
				frame.PC++
				return nil
			}
		}

		// 复杂情况fallback
		result, err := SubtractValues(valueB, valueC)
		if err != nil {
			return err
		}
		frame.Registers[instruction.A] = result
		frame.PC++
		return nil

	case OP_MUL:
		// 内联MUL指令的快速路径
		valueB := frame.Registers[instruction.B]
		valueC := frame.Registers[instruction.C]

		// 小整数快速路径
		if valueB.IsSmallInt() && valueC.IsSmallInt() {
			aInt := valueB.AsSmallInt()
			bInt := valueC.AsSmallInt()
			result64 := int64(aInt) * int64(bInt)

			if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
				frame.Registers[instruction.A] = NewSmallIntValue(int32(result64))
				frame.PC++
				return nil
			}
		}

		// 复杂情况fallback
		result, err := MultiplyValues(valueB, valueC)
		if err != nil {
			return err
		}
		frame.Registers[instruction.A] = result
		frame.PC++
		return nil

	case OP_CALL:
		return e.executeCallOptimized(instruction)

	case OP_RETURN:
		return e.executeReturnOptimized(instruction)

	case OP_HALT:
		// 归还栈帧
		GlobalFramePool.Put(e.CurrentFrame)
		e.CurrentFrame = nil
		return nil

	default:
		return fmt.Errorf("unknown opcode: %d", instruction.OpCode)
	}
}

// 性能对比测试相关

// BenchmarkFramePoolVsNew 测试栈帧池 vs 直接创建的性能
func BenchmarkFramePoolVsNew() {
	// 这个函数可以在测试文件中调用来验证优化效果
}
