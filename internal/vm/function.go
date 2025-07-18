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
	OP_MOD                  // MOD A B C : R(A) := R(B) % R(C)
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
	OP_NEW_ARRAY               // NEW_ARRAY A B : R(A) := array(length=B)
	OP_NEW_ARRAY_WITH_CAPACITY // NEW_ARRAY_WITH_CAPACITY A B C : R(A) := array(capacity=R(B), default=R(C))
	OP_ARRAY_GET               // ARRAY_GET A B C : R(A) := R(B)[R(C)]
	OP_ARRAY_SET               // ARRAY_SET A B C : R(A)[R(B)] := R(C)
	OP_ARRAY_LEN               // ARRAY_LEN A B : R(A) := len(R(B))

	// GC相关指令
	OP_GC_WRITE_BARRIER // GC写屏障: GC_WRITE_BARRIER A B : WriteBarrier(R(A), R(B))
	OP_GC_INC_REF       // 增加引用计数: GC_INC_REF A : IncRef(R(A))
	OP_GC_DEC_REF       // 减少引用计数: GC_DEC_REF A : DecRef(R(A))
	OP_GC_ALLOC         // GC分配对象: GC_ALLOC A B : R(A) := GCAlloc(type=B)
	OP_GC_COLLECT       // 触发GC: GC_COLLECT : TriggerGC()
	OP_GC_CHECK         // GC检查: GC_CHECK A : R(A) := IsValidPtr(R(A))
	OP_GC_PIN           // 固定对象: GC_PIN A : Pin(R(A))
	OP_GC_UNPIN         // 取消固定: GC_UNPIN A : Unpin(R(A))

	// 闭包指令
	OP_MAKE_CLOSURE  // 创建闭包: MAKE_CLOSURE A B C : R(A) := Closure(function=R(B), capture_count=C, captures=R(B+1)...R(B+C))
	OP_GET_UPVALUE   // 获取upvalue: GET_UPVALUE A B : R(A) := Upvalue[B].Get()
	OP_SET_UPVALUE   // 设置upvalue: SET_UPVALUE A B : Upvalue[B].Set(R(A))
	OP_CLOSE_UPVALUE // 关闭upvalue: CLOSE_UPVALUE A : Close upvalues >= A

	// 内存管理指令
	OP_WEAK_REF // 创建弱引用: WEAK_REF A B : R(A) := WeakRef(R(B))
	OP_WEAK_GET // 获取弱引用值: WEAK_GET A B : R(A) := WeakGet(R(B))

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

	// 代码和常量（使用ValueGC）
	Instructions []Instruction // 指令序列
	Constants    []ValueGC     // 常量表（使用GC安全的ValueGC）

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
		Constants:    make([]ValueGC, 0),
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
func (f *Function) AddConstant(value ValueGC) int {
	f.Constants = append(f.Constants, value)
	return len(f.Constants) - 1
}

// GetConstant 获取常量
func (f *Function) GetConstant(index int) ValueGC {
	if index < 0 || index >= len(f.Constants) {
		return NewNilValueGC()
	}
	return f.Constants[index]
}

// 便利方法：添加不同类型的常量

func (f *Function) AddNumberConstant(n float64) int {
	return f.AddConstant(NewNumberValueGC(n))
}

func (f *Function) AddStringConstant(s string) int {
	return f.AddConstant(NewStringValueGC(s))
}

func (f *Function) AddBoolConstant(b bool) int {
	return f.AddConstant(NewBoolValueGC(b))
}
