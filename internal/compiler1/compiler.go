package compiler1

import (
	"fmt"

	"github.com/zhnt/aql/internal/parser1"
	"github.com/zhnt/aql/internal/vm"
)

// Compiler AQL编译器，将AST编译为VM字节码
type Compiler struct {
	constants    []vm.ValueGC    // 常量池
	symbolTable  *SymbolTable    // 符号表
	scopes       []*CompileScope // 作用域栈
	scopeIndex   int             // 当前作用域索引
	nextRegister int             // 下一个可用寄存器
	maxRegisters int             // 最大寄存器使用数
	loopStack    []*LoopContext  // 循环栈，用于break/continue

	// 寄存器管理优化
	freeRegisters []int // 空闲寄存器池
	registerStack []int // 寄存器栈，用于嵌套表达式
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
	// 寄存器状态保存
	savedNextRegister int
	savedMaxRegisters int
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
		constants:     make([]vm.ValueGC, 0),
		symbolTable:   NewSymbolTable(),
		scopes:        []*CompileScope{mainScope},
		scopeIndex:    0,
		nextRegister:  0,
		maxRegisters:  0,
		loopStack:     make([]*LoopContext, 0),
		freeRegisters: make([]int, 0),
		registerStack: make([]int, 0),
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

	// 动态计算MaxStackSize，确保足够的寄存器空间
	calculatedSize := c.calculateOptimalStackSize()
	function.MaxStackSize = calculatedSize

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

	// 为局部变量分配固定的寄存器
	var targetReg int
	if symbol.Scope == LOCAL_SCOPE {
		// 局部变量使用固定的寄存器位置，确保不会被临时计算重用
		targetReg = symbol.Index
		if reg != targetReg {
			c.emit(vm.OP_MOVE, targetReg, reg, 0) // 移动到固定位置
		}
		// 确保寄存器分配器不会重用局部变量寄存器
		if c.nextRegister <= targetReg {
			c.nextRegister = targetReg + 1
		}
	} else {
		targetReg = reg
	}

	// 发射存储指令
	if symbol.Scope == GLOBAL_SCOPE {
		c.emit(vm.OP_SET_GLOBAL, targetReg, symbol.Index) // G(symbol.Index) := R[targetReg]
	} else {
		c.emit(vm.OP_SET_LOCAL, targetReg, symbol.Index) // L(symbol.Index) := R[targetReg]
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

	// 为局部变量分配固定的寄存器
	var targetReg int
	if symbol.Scope == LOCAL_SCOPE {
		// 局部变量使用固定的寄存器位置，确保不会被临时计算重用
		targetReg = symbol.Index
		if reg != targetReg {
			c.emit(vm.OP_MOVE, targetReg, reg, 0) // 移动到固定位置
		}
		// 确保寄存器分配器不会重用局部变量寄存器
		if c.nextRegister <= targetReg {
			c.nextRegister = targetReg + 1
		}
	} else {
		targetReg = reg
	}

	if symbol.Scope == GLOBAL_SCOPE {
		c.emit(vm.OP_SET_GLOBAL, targetReg, symbol.Index) // G(symbol.Index) := R[targetReg]
	} else {
		c.emit(vm.OP_SET_LOCAL, targetReg, symbol.Index) // L(symbol.Index) := R[targetReg]
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
	// 特殊处理具名函数定义
	if funcLit, ok := stmt.Expression.(*parser1.FunctionLiteral); ok && funcLit.Name != nil {
		return c.compileNamedFunctionDefinition(funcLit)
	}

	reg, err := c.compileExpression(stmt.Expression)
	if err != nil {
		return err
	}

	// 找到一个安全的寄存器来存储结果，避免覆盖局部变量
	resultReg := c.findSafeResultRegister()
	if reg != resultReg {
		c.emit(vm.OP_MOVE, resultReg, reg, 0) // R[resultReg] := R[reg]
	}

	// 表达式语句的结果被丢弃（但已经保存在结果寄存器中）
	c.emit(vm.OP_POP)
	// 在表达式语句结束后重置寄存器分配器
	c.resetRegisters()
	return nil
}

// findSafeResultRegister 找到一个安全的寄存器来存储表达式结果
func (c *Compiler) findSafeResultRegister() int {
	// 找到所有局部变量的最大寄存器索引
	maxLocalReg := -1
	if c.symbolTable != nil {
		for _, symbol := range c.symbolTable.store {
			if symbol.Scope == LOCAL_SCOPE && symbol.Index > maxLocalReg {
				maxLocalReg = symbol.Index
			}
		}
	}

	// 如果没有局部变量，使用寄存器0
	if maxLocalReg == -1 {
		return 0
	}

	// 否则使用局部变量后的第一个寄存器
	return maxLocalReg + 1
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
	case *parser1.IndexAssignmentStatement:
		return c.compileIndexAssignmentExpression(expr)
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
	case *parser1.ArrayConstructor:
		return c.compileArrayConstructor(expr)
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
	switch symbol.Scope {
	case GLOBAL_SCOPE:
		c.emit(vm.OP_GET_GLOBAL, reg, symbol.Index) // R[reg] := G(symbol.Index)
	case FREE_SCOPE:
		c.emit(vm.OP_GET_UPVALUE, reg, symbol.Index) // R[reg] := Upvalue[symbol.Index].Get()
	default: // LOCAL_SCOPE 和其他
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
	switch symbol.Scope {
	case GLOBAL_SCOPE:
		c.emit(vm.OP_SET_GLOBAL, valueReg, symbol.Index) // G(symbol.Index) := R[valueReg]
	case FREE_SCOPE:
		c.emit(vm.OP_SET_UPVALUE, valueReg, symbol.Index) // Upvalue[symbol.Index].Set(R[valueReg])
	default: // LOCAL_SCOPE 和其他
		c.emit(vm.OP_SET_LOCAL, valueReg, symbol.Index) // L(symbol.Index) := R[valueReg]
	}

	// 赋值表达式的结果就是被赋的值
	return valueReg, nil
}

// compileIndexAssignmentExpression 编译索引赋值表达式
func (c *Compiler) compileIndexAssignmentExpression(expr *parser1.IndexAssignmentStatement) (int, error) {
	// 编译右值表达式
	valueReg, err := c.compileExpression(expr.Value)
	if err != nil {
		return -1, err
	}

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

	// 编译索引表达式
	indexReg, err := c.compileExpression(indexExpr.Index)
	if err != nil {
		return -1, err
	}

	// 发射数组设置指令: ARRAY_SET arrayReg, indexReg, valueReg
	c.emit(vm.OP_ARRAY_SET, arrayReg, indexReg, valueReg)

	// 索引赋值表达式的结果就是被赋的值
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
	case "%":
		c.emit(vm.OP_MOD, resultReg, leftReg, rightReg) // R[resultReg] := R[leftReg] % R[rightReg]
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
	// 分配结果寄存器
	resultReg := c.allocateRegister()

	// 存储所有跳转位置，用于后续回填
	var jumpToEndPositions []int

	// 编译主if条件
	conditionReg, err := c.compileExpression(expr.Condition)
	if err != nil {
		return -1, err
	}

	// 发射条件跳转指令：如果条件为假，跳过if块
	jumpIfFalsePos := c.emit(vm.OP_JUMP_IF_FALSE, conditionReg, 9999)

	// 编译if块（consequence）
	err = c.compileIfBlock(expr.Consequence, resultReg)
	if err != nil {
		return -1, err
	}

	// 发射无条件跳转指令：跳到最终结束位置
	jumpToEndPos := c.emit(vm.OP_JUMP, 9999)
	jumpToEndPositions = append(jumpToEndPositions, jumpToEndPos)

	// 回填第一个条件跳转的目标地址
	currentPos := len(c.currentInstructions())
	jumpTarget := currentPos - jumpIfFalsePos
	c.scopes[c.scopeIndex].instructions[jumpIfFalsePos].Bx = jumpTarget

	// 编译所有elif分支
	for _, elifBranch := range expr.ElifBranches {
		// 编译elif条件
		elifConditionReg, err := c.compileExpression(elifBranch.Condition)
		if err != nil {
			return -1, err
		}

		// 发射条件跳转指令：如果elif条件为假，跳过elif块
		elifJumpIfFalsePos := c.emit(vm.OP_JUMP_IF_FALSE, elifConditionReg, 9999)

		// 编译elif块
		err = c.compileIfBlock(elifBranch.Consequence, resultReg)
		if err != nil {
			return -1, err
		}

		// 发射无条件跳转指令：跳到最终结束位置
		jumpToEndPos := c.emit(vm.OP_JUMP, 9999)
		jumpToEndPositions = append(jumpToEndPositions, jumpToEndPos)

		// 回填elif条件跳转的目标地址
		currentPos = len(c.currentInstructions())
		jumpTarget = currentPos - elifJumpIfFalsePos
		c.scopes[c.scopeIndex].instructions[elifJumpIfFalsePos].Bx = jumpTarget
	}

	// 如果有else块，编译它
	if expr.Alternative != nil {
		err = c.compileIfBlock(expr.Alternative, resultReg)
		if err != nil {
			return -1, err
		}
	} else {
		// 没有else块，设置结果为nil
		c.emit(vm.OP_LOADK, resultReg, c.addConstant(vm.NewNilValue()))
	}

	// 回填所有跳转到结束位置的指令
	currentPos = len(c.currentInstructions())
	for _, jumpPos := range jumpToEndPositions {
		jumpTarget = currentPos - jumpPos
		c.scopes[c.scopeIndex].instructions[jumpPos].Bx = jumpTarget
	}

	return resultReg, nil
}

// compileIfBlock 编译if/elif/else块的通用方法
func (c *Compiler) compileIfBlock(block *parser1.BlockStatement, resultReg int) error {
	ifStmts := block.Statements
	for i, stmt := range ifStmts {
		if i == len(ifStmts)-1 {
			// 最后一个语句，特殊处理
			if exprStmt, ok := stmt.(*parser1.ExpressionStatement); ok {
				// 编译表达式并将结果存储到resultReg
				reg, err := c.compileExpression(exprStmt.Expression)
				if err != nil {
					return err
				}
				c.emit(vm.OP_MOVE, resultReg, reg, 0)
			} else {
				// 非表达式语句，编译它并设置结果为nil
				err := c.compileStatement(stmt)
				if err != nil {
					return err
				}
				c.emit(vm.OP_LOADK, resultReg, c.addConstant(vm.NewNilValue()))
			}
		} else {
			// 其他语句正常编译
			err := c.compileStatement(stmt)
			if err != nil {
				return err
			}
		}
	}

	// 如果块为空，设置结果为nil
	if len(ifStmts) == 0 {
		c.emit(vm.OP_LOADK, resultReg, c.addConstant(vm.NewNilValue()))
	}

	return nil
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
	// 进入新的编译作用域
	c.enterScope()

	// 创建新的符号表作用域
	enclosedSymbolTable := NewEnclosedSymbolTable(c.symbolTable)
	c.symbolTable = enclosedSymbolTable

	// 创建新函数
	var functionName string
	if expr.Name != nil {
		functionName = expr.Name.Value
	} else {
		functionName = "<anonymous>"
	}

	function := vm.NewFunction(functionName)
	function.ParamCount = len(expr.Parameters)

	// 先定义参数为局部变量（从索引0开始）
	for _, param := range expr.Parameters {
		c.symbolTable.Define(param.Value)
	}

	// 重置寄存器分配器，从参数后开始
	c.nextRegister = function.ParamCount

	// 如果是具名函数，为了支持递归，我们需要在函数体内定义函数名
	// 但不应该占用局部变量的索引，因为函数名在编译时已经确定
	if expr.Name != nil {
		// 将函数名添加到符号表，但使用特殊的处理
		// 这里暂时跳过，因为递归调用需要特殊处理
		// TODO: 实现正确的递归函数支持
	}

	// 编译函数体
	err := c.compileBlockStatement(expr.Body)
	if err != nil {
		return -1, err
	}

	// 如果函数体没有显式return，添加隐式return nil
	lastInst := c.scopes[c.scopeIndex].lastInstruction
	if lastInst.OpCode != vm.OP_RETURN {
		nilReg := c.allocateRegister()
		c.emit(vm.OP_LOADK, nilReg, c.addConstant(vm.NewNilValueGC()))
		c.emit(vm.OP_RETURN, nilReg, 1, 0)
	}

	// 设置函数的指令和最大栈大小
	function.Instructions = c.currentInstructions()
	function.Constants = c.constants
	function.MaxStackSize = c.maxRegisters

	// 检查是否有自由变量（需要创建闭包）
	freeSymbols := c.symbolTable.FreeSymbols
	numFreeVars := len(freeSymbols)

	// 退出作用域
	c.leaveScope()
	c.symbolTable = c.symbolTable.Outer

	// 将编译好的函数注册到全局Function注册表
	functionID := vm.RegisterFunction(function)

	if numFreeVars == 0 {
		// 没有自由变量，创建普通函数
		functionValue := vm.NewFunctionValueGCFromID(functionID)
		constIndex := c.addConstant(functionValue)

		// 重要修复：确保函数对象不会分配到已被局部变量占用的寄存器
		// 扫描当前符号表的局部变量，找到最大的索引
		maxLocalIndex := -1
		if c.symbolTable != nil {
			for _, symbol := range c.symbolTable.store {
				if symbol.Scope == LOCAL_SCOPE && symbol.Index > maxLocalIndex {
					maxLocalIndex = symbol.Index
				}
			}
		}

		// 如果有局部变量，确保函数对象的寄存器在所有局部变量之后
		if maxLocalIndex >= 0 && c.nextRegister <= maxLocalIndex {
			c.nextRegister = maxLocalIndex + 1
		}

		reg := c.allocateRegister()
		c.emit(vm.OP_LOADK, reg, constIndex)
		return reg, nil
	} else {
		// 有自由变量，需要创建闭包
		// 先加载函数到寄存器
		functionValue := vm.NewFunctionValueGCFromID(functionID)
		constIndex := c.addConstant(functionValue)

		// 重要修复：确保函数对象不会分配到已被局部变量占用的寄存器
		// 扫描当前符号表的局部变量，找到最大的索引
		maxLocalIndex := -1
		if c.symbolTable != nil {
			for _, symbol := range c.symbolTable.store {
				if symbol.Scope == LOCAL_SCOPE && symbol.Index > maxLocalIndex {
					maxLocalIndex = symbol.Index
				}
			}
		}

		// 如果有局部变量，确保函数对象的寄存器在所有局部变量之后
		if maxLocalIndex >= 0 && c.nextRegister <= maxLocalIndex {
			c.nextRegister = maxLocalIndex + 1
		}

		funcReg := c.allocateRegister()
		c.emit(vm.OP_LOADK, funcReg, constIndex)

		// 加载自由变量到寄存器 - 修复后的版本
		captureRegs := make([]int, numFreeVars)
		for i, freeVar := range freeSymbols {
			captureReg := c.allocateRegister()

			// 关键修复：需要在当前作用域中正确地获取自由变量
			// 自由变量对于当前作用域来说应该通过符号表解析来访问

			// 在当前符号表中查找这个变量（应该被解析为FREE_SCOPE）
			currentSymbol, ok := c.symbolTable.Resolve(freeVar.Name)
			if !ok {
				// 如果找不到，说明编译器逻辑有问题
				return -1, &CompilationError{
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
				return -1, &CompilationError{
					Message: "unsupported scope for free variable: " + string(currentSymbol.Scope),
				}
			}

			captureRegs[i] = captureReg
		}

		// 将捕获变量移动到连续的寄存器位置
		// MAKE_CLOSURE 期望: function在B，捕获变量在B+1, B+2, ...
		baseReg := funcReg + 1
		for i, captureReg := range captureRegs {
			targetReg := baseReg + i
			if captureReg != targetReg {
				c.emit(vm.OP_MOVE, targetReg, captureReg, 0)
			}
		}

		// 发射创建闭包指令
		closureReg := c.allocateRegister()
		c.emit(vm.OP_MAKE_CLOSURE, closureReg, funcReg, numFreeVars)

		return closureReg, nil
	}
}

func (c *Compiler) compileCallExpression(expr *parser1.CallExpression) (int, error) {
	// 编译函数表达式
	funcReg, err := c.compileExpression(expr.Function)
	if err != nil {
		return -1, err
	}

	// 编译参数
	argCount := len(expr.Arguments)
	for i, arg := range expr.Arguments {
		argReg, err := c.compileExpression(arg)
		if err != nil {
			return -1, err
		}

		// 将参数移动到函数寄存器之后的位置
		// CALL指令期望: R(A) = 函数, R(A+1) = 参数1, R(A+2) = 参数2, ...
		targetReg := funcReg + 1 + i
		if argReg != targetReg {
			c.emit(vm.OP_MOVE, targetReg, argReg, 0)
		}
	}

	// 发射CALL指令
	// CALL A B C: 调用R(A)，参数数量为B-1，期望返回值数量为C
	c.emit(vm.OP_CALL, funcReg, argCount+1, 1) // +1 because B includes the function itself

	// 调用后，结果在funcReg位置
	return funcReg, nil
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

// compileArrayConstructor 编译数组构造器 Array(capacity) 或 Array(capacity, defaultValue)
func (c *Compiler) compileArrayConstructor(expr *parser1.ArrayConstructor) (int, error) {
	// 编译容量表达式
	capacityReg, err := c.compileExpression(expr.Capacity)
	if err != nil {
		return -1, err
	}

	// 分配结果寄存器
	arrayReg := c.allocateRegister()

	// 编译默认值表达式（如果有）
	if expr.DefaultValue != nil {
		defaultValueReg, err := c.compileExpression(expr.DefaultValue)
		if err != nil {
			return -1, err
		}

		// 发射创建带默认值的数组指令: NEW_ARRAY_WITH_CAPACITY arrayReg, capacityReg, defaultValueReg
		c.emit(vm.OP_NEW_ARRAY_WITH_CAPACITY, arrayReg, capacityReg, defaultValueReg)
	} else {
		// 修复：加载nil常量到寄存器，然后传递寄存器号
		nilReg := c.allocateRegister()
		nilConstIndex := c.addConstant(vm.NewNilValueGC())
		c.emit(vm.OP_LOADK, nilReg, nilConstIndex)

		// 发射创建空数组指令: NEW_ARRAY_WITH_CAPACITY arrayReg, capacityReg, nilReg
		c.emit(vm.OP_NEW_ARRAY_WITH_CAPACITY, arrayReg, capacityReg, nilReg)
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

	// 分配结果寄存器，确保不会与之前的寄存器冲突
	resultReg := c.allocateRegister()

	// 发射数组获取指令: ARRAY_GET resultReg, leftReg, indexReg
	c.emit(vm.OP_ARRAY_GET, resultReg, leftReg, indexReg)

	return resultReg, nil
}

// 辅助方法

// allocateRegister 分配一个新寄存器（改进版，避免参数冲突）
func (c *Compiler) allocateRegister() int {
	// 首先尝试从空闲寄存器池中获取
	if len(c.freeRegisters) > 0 {
		for i := len(c.freeRegisters) - 1; i >= 0; i-- {
			reg := c.freeRegisters[i]

			// 确保不会重用局部变量或参数的寄存器
			if c.isRegisterConflict(reg) {
				continue
			}

			// 从空闲池中移除这个寄存器
			c.freeRegisters = append(c.freeRegisters[:i], c.freeRegisters[i+1:]...)
			return reg
		}
	}

	// 如果没有合适的空闲寄存器，分配新的
	// 重要：从安全位置开始分配，避免与参数和局部变量冲突
	minSafeReg := c.getMinimumSafeRegister()
	if c.nextRegister < minSafeReg {
		c.nextRegister = minSafeReg
	}

	reg := c.nextRegister
	c.nextRegister++
	if c.nextRegister > c.maxRegisters {
		c.maxRegisters = c.nextRegister
	}
	return reg
}

// getMinimumSafeRegister 获取最小安全寄存器位置
func (c *Compiler) getMinimumSafeRegister() int {
	// 找到所有局部变量和参数的最大寄存器索引
	maxReservedReg := -1

	if c.symbolTable != nil {
		for _, symbol := range c.symbolTable.store {
			if symbol.Scope == LOCAL_SCOPE && symbol.Index > maxReservedReg {
				maxReservedReg = symbol.Index
			}
		}
	}

	// 为了安全，临时寄存器从保留寄存器后的位置开始分配
	return maxReservedReg + 1
}

// isRegisterConflict 检查寄存器是否与局部变量或参数冲突（改进版）
func (c *Compiler) isRegisterConflict(reg int) bool {
	if c.symbolTable == nil {
		return false
	}

	// 检查所有符号表层次的局部变量
	currentSymbolTable := c.symbolTable
	for currentSymbolTable != nil {
		for _, symbol := range currentSymbolTable.store {
			if symbol.Scope == LOCAL_SCOPE && symbol.Index == reg {
				return true
			}
		}
		currentSymbolTable = currentSymbolTable.Outer
	}

	return false
}

// releaseRegister 释放寄存器（优化版）
func (c *Compiler) releaseRegister(reg int) {
	if reg >= 0 && reg < c.nextRegister {
		// 将寄存器添加到空闲池
		c.freeRegisters = append(c.freeRegisters, reg)
	}
}

// pushRegister 将寄存器推入栈（用于嵌套表达式）
func (c *Compiler) pushRegister(reg int) {
	c.registerStack = append(c.registerStack, reg)
}

// popRegister 从栈中弹出寄存器并释放
func (c *Compiler) popRegister() int {
	if len(c.registerStack) == 0 {
		return -1
	}

	reg := c.registerStack[len(c.registerStack)-1]
	c.registerStack = c.registerStack[:len(c.registerStack)-1]
	c.releaseRegister(reg)
	return reg
}

// resetRegisters 重置寄存器分配器（用于语句之间）
func (c *Compiler) resetRegisters() {
	c.nextRegister = 0
	c.freeRegisters = c.freeRegisters[:0] // 清空但保留容量
	c.registerStack = c.registerStack[:0] // 清空但保留容量
}

// addConstant 添加常量到常量池
func (c *Compiler) addConstant(obj vm.ValueGC) int {
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

// enterScope 进入新的编译作用域
func (c *Compiler) enterScope() {
	scope := &CompileScope{
		instructions:        make([]vm.Instruction, 0),
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
		// 保存当前寄存器状态
		savedNextRegister: c.nextRegister,
		savedMaxRegisters: c.maxRegisters,
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++

	// 重置寄存器分配器
	c.nextRegister = 0
	c.maxRegisters = 0
}

// leaveScope 离开当前编译作用域
func (c *Compiler) leaveScope() []vm.Instruction {
	instructions := c.currentInstructions()

	// 恢复上一个作用域的寄存器状态
	if c.scopeIndex > 0 {
		parentScope := c.scopes[c.scopeIndex-1]

		// 重要修复：我们需要更新父作用域的寄存器状态，确保后续的寄存器分配
		// 能够正确地继续，而不是重用已经被占用的寄存器

		// 首先恢复到父作用域的状态
		c.nextRegister = parentScope.savedNextRegister
		c.maxRegisters = parentScope.savedMaxRegisters

		// 然后更新父作用域的状态，确保后续的寄存器分配不会冲突
		// 重新扫描父作用域的局部变量，更新nextRegister
		maxLocalIndex := -1
		if c.symbolTable != nil {
			for _, symbol := range c.symbolTable.store {
				if symbol.Scope == LOCAL_SCOPE && symbol.Index > maxLocalIndex {
					maxLocalIndex = symbol.Index
				}
			}
		}

		// 确保nextRegister至少在所有局部变量之后
		if maxLocalIndex >= 0 && c.nextRegister <= maxLocalIndex {
			c.nextRegister = maxLocalIndex + 1
		}
	}

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--

	return instructions
}

// compileNamedFunctionDefinition 编译具名函数定义
func (c *Compiler) compileNamedFunctionDefinition(funcLit *parser1.FunctionLiteral) error {
	// 先定义函数名，让函数体内可以引用自己（支持递归）
	symbol := c.symbolTable.Define(funcLit.Name.Value)

	// 编译函数字面量
	funcReg, err := c.compileFunctionLiteral(funcLit)
	if err != nil {
		return err
	}

	// 为局部函数分配固定的寄存器，与变量使用相同的策略
	var targetReg int
	if symbol.Scope == LOCAL_SCOPE {
		// 局部函数使用固定的寄存器位置，确保不会被临时计算重用
		targetReg = symbol.Index
		if funcReg != targetReg {
			c.emit(vm.OP_MOVE, targetReg, funcReg, 0) // 移动到固定位置
		}
		// 确保寄存器分配器不会重用局部函数寄存器
		if c.nextRegister <= targetReg {
			c.nextRegister = targetReg + 1
		}
	} else {
		targetReg = funcReg
	}

	// 重要修复：无论是普通函数还是闭包，都要存储到符号对应的位置
	// 这确保了后续的标识符解析能够获取到正确的对象
	if symbol.Scope == GLOBAL_SCOPE {
		c.emit(vm.OP_SET_GLOBAL, targetReg, symbol.Index)
	} else {
		c.emit(vm.OP_SET_LOCAL, targetReg, symbol.Index)
	}

	return nil
}

// ByteCode 编译结果
type ByteCode struct {
	Instructions []vm.Instruction
	Constants    []vm.ValueGC
}

// ByteCode 返回编译后的字节码
func (c *Compiler) ByteCode() *ByteCode {
	return &ByteCode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
	}
}

// calculateOptimalStackSize 计算最优的栈大小
func (c *Compiler) calculateOptimalStackSize() int {
	// 基于实际使用的最大寄存器数量计算
	baseSize := c.maxRegisters

	// 添加安全余量：50%的缓冲区，最少256个寄存器
	safetyMargin := int(float64(baseSize) * 0.5)
	optimalSize := baseSize + safetyMargin

	// 设置最小值和最大值
	const minStackSize = 256
	const maxStackSize = 8192

	if optimalSize < minStackSize {
		optimalSize = minStackSize
	} else if optimalSize > maxStackSize {
		optimalSize = maxStackSize
	}

	return optimalSize
}
