package lexer1

import "fmt"

// TokenType Token类型枚举
type TokenType int

const (
	// 特殊Token
	ILLEGAL TokenType = iota // 非法token
	EOF                      // 文件结束

	// 标识符和字面量
	IDENT  // 标识符: add, foobar, x, y, ...
	INT    // 整数: 1343456
	FLOAT  // 浮点数: 13.45
	STRING // 字符串: "foobar"
	BOOL   // 布尔值: true, false

	// 运算符
	ASSIGN   // =
	PLUS     // +
	MINUS    // -
	BANG     // !
	ASTERISK // *
	SLASH    // /
	PERCENT  // %

	// 比较运算符
	LT     // <
	GT     // >
	EQ     // ==
	NOT_EQ // !=
	LTE    // <=
	GTE    // >=

	// 逻辑运算符
	AND // &&
	OR  // ||

	// 分隔符
	COMMA     // ,
	SEMICOLON // ;
	COLON     // :
	DOT       // .

	// 括号
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]

	// 关键字
	FUNCTION // function
	LET      // let
	CONST    // const
	IF       // if
	ELSE     // else
	RETURN   // return
	TRUE     // true
	FALSE    // false
	NULL     // null
	FOR      // for
	WHILE    // while
	BREAK    // break
	CONTINUE // continue
	ARRAY    // Array (预分配容量数组构造器)

	// 异步相关关键字
	ASYNC // async
	AWAIT // await
	YIELD // yield

	// 类型相关
	TYPE      // type
	INTERFACE // interface
	CLASS     // class
	EXTENDS   // extends

	// AI服务相关
	AT_SYMBOL // @
	PIPE      // |>

	// 导入导出
	IMPORT // import
	EXPORT // export
	FROM   // from

	// 异常处理
	TRY     // try
	CATCH   // catch
	FINALLY // finally
	THROW   // throw

	// 注释
	COMMENT // // 单行注释
)

// Token 表示一个词法单元
type Token struct {
	Type     TokenType // token类型
	Literal  string    // 字面值
	Line     int       // 行号
	Column   int       // 列号
	Position int       // 在源码中的位置
}

// String 返回TokenType的字符串表示
func (t TokenType) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case STRING:
		return "STRING"
	case BOOL:
		return "BOOL"
	case ASSIGN:
		return "ASSIGN"
	case PLUS:
		return "PLUS"
	case MINUS:
		return "MINUS"
	case BANG:
		return "BANG"
	case ASTERISK:
		return "ASTERISK"
	case SLASH:
		return "SLASH"
	case PERCENT:
		return "PERCENT"
	case LT:
		return "LT"
	case GT:
		return "GT"
	case EQ:
		return "EQ"
	case NOT_EQ:
		return "NOT_EQ"
	case LTE:
		return "LTE"
	case GTE:
		return "GTE"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case COMMA:
		return "COMMA"
	case SEMICOLON:
		return "SEMICOLON"
	case COLON:
		return "COLON"
	case DOT:
		return "DOT"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case FUNCTION:
		return "FUNCTION"
	case LET:
		return "LET"
	case CONST:
		return "CONST"
	case IF:
		return "IF"
	case ELSE:
		return "ELSE"
	case RETURN:
		return "RETURN"
	case TRUE:
		return "TRUE"
	case FALSE:
		return "FALSE"
	case NULL:
		return "NULL"
	case FOR:
		return "FOR"
	case WHILE:
		return "WHILE"
	case BREAK:
		return "BREAK"
	case CONTINUE:
		return "CONTINUE"
	case ARRAY:
		return "ARRAY"
	case ASYNC:
		return "ASYNC"
	case AWAIT:
		return "AWAIT"
	case YIELD:
		return "YIELD"
	case TYPE:
		return "TYPE"
	case INTERFACE:
		return "INTERFACE"
	case CLASS:
		return "CLASS"
	case EXTENDS:
		return "EXTENDS"
	case AT_SYMBOL:
		return "AT_SYMBOL"
	case PIPE:
		return "PIPE"
	case IMPORT:
		return "IMPORT"
	case EXPORT:
		return "EXPORT"
	case FROM:
		return "FROM"
	case TRY:
		return "TRY"
	case CATCH:
		return "CATCH"
	case FINALLY:
		return "FINALLY"
	case THROW:
		return "THROW"
	case COMMENT:
		return "COMMENT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(t))
	}
}

// String 返回Token的字符串表示
func (t Token) String() string {
	return fmt.Sprintf("{Type:%s, Literal:%q, Line:%d, Column:%d}",
		t.Type, t.Literal, t.Line, t.Column)
}

// IsKeyword 检查标识符是否为关键字
func IsKeyword(ident string) (TokenType, bool) {
	keywords := map[string]TokenType{
		"function":  FUNCTION,
		"let":       LET,
		"const":     CONST,
		"if":        IF,
		"else":      ELSE,
		"return":    RETURN,
		"true":      TRUE,
		"false":     FALSE,
		"null":      NULL,
		"for":       FOR,
		"while":     WHILE,
		"break":     BREAK,
		"continue":  CONTINUE,
		"Array":     ARRAY,
		"async":     ASYNC,
		"await":     AWAIT,
		"yield":     YIELD,
		"type":      TYPE,
		"interface": INTERFACE,
		"class":     CLASS,
		"extends":   EXTENDS,
		"import":    IMPORT,
		"export":    EXPORT,
		"from":      FROM,
		"try":       TRY,
		"catch":     CATCH,
		"finally":   FINALLY,
		"throw":     THROW,
	}

	tokenType, exists := keywords[ident]
	return tokenType, exists
}

// NewToken 创建新的Token
func NewToken(tokenType TokenType, literal string, line, column, position int) Token {
	return Token{
		Type:     tokenType,
		Literal:  literal,
		Line:     line,
		Column:   column,
		Position: position,
	}
}
