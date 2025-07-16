package parser1

import (
	"fmt"

	"github.com/zhnt/aql/internal/lexer1"
)

// 运算符优先级
const (
	_ int = iota
	LOWEST
	COMMA       // , (最低优先级)
	ASSIGN      // =
	PIPE        // |>
	OR          // ||
	AND         // &&
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // array[index]
	PROPERTY    // obj.property
)

// 运算符优先级映射
var precedences = map[lexer1.TokenType]int{
	lexer1.COMMA:    COMMA, // 逗号优先级
	lexer1.ASSIGN:   ASSIGN,
	lexer1.PIPE:     PIPE,
	lexer1.OR:       OR,
	lexer1.AND:      AND,
	lexer1.EQ:       EQUALS,
	lexer1.NOT_EQ:   EQUALS,
	lexer1.LT:       LESSGREATER,
	lexer1.GT:       LESSGREATER,
	lexer1.LTE:      LESSGREATER,
	lexer1.GTE:      LESSGREATER,
	lexer1.PLUS:     SUM,
	lexer1.MINUS:    SUM,
	lexer1.SLASH:    PRODUCT,
	lexer1.ASTERISK: PRODUCT,
	lexer1.PERCENT:  PRODUCT,
	lexer1.LPAREN:   CALL,
	lexer1.LBRACKET: INDEX,
	lexer1.DOT:      PROPERTY,
}

// Parser 语法分析器
type Parser struct {
	l *lexer1.Lexer

	curToken  lexer1.Token
	peekToken lexer1.Token

	errors []string

	prefixParseFns map[lexer1.TokenType]prefixParseFn
	infixParseFns  map[lexer1.TokenType]infixParseFn
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

// New 创建新的语法分析器
func New(l *lexer1.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// 初始化解析函数映射
	p.prefixParseFns = make(map[lexer1.TokenType]prefixParseFn)
	p.registerPrefix(lexer1.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer1.INT, p.parseIntegerLiteral)
	p.registerPrefix(lexer1.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer1.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer1.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer1.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer1.NULL, p.parseNullLiteral)
	p.registerPrefix(lexer1.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer1.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer1.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer1.IF, p.parseIfExpression)
	p.registerPrefix(lexer1.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(lexer1.ASYNC, p.parseAsyncFunctionLiteral)
	p.registerPrefix(lexer1.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(lexer1.ARRAY, p.parseArrayConstructor)
	p.registerPrefix(lexer1.LBRACE, p.parseObjectLiteral)
	p.registerPrefix(lexer1.AWAIT, p.parseAwaitExpression)
	p.registerPrefix(lexer1.YIELD, p.parseYieldExpression)
	p.registerPrefix(lexer1.AT_SYMBOL, p.parseServiceCallExpression)

	p.infixParseFns = make(map[lexer1.TokenType]infixParseFn)
	p.registerInfix(lexer1.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer1.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer1.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer1.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer1.PERCENT, p.parseInfixExpression)
	p.registerInfix(lexer1.EQ, p.parseInfixExpression)
	p.registerInfix(lexer1.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer1.LT, p.parseInfixExpression)
	p.registerInfix(lexer1.GT, p.parseInfixExpression)
	p.registerInfix(lexer1.LTE, p.parseInfixExpression)
	p.registerInfix(lexer1.GTE, p.parseInfixExpression)
	p.registerInfix(lexer1.AND, p.parseInfixExpression)
	p.registerInfix(lexer1.OR, p.parseInfixExpression)
	p.registerInfix(lexer1.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer1.LBRACKET, p.parseIndexExpression)
	p.registerInfix(lexer1.DOT, p.parsePropertyExpression)
	p.registerInfix(lexer1.PIPE, p.parsePipeExpression)
	p.registerInfix(lexer1.ASSIGN, p.parseAssignmentExpression)
	// 移除通用逗号运算符 - 逗号只作为分隔符使用
	// p.registerInfix(lexer1.COMMA, p.parseCommaExpression)

	// 读取两个token，设置curToken和peekToken
	p.nextToken()
	p.nextToken()

	// 跳过开头的注释
	for p.curToken.Type == lexer1.COMMENT {
		p.nextToken()
	}

	return p
}

// 注册前缀解析函数
func (p *Parser) registerPrefix(tokenType lexer1.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

// 注册中缀解析函数
func (p *Parser) registerInfix(tokenType lexer1.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// nextToken 前进到下一个token
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()

	// 跳过注释token
	for p.peekToken.Type == lexer1.COMMENT {
		p.peekToken = p.l.NextToken()
	}
}

// ParseProgram 解析程序
func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.Statements = []Statement{}

	for !p.curTokenIs(lexer1.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseStatement 解析语句
func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case lexer1.LET:
		return p.parseLetStatement()
	case lexer1.CONST:
		return p.parseConstStatement()
	case lexer1.RETURN:
		return p.parseReturnStatement()
	case lexer1.WHILE:
		return p.parseWhileStatement()
	case lexer1.FOR:
		return p.parseForStatement()
	case lexer1.BREAK:
		return p.parseBreakStatement()
	case lexer1.CONTINUE:
		return p.parseContinueStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseLetStatement 解析let语句
func (p *Parser) parseLetStatement() *LetStatement {
	stmt := &LetStatement{Token: p.curToken}

	if !p.expectPeek(lexer1.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer1.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer1.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseConstStatement 解析const语句
func (p *Parser) parseConstStatement() *ConstStatement {
	stmt := &ConstStatement{Token: p.curToken}

	if !p.expectPeek(lexer1.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer1.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer1.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseReturnStatement 解析return语句
func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer1.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseExpressionStatement 解析表达式语句
func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	// 普通表达式语句（让赋值通过正常的表达式解析流程处理）
	stmt := &ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer1.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseAssignmentStatement 解析赋值语句并包装为表达式语句
func (p *Parser) parseAssignmentStatement() *ExpressionStatement {
	stmt := &ExpressionStatement{Token: p.curToken}

	assignStmt := &AssignmentStatement{Token: p.curToken}
	assignStmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer1.ASSIGN) {
		return nil
	}

	p.nextToken()
	assignStmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer1.SEMICOLON) {
		p.nextToken()
	}

	// 将赋值语句作为表达式包装
	stmt.Expression = assignStmt
	return stmt
}

// parseIfStatement 解析if语句
func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{Token: p.curToken}

	if !p.expectPeek(lexer1.LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer1.RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer1.LBRACE) {
		return nil
	}

	stmt.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(lexer1.ELSE) {
		p.nextToken()

		if !p.expectPeek(lexer1.LBRACE) {
			return nil
		}

		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

// parseWhileStatement 解析while语句
func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{Token: p.curToken}

	if !p.expectPeek(lexer1.LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer1.RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer1.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseForStatement 解析for语句
func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{Token: p.curToken}

	if !p.expectPeek(lexer1.LPAREN) {
		return nil
	}

	// 解析初始化语句
	p.nextToken()
	if !p.curTokenIs(lexer1.SEMICOLON) {
		stmt.Init = p.parseForInitStatement()
		// 解析完初始化语句后，期望分号
		if !p.expectPeek(lexer1.SEMICOLON) {
			return nil
		}
	}
	// 如果当前已经是分号，直接跳过

	// 解析条件表达式
	p.nextToken()
	if !p.curTokenIs(lexer1.SEMICOLON) {
		stmt.Condition = p.parseExpression(LOWEST)
		// 解析完条件表达式后，期望分号
		if !p.expectPeek(lexer1.SEMICOLON) {
			return nil
		}
	}
	// 如果当前已经是分号，直接跳过

	// 解析更新表达式
	p.nextToken()
	if !p.curTokenIs(lexer1.RPAREN) && !p.curTokenIs(lexer1.SEMICOLON) {
		stmt.Update = p.parseExpression(LOWEST)
		// 解析完更新表达式后，期望右括号
		if !p.expectPeek(lexer1.RPAREN) {
			return nil
		}
	} else if p.curTokenIs(lexer1.SEMICOLON) {
		// 如果遇到分号，跳过分号找到右括号
		if !p.expectPeek(lexer1.RPAREN) {
			return nil
		}
	}
	// 如果当前已经是右括号，直接跳过

	if !p.expectPeek(lexer1.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseBreakStatement 解析break语句
func (p *Parser) parseBreakStatement() *BreakStatement {
	stmt := &BreakStatement{Token: p.curToken}

	if p.peekTokenIs(lexer1.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseContinueStatement 解析continue语句
func (p *Parser) parseContinueStatement() *ContinueStatement {
	stmt := &ContinueStatement{Token: p.curToken}

	if p.peekTokenIs(lexer1.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseForInitStatement 解析for循环的初始化语句（不自动消费分号）
func (p *Parser) parseForInitStatement() Statement {
	switch p.curToken.Type {
	case lexer1.LET:
		return p.parseForLetStatement()
	case lexer1.CONST:
		return p.parseForConstStatement()
	default:
		// 赋值表达式
		return p.parseForExpressionStatement()
	}
}

// parseForLetStatement 解析for循环中的let语句（不消费分号）
func (p *Parser) parseForLetStatement() *LetStatement {
	stmt := &LetStatement{Token: p.curToken}

	if !p.expectPeek(lexer1.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer1.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	// 注意：不消费分号，留给for循环解析器处理
	return stmt
}

// parseForConstStatement 解析for循环中的const语句（不消费分号）
func (p *Parser) parseForConstStatement() *ConstStatement {
	stmt := &ConstStatement{Token: p.curToken}

	if !p.expectPeek(lexer1.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer1.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	// 注意：不消费分号，留给for循环解析器处理
	return stmt
}

// parseForExpressionStatement 解析for循环中的表达式语句（不消费分号）
func (p *Parser) parseForExpressionStatement() *ExpressionStatement {
	// 检查是否为简单赋值语句 (identifier = expression)
	if p.curToken.Type == lexer1.IDENT && p.peekToken.Type == lexer1.ASSIGN {
		// 解析为赋值语句，然后包装在表达式语句中
		return p.parseForAssignmentStatement()
	}

	// 普通表达式语句
	stmt := &ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)

	// 注意：不消费分号，留给for循环解析器处理
	return stmt
}

// parseForAssignmentStatement 解析for循环中的赋值语句（不消费分号）
func (p *Parser) parseForAssignmentStatement() *ExpressionStatement {
	stmt := &ExpressionStatement{Token: p.curToken}

	assignStmt := &AssignmentStatement{Token: p.curToken}
	assignStmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer1.ASSIGN) {
		return nil
	}

	p.nextToken()
	assignStmt.Value = p.parseExpression(LOWEST)

	// 注意：不消费分号，留给for循环解析器处理

	// 将赋值语句作为表达式包装
	stmt.Expression = assignStmt
	return stmt
}

// parseBlockStatement 解析代码块语句
func (p *Parser) parseBlockStatement() *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.nextToken()

	for !p.curTokenIs(lexer1.RBRACE) && !p.curTokenIs(lexer1.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// Errors 返回解析错误
func (p *Parser) Errors() []string {
	return p.errors
}

// peekError 记录peek错误
func (p *Parser) peekError(t lexer1.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

// noPrefixParseFnError 记录前缀解析函数缺失错误
func (p *Parser) noPrefixParseFnError(t lexer1.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

// curTokenIs 检查当前token类型
func (p *Parser) curTokenIs(t lexer1.TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs 检查下一个token类型
func (p *Parser) peekTokenIs(t lexer1.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek 检查下一个token是否符合预期，如果是则前进
func (p *Parser) expectPeek(t lexer1.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// peekPrecedence 获取下一个token的优先级
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// curPrecedence 获取当前token的优先级
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}
