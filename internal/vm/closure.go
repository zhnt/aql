package vm

import "fmt"

// Closure 闭包对象，包含函数和捕获的变量
type Closure struct {
	Function *Function          // 闭包函数
	Captures map[string]ValueGC // 捕获的变量 {变量名: 值}
}

// NewClosure 创建新闭包
func NewClosure(function *Function, captures map[string]ValueGC) *Closure {
	if captures == nil {
		captures = make(map[string]ValueGC)
	}
	return &Closure{
		Function: function,
		Captures: captures,
	}
}

// GetCapture 获取捕获的变量
func (c *Closure) GetCapture(name string) (ValueGC, bool) {
	value, exists := c.Captures[name]
	return value, exists
}

// SetCapture 设置捕获的变量
func (c *Closure) SetCapture(name string, value ValueGC) {
	c.Captures[name] = value
}

// String 闭包的字符串表示
func (c *Closure) String() string {
	return fmt.Sprintf("closure:%s", c.Function.Name)
}

// CapturedVariable 捕获变量的元信息
type CapturedVariable struct {
	Name  string // 变量名
	Index int    // 在外部函数中的寄存器索引
}

// FunctionWithCaptures 带捕获信息的函数
type FunctionWithCaptures struct {
	*Function                      // 嵌入原始Function
	CaptureVars []CapturedVariable // 需要捕获的变量列表
}

// NewFunctionWithCaptures 创建带捕获信息的函数
func NewFunctionWithCaptures(function *Function, captures []CapturedVariable) *FunctionWithCaptures {
	return &FunctionWithCaptures{
		Function:    function,
		CaptureVars: captures,
	}
}
