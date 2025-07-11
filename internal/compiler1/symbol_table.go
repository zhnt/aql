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
}

// SymbolTable 符号表
type SymbolTable struct {
	Outer          *SymbolTable
	store          map[string]Symbol
	numDefinitions int
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
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		if !ok {
			return obj, ok
		}

		if obj.Scope == GLOBAL_SCOPE || obj.Scope == BUILTIN_SCOPE {
			return obj, ok
		}

		// 自由变量
		free := s.defineFree(obj)
		return free, true
	}

	return obj, ok
}

// defineFree 定义自由变量
func (s *SymbolTable) defineFree(original Symbol) Symbol {
	s.store[original.Name] = Symbol{
		Name:  original.Name,
		Index: len(s.store),
		Scope: FREE_SCOPE,
	}

	return s.store[original.Name]
}
