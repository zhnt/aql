package vm

import (
	"fmt"
)

// StackFrame AQL栈帧，使用优化的Value系统
type StackFrame struct {
	// 函数信息
	Function *Function
	PC       int // 程序计数器

	// 寄存器/局部变量（使用优化的Value）
	Registers []Value
	Base      int // 寄存器基址

	// 调用上下文
	Caller       *StackFrame // 调用者栈帧
	ReturnAddr   int         // 返回地址
	ExpectedRets int         // 期望返回值数量
}

// NewStackFrame 创建新栈帧
func NewStackFrame(function *Function, caller *StackFrame, returnAddr int) *StackFrame {
	// 计算寄存器大小，至少需要函数的最大栈大小
	regSize := function.MaxStackSize
	if regSize < 16 {
		regSize = 16 // 最小寄存器数量
	}

	frame := &StackFrame{
		Function:     function,
		PC:           0,
		Registers:    make([]Value, regSize),
		Base:         0,
		Caller:       caller,
		ReturnAddr:   returnAddr,
		ExpectedRets: 1,
	}

	// 初始化寄存器为nil值
	for i := range frame.Registers {
		frame.Registers[i] = NewNilValue()
	}

	return frame
}

// GetRegister 获取寄存器值
func (sf *StackFrame) GetRegister(index int) Value {
	if index < 0 || index >= len(sf.Registers) {
		return NewNilValue()
	}
	return sf.Registers[index]
}

// SetRegister 设置寄存器值
func (sf *StackFrame) SetRegister(index int, value Value) error {
	if index < 0 || index >= len(sf.Registers) {
		return fmt.Errorf("register index %d out of bounds", index)
	}
	sf.Registers[index] = value
	return nil
}

// GetConstant 获取函数常量
func (sf *StackFrame) GetConstant(index int) Value {
	return sf.Function.GetConstant(index)
}

// GetInstruction 获取当前指令
func (sf *StackFrame) GetInstruction() Instruction {
	if sf.PC < 0 || sf.PC >= len(sf.Function.Instructions) {
		return Instruction{OpCode: OP_HALT}
	}
	return sf.Function.Instructions[sf.PC]
}

// String 栈帧的字符串表示
func (sf *StackFrame) String() string {
	return fmt.Sprintf("StackFrame{func=%s, PC=%d}", sf.Function.Name, sf.PC)
}

// 优化方法：批量寄存器操作

// CopyRegisters 批量复制寄存器（用于函数调用参数传递）
func (sf *StackFrame) CopyRegisters(srcStart, dstStart, count int) error {
	for i := 0; i < count; i++ {
		srcIdx := srcStart + i
		dstIdx := dstStart + i

		if srcIdx >= 0 && srcIdx < len(sf.Registers) &&
			dstIdx >= 0 && dstIdx < len(sf.Registers) {
			sf.Registers[dstIdx] = sf.Registers[srcIdx]
		}
	}
	return nil
}

// ClearRegisters 清空指定范围的寄存器
func (sf *StackFrame) ClearRegisters(start, count int) {
	nilValue := NewNilValue()
	for i := 0; i < count; i++ {
		idx := start + i
		if idx >= 0 && idx < len(sf.Registers) {
			sf.Registers[idx] = nilValue
		}
	}
}

// SetParameters 设置函数参数
func (sf *StackFrame) SetParameters(params []Value) {
	for i, param := range params {
		if i < len(sf.Registers) {
			sf.Registers[i] = param
		}
	}
}

// GetReturnValues 获取返回值
func (sf *StackFrame) GetReturnValues(count int) []Value {
	if count <= 0 {
		return nil
	}

	results := make([]Value, count)
	for i := 0; i < count; i++ {
		if i < len(sf.Registers) {
			results[i] = sf.Registers[i]
		} else {
			results[i] = NewNilValue()
		}
	}
	return results
}
