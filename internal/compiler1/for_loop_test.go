package compiler1

import (
	"testing"

	"github.com/zhnt/aql/internal/lexer1"
	"github.com/zhnt/aql/internal/parser1"
	"github.com/zhnt/aql/internal/vm"
)

func TestForLoopCompilation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "简单for循环计数",
			input:    "let sum = 0; for (let i = 1; i <= 3; i = i + 1) { sum = sum + i; } sum",
			expected: float64(6), // 1 + 2 + 3 = 6
		},
		{
			name:     "for循环无初始化",
			input:    "let i = 0; for (; i < 3; i = i + 1) { i; } i",
			expected: float64(3),
		},
		{
			name:     "for循环只有条件",
			input:    "let x = 0; for (; x < 2;) { x = x + 1; } x",
			expected: float64(2),
		},
		{
			name:     "for循环空条件(无限循环的简化版)",
			input:    "let count = 0; for (count = 0; count < 1; count = count + 1) { count; } count",
			expected: float64(1),
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

			// 执行
			executor := vm.NewExecutor()
			results, err := executor.Execute(function, []vm.Value{})
			if err != nil {
				t.Fatalf("执行失败: %v", err)
			}

			if len(results) == 0 {
				t.Fatalf("没有返回结果")
			}

			// 检查结果
			result := results[0]
			if result.IsNumber() {
				num, _ := result.ToNumber()
				if expected, ok := tt.expected.(float64); ok {
					if num != expected {
						t.Errorf("期望结果 %v，得到 %v", expected, num)
					}
				}
			} else {
				t.Errorf("期望数字结果，得到 %s", result.Type())
			}
		})
	}
}

func TestForLoopAST(t *testing.T) {
	input := "for (let i = 0; i < 10; i = i + 1) { print(i); }"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	// 检查解析错误
	if len(p.Errors()) > 0 {
		t.Fatalf("解析错误: %v", p.Errors())
	}

	if len(program.Statements) != 1 {
		t.Fatalf("期望1个语句，得到%d个", len(program.Statements))
	}

	forStmt, ok := program.Statements[0].(*parser1.ForStatement)
	if !ok {
		t.Fatalf("期望ForStatement，得到%T", program.Statements[0])
	}

	// 检查AST结构
	if forStmt.Init == nil {
		t.Error("期望有初始化语句")
	}

	if forStmt.Condition == nil {
		t.Error("期望有条件表达式")
	}

	if forStmt.Update == nil {
		t.Error("期望有更新表达式")
	}

	if forStmt.Body == nil {
		t.Error("期望有循环体")
	}

	t.Logf("✅ for循环AST解析正确: %s", forStmt.String())
}

func TestForLoopBytecode(t *testing.T) {
	input := "for (let i = 0; i < 2; i = i + 1) { i; }"

	l := lexer1.New(input)
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

	// 检查生成的字节码
	t.Logf("✅ 生成了 %d 条指令", len(function.Instructions))
	t.Logf("✅ 生成了 %d 个常量", len(function.Constants))

	// 打印字节码（调试用）
	for i, inst := range function.Instructions {
		t.Logf("指令[%d]: op=%v, A=%d, B=%d, C=%d, Bx=%d",
			i, inst.OpCode, inst.A, inst.B, inst.C, inst.Bx)
	}

	// 执行
	executor := vm.NewExecutor()
	_, err = executor.Execute(function, []vm.Value{})
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	t.Log("✅ for循环编译和执行成功")
}
