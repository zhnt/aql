package lexer1

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Lexer 词法分析器
type Lexer struct {
	input        string // 输入源码
	position     int    // 当前位置（指向当前字符）
	readPosition int    // 当前读取位置（指向当前字符的下一个字符）
	ch           byte   // 当前正在检查的字符
	line         int    // 当前行号
	column       int    // 当前列号
}

// New 创建新的词法分析器
func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar() // 读取第一个字符
	return l
}

// readChar 读取下一个字符并移动position指针
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII码的NUL字符，表示"EOF"
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++

	// 更新行号和列号
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar 查看下一个字符但不移动指针
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// skipWhitespace 跳过空白字符
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier 读取标识符
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber 读取数字（支持整数和浮点数）
func (l *Lexer) readNumber() (string, TokenType) {
	position := l.position
	tokenType := INT

	// 读取整数部分
	for isDigit(l.ch) {
		l.readChar()
	}

	// 检查是否为浮点数
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = FLOAT
		l.readChar() // 跳过 '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position], tokenType
}

// readString 读取字符串字面量
func (l *Lexer) readString() string {
	position := l.position + 1 // 跳过开始的引号
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		// 处理转义字符
		if l.ch == '\\' {
			l.readChar() // 跳过转义字符
		}
	}
	return l.input[position:l.position]
}

// readSingleLineComment 读取单行注释
func (l *Lexer) readSingleLineComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.position]
}

// NextToken 获取下一个token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	line, column := l.line, l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(EQ, string(ch)+string(l.ch), line, column, l.position-1)
		} else {
			tok = NewToken(ASSIGN, string(l.ch), line, column, l.position)
		}
	case '+':
		tok = NewToken(PLUS, string(l.ch), line, column, l.position)
	case '-':
		tok = NewToken(MINUS, string(l.ch), line, column, l.position)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(NOT_EQ, string(ch)+string(l.ch), line, column, l.position-1)
		} else {
			tok = NewToken(BANG, string(l.ch), line, column, l.position)
		}
	case '*':
		tok = NewToken(ASTERISK, string(l.ch), line, column, l.position)
	case '/':
		if l.peekChar() == '/' {
			// 单行注释
			literal := l.readSingleLineComment()
			tok = NewToken(COMMENT, literal, line, column, l.position-len(literal))
			return tok // 直接返回，不调用readChar
		} else {
			tok = NewToken(SLASH, string(l.ch), line, column, l.position)
		}
	case '%':
		tok = NewToken(PERCENT, string(l.ch), line, column, l.position)
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(LTE, string(ch)+string(l.ch), line, column, l.position-1)
		} else {
			tok = NewToken(LT, string(l.ch), line, column, l.position)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = NewToken(GTE, string(ch)+string(l.ch), line, column, l.position-1)
		} else {
			tok = NewToken(GT, string(l.ch), line, column, l.position)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = NewToken(AND, string(ch)+string(l.ch), line, column, l.position-1)
		} else {
			tok = NewToken(ILLEGAL, string(l.ch), line, column, l.position)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = NewToken(OR, string(ch)+string(l.ch), line, column, l.position-1)
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = NewToken(PIPE, string(ch)+string(l.ch), line, column, l.position-1)
		} else {
			tok = NewToken(ILLEGAL, string(l.ch), line, column, l.position)
		}
	case ',':
		tok = NewToken(COMMA, string(l.ch), line, column, l.position)
	case ';':
		tok = NewToken(SEMICOLON, string(l.ch), line, column, l.position)
	case ':':
		tok = NewToken(COLON, string(l.ch), line, column, l.position)
	case '.':
		tok = NewToken(DOT, string(l.ch), line, column, l.position)
	case '(':
		tok = NewToken(LPAREN, string(l.ch), line, column, l.position)
	case ')':
		tok = NewToken(RPAREN, string(l.ch), line, column, l.position)
	case '{':
		tok = NewToken(LBRACE, string(l.ch), line, column, l.position)
	case '}':
		tok = NewToken(RBRACE, string(l.ch), line, column, l.position)
	case '[':
		tok = NewToken(LBRACKET, string(l.ch), line, column, l.position)
	case ']':
		tok = NewToken(RBRACKET, string(l.ch), line, column, l.position)
	case '@':
		tok = NewToken(AT_SYMBOL, string(l.ch), line, column, l.position)
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Line = line
		tok.Column = column
		tok.Position = l.position - len(tok.Literal) - 1
	case 0:
		tok = NewToken(EOF, "", line, column, l.position)
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Line = line
			tok.Column = column
			tok.Position = l.position - len(tok.Literal)

			// 检查是否为关键字
			if tokenType, isKeyword := IsKeyword(tok.Literal); isKeyword {
				tok.Type = tokenType
			} else {
				tok.Type = IDENT
			}
			return tok // 不需要调用readChar，readIdentifier已经移动了指针
		} else if isDigit(l.ch) {
			tok.Literal, tok.Type = l.readNumber()
			tok.Line = line
			tok.Column = column
			tok.Position = l.position - len(tok.Literal)
			return tok // 不需要调用readChar，readNumber已经移动了指针
		} else {
			tok = NewToken(ILLEGAL, string(l.ch), line, column, l.position)
		}
	}

	l.readChar()
	return tok
}

// GetAllTokens 获取所有token（用于调试和测试）
func (l *Lexer) GetAllTokens() []Token {
	var tokens []Token

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	return tokens
}

// PrintTokens 打印所有token（调试用）
func (l *Lexer) PrintTokens() {
	for {
		tok := l.NextToken()
		fmt.Printf("%s\n", tok)
		if tok.Type == EOF {
			break
		}
	}
}

// 辅助函数

// isLetter 检查字符是否为字母
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit 检查字符是否为数字
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// isAlphaNumeric 检查字符是否为字母或数字
func isAlphaNumeric(ch byte) bool {
	return isLetter(ch) || isDigit(ch)
}

// IsValidIdentifier 检查字符串是否为有效的标识符
func IsValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	// 第一个字符必须是字母或下划线
	r, size := utf8.DecodeRuneInString(s)
	if !unicode.IsLetter(r) && r != '_' {
		return false
	}

	// 其余字符必须是字母、数字或下划线
	for i := size; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
		i += size
	}

	return true
}
