package compiler1

import (
	"fmt"

	"github.com/zhnt/aql/internal/parser1"
	"github.com/zhnt/aql/internal/vm"
)

// Compiler AQL编译器，将AST编译为VM字节码
type Compiler struct {
	constants    []vm.Value      // 常量池
	symbolTable  *SymbolTable    // 符号表
	scopes       []*CompileScope // 作用域栈
	scopeIndex   int             // 当前作用域索引
	nextRegister int             // 下一个可用寄存器
	maxRegisters int             // 最大寄存器使用数
	loopStack    []*LoopContext  // 循环栈，用于break/continue
}

// LoopContext 循环上下文，用于处理break/continue
type LoopContext struct {
	breakJumps    []int // 需要回填的break跳转位置
	continueJumps []int // 需要回填的continue跳转位置
	updateStart   int   // 循环更新部分开始位置
}

// CompileScope 编译作用域
type CompileScope struct {
	instructions        []vm.Instruction // 当前作用域的指令
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

// EmittedInstruction 已发射的指令信息
type EmittedInstruction struct {
	OpCode   vm.OpCode
	Position int
}

// CompilationError 编译错误
type CompilationError struct {
	Message string
	Node    parser1.Node
}

func (ce *CompilationError) Error() string {
	return fmt.Sprintf("编译错误: %s", ce.Message)
}

// New 创建新的编译器
func New() *Compiler {
	mainScope := &CompileScope{
		instructions:        make([]vm.Instruction, 0),
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	return &Compiler{
		constants:    make([]vm.Value, 0),
		symbolTable:  NewSymbolTable(),
		scopes:       []*CompileScope{mainScope},
		scopeIndex:   0,
		nextRegister: 0,
		maxRegisters: 0,
		loopStack:    make([]*LoopContext, 0),
	}
}

// Compile 编译程序为VM函数
func (c *Compiler) Compile(node parser1.Node) (*vm.Function, error) {
	switch node := node.(type) {
	case *parser1.Program:
		return c.compileProgram(node)
	default:
		return nil, &CompilationError{
			Message: fmt.Sprintf("unsupported node type: %T", node),
			Node:    node,
		}
	}
}

// compileProgram 编译程序
func (c *Compiler) compileProgram(program *parser1.Program) (*vm.Function, error) {
	for _, stmt := range program.Statements {
		err := c.compileStatement(stmt)
		if err != nil {
			return nil, err
		}
	}

	// 创建主函数
	function := vm.NewFunction("main")
	function.Instructions = c.currentInstructions()
	function.Constants = c.constants
	function.MaxStackSize = 256 // 临时设置，后续可优化

	return function, nil
}

// compileStatement 编译语句
func (c *Compiler) compileStatement(stmt parser1.Statement) error {
	switch stmt := stmt.(type) {
	case *parser1.LetStatement:
		return c.compileLetStatement(stmt)
	case *parser1.ConstStatement:
		return c.compileConstStatement(stmt)
	case *parser1.ReturnStatement:
		return c.compileReturnStatement(stmt)
	case *parser1.ExpressionStatement:
		return c.compileExpressionStatement(stmt)
	case *parser1.ForStatement:
		return c.compileForStatement(stmt)
	case *parser1.WhileStatement:
		return c.compileWhileStatement(stmt)
	case *parser1.BreakStatement:
		return c.compileBreakStatement(stmt)
	case *parser1.ContinueStatement:
		return c.compileContinueStatement(stmt)
	default:
		return &CompilationError{
			Message: fmt.Sprintf("unsupported statement type: %T", stmt),
			Node:    stmt,
		}
	}
}

// compileLetStatement 编译let语句
func (c *Compiler) compileLetStatement(stmt *parser1.LetStatement) error {
	// 编译右值表达式
	reg, err := c.compileExpression(stmt.Value)
	if err != nil {
		return err
	}

	// 定义符号
	symbol := c.symbolTable.Define(stmt.Name.Value)

	// 发射存储指令
	if symbol.Scope == GLOBAL_SCOPE {
		c.emit(vm.OP_SET_GLOBAL, reg, symbol.Index) // G(symbol.Index) := R[reg]
	} else {
		c.emit(vm.OP_SET_LOCAL, reg, symbol.Index) // L(symbol.Index) := R[reg]
	}

	return nil
}

// compileConstStatement 编译const语句
func (c *Compiler) compileConstStatement(stmt *parser1.ConstStatement) error {
	// const的编译与let相同，但符号标记为常量
	reg, err := c.compileExpression(stmt.Value)
	if err != nil {
		return err
	}

	symbol := c.symbolTable.Define(stmt.Name.Value)
	symbol.IsConstant = true

	if symbol.Scope == GLOBAL_SCOPE {
		c.emit(vm.OP_SET_GLOBAL, reg, symbol.Index) // G(symbol.Index) := R[reg]
	} else {
		c.emit(vm.OP_SET_LOCAL, reg, symbol.Index) // L(symbol.Index) := R[reg]
	}

	return nil
}

// compileReturnStatement 编译return语句
func (c *Compiler) compileReturnStatement(stmt *parser1.ReturnStatement) error {
	if stmt.ReturnValue != nil {
		reg, err := c.compileExpression(stmt.ReturnValue)
		if err != nil {
			return err
		}
		c.emit(vm.OP_RETURN, reg, 1, 0) // return R[reg], 1个返回值
	} else {
		// 没有返回值，返回nil
		nilReg := c.allocateRegister()
		c.emit(vm.OP_LOADK, nilReg, c.addConstant(vm.NewNilValue()))
		c.emit(vm.OP_RETURN, nilReg, 1, 0) // return R[nilReg], 1个返回值
	}
	return nil
}

// compileExpressionStatement 编译表达式语句
func (c *Compiler) compileExpressionStatement(stmt *parser1.ExpressionStatement) error {
	reg, err := c.compileExpression(stmt.Expression)
	if err != nil {
		return err
	}

	// 将结果移动到寄存器0，以便VM能正确返回结果
	if reg != 0 {
		c.emit(vm.OP_MOVE, 0, reg, 0) // R0 := R[reg]
	}

	// 表达式语句的结果被丢弃（但已经保存在R0中）
	c.emit(vm.OP_POP)
	// 在表达式语句结束后重置寄存器分配器
	c.resetRegisters()
	return nil
}

// compileExpression 编译表达式，返回结果所在的寄存器号
func (c *Compiler) compileExpression(expr parser1.Expression) (int, error) {
	switch expr := expr.(type) {
	case *parser1.IntegerLiteral:
		return c.compileIntegerLiteral(expr)
	case *parser1.FloatLiteral:
		return c.compileFloatLiteral(expr)
	case *parser1.StringLiteral:
		return c.compileStringLiteral(expr)
	case *parser1.BooleanLiteral:
		return c.compileBooleanLiteral(expr)
	case *parser1.NullLiteral:
		return c.compileNullLiteral(expr)
	case *parser1.Identifier:
		return c.compileIdentifier(expr)
	case *parser1.AssignmentStatement:
		return c.compileAssignmentExpression(expr)
	case *parser1.InfixExpression:
		return c.compileInfixExpression(expr)
	case *parser1.PrefixExpression:
		return c.compilePrefixExpression(expr)
	case *parser1.IfStatement:
		return c.compileIfExpression(expr)
	case *parser1.FunctionLiteral:
		return c.compileFunctionLiteral(expr)
	case *parser1.CallExpression:
		return c.compileCallExpression(expr)
	case *parser1.ArrayLiteral:
		return c.compileArrayLiteral(expr)
	case *parser1.IndexExpression:
		return c.compileIndexExpression(expr)
	default:
		return -1, &CompilationError{
			Message: fmt.Sprintf("unsupported expression type: %T", expr),
			Node:    expr,
		}
	}
}

// 字面量编译方法

func (c *Compiler) compileIntegerLiteral(expr *parser1.IntegerLiteral) (int, error) {
	integer := vm.NewNumberValue(float64(expr.Value))
	constIndex := c.addConstant(integer)
	reg := c.allocateRegister()
	c.emit(vm.OP_LOADK, reg, constIndex)
	return reg, nil
}

func (c *Compiler) compileFloatLiteral(expr *parser1.FloatLiteral) (int, error) {
	float := vm.NewNumberValue(expr.Value)
	constIndex := c.addConstant(float)
	reg := c.allocateRegister()
	c.emit(vm.OP_LOADK, reg, constIndex)
	return reg, nil
}

func (c *Compiler) compileStringLiteral(expr *parser1.StringLiteral) (int, error) {
	str := vm.NewStringValue(expr.Value)
	constIndex := c.addConstant(str)
	reg := c.allocateRegister()
	c.emit(vm.OP_LOADK, reg, constIndex)
	return reg, nil
}

func (c *Compiler) compileBooleanLiteral(expr *parser1.BooleanLiteral) (int, error) {
	boolean := vm.NewBoolValue(expr.Value)
	constIndex := c.addConstant(boolean)
	reg := c.allocateRegister()
	c.emit(vm.OP_LOADK, reg, constIndex)
	return reg, nil
}

func (c *Compiler) compileNullLiteral(expr *parser1.NullLiteral) (int, error) {
	null := vm.NewNilValue()
	constIndex := c.addConstant(null)
	reg := c.allocateRegister()
	c.emit(vm.OP_LOADK, reg, constIndex)
	return reg, nil
}

func (c *Compiler) compileIdentifier(expr *parser1.Identifier) (int, error) {
	symbol, ok := c.symbolTable.Resolve(expr.Value)
	if !ok {
		return -1, &CompilationError{
			Message: fmt.Sprintf("undefined variable: %s", expr.Value),
			Node:    expr,
		}
	}

	reg := c.allocateRegister()
	if symbol.Scope == GLOBAL_SCOPE {
		c.emit(vm.OP_GET_GLOBAL, reg, symbol.Index) // R[reg] := G(symbol.Index)
	} else {
		c.emit(vm.OP_GET_LOCAL, reg, symbol.Index) // R[reg] := L(symbol.Index)
	}

	return reg, nil
}

// compileAssignmentExpression 编译赋值表达式
func (c *Compiler) compileAssignmentExpression(expr *parser1.AssignmentStatement) (int, error) {
	// 编译右值表达式
	valueReg, err := c.compileExpression(expr.Value)
	if err != nil {
		return -1, err
	}

	// 检查变量是否已存在
	symbol, ok := c.symbolTable.Resolve(expr.Name.Value)
	if !ok {
		// 变量不存在，定义新变量（Python风格）
		symbol = c.symbolTable.Define(expr.Name.Value)
	}

	// 发射存储指令
	if symbol.Scope == GLOBAL_SCOPE {
		c.emit(vm.OP_SET_GLOBAL, valueReg, symbol.Index) // G(symbol.Index) := R[valueReg]
	} else {
		c.emit(vm.OP_SET_LOCAL, valueReg, symbol.Index) // L(symbol.Index) := R[valueReg]
	}

	// 赋值表达式的结果就是被赋的值
	return valueReg, nil
}

// 运算表达式编译方法

func (c *Compiler) compileInfixExpression(expr *parser1.InfixExpression) (int, error) {
	// 特殊处理比较运算符的短路求值
	if expr.Operator == "<" {
		leftReg, err := c.compileExpression(expr.Left)
		if err != nil {
			return -1, err
		}

		rightReg, err := c.compileExpression(expr.Right)
		if err != nil {
			return -1, err
		}

		resultReg := c.allocateRegister()
		c.emit(vm.OP_LT, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] < R[rightReg]
		return resultReg, nil
	}

	// 处理其他运算符
	leftReg, err := c.compileExpression(expr.Left)
	if err != nil {
		return -1, err
	}

	rightReg, err := c.compileExpression(expr.Right)
	if err != nil {
		return -1, err
	}

	resultReg := c.allocateRegister()
	switch expr.Operator {
	case "+":
		c.emit(vm.OP_ADD, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] + R[rightReg]
	case "-":
		c.emit(vm.OP_SUB, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] - R[rightReg]
	case "*":
		c.emit(vm.OP_MUL, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] * R[rightReg]
	case "/":
		c.emit(vm.OP_DIV, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] / R[rightReg]
	case "==":
		c.emit(vm.OP_EQ, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] == R[rightReg]
	case "!=":
		c.emit(vm.OP_NEQ, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] != R[rightReg]
	case ">":
		c.emit(vm.OP_GT, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] > R[rightReg]
	case "<=":
		c.emit(vm.OP_LTE, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] <= R[rightReg]
	case ">=":
		c.emit(vm.OP_GTE, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] >= R[rightReg]
	default:
		return -1, &CompilationError{
			Message: fmt.Sprintf("unknown operator: %s", expr.Operator),
			Node:    expr,
		}
	}

	return resultReg, nil
}

func (c *Compiler) compilePrefixExpression(expr *parser1.PrefixExpression) (int, error) {
	rightReg, err := c.compileExpression(expr.Right)
	if err != nil {
		return -1, err
	}

	resultReg := c.allocateRegister()
	switch expr.Operator {
	case "!":
		c.emit(vm.OP_NOT, resultReg, rightReg, 0) // R[resultReg] := !R[rightReg]
	case "-":
		c.emit(vm.OP_NEG, resultReg, rightReg, 0) // R[resultReg] := -R[rightReg]
	default:
		return -1, &CompilationError{
			Message: fmt.Sprintf("unknown operator: %s", expr.Operator),
			Node:    expr,
		}
	}

	return resultReg, nil
}

// 复杂表达式编译方法（临时简化实现）

func (c *Compiler) compileIfExpression(expr *parser1.IfStatement) (int, error) {
	// 编译条件表达式
	conditionReg, err := c.compileExpression(expr.Condition)
	if err != nil {
		return -1, err
	}

	// 发射条件跳转指令：如果条件为假，跳过if块
	jumpIfFalsePos := c.emit(vm.OP_JUMP_IF_FALSE, conditionReg, 9999)

	// 分配结果寄存器
	resultReg := c.allocateRegister()

	// 编译if块（consequence）- 只编译除了最后一个表达式语句之外的所有语句
	ifStmts := expr.Consequence.Statements
	for i, stmt := range ifStmts {
		if i == len(ifStmts)-1 {
			// 最后一个语句，特殊处理
			if exprStmt, ok := stmt.(*parser1.ExpressionStatement); ok {
				// 编译表达式并将结果存储到resultReg
				reg, err := c.compileExpression(exprStmt.Expression)
				if err != nil {
					return -1, err
				}
				c.emit(vm.OP_MOVE, resultReg, reg, 0)
			} else {
				// 非表达式语句，编译它并设置结果为nil
				err := c.compileStatement(stmt)
				if err != nil {
					return -1, err
				}
				c.emit(vm.OP_LOADK, resultReg, c.addConstant(vm.NewNilValue()))
			}
		} else {
			// 其他语句正常编译
			err := c.compileStatement(stmt)
			if err != nil {
				return -1, err
			}
		}
	}

	// 如果if块为空，设置结果为nil
	if len(ifStmts) == 0 {
		c.emit(vm.OP_LOADK, resultReg, c.addConstant(vm.NewNilValue()))
	}

	var jumpOverElsePos int = -1

	// 如果有else块
	if expr.Alternative != nil {
		// 发射无条件跳转指令：跳过else块
		jumpOverElsePos = c.emit(vm.OP_JUMP, 9999)
	}

	// 回填第一个跳转指令的目标地址
	currentPos := len(c.currentInstructions())
	jumpTarget := currentPos - jumpIfFalsePos
	c.scopes[c.scopeIndex].instructions[jumpIfFalsePos].Bx = jumpTarget

	// 如果有else块，编译它
	if expr.Alternative != nil {
		elseStmts := expr.Alternative.Statements
		for i, stmt := range elseStmts {
			if i == len(elseStmts)-1 {
				// 最后一个语句，特殊处理
				if exprStmt, ok := stmt.(*parser1.ExpressionStatement); ok {
					// 编译表达式并将结果存储到resultReg
					reg, err := c.compileExpression(exprStmt.Expression)
					if err != nil {
						return -1, err
					}
					c.emit(vm.OP_MOVE, resultReg, reg, 0)
				} else {
					// 非表达式语句，编译它并设置结果为nil
					err := c.compileStatement(stmt)
					if err != nil {
						return -1, err
					}
					c.emit(vm.OP_LOADK, resultReg, c.addConstant(vm.NewNilValue()))
				}
			} else {
				// 其他语句正常编译
				err := c.compileStatement(stmt)
				if err != nil {
					return -1, err
				}
			}
		}

		// 如果else块为空，设置结果为nil
		if len(elseStmts) == 0 {
			c.emit(vm.OP_LOADK, resultReg, c.addConstant(vm.NewNilValue()))
		}

		// 回填跳过else块的跳转指令
		currentPos = len(c.currentInstructions())
		jumpTarget = currentPos - jumpOverElsePos
		c.scopes[c.scopeIndex].instructions[jumpOverElsePos].Bx = jumpTarget
	}

	return resultReg, nil
}

// compileCommaExpression 已移除 - AQL不再支持逗号运算符

// compileForStatement 编译for循环语句
// for (init; condition; update) { body }
func (c *Compiler) compileForStatement(stmt *parser1.ForStatement) error {
	// 1. 编译初始化语句
	if stmt.Init != nil {
		err := c.compileStatement(stmt.Init)
		if err != nil {
			return err
		}
	}

	// 2. 创建循环上下文并推入循环栈
	loopContext := &LoopContext{
		breakJumps:    make([]int, 0),
		continueJumps: make([]int, 0),
		updateStart:   -1,
	}
	c.loopStack = append(c.loopStack, loopContext)

	// 3. 记录循环开始位置
	loopStart := len(c.currentInstructions())

	var conditionJumpPos int = -1

	// 4. 编译条件表达式
	if stmt.Condition != nil {
		conditionReg, err := c.compileExpression(stmt.Condition)
		if err != nil {
			return err
		}

		// 如果条件为假，跳转到循环结束（占位符，稍后回填）
		conditionJumpPos = c.emit(vm.OP_JUMP_IF_FALSE, conditionReg, 9999)
	}

	// 5. 编译循环体
	err := c.compileBlockStatement(stmt.Body)
	if err != nil {
		return err
	}

	// 6. 记录更新部分开始位置
	updateStart := len(c.currentInstructions())
	loopContext.updateStart = updateStart

	// 7. 编译更新表达式
	if stmt.Update != nil {
		_, err := c.compileExpression(stmt.Update)
		if err != nil {
			return err
		}
	}

	// 8. 发射跳转指令：跳回循环开始
	currentPos := len(c.currentInstructions())
	jumpBackOffset := loopStart - currentPos
	c.emit(vm.OP_JUMP, jumpBackOffset)

	// 9. 记录循环结束位置
	loopEnd := len(c.currentInstructions())

	// 10. 回填条件跳转指令
	if conditionJumpPos != -1 {
		jumpTarget := loopEnd - conditionJumpPos
		c.scopes[c.scopeIndex].instructions[conditionJumpPos].Bx = jumpTarget
	}

	// 11. 回填所有break跳转（跳转到循环结束）
	for _, breakJumpPos := range loopContext.breakJumps {
		jumpTarget := loopEnd - breakJumpPos
		c.scopes[c.scopeIndex].instructions[breakJumpPos].Bx = jumpTarget
	}

	// 12. 回填所有continue跳转（跳转到更新部分）
	for _, continueJumpPos := range loopContext.continueJumps {
		jumpTarget := updateStart - continueJumpPos
		c.scopes[c.scopeIndex].instructions[continueJumpPos].Bx = jumpTarget
	}

	// 13. 弹出循环上下文
	c.loopStack = c.loopStack[:len(c.loopStack)-1]

	return nil
}

// compileWhileStatement 编译while循环语句
// while (condition) { body }
func (c *Compiler) compileWhileStatement(stmt *parser1.WhileStatement) error {
	// 1. 创建循环上下文并推入循环栈
	loopContext := &LoopContext{
		breakJumps:    make([]int, 0),
		continueJumps: make([]int, 0),
		updateStart:   -1, // while循环没有更新部分
	}
	c.loopStack = append(c.loopStack, loopContext)

	// 2. 记录循环开始位置（条件检查）
	loopStart := len(c.currentInstructions())
	loopContext.updateStart = loopStart // continue跳回到条件检查

	// 3. 编译条件表达式
	conditionReg, err := c.compileExpression(stmt.Condition)
	if err != nil {
		return err
	}

	// 4. 如果条件为假，跳转到循环结束（占位符，稍后回填）
	conditionJumpPos := c.emit(vm.OP_JUMP_IF_FALSE, conditionReg, 9999)

	// 5. 编译循环体
	err = c.compileBlockStatement(stmt.Body)
	if err != nil {
		return err
	}

	// 6. 发射跳转指令：跳回循环开始（条件检查）
	currentPos := len(c.currentInstructions())
	jumpBackOffset := loopStart - currentPos
	c.emit(vm.OP_JUMP, jumpBackOffset)

	// 7. 记录循环结束位置
	loopEnd := len(c.currentInstructions())

	// 8. 回填条件跳转指令
	jumpTarget := loopEnd - conditionJumpPos
	c.scopes[c.scopeIndex].instructions[conditionJumpPos].Bx = jumpTarget

	// 9. 回填所有break跳转（跳转到循环结束）
	for _, breakJumpPos := range loopContext.breakJumps {
		jumpTarget := loopEnd - breakJumpPos
		c.scopes[c.scopeIndex].instructions[breakJumpPos].Bx = jumpTarget
	}

	// 10. 回填所有continue跳转（跳转到条件检查）
	for _, continueJumpPos := range loopContext.continueJumps {
		jumpTarget := loopStart - continueJumpPos
		c.scopes[c.scopeIndex].instructions[continueJumpPos].Bx = jumpTarget
	}

	// 11. 弹出循环上下文
	c.loopStack = c.loopStack[:len(c.loopStack)-1]

	return nil
}

// compileBreakStatement 编译break语句
func (c *Compiler) compileBreakStatement(stmt *parser1.BreakStatement) error {
	// 检查是否在循环中
	if len(c.loopStack) == 0 {
		return &CompilationError{
			Message: "break statement not within a loop",
			Node:    stmt,
		}
	}

	// 获取当前循环上下文
	currentLoop := c.loopStack[len(c.loopStack)-1]

	// 发射跳转指令，跳转到循环结束（占位符，稍后回填）
	jumpPos := c.emit(vm.OP_JUMP, 9999)
	currentLoop.breakJumps = append(currentLoop.breakJumps, jumpPos)

	return nil
}

// compileContinueStatement 编译continue语句
func (c *Compiler) compileContinueStatement(stmt *parser1.ContinueStatement) error {
	// 检查是否在循环中
	if len(c.loopStack) == 0 {
		return &CompilationError{
			Message: "continue statement not within a loop",
			Node:    stmt,
		}
	}

	// 获取当前循环上下文
	currentLoop := c.loopStack[len(c.loopStack)-1]

	// 发射跳转指令，跳转到循环更新部分（占位符，稍后回填）
	jumpPos := c.emit(vm.OP_JUMP, 9999)
	currentLoop.continueJumps = append(currentLoop.continueJumps, jumpPos)

	return nil
}

// compileBlockStatement 编译代码块语句
func (c *Compiler) compileBlockStatement(block *parser1.BlockStatement) error {
	for _, stmt := range block.Statements {
		err := c.compileStatement(stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileFunctionLiteral(expr *parser1.FunctionLiteral) (int, error) {
	// 简化的函数字面量编译
	// TODO: 实现函数编译
	return -1, &CompilationError{
		Message: "function literals not yet implemented",
		Node:    expr,
	}
}

func (c *Compiler) compileCallExpression(expr *parser1.CallExpression) (int, error) {
	// 简化的函数调用编译
	// TODO: 实现函数调用
	return -1, &CompilationError{
		Message: "function calls not yet implemented",
		Node:    expr,
	}
}

func (c *Compiler) compileArrayLiteral(expr *parser1.ArrayLiteral) (int, error) {
	// 创建新数组
	length := len(expr.Elements)
	arrayReg := c.allocateRegister()

	// 发射创建数组指令: NEW_ARRAY arrayReg, length
	c.emit(vm.OP_NEW_ARRAY, arrayReg, length, 0)

	// 编译并设置每个元素
	for i, element := range expr.Elements {
		// 编译元素表达式
		elementReg, err := c.compileExpression(element)
		if err != nil {
			return -1, err
		}

		// 创建索引常量
		indexReg := c.allocateRegister()
		indexConstIndex := c.addConstant(vm.NewSmallIntValue(int32(i)))
		c.emit(vm.OP_LOADK, indexReg, indexConstIndex)

		// 设置数组元素: ARRAY_SET arrayReg, indexReg, elementReg
		c.emit(vm.OP_ARRAY_SET, arrayReg, indexReg, elementReg)
	}

	return arrayReg, nil
}

func (c *Compiler) compileIndexExpression(expr *parser1.IndexExpression) (int, error) {
	// 编译被索引的表达式（数组）
	leftReg, err := c.compileExpression(expr.Left)
	if err != nil {
		return -1, err
	}

	// 编译索引表达式
	indexReg, err := c.compileExpression(expr.Index)
	if err != nil {
		return -1, err
	}

	// 分配结果寄存器
	resultReg := c.allocateRegister()

	// 发射数组获取指令: ARRAY_GET resultReg, leftReg, indexReg
	c.emit(vm.OP_ARRAY_GET, resultReg, leftReg, indexReg)

	return resultReg, nil
}

// 辅助方法

// allocateRegister 分配一个新寄存器
func (c *Compiler) allocateRegister() int {
	reg := c.nextRegister
	c.nextRegister++
	if c.nextRegister > c.maxRegisters {
		c.maxRegisters = c.nextRegister
	}
	return reg
}

// releaseRegister 释放寄存器（简化版，重置到某个位置）
func (c *Compiler) releaseRegister(reg int) {
	// 简化的寄存器释放：如果是最后分配的寄存器，可以回退
	if reg == c.nextRegister-1 {
		c.nextRegister = reg
	}
}

// resetRegisters 重置寄存器分配器（用于语句之间）
func (c *Compiler) resetRegisters() {
	c.nextRegister = 0
}

// addConstant 添加常量到常量池
func (c *Compiler) addConstant(obj vm.Value) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

// emit 发射指令
func (c *Compiler) emit(op vm.OpCode, operands ...int) int {
	ins := c.makeInstruction(op, operands...)
	pos := c.addInstruction(ins)
	c.setLastInstruction(op, pos)
	return pos
}

// makeInstruction 创建指令
func (c *Compiler) makeInstruction(op vm.OpCode, operands ...int) vm.Instruction {
	inst := vm.Instruction{OpCode: op}

	if len(operands) > 0 {
		inst.A = operands[0]
	}
	if len(operands) > 1 {
		inst.B = operands[1]
	}
	if len(operands) > 2 {
		inst.C = operands[2]
	}

	// 对于使用Bx的指令，将B设置为Bx
	if len(operands) == 2 && (op == vm.OP_LOADK || op == vm.OP_GET_GLOBAL || op == vm.OP_SET_GLOBAL ||
		op == vm.OP_JUMP_IF_FALSE || op == vm.OP_JUMP_IF_TRUE) {
		inst.Bx = operands[1]
		inst.B = 0
		inst.C = 0
	}

	// NEW_ARRAY使用B字段存储长度，不需要特殊处理

	// 对于只使用Bx的指令（如JUMP）
	if len(operands) == 1 && op == vm.OP_JUMP {
		inst.Bx = operands[0]
		inst.A = 0
		inst.B = 0
		inst.C = 0
	}

	return inst
}

// addInstruction 添加指令到当前作用域
func (c *Compiler) addInstruction(ins vm.Instruction) int {
	posNewInstruction := len(c.currentInstructions())
	c.scopes[c.scopeIndex].instructions = append(c.scopes[c.scopeIndex].instructions, ins)
	return posNewInstruction
}

// setLastInstruction 设置最后发射的指令信息
func (c *Compiler) setLastInstruction(op vm.OpCode, pos int) {
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{OpCode: op, Position: pos}

	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
}

// currentInstructions 获取当前作用域的指令
func (c *Compiler) currentInstructions() []vm.Instruction {
	return c.scopes[c.scopeIndex].instructions
}

// ByteCode 编译结果
type ByteCode struct {
	Instructions []vm.Instruction
	Constants    []vm.Value
}

// ByteCode 返回编译后的字节码
func (c *Compiler) ByteCode() *ByteCode {
	return &ByteCode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
	}
}
