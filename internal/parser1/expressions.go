package parser1

import (
	"strconv"

	"github.com/zhnt/aql/internal/lexer1"
)

// parseExpression 解析表达式 (Pratt解析算法)
func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	for !p.peekTokenIs(lexer1.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// parseIdentifier 解析标识符
func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parseIntegerLiteral 解析整数字面量
func (p *Parser) parseIntegerLiteral() Expression {
	lit := &IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := "could not parse " + p.curToken.Literal + " as integer"
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

// parseFloatLiteral 解析浮点数字面量
func (p *Parser) parseFloatLiteral() Expression {
	lit := &FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := "could not parse " + p.curToken.Literal + " as float"
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

// parseStringLiteral 解析字符串字面量
func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// parseBooleanLiteral 解析布尔字面量
func (p *Parser) parseBooleanLiteral() Expression {
	return &BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(lexer1.TRUE)}
}

// parseNullLiteral 解析null字面量
func (p *Parser) parseNullLiteral() Expression {
	return &NullLiteral{Token: p.curToken}
}

// parsePrefixExpression 解析前缀表达式
func (p *Parser) parsePrefixExpression() Expression {
	expression := &PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// parseInfixExpression 解析中缀表达式
func (p *Parser) parseInfixExpression(left Expression) Expression {
	expression := &InfixExpression{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

// parseGroupedExpression 解析分组表达式 (括号表达式)
func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer1.RPAREN) {
		return nil
	}

	return exp
}

// parseIfExpression 解析if表达式
func (p *Parser) parseIfExpression() Expression {
	statement := p.parseIfStatement()
	return statement // IfStatement实现了Expression接口
}

// parseFunctionLiteral 解析函数字面量
func (p *Parser) parseFunctionLiteral() Expression {
	lit := &FunctionLiteral{Token: p.curToken}

	// 检查是否有函数名（可选）
	if p.peekTokenIs(lexer1.IDENT) {
		p.nextToken()
		lit.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if !p.expectPeek(lexer1.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(lexer1.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

// parseAsyncFunctionLiteral 解析异步函数字面量
func (p *Parser) parseAsyncFunctionLiteral() Expression {
	if !p.expectPeek(lexer1.FUNCTION) {
		return nil
	}

	lit := &FunctionLiteral{Token: p.curToken, IsAsync: true}

	// 检查是否有函数名（可选）
	if p.peekTokenIs(lexer1.IDENT) {
		p.nextToken()
		lit.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if !p.expectPeek(lexer1.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(lexer1.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

// parseFunctionParameters 解析函数参数
func (p *Parser) parseFunctionParameters() []*Identifier {
	identifiers := []*Identifier{}

	if p.peekTokenIs(lexer1.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(lexer1.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(lexer1.RPAREN) {
		return nil
	}

	return identifiers
}

// parseCallExpression 解析函数调用表达式
func (p *Parser) parseCallExpression(fn Expression) Expression {
	exp := &CallExpression{Token: p.curToken, Function: fn}
	exp.Arguments = p.parseExpressionList(lexer1.RPAREN)
	return exp
}

// parseExpressionList 解析表达式列表
func (p *Parser) parseExpressionList(end lexer1.TokenType) []Expression {
	args := []Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return args
	}

	p.nextToken()
	// 逗号只作为分隔符，不是运算符
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer1.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return args
}

// parseArrayLiteral 解析数组字面量
func (p *Parser) parseArrayLiteral() Expression {
	array := &ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(lexer1.RBRACKET)
	return array
}

// parseArrayConstructor 解析数组构造器 Array(capacity) 或 Array(capacity, defaultValue)
func (p *Parser) parseArrayConstructor() Expression {
	arrayConstructor := &ArrayConstructor{Token: p.curToken}

	if !p.expectPeek(lexer1.LPAREN) {
		return nil
	}

	// 解析容量参数
	p.nextToken()
	arrayConstructor.Capacity = p.parseExpression(LOWEST)

	// 检查是否有默认值参数
	if p.peekTokenIs(lexer1.COMMA) {
		p.nextToken() // 跳过逗号
		p.nextToken() // 移到默认值
		arrayConstructor.DefaultValue = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(lexer1.RPAREN) {
		return nil
	}

	return arrayConstructor
}

// parseIndexExpression 解析索引表达式
func (p *Parser) parseIndexExpression(left Expression) Expression {
	exp := &IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer1.RBRACKET) {
		return nil
	}

	return exp
}

// parseObjectLiteral 解析对象字面量
func (p *Parser) parseObjectLiteral() Expression {
	obj := &ObjectLiteral{Token: p.curToken}
	obj.Pairs = make(map[Expression]Expression)

	for !p.peekTokenIs(lexer1.RBRACE) && !p.peekTokenIs(lexer1.EOF) {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		if !p.expectPeek(lexer1.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		obj.Pairs[key] = value

		if !p.peekTokenIs(lexer1.RBRACE) && !p.expectPeek(lexer1.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(lexer1.RBRACE) {
		return nil
	}

	return obj
}

// parsePropertyExpression 解析属性访问表达式
func (p *Parser) parsePropertyExpression(left Expression) Expression {
	exp := &PropertyExpression{Token: p.curToken, Object: left}

	if !p.expectPeek(lexer1.IDENT) {
		return nil
	}

	exp.Property = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}

// parseAwaitExpression 解析await表达式
func (p *Parser) parseAwaitExpression() Expression {
	exp := &AwaitExpression{Token: p.curToken}

	p.nextToken()
	exp.Expression = p.parseExpression(PREFIX)

	return exp
}

// parseYieldExpression 解析yield表达式
func (p *Parser) parseYieldExpression() Expression {
	exp := &YieldExpression{Token: p.curToken}

	if !p.peekTokenIs(lexer1.SEMICOLON) && !p.peekTokenIs(lexer1.RBRACE) {
		p.nextToken()
		exp.Expression = p.parseExpression(LOWEST)
	}

	return exp
}

// parseServiceCallExpression 解析AI服务调用表达式
func (p *Parser) parseServiceCallExpression() Expression {
	exp := &ServiceCallExpression{Token: p.curToken}

	if !p.expectPeek(lexer1.IDENT) {
		return nil
	}

	exp.Service = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer1.DOT) {
		return nil
	}

	if !p.expectPeek(lexer1.IDENT) {
		return nil
	}

	exp.Method = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer1.LPAREN) {
		return nil
	}

	exp.Arguments = p.parseExpressionList(lexer1.RPAREN)

	return exp
}

// parsePipeExpression 解析管道表达式
func (p *Parser) parsePipeExpression(left Expression) Expression {
	exp := &PipeExpression{Token: p.curToken, Left: left}

	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)

	return exp
}

// parseAssignmentExpression 解析赋值表达式 (中缀解析函数)
func (p *Parser) parseAssignmentExpression(left Expression) Expression {
	// 检查左侧是否为有效的赋值目标
	switch leftExpr := left.(type) {
	case *Identifier:
		// 简单变量赋值: x = value
		exp := &AssignmentStatement{
			Token: p.curToken, // = token
			Name:  leftExpr,
		}

		p.nextToken()
		exp.Value = p.parseExpression(LOWEST)

		return exp
	case *IndexExpression:
		// 索引赋值: arr[index] = value
		exp := &IndexAssignmentStatement{
			Token: p.curToken, // = token
			Left:  leftExpr,
		}

		p.nextToken()
		exp.Value = p.parseExpression(LOWEST)

		return exp
	default:
		p.errors = append(p.errors, "invalid assignment target")
		return nil
	}
}

// parseCommaExpression 已移除 - AQL不再支持逗号运算符
// 逗号只作为分隔符使用（数组、函数参数、对象属性等）
