package compiler1

import (
	"github.com/zhnt/aql/internal/vm"
)

// SimpleOptimizer 简单的编译优化器
type SimpleOptimizer struct {
	// 配置
	EnableInlining   bool
	EnableStackAlloc bool
	MaxInlineSize    int
	MaxInlineDepth   int

	// 统计
	InlineCount     int
	StackAllocCount int
}

// NewSimpleOptimizer 创建新的简单优化器
func NewSimpleOptimizer() *SimpleOptimizer {
	return &SimpleOptimizer{
		EnableInlining:   true,
		EnableStackAlloc: true,
		MaxInlineSize:    10, // 最多10条指令
		MaxInlineDepth:   3,  // 最多3层内联
		InlineCount:      0,
		StackAllocCount:  0,
	}
}

// OptimizeConfig 优化配置
type OptimizeConfig struct {
	EnableInlining   bool
	EnableStackAlloc bool
	MaxInlineSize    int
}

// DefaultOptimizeConfig 默认优化配置
func DefaultOptimizeConfig() *OptimizeConfig {
	return &OptimizeConfig{
		EnableInlining:   true,
		EnableStackAlloc: true,
		MaxInlineSize:    10,
	}
}

// CanInlineFunction 检查函数是否可以内联
func (opt *SimpleOptimizer) CanInlineFunction(function *vm.Function, freeVars int) bool {
	if !opt.EnableInlining {
		return false
	}

	// 基本条件：
	// 1. 函数足够小
	// 2. 没有自由变量或很少自由变量
	// 3. 没有复杂的控制流

	if len(function.Instructions) > opt.MaxInlineSize {
		return false
	}

	if freeVars > 2 {
		return false
	}

	// 检查指令复杂度
	for _, inst := range function.Instructions {
		switch inst.OpCode {
		case vm.OP_CALL, vm.OP_MAKE_CLOSURE:
			return false // 避免嵌套调用的内联
		case vm.OP_JUMP, vm.OP_JUMP_IF_FALSE, vm.OP_JUMP_IF_TRUE:
			return false // 避免复杂控制流的内联
		}
	}

	return true
}

// CanStackAllocClosure 检查闭包是否可以栈分配
func (opt *SimpleOptimizer) CanStackAllocClosure(freeVars []*Symbol, callSite string) bool {
	if !opt.EnableStackAlloc {
		return false
	}

	// 栈分配条件：
	// 1. 所有捕获变量都是局部的
	// 2. 捕获变量数量不多
	// 3. 在当前函数作用域内使用

	if len(freeVars) > 3 {
		return false
	}

	for _, freeVar := range freeVars {
		if freeVar.Scope == GLOBAL_SCOPE {
			return false
		}
	}

	return true
}

// EstimatePerformanceBenefit 估算性能收益
func (opt *SimpleOptimizer) EstimatePerformanceBenefit(optimizationType string, context map[string]interface{}) int {
	switch optimizationType {
	case "inline":
		// 内联的收益：减少函数调用开销
		return 50
	case "stack_alloc":
		// 栈分配的收益：减少堆分配开销
		return 30
	default:
		return 0
	}
}

// ApplyOptimizations 应用优化
func (opt *SimpleOptimizer) ApplyOptimizations(compiler *Compiler, function *vm.Function, freeVars []*Symbol) (*OptimizedFunction, error) {
	optimized := &OptimizedFunction{
		Original:      function,
		Optimizations: make([]string, 0),
	}

	// 尝试内联
	if opt.CanInlineFunction(function, len(freeVars)) {
		optimized.CanInline = true
		optimized.Optimizations = append(optimized.Optimizations, "inline")
		opt.InlineCount++
	}

	// 尝试栈分配
	if opt.CanStackAllocClosure(freeVars, "") {
		optimized.CanStackAlloc = true
		optimized.Optimizations = append(optimized.Optimizations, "stack_alloc")
		opt.StackAllocCount++
	}

	return optimized, nil
}

// OptimizedFunction 优化后的函数信息
type OptimizedFunction struct {
	Original         *vm.Function
	CanInline        bool
	CanStackAlloc    bool
	Optimizations    []string
	EstimatedBenefit int
}

// GetStats 获取优化统计信息
func (opt *SimpleOptimizer) GetStats() map[string]int {
	return map[string]int{
		"inline_count":      opt.InlineCount,
		"stack_alloc_count": opt.StackAllocCount,
	}
}

// Reset 重置统计信息
func (opt *SimpleOptimizer) Reset() {
	opt.InlineCount = 0
	opt.StackAllocCount = 0
}
