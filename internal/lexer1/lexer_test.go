package lexer1

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `let five = 5;
let ten = 10;
let add = function(x, y) {
	x + y;
};
let result = add(five, ten);
!-/*5;
5 < 10 > 5;

if (5 < 10) {
	return true;
} else {
	return false;
}

10 == 10;
10 != 9;
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{LET, "let"},
		{IDENT, "five"},
		{ASSIGN, "="},
		{INT, "5"},
		{SEMICOLON, ";"},
		{LET, "let"},
		{IDENT, "ten"},
		{ASSIGN, "="},
		{INT, "10"},
		{SEMICOLON, ";"},
		{LET, "let"},
		{IDENT, "add"},
		{ASSIGN, "="},
		{FUNCTION, "function"},
		{LPAREN, "("},
		{IDENT, "x"},
		{COMMA, ","},
		{IDENT, "y"},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{IDENT, "x"},
		{PLUS, "+"},
		{IDENT, "y"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{SEMICOLON, ";"},
		{LET, "let"},
		{IDENT, "result"},
		{ASSIGN, "="},
		{IDENT, "add"},
		{LPAREN, "("},
		{IDENT, "five"},
		{COMMA, ","},
		{IDENT, "ten"},
		{RPAREN, ")"},
		{SEMICOLON, ";"},
		{BANG, "!"},
		{MINUS, "-"},
		{SLASH, "/"},
		{ASTERISK, "*"},
		{INT, "5"},
		{SEMICOLON, ";"},
		{INT, "5"},
		{LT, "<"},
		{INT, "10"},
		{GT, ">"},
		{INT, "5"},
		{SEMICOLON, ";"},
		{IF, "if"},
		{LPAREN, "("},
		{INT, "5"},
		{LT, "<"},
		{INT, "10"},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{RETURN, "return"},
		{TRUE, "true"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{ELSE, "else"},
		{LBRACE, "{"},
		{RETURN, "return"},
		{FALSE, "false"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{INT, "10"},
		{EQ, "=="},
		{INT, "10"},
		{SEMICOLON, ";"},
		{INT, "10"},
		{NOT_EQ, "!="},
		{INT, "9"},
		{SEMICOLON, ";"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestAsyncAwaitTokens(t *testing.T) {
	input := `async function fetchData() {
		let result = await fetch("/api/data");
		yield result;
	}`

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{ASYNC, "async"},
		{FUNCTION, "function"},
		{IDENT, "fetchData"},
		{LPAREN, "("},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{LET, "let"},
		{IDENT, "result"},
		{ASSIGN, "="},
		{AWAIT, "await"},
		{IDENT, "fetch"},
		{LPAREN, "("},
		{STRING, "/api/data"},
		{RPAREN, ")"},
		{SEMICOLON, ";"},
		{YIELD, "yield"},
		{IDENT, "result"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestAIServiceTokens(t *testing.T) {
	input := `let response = await @openai.chat("Hello");
	data |> @preprocess.clean() |> @ai.analyze();`

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{LET, "let"},
		{IDENT, "response"},
		{ASSIGN, "="},
		{AWAIT, "await"},
		{AT_SYMBOL, "@"},
		{IDENT, "openai"},
		{DOT, "."},
		{IDENT, "chat"},
		{LPAREN, "("},
		{STRING, "Hello"},
		{RPAREN, ")"},
		{SEMICOLON, ";"},
		{IDENT, "data"},
		{PIPE, "|>"},
		{AT_SYMBOL, "@"},
		{IDENT, "preprocess"},
		{DOT, "."},
		{IDENT, "clean"},
		{LPAREN, "("},
		{RPAREN, ")"},
		{PIPE, "|>"},
		{AT_SYMBOL, "@"},
		{IDENT, "ai"},
		{DOT, "."},
		{IDENT, "analyze"},
		{LPAREN, "("},
		{RPAREN, ")"},
		{SEMICOLON, ";"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestNumberTokens(t *testing.T) {
	input := `42 3.14 0 123.456`

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{INT, "42"},
		{FLOAT, "3.14"},
		{INT, "0"},
		{FLOAT, "123.456"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestStringTokens(t *testing.T) {
	input := `"hello world" "test string" ""`

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{STRING, "hello world"},
		{STRING, "test string"},
		{STRING, ""},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestLogicalOperators(t *testing.T) {
	input := `&& || <= >= == !=`

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{AND, "&&"},
		{OR, "||"},
		{LTE, "<="},
		{GTE, ">="},
		{EQ, "=="},
		{NOT_EQ, "!="},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestComments(t *testing.T) {
	input := `let x = 5; // this is a comment
	// another comment
	let y = 10;`

	expectedTokens := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{LET, "let"},
		{IDENT, "x"},
		{ASSIGN, "="},
		{INT, "5"},
		{SEMICOLON, ";"},
		{COMMENT, "// this is a comment"},
		{COMMENT, "// another comment"},
		{LET, "let"},
		{IDENT, "y"},
		{ASSIGN, "="},
		{INT, "10"},
		{SEMICOLON, ";"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestComplexAQLProgram(t *testing.T) {
	input := `// AQL示例程序
	import { openai } from "@ai/openai";
	
	async function processText(input: string) {
		try {
			let cleaned = input |> @preprocess.clean();
			let result = await @openai.chat({
				model: "gpt-4",
				prompt: cleaned
			});
			return result;
		} catch (error) {
			throw error;
		}
	}
	
	export { processText };`

	l := New(input)

	// 获取所有token进行验证
	tokens := l.GetAllTokens()

	// 验证关键token存在
	expectedTypes := []TokenType{
		COMMENT, IMPORT, LBRACE, IDENT, RBRACE, FROM, STRING, SEMICOLON,
		ASYNC, FUNCTION, IDENT, LPAREN, IDENT, COLON, IDENT, RPAREN, LBRACE,
		TRY, LBRACE,
		LET, IDENT, ASSIGN, IDENT, PIPE, AT_SYMBOL, IDENT, DOT, IDENT, LPAREN, RPAREN, SEMICOLON,
		LET, IDENT, ASSIGN, AWAIT, AT_SYMBOL, IDENT, DOT, IDENT, LPAREN,
		LBRACE,
		IDENT, COLON, STRING, COMMA,
		IDENT, COLON, IDENT,
		RBRACE,
		RPAREN, SEMICOLON,
		RETURN, IDENT, SEMICOLON,
		RBRACE, CATCH, LPAREN, IDENT, RPAREN, LBRACE,
		THROW, IDENT, SEMICOLON,
		RBRACE,
		RBRACE,
		EXPORT, LBRACE, IDENT, RBRACE, SEMICOLON, EOF,
	}

	if len(tokens) < len(expectedTypes) {
		t.Fatalf("Expected at least %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	// 验证最后一个token是EOF
	if tokens[len(tokens)-1].Type != EOF {
		t.Fatalf("Expected last token to be EOF, got %q", tokens[len(tokens)-1].Type)
	}

	// 验证没有ILLEGAL token
	for i, tok := range tokens {
		if tok.Type == ILLEGAL {
			t.Fatalf("Found illegal token at position %d: %s", i, tok)
		}
	}
}

func TestLineAndColumnTracking(t *testing.T) {
	input := `let x = 5;
let y = 10;`

	l := New(input)

	// 第一行的token
	tok := l.NextToken() // let
	if tok.Line != 1 || tok.Column != 1 {
		t.Fatalf("Expected line=1, column=1, got line=%d, column=%d", tok.Line, tok.Column)
	}

	tok = l.NextToken() // x
	if tok.Line != 1 || tok.Column != 5 {
		t.Fatalf("Expected line=1, column=5, got line=%d, column=%d", tok.Line, tok.Column)
	}

	// 跳到第二行
	for tok.Line == 1 {
		tok = l.NextToken()
	}

	// 第二行的token
	if tok.Type == LET && (tok.Line != 2 || tok.Column != 1) {
		t.Fatalf("Expected line=2, column=1, got line=%d, column=%d", tok.Line, tok.Column)
	}
}

func TestErrorHandling(t *testing.T) {
	input := `let x = 5; & | #`

	l := New(input)

	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	// 验证illegal token的存在
	illegalCount := 0
	for _, tok := range tokens {
		if tok.Type == ILLEGAL {
			illegalCount++
		}
	}

	if illegalCount == 0 {
		t.Fatalf("Expected some illegal tokens for invalid characters")
	}
}

// 基准测试
func BenchmarkLexer(b *testing.B) {
	input := `
	async function processData(input: string) {
		let cleaned = input |> @preprocess.clean();
		let analyzed = await @ai.gpt4.analyze(cleaned);
		let formatted = analyzed |> @postprocess.format();
		return formatted;
	}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}

// 测试辅助函数
func TestIsValidIdentifier(t *testing.T) {
	validTests := []string{
		"hello",
		"_test",
		"test123",
		"Test_Case",
		"中文标识符",
	}

	invalidTests := []string{
		"",
		"123abc",
		"test-case",
		"test space",
		"test.case",
	}

	for _, test := range validTests {
		if !IsValidIdentifier(test) {
			t.Fatalf("Expected %q to be a valid identifier", test)
		}
	}

	for _, test := range invalidTests {
		if IsValidIdentifier(test) {
			t.Fatalf("Expected %q to be an invalid identifier", test)
		}
	}
}

func TestKeywordRecognition(t *testing.T) {
	keywords := map[string]TokenType{
		"function": FUNCTION,
		"let":      LET,
		"const":    CONST,
		"async":    ASYNC,
		"await":    AWAIT,
		"yield":    YIELD,
		"if":       IF,
		"else":     ELSE,
		"return":   RETURN,
		"true":     TRUE,
		"false":    FALSE,
		"null":     NULL,
	}

	for keyword, expectedType := range keywords {
		tokenType, isKeyword := IsKeyword(keyword)
		if !isKeyword {
			t.Fatalf("Expected %q to be recognized as keyword", keyword)
		}
		if tokenType != expectedType {
			t.Fatalf("Expected %q to be %s, got %s", keyword, expectedType, tokenType)
		}
	}

	// 测试非关键字
	if _, isKeyword := IsKeyword("notAKeyword"); isKeyword {
		t.Fatalf("Expected 'notAKeyword' to not be recognized as keyword")
	}
}
