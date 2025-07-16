package compiler1

// SymbolScope 符号作用域类型
type SymbolScope string

const (
	GLOBAL_SCOPE  SymbolScope = "GLOBAL"
	LOCAL_SCOPE   SymbolScope = "LOCAL"
	BUILTIN_SCOPE SymbolScope = "BUILTIN"
	FREE_SCOPE    SymbolScope = "FREE"
)

// Symbol 符号定义
type Symbol struct {
	Name       string
	Scope      SymbolScope
	Index      int
	IsConstant bool
	IsCaptured bool // 是否被内部函数捕获
}

// SymbolTable 符号表
type SymbolTable struct {
	Outer          *SymbolTable
	store          map[string]Symbol
	numDefinitions int
	FreeSymbols    []Symbol // 自由变量（外部捕获的变量）
}

// NewSymbolTable 创建新的符号表
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		store:          make(map[string]Symbol),
		numDefinitions: 0,
	}
}

// NewEnclosedSymbolTable 创建封闭的符号表（用于函数作用域）
func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

// Define 定义新符号
func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{
		Name:       name,
		Index:      s.numDefinitions,
		IsConstant: false,
	}

	if s.Outer == nil {
		symbol.Scope = GLOBAL_SCOPE
	} else {
		symbol.Scope = LOCAL_SCOPE
	}

	s.store[name] = symbol
	s.numDefinitions++

	return symbol
}

// DefineBuiltin 定义内建符号
func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{
		Name:  name,
		Index: index,
		Scope: BUILTIN_SCOPE,
	}
	s.store[name] = symbol
	return symbol
}

// Resolve 解析符号
func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if ok {
		return obj, ok
	}

	if s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		if !ok {
			return obj, ok
		}

		if obj.Scope == GLOBAL_SCOPE || obj.Scope == BUILTIN_SCOPE {
			return obj, ok
		}

		// 标记外部变量为被捕获（如果是局部变量）
		if obj.Scope == LOCAL_SCOPE && !obj.IsCaptured {
			obj.IsCaptured = true
			s.Outer.store[name] = obj // 更新外部符号表
		}

		// 创建自由变量
		free := s.defineFree(obj)
		return free, true
	}

	return obj, ok
}

// defineFree 定义自由变量
func (s *SymbolTable) defineFree(original Symbol) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)

	symbol := Symbol{
		Name:       original.Name,
		Index:      len(s.FreeSymbols) - 1,
		Scope:      FREE_SCOPE,
		IsConstant: original.IsConstant,
		IsCaptured: false,
	}

	s.store[original.Name] = symbol
	return symbol
}
