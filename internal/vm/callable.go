package vm

import "fmt"

// Callable 统一的可调用对象（函数和闭包）
// 学习Lua的简洁设计：函数和闭包使用同一个结构
type Callable struct {
	Function *Function  // 函数对象（复用现有）
	Upvalues []*Upvalue // upvalue数组（可为空）

	// 可选的优化信息
	IsInlinable bool  // 是否可内联
	CallCount   int32 // 调用计数（用于JIT决策）
}

// Upvalue 变量捕获容器（简化版，类似Lua）
type Upvalue struct {
	// 值存储（二选一）
	Stack *ValueGC // 指向栈变量（开放状态）
	Value ValueGC  // 堆上值（关闭状态）

	// 状态
	IsClosed bool // 是否已关闭到堆

	// 调试信息（可选）
	Name string // 变量名（仅用于调试）
}

// NewCallableFunction 创建普通函数（无upvalue）
func NewCallableFunction(function *Function) *Callable {
	return &Callable{
		Function: function,
		Upvalues: nil, // 普通函数无upvalue
	}
}

// NewCallableClosure 创建闭包（有upvalue）
func NewCallableClosure(function *Function, upvalues []*Upvalue) *Callable {
	return &Callable{
		Function: function,
		Upvalues: upvalues,
	}
}

// IsClosure 检查是否为闭包
func (c *Callable) IsClosure() bool {
	return len(c.Upvalues) > 0
}

// String 返回可调用对象的字符串表示
func (c *Callable) String() string {
	if c.IsClosure() {
		return fmt.Sprintf("closure:%s", c.Function.Name)
	}
	return fmt.Sprintf("function:%s", c.Function.Name)
}

// Upvalue操作（极简API）

// Get 获取upvalue的值
func (uv *Upvalue) Get() ValueGC {
	if uv.IsClosed {
		return uv.Value
	}
	if uv.Stack == nil {
		return NewNilValueGC()
	}
	return *uv.Stack
}

// Set 设置upvalue的值
func (uv *Upvalue) Set(value ValueGC) {
	if uv.IsClosed {
		uv.Value = value
	} else if uv.Stack != nil {
		*uv.Stack = value
	}
}

// Close 关闭upvalue到堆（栈帧销毁时调用）
func (uv *Upvalue) Close() {
	if !uv.IsClosed && uv.Stack != nil {
		// 将栈上的值复制到堆
		uv.Value = *uv.Stack
		uv.Stack = nil
		uv.IsClosed = true
	}
}

// IsOpen 检查upvalue是否处于开放状态
func (uv *Upvalue) IsOpen() bool {
	return !uv.IsClosed
}

// NewUpvalue 创建新的upvalue
func NewUpvalue(name string, stackSlot *ValueGC) *Upvalue {
	return &Upvalue{
		Stack:    stackSlot,
		Value:    NewNilValueGC(),
		IsClosed: false,
		Name:     name,
	}
}

// String 返回upvalue的字符串表示
func (uv *Upvalue) String() string {
	status := "open"
	if uv.IsClosed {
		status = "closed"
	}
	return fmt.Sprintf("upvalue:%s(%s)", uv.Name, status)
}
