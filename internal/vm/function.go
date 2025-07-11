package vm

// OpCode VM指令操作码
type OpCode uint8

const (
	// Lua风格基础指令
	OP_MOVE   OpCode = iota // MOVE A B : R(A) := R(B)
	OP_LOADK                // LOADK A Bx : R(A) := K(Bx)
	OP_ADD                  // ADD A B C : R(A) := R(B) + R(C)
	OP_SUB                  // SUB A B C : R(A) := R(B) - R(C)
	OP_MUL                  // MUL A B C : R(A) := R(B) * R(C)
	OP_DIV                  // DIV A B C : R(A) := R(B) / R(C)
	OP_CALL                 // CALL A B C : R(A) := R(A)(R(A+1), ..., R(A+B-1))
	OP_RETURN               // RETURN A B : return R(A), ..., R(A+B-2)
	OP_HALT                 // HALT : 停止执行

	// 比较指令
	OP_EQ  // EQ A B C : R(A) := R(B) == R(C)
	OP_NEQ // NEQ A B C : R(A) := R(B) != R(C)
	OP_LT  // LT A B C : R(A) := R(B) < R(C)
	OP_GT  // GT A B C : R(A) := R(B) > R(C)
	OP_LTE // LTE A B C : R(A) := R(B) <= R(C)
	OP_GTE // GTE A B C : R(A) := R(B) >= R(C)

	// 逻辑指令
	OP_NOT // NOT A B : R(A) := !R(B)
	OP_NEG // NEG A B : R(A) := -R(B)

	// 变量访问指令
	OP_GET_GLOBAL // GET_GLOBAL A Bx : R(A) := G(Bx)
	OP_SET_GLOBAL // SET_GLOBAL A Bx : G(Bx) := R(A)
	OP_GET_LOCAL  // GET_LOCAL A B : R(A) := L(B)
	OP_SET_LOCAL  // SET_LOCAL A B : L(B) := R(A)

	// 栈操作指令
	OP_POP // POP : 弹出栈顶

	// 跳转指令
	OP_JUMP          // JUMP Bx : PC := PC + Bx
	OP_JUMP_IF_FALSE // JUMP_IF_FALSE A Bx : if !R(A) then PC := PC + Bx
	OP_JUMP_IF_TRUE  // JUMP_IF_TRUE A Bx : if R(A) then PC := PC + Bx

	// 数组操作指令
	OP_NEW_ARRAY // NEW_ARRAY A B : R(A) := array(length=B)
	OP_ARRAY_GET // ARRAY_GET A B C : R(A) := R(B)[R(C)]
	OP_ARRAY_SET // ARRAY_SET A B C : R(A)[R(B)] := R(C)
	OP_ARRAY_LEN // ARRAY_LEN A B : R(A) := len(R(B))

	// AQL扩展指令（为将来准备）
	OP_ASYNC_CALL // 异步函数调用
	OP_AWAIT      // await操作
	OP_YIELD      // 协程yield
)

// Instruction VM指令表示
type Instruction struct {
	OpCode OpCode
	A      int // 第一个操作数
	B      int // 第二个操作数
	C      int // 第三个操作数
	Bx     int // 扩展操作数（用于常量索引等）
}

// Function AQL函数表示，使用优化的Value系统
type Function struct {
	// 基础信息
	Name         string // 函数名称
	ParamCount   int    // 参数数量
	IsVarArg     bool   // 是否支持变参
	MaxStackSize int    // 最大栈大小

	// 代码和常量（使用优化的Value）
	Instructions []Instruction // 指令序列
	Constants    []Value       // 常量表（使用优化的Value）

	// 调试信息
	Source      string // 源文件路径
	LineNumbers []int  // 行号映射

	// 异步支持（为将来准备）
	IsAsync bool // 是否为异步函数
}

// NewFunction 创建新函数
func NewFunction(name string) *Function {
	return &Function{
		Name:         name,
		ParamCount:   0,
		IsVarArg:     false,
		MaxStackSize: 0,
		Instructions: make([]Instruction, 0),
		Constants:    make([]Value, 0),
		LineNumbers:  make([]int, 0),
		IsAsync:      false,
	}
}

// AddInstruction 添加指令
func (f *Function) AddInstruction(op OpCode, a, b, c int) {
	f.Instructions = append(f.Instructions, Instruction{
		OpCode: op,
		A:      a,
		B:      b,
		C:      c,
	})
}

// AddInstructionBx 添加带扩展操作数的指令
func (f *Function) AddInstructionBx(op OpCode, a, bx int) {
	f.Instructions = append(f.Instructions, Instruction{
		OpCode: op,
		A:      a,
		Bx:     bx,
	})
}

// AddConstant 添加常量，返回索引
func (f *Function) AddConstant(value Value) int {
	f.Constants = append(f.Constants, value)
	return len(f.Constants) - 1
}

// GetConstant 获取常量
func (f *Function) GetConstant(index int) Value {
	if index < 0 || index >= len(f.Constants) {
		return NewNilValue()
	}
	return f.Constants[index]
}

// 便利方法：添加不同类型的常量

func (f *Function) AddNumberConstant(n float64) int {
	return f.AddConstant(NewNumberValue(n))
}

func (f *Function) AddStringConstant(s string) int {
	return f.AddConstant(NewStringValue(s))
}

func (f *Function) AddBoolConstant(b bool) int {
	return f.AddConstant(NewBoolValue(b))
}
