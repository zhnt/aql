package compiler1

import (
	"github.com/zhnt/aql/internal/vm"
)

// 扩展现有的编译器支持闭包

// emitMakeClosureInstructions 生成创建闭包的指令（新版本）
func (c *Compiler) emitMakeClosureInstructions(functionID int, freeVars []*Symbol) (int, error) {
	// 1. 加载函数
	functionValue := vm.NewFunctionValueGCFromID(functionID)
	constIndex := c.addConstant(functionValue)
	funcReg := c.allocateRegister()
	c.emit(vm.OP_LOADK, funcReg, constIndex)

	// 2. 加载捕获变量
	for i, freeVar := range freeVars {
		captureReg := c.allocateRegister()
		if freeVar.Scope == GLOBAL_SCOPE {
			c.emit(vm.OP_GET_GLOBAL, captureReg, freeVar.Index)
		} else {
			c.emit(vm.OP_GET_LOCAL, captureReg, freeVar.Index)
		}

		// 移动到连续位置
		expectedReg := funcReg + 1 + i
		if captureReg != expectedReg {
			c.emit(vm.OP_MOVE, expectedReg, captureReg, 0)
		}
	}

	// 3. 创建闭包
	closureReg := c.allocateRegister()
	c.emit(vm.OP_MAKE_CLOSURE, closureReg, funcReg, len(freeVars))

	return closureReg, nil
}

// analyzeFreeVariables 分析自由变量（简化版）
func (c *Compiler) analyzeFreeVariables(freeSymbols []*Symbol) []string {
	freeVarNames := make([]string, len(freeSymbols))
	for i, symbol := range freeSymbols {
		freeVarNames[i] = symbol.Name
	}
	return freeVarNames
}

// shouldInline 简单的内联决策（可选优化）
func (c *Compiler) shouldInline(function *vm.Function) bool {
	// 简单的内联条件：
	// 1. 小函数（少于10条指令）
	// 2. 无捕获变量
	return len(function.Instructions) <= 10 &&
		(c.symbolTable.FreeSymbols == nil || len(c.symbolTable.FreeSymbols) == 0)
}

// shouldStackAlloc 简单的栈分配决策（可选优化）
func (c *Compiler) shouldStackAlloc(freeVars []*Symbol) bool {
	// 如果所有自由变量都是局部的，可以考虑栈分配
	for _, freeVar := range freeVars {
		if freeVar.Scope == GLOBAL_SCOPE {
			return false
		}
	}
	return true
}

// estimateClosureCost 估算闭包成本（可选优化）
func (c *Compiler) estimateClosureCost(freeVars []*Symbol) int {
	// 简单的成本模型：
	// 基础成本 + 每个捕获变量的成本
	baseCost := 10
	varCost := len(freeVars) * 2

	// 全局变量成本更高
	for _, freeVar := range freeVars {
		if freeVar.Scope == GLOBAL_SCOPE {
			varCost += 5
		}
	}

	return baseCost + varCost
}
