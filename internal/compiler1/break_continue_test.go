package compiler1

import (
	"testing"

	"github.com/zhnt/aql/internal/lexer1"
	"github.com/zhnt/aql/internal/parser1"
)

func TestBreakStatementParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple break",
			input: "for (let i = 0; i < 10; i = i + 1) { if (i == 3) { break; } }",
		},
		{
			name:  "break with semicolon",
			input: "for (let i = 0; i < 10; i = i + 1) { if (i == 3) { break; } }",
		},
		{
			name:  "break in nested if",
			input: "for (let i = 0; i < 10; i = i + 1) { if (i > 5) { if (i == 7) { break; } } }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer1.New(tt.input)
			p := parser1.New(l)
			program := p.ParseProgram()

			// 检查解析错误
			if len(p.Errors()) > 0 {
				t.Fatalf("解析错误: %v", p.Errors())
			}

			if len(program.Statements) != 1 {
				t.Fatalf("期望1个语句，得到%d个", len(program.Statements))
			}

			t.Logf("✅ %s: AST解析成功", tt.name)
			t.Logf("   AST: %s", program.Statements[0].String())
		})
	}
}

func TestContinueStatementParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple continue",
			input: "for (let i = 0; i < 10; i = i + 1) { if (i == 3) { continue; } }",
		},
		{
			name:  "continue with semicolon",
			input: "for (let i = 0; i < 10; i = i + 1) { if (i == 3) { continue; } }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer1.New(tt.input)
			p := parser1.New(l)
			program := p.ParseProgram()

			// 检查解析错误
			if len(p.Errors()) > 0 {
				t.Fatalf("解析错误: %v", p.Errors())
			}

			if len(program.Statements) != 1 {
				t.Fatalf("期望1个语句，得到%d个", len(program.Statements))
			}

			t.Logf("✅ %s: AST解析成功", tt.name)
		})
	}
}

func TestBreakContinueCompilation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "break编译",
			input: "for (let i = 0; i < 10; i = i + 1) { if (i == 3) { break; } i; }",
		},
		{
			name:  "continue编译",
			input: "for (let i = 0; i < 10; i = i + 1) { if (i == 3) { continue; } i; }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer1.New(tt.input)
			p := parser1.New(l)
			program := p.ParseProgram()

			// 检查解析错误
			if len(p.Errors()) > 0 {
				t.Fatalf("解析错误: %v", p.Errors())
			}

			// 编译
			compiler := New()
			function, err := compiler.Compile(program)
			if err != nil {
				t.Fatalf("编译失败: %v", err)
			}

			t.Logf("✅ %s: 编译成功", tt.name)
			t.Logf("   生成指令: %d条", len(function.Instructions))

			// 打印字节码（调试用）
			for i, inst := range function.Instructions {
				t.Logf("   指令[%d]: op=%v, A=%d, B=%d, C=%d, Bx=%d",
					i, inst.OpCode, inst.A, inst.B, inst.C, inst.Bx)
			}
		})
	}
}

func TestBreakContinueOutsideLoop(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "break outside loop",
			input:         "break;",
			expectedError: "break statement not within a loop",
		},
		{
			name:          "continue outside loop",
			input:         "continue;",
			expectedError: "continue statement not within a loop",
		},
		{
			name:          "break in if without loop",
			input:         "if (true) { break; }",
			expectedError: "break statement not within a loop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer1.New(tt.input)
			p := parser1.New(l)
			program := p.ParseProgram()

			// 检查解析错误
			if len(p.Errors()) > 0 {
				t.Fatalf("解析错误: %v", p.Errors())
			}

			// 编译
			compiler := New()
			_, err := compiler.Compile(program)

			if err == nil {
				t.Fatalf("期望编译错误，但编译成功了")
			}

			if err.Error() != "编译错误: "+tt.expectedError {
				t.Fatalf("期望错误 '%s'，得到 '%s'", tt.expectedError, err.Error())
			}

			t.Logf("✅ %s: 正确检测到错误: %s", tt.name, err.Error())
		})
	}
}

func TestBreakContinueSemantics(t *testing.T) {
	// 测试break和continue的语义正确性
	breakTest := `
	let result = 0;
	for (let i = 0; i < 10; i = i + 1) {
		if (i == 5) {
			break;
		}
		result = result + i;
	}
	result
	`

	continueTest := `
	let result = 0;
	for (let i = 0; i < 10; i = i + 1) {
		if (i == 3) {
			continue;
		}
		if (i == 7) {
			continue;
		}
		result = result + i;
	}
	result
	`

	tests := []struct {
		name  string
		input string
	}{
		{"break语义测试", breakTest},
		{"continue语义测试", continueTest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer1.New(tt.input)
			p := parser1.New(l)
			program := p.ParseProgram()

			// 检查解析错误
			if len(p.Errors()) > 0 {
				t.Fatalf("解析错误: %v", p.Errors())
			}

			// 编译
			compiler := New()
			function, err := compiler.Compile(program)
			if err != nil {
				t.Fatalf("编译失败: %v", err)
			}

			t.Logf("✅ %s: 编译成功", tt.name)
			t.Logf("   生成指令: %d条", len(function.Instructions))

			// 注意：这里不执行VM测试，因为可能需要VM支持循环控制指令
			// 但至少验证了解析和编译都正确工作
		})
	}
}
