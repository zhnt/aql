package parser1

import (
	"fmt"
	"strings"

	"github.com/zhnt/aql/internal/lexer1"
)

// Node AST节点接口
type Node interface {
	String() string
	TokenLiteral() string
}

// Statement 语句节点接口
type Statement interface {
	Node
	statementNode()
}

// Expression 表达式节点接口
type Expression interface {
	Node
	expressionNode()
}

// =============================================================================
// 程序和语句节点
// =============================================================================

// Program 程序根节点
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out strings.Builder
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// LetStatement let语句节点
type LetStatement struct {
	Token lexer1.Token // LET token
	Name  *Identifier  // 变量名
	Value Expression   // 值表达式
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	var out strings.Builder
	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")
	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

// ConstStatement const语句节点
type ConstStatement struct {
	Token lexer1.Token // CONST token
	Name  *Identifier  // 变量名
	Value Expression   // 值表达式
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) String() string {
	var out strings.Builder
	out.WriteString(cs.TokenLiteral() + " ")
	out.WriteString(cs.Name.String())
	out.WriteString(" = ")
	if cs.Value != nil {
		out.WriteString(cs.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

// ReturnStatement return语句节点
type ReturnStatement struct {
	Token       lexer1.Token // RETURN token
	ReturnValue Expression   // 返回值表达式
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out strings.Builder
	out.WriteString(rs.TokenLiteral() + " ")
	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}
	out.WriteString(";")
	return out.String()
}

// ExpressionStatement 表达式语句节点
type ExpressionStatement struct {
	Token      lexer1.Token // 表达式的第一个token
	Expression Expression   // 表达式
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// AssignmentStatement 赋值语句节点 (x = value)
type AssignmentStatement struct {
	Token lexer1.Token // 标识符token
	Name  *Identifier  // 变量名
	Value Expression   // 值表达式
}

func (as *AssignmentStatement) statementNode()       {}
func (as *AssignmentStatement) expressionNode()      {} // 同时作为表达式
func (as *AssignmentStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignmentStatement) String() string {
	var out strings.Builder
	out.WriteString(as.Name.String())
	out.WriteString(" = ")
	if as.Value != nil {
		out.WriteString(as.Value.String())
	}
	return out.String()
}

// BlockStatement 代码块语句节点
type BlockStatement struct {
	Token      lexer1.Token // { token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out strings.Builder
	out.WriteString("{")
	for _, s := range bs.Statements {
		out.WriteString(" ")
		out.WriteString(s.String())
	}
	out.WriteString("}")
	return out.String()
}

// IfStatement if语句节点
type IfStatement struct {
	Token       lexer1.Token    // IF token
	Condition   Expression      // 条件表达式
	Consequence *BlockStatement // if块
	Alternative *BlockStatement // else块
}

func (ifs *IfStatement) statementNode()       {}
func (ifs *IfStatement) expressionNode()      {} // IfStatement同时作为表达式
func (ifs *IfStatement) TokenLiteral() string { return ifs.Token.Literal }
func (ifs *IfStatement) String() string {
	var out strings.Builder
	out.WriteString("if")
	out.WriteString(ifs.Condition.String())
	out.WriteString(" ")
	out.WriteString(ifs.Consequence.String())
	if ifs.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ifs.Alternative.String())
	}
	return out.String()
}

// WhileStatement while语句节点
type WhileStatement struct {
	Token     lexer1.Token    // WHILE token
	Condition Expression      // 条件表达式
	Body      *BlockStatement // 循环体
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) String() string {
	var out strings.Builder
	out.WriteString("while ")
	out.WriteString(ws.Condition.String())
	out.WriteString(" ")
	out.WriteString(ws.Body.String())
	return out.String()
}

// ForStatement for语句节点
type ForStatement struct {
	Token     lexer1.Token    // FOR token
	Init      Statement       // 初始化语句
	Condition Expression      // 条件表达式
	Update    Expression      // 更新表达式
	Body      *BlockStatement // 循环体
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	var out strings.Builder
	out.WriteString("for (")
	if fs.Init != nil {
		out.WriteString(fs.Init.String())
	}
	out.WriteString("; ")
	if fs.Condition != nil {
		out.WriteString(fs.Condition.String())
	}
	out.WriteString("; ")
	if fs.Update != nil {
		out.WriteString(fs.Update.String())
	}
	out.WriteString(") ")
	out.WriteString(fs.Body.String())
	return out.String()
}

// BreakStatement break语句节点
type BreakStatement struct {
	Token lexer1.Token // BREAK token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break" }

// ContinueStatement continue语句节点
type ContinueStatement struct {
	Token lexer1.Token // CONTINUE token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return "continue" }

// =============================================================================
// 表达式节点
// =============================================================================

// Identifier 标识符节点
type Identifier struct {
	Token lexer1.Token // IDENT token
	Value string       // 标识符值
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral 整数字面量节点
type IntegerLiteral struct {
	Token lexer1.Token // INT token
	Value int64        // 整数值
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral 浮点数字面量节点
type FloatLiteral struct {
	Token lexer1.Token // FLOAT token
	Value float64      // 浮点数值
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral 字符串字面量节点
type StringLiteral struct {
	Token lexer1.Token // STRING token
	Value string       // 字符串值
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return fmt.Sprintf(`"%s"`, sl.Value) }

// BooleanLiteral 布尔字面量节点
type BooleanLiteral struct {
	Token lexer1.Token // TRUE/FALSE token
	Value bool         // 布尔值
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

// NullLiteral null字面量节点
type NullLiteral struct {
	Token lexer1.Token // NULL token
}

func (nl *NullLiteral) expressionNode()      {}
func (nl *NullLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NullLiteral) String() string       { return nl.Token.Literal }

// PrefixExpression 前缀表达式节点 (!x, -x)
type PrefixExpression struct {
	Token    lexer1.Token // 前缀token (!、-)
	Operator string       // 运算符
	Right    Expression   // 右表达式
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")
	return out.String()
}

// InfixExpression 中缀表达式节点 (x + y, x == y)
type InfixExpression struct {
	Token    lexer1.Token // 运算符token
	Left     Expression   // 左表达式
	Operator string       // 运算符
	Right    Expression   // 右表达式
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")
	return out.String()
}

// CommaExpression 已移除 - AQL不再支持逗号运算符
// 逗号只作为分隔符使用（数组、函数参数、对象属性等）

// CallExpression 函数调用表达式节点
type CallExpression struct {
	Token     lexer1.Token // ( token
	Function  Expression   // 函数表达式 (标识符或函数字面量)
	Arguments []Expression // 参数列表
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out strings.Builder
	var args []string
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// FunctionLiteral 函数字面量节点
type FunctionLiteral struct {
	Token      lexer1.Token    // FUNCTION token
	Name       *Identifier     // 函数名（可选）
	Parameters []*Identifier   // 参数列表
	Body       *BlockStatement // 函数体
	IsAsync    bool            // 是否为异步函数
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out strings.Builder
	var params []string
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}
	if fl.IsAsync {
		out.WriteString("async ")
	}
	out.WriteString(fl.TokenLiteral())
	if fl.Name != nil {
		out.WriteString(" ")
		out.WriteString(fl.Name.String())
	}
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")
	out.WriteString(fl.Body.String())
	return out.String()
}

// ArrayLiteral 数组字面量节点
type ArrayLiteral struct {
	Token    lexer1.Token // [ token
	Elements []Expression // 数组元素
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out strings.Builder
	var elements []string
	for _, e := range al.Elements {
		elements = append(elements, e.String())
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")
	return out.String()
}

// IndexExpression 索引表达式节点 (array[index])
type IndexExpression struct {
	Token lexer1.Token // [ token
	Left  Expression   // 被索引的表达式
	Index Expression   // 索引表达式
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")
	return out.String()
}

// =============================================================================
// AQL特定语法节点
// =============================================================================

// AwaitExpression await表达式节点
type AwaitExpression struct {
	Token      lexer1.Token // AWAIT token
	Expression Expression   // 被await的表达式
}

func (ae *AwaitExpression) expressionNode()      {}
func (ae *AwaitExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AwaitExpression) String() string {
	var out strings.Builder
	out.WriteString("await ")
	out.WriteString(ae.Expression.String())
	return out.String()
}

// YieldExpression yield表达式节点
type YieldExpression struct {
	Token      lexer1.Token // YIELD token
	Expression Expression   // 被yield的表达式
}

func (ye *YieldExpression) expressionNode()      {}
func (ye *YieldExpression) TokenLiteral() string { return ye.Token.Literal }
func (ye *YieldExpression) String() string {
	var out strings.Builder
	out.WriteString("yield ")
	if ye.Expression != nil {
		out.WriteString(ye.Expression.String())
	}
	return out.String()
}

// ServiceCallExpression AI服务调用表达式节点 (@service.method())
type ServiceCallExpression struct {
	Token     lexer1.Token // @ token
	Service   *Identifier  // 服务名
	Method    *Identifier  // 方法名
	Arguments []Expression // 参数列表
}

func (sce *ServiceCallExpression) expressionNode()      {}
func (sce *ServiceCallExpression) TokenLiteral() string { return sce.Token.Literal }
func (sce *ServiceCallExpression) String() string {
	var out strings.Builder
	var args []string
	for _, a := range sce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString("@")
	out.WriteString(sce.Service.String())
	out.WriteString(".")
	out.WriteString(sce.Method.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// PipeExpression 管道表达式节点 (data |> func())
type PipeExpression struct {
	Token lexer1.Token // |> token
	Left  Expression   // 左表达式(数据)
	Right Expression   // 右表达式(函数调用)
}

func (pe *PipeExpression) expressionNode()      {}
func (pe *PipeExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PipeExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(pe.Left.String())
	out.WriteString(" |> ")
	out.WriteString(pe.Right.String())
	out.WriteString(")")
	return out.String()
}

// ObjectLiteral 对象字面量节点
type ObjectLiteral struct {
	Token lexer1.Token              // { token
	Pairs map[Expression]Expression // 键值对
}

func (ol *ObjectLiteral) expressionNode()      {}
func (ol *ObjectLiteral) TokenLiteral() string { return ol.Token.Literal }
func (ol *ObjectLiteral) String() string {
	var out strings.Builder
	var pairs []string
	for key, value := range ol.Pairs {
		pairs = append(pairs, key.String()+": "+value.String())
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// PropertyExpression 属性访问表达式节点 (obj.prop)
type PropertyExpression struct {
	Token    lexer1.Token // . token
	Object   Expression   // 对象表达式
	Property *Identifier  // 属性名
}

func (pe *PropertyExpression) expressionNode()      {}
func (pe *PropertyExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PropertyExpression) String() string {
	var out strings.Builder
	out.WriteString(pe.Object.String())
	out.WriteString(".")
	out.WriteString(pe.Property.String())
	return out.String()
}
