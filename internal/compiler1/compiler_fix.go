package compiler1

import (
	"github.com/zhnt/aql/internal/vm"
)

// 修复自由变量捕获的问题
// 问题：在生成闭包时，自由变量的捕获使用了错误的作用域索引

// 修复后的compileFunctionLiteral方法中的自由变量处理部分
func (c *Compiler) generateFreeVariableCaptureFixed(freeSymbols []Symbol, funcReg int) ([]int, error) {
	numFreeVars := len(freeSymbols)
	captureRegs := make([]int, numFreeVars)

	for i, freeVar := range freeSymbols {
		captureReg := c.allocateRegister()

		// 关键修复：需要在当前作用域中正确地获取自由变量
		// 自由变量对于当前作用域来说应该通过符号表解析来访问

		// 在当前符号表中查找这个变量（应该被解析为FREE_SCOPE）
		currentSymbol, ok := c.symbolTable.Resolve(freeVar.Name)
		if !ok {
			// 如果找不到，说明编译器逻辑有问题
			return nil, &CompilationError{
				Message: "free variable not found in current scope: " + freeVar.Name,
			}
		}

		// 根据当前符号表中的解析结果生成指令
		switch currentSymbol.Scope {
		case GLOBAL_SCOPE:
			c.emit(vm.OP_GET_GLOBAL, captureReg, currentSymbol.Index)
		case FREE_SCOPE:
			c.emit(vm.OP_GET_UPVALUE, captureReg, currentSymbol.Index)
		case LOCAL_SCOPE:
			c.emit(vm.OP_GET_LOCAL, captureReg, currentSymbol.Index)
		default:
			return nil, &CompilationError{
				Message: "unsupported scope for free variable: " + string(currentSymbol.Scope),
			}
		}

		captureRegs[i] = captureReg
	}

	return captureRegs, nil
}

// 辅助函数：创建CompilationError
func (c *Compiler) newCompilationError(message string) *CompilationError {
	return &CompilationError{
		Message: message,
	}
}
