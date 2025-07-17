package compiler1

import (
	"github.com/zhnt/aql/internal/parser1"
	"github.com/zhnt/aql/internal/vm"
)

// 改进的寄存器分配策略
// 解决数组访问和赋值时的寄存器冲突问题

// RegisterAllocationContext 寄存器分配上下文
type RegisterAllocationContext struct {
	// 保留的寄存器集合 - 这些寄存器不应被重用
	reservedRegisters map[int]bool
	// 临时寄存器集合 - 用于复杂表达式的临时计算
	tempRegisters []int
	// 当前表达式的深度 - 用于嵌套表达式管理
	expressionDepth int
}

// 创建新的寄存器分配上下文
func (c *Compiler) newRegisterContext() *RegisterAllocationContext {
	return &RegisterAllocationContext{
		reservedRegisters: make(map[int]bool),
		tempRegisters:     make([]int, 0),
		expressionDepth:   0,
	}
}

// 改进的allocateRegister - 支持寄存器保留
func (c *Compiler) allocateRegisterWithContext(ctx *RegisterAllocationContext) int {
	// 首先尝试从空闲寄存器池中获取
	if len(c.freeRegisters) > 0 {
		for i := len(c.freeRegisters) - 1; i >= 0; i-- {
			reg := c.freeRegisters[i]

			// 检查是否与局部变量冲突
			if c.isLocalVariableRegister(reg) {
				continue
			}

			// 检查是否在保留寄存器集合中
			if ctx != nil && ctx.reservedRegisters[reg] {
				continue
			}

			// 找到可用的寄存器，从空闲池中移除
			c.freeRegisters = append(c.freeRegisters[:i], c.freeRegisters[i+1:]...)
			return reg
		}
	}

	// 如果没有合适的空闲寄存器，分配新的
	reg := c.nextRegister
	c.nextRegister++
	if c.nextRegister > c.maxRegisters {
		c.maxRegisters = c.nextRegister
	}
	return reg
}

// 检查寄存器是否被局部变量占用
func (c *Compiler) isLocalVariableRegister(reg int) bool {
	if c.symbolTable == nil {
		return false
	}

	for _, symbol := range c.symbolTable.store {
		if symbol.Scope == LOCAL_SCOPE && symbol.Index == reg {
			return true
		}
	}
	return false
}

// 保留寄存器 - 防止被重用
func (c *Compiler) reserveRegister(ctx *RegisterAllocationContext, reg int) {
	if ctx != nil {
		ctx.reservedRegisters[reg] = true
	}
}

// 释放保留的寄存器
func (c *Compiler) releaseReservedRegister(ctx *RegisterAllocationContext, reg int) {
	if ctx != nil {
		delete(ctx.reservedRegisters, reg)
	}
}

// 改进的compileIndexExpression - 避免寄存器冲突
func (c *Compiler) compileIndexExpressionSafe(expr *parser1.IndexExpression) (int, error) {
	// 创建寄存器分配上下文
	ctx := c.newRegisterContext()

	// 编译被索引的表达式（数组）
	leftReg, err := c.compileExpression(expr.Left)
	if err != nil {
		return -1, err
	}

	// 保留数组寄存器，防止被后续操作重用
	c.reserveRegister(ctx, leftReg)

	// 编译索引表达式
	indexReg, err := c.compileExpression(expr.Index)
	if err != nil {
		return -1, err
	}

	// 保留索引寄存器
	c.reserveRegister(ctx, indexReg)

	// 分配结果寄存器（不会与之前的寄存器冲突）
	resultReg := c.allocateRegisterWithContext(ctx)

	// 发射数组获取指令: ARRAY_GET resultReg, leftReg, indexReg
	c.emit(vm.OP_ARRAY_GET, resultReg, leftReg, indexReg)

	// 释放保留的寄存器
	c.releaseReservedRegister(ctx, leftReg)
	c.releaseReservedRegister(ctx, indexReg)

	return resultReg, nil
}

// 改进的compileIndexAssignmentExpression - 避免寄存器冲突
func (c *Compiler) compileIndexAssignmentExpressionSafe(expr *parser1.IndexAssignmentStatement) (int, error) {
	// 创建寄存器分配上下文
	ctx := c.newRegisterContext()

	// 编译右值表达式
	valueReg, err := c.compileExpression(expr.Value)
	if err != nil {
		return -1, err
	}

	// 保留值寄存器
	c.reserveRegister(ctx, valueReg)

	// 编译索引表达式（应该是IndexExpression）
	indexExpr, ok := expr.Left.(*parser1.IndexExpression)
	if !ok {
		return -1, &CompilationError{
			Message: "invalid index assignment target",
			Node:    expr.Left,
		}
	}

	// 编译被索引的表达式（数组）
	arrayReg, err := c.compileExpression(indexExpr.Left)
	if err != nil {
		return -1, err
	}

	// 保留数组寄存器
	c.reserveRegister(ctx, arrayReg)

	// 编译索引表达式
	indexReg, err := c.compileExpression(indexExpr.Index)
	if err != nil {
		return -1, err
	}

	// 保留索引寄存器
	c.reserveRegister(ctx, indexReg)

	// 发射数组设置指令: ARRAY_SET arrayReg, indexReg, valueReg
	c.emit(vm.OP_ARRAY_SET, arrayReg, indexReg, valueReg)

	// 释放所有保留的寄存器
	c.releaseReservedRegister(ctx, valueReg)
	c.releaseReservedRegister(ctx, arrayReg)
	c.releaseReservedRegister(ctx, indexReg)

	// 索引赋值表达式的结果就是被赋的值
	return valueReg, nil
}

// 改进的闭包编译 - upvalue优先分配策略
func (c *Compiler) compileClosureWithFixedUpvalues(functionID int, freeSymbols []*Symbol) (int, error) {
	// 为upvalue预留固定的寄存器范围
	upvalueBaseReg := c.getUpvalueBaseRegister()

	// 先加载函数到寄存器
	functionValue := vm.NewFunctionValueGCFromID(functionID)
	constIndex := c.addConstant(functionValue)

	// 确保函数寄存器不会与upvalue冲突
	funcReg := c.allocateRegisterAfter(upvalueBaseReg + len(freeSymbols))
	c.emit(vm.OP_LOADK, funcReg, constIndex)

	// 按固定顺序加载upvalue
	for i, freeVar := range freeSymbols {
		captureReg := upvalueBaseReg + i

		// 在当前符号表中查找这个变量
		currentSymbol, ok := c.symbolTable.Resolve(freeVar.Name)
		if !ok {
			return -1, &CompilationError{
				Message: "free variable not found in current scope: " + freeVar.Name,
			}
		}

		// 根据符号类型生成加载指令
		switch currentSymbol.Scope {
		case GLOBAL_SCOPE:
			c.emit(vm.OP_GET_GLOBAL, captureReg, currentSymbol.Index)
		case FREE_SCOPE:
			c.emit(vm.OP_GET_UPVALUE, captureReg, currentSymbol.Index)
		case LOCAL_SCOPE:
			c.emit(vm.OP_GET_LOCAL, captureReg, currentSymbol.Index)
		}
	}

	// 创建闭包
	closureReg := c.allocateRegisterAfter(funcReg)
	c.emit(vm.OP_MAKE_CLOSURE, closureReg, funcReg, len(freeSymbols))

	return closureReg, nil
}

// 获取upvalue基础寄存器位置
func (c *Compiler) getUpvalueBaseRegister() int {
	// 找到所有局部变量的最大索引
	maxLocalIndex := -1
	if c.symbolTable != nil {
		for _, symbol := range c.symbolTable.store {
			if symbol.Scope == LOCAL_SCOPE && symbol.Index > maxLocalIndex {
				maxLocalIndex = symbol.Index
			}
		}
	}

	// upvalue从局部变量后开始分配
	return maxLocalIndex + 1
}

// 在指定寄存器之后分配寄存器
func (c *Compiler) allocateRegisterAfter(minReg int) int {
	if c.nextRegister <= minReg {
		c.nextRegister = minReg + 1
	}

	reg := c.nextRegister
	c.nextRegister++
	if c.nextRegister > c.maxRegisters {
		c.maxRegisters = c.nextRegister
	}
	return reg
}

// 智能寄存器分配策略 - 考虑变量生命周期
func (c *Compiler) allocateRegisterWithLifetime(lifetime int) int {
	// 根据变量生命周期选择合适的寄存器
	// 短生命周期变量优先使用临时寄存器
	// 长生命周期变量使用固定寄存器

	if lifetime <= 3 {
		// 短生命周期 - 使用临时寄存器
		return c.allocateRegister()
	} else {
		// 长生命周期 - 使用固定寄存器
		reg := c.nextRegister
		c.nextRegister++
		if c.nextRegister > c.maxRegisters {
			c.maxRegisters = c.nextRegister
		}
		return reg
	}
}
