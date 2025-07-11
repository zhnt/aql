package compiler1

import (
	"testing"

	"github.com/zhnt/aql/internal/lexer1"
	"github.com/zhnt/aql/internal/parser1"
	"github.com/zhnt/aql/internal/vm"
)

func TestWhileLoopCompilation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "简单while循环计数",
			input:    "let sum = 0; let i = 1; while (i <= 3) { sum = sum + i; i = i + 1; } sum",
			expected: float64(6), // 1 + 2 + 3 = 6
		},
		{
			name:     "while循环递减",
			input:    "let result = 1; let count = 3; while (count > 0) { result = result * count; count = count - 1; } result",
			expected: float64(6), // 3! = 3 * 2 * 1 = 6
		},
		{
			name:     "条件为假的while循环",
			input:    "let x = 5; while (x < 5) { x = x + 1; } x",
			expected: float64(5), // 条件为假，循环体不执行
		},
		{
			name:     "while循环布尔条件",
			input:    "let done = false; let count = 0; while (!done) { count = count + 1; if (count >= 3) { done = true; } } count",
			expected: float64(3),
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

func TestWhileLoopAST(t *testing.T) {
	input := "while (x < 10) { x = x + 1; }"

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

	whileStmt, ok := program.Statements[0].(*parser1.WhileStatement)
	if !ok {
		t.Fatalf("期望WhileStatement，得到%T", program.Statements[0])
	}

	// 检查AST结构
	if whileStmt.Condition == nil {
		t.Error("期望有条件表达式")
	}

	if whileStmt.Body == nil {
		t.Error("期望有循环体")
	}

	t.Logf("✅ while循环AST解析正确: %s", whileStmt.String())
}

func TestWhileLoopBreakContinue(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "while循环中的break",
			input: "let sum = 0; let i = 0; while (i < 10) { if (i == 3) { break; } sum = sum + i; i = i + 1; } sum",
		},
		{
			name:  "while循环中的continue",
			input: "let sum = 0; let i = 0; while (i < 5) { i = i + 1; if (i == 3) { continue; } sum = sum + i; } sum",
		},
		{
			name:  "嵌套while循环中的break",
			input: "let result = 0; let i = 0; while (i < 3) { let j = 0; while (j < 5) { if (j == 2) { break; } result = result + 1; j = j + 1; } i = i + 1; } result",
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

			// 验证执行不报错
			executor := vm.NewExecutor()
			_, err = executor.Execute(function, []vm.Value{})
			if err != nil {
				t.Fatalf("执行失败: %v", err)
			}

			t.Logf("✅ %s: 执行成功", tt.name)
		})
	}
}

func TestWhileLoopBytecode(t *testing.T) {
	input := "let x = 0; while (x < 3) { x = x + 1; } x"

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
	results, err := executor.Execute(function, []vm.Value{})
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	// 验证结果
	if len(results) > 0 && results[0].IsNumber() {
		num, _ := results[0].ToNumber()
		if num != 3 {
			t.Errorf("期望结果3，得到%v", num)
		}
	}

	t.Log("✅ while循环编译和执行成功")
}

func TestWhileLoopSyntaxVariants(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "简单条件",
			input:       "while (true) { break; }",
			description: "最简单的while循环",
		},
		{
			name:        "复杂条件",
			input:       "let x = 5; while (x > 0) { x = x - 1; }",
			description: "简单比较条件",
		},
		{
			name:        "嵌套循环",
			input:       "let i = 0; let j = 0; while (i < 3) { while (j < 2) { j = j + 1; } i = i + 1; }",
			description: "嵌套while循环",
		},
		{
			name:        "空循环体",
			input:       "while (false) { }",
			description: "空的循环体",
		},
		{
			name:        "条件中的函数调用",
			input:       "let done = false; while (!done) { done = true; }",
			description: "条件中使用逻辑运算",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("测试: %s - %s", tt.name, tt.description)

			// 解析
			l := lexer1.New(tt.input)
			p := parser1.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("解析失败 %s: %v", tt.input, p.Errors())
			}

			// 编译
			compiler := New()
			function, err := compiler.Compile(program)
			if err != nil {
				t.Fatalf("编译失败 %s: %v", tt.input, err)
			}

			t.Logf("✅ %s: 解析和编译成功", tt.name)
			t.Logf("   AST: %s", program.Statements[0].String())
			t.Logf("   生成指令: %d条", len(function.Instructions))
		})
	}
}

func TestWhileVsForLoop(t *testing.T) {
	// 对比测试：相同逻辑的while循环和for循环应该产生相同结果

	whileVersion := `
		let sum = 0;
		let i = 1;
		while (i <= 5) {
			sum = sum + i;
			i = i + 1;
		}
		sum
	`

	forVersion := `
		let sum = 0;
		for (let i = 1; i <= 5; i = i + 1) {
			sum = sum + i;
		}
		sum
	`

	// 编译和执行while版本
	l1 := lexer1.New(whileVersion)
	p1 := parser1.New(l1)
	program1 := p1.ParseProgram()

	if len(p1.Errors()) > 0 {
		t.Fatalf("while版本解析错误: %v", p1.Errors())
	}

	compiler1 := New()
	function1, err := compiler1.Compile(program1)
	if err != nil {
		t.Fatalf("while版本编译失败: %v", err)
	}

	executor1 := vm.NewExecutor()
	results1, err := executor1.Execute(function1, []vm.Value{})
	if err != nil {
		t.Fatalf("while版本执行失败: %v", err)
	}

	// 编译和执行for版本
	l2 := lexer1.New(forVersion)
	p2 := parser1.New(l2)
	program2 := p2.ParseProgram()

	if len(p2.Errors()) > 0 {
		t.Fatalf("for版本解析错误: %v", p2.Errors())
	}

	compiler2 := New()
	function2, err := compiler2.Compile(program2)
	if err != nil {
		t.Fatalf("for版本编译失败: %v", err)
	}

	executor2 := vm.NewExecutor()
	results2, err := executor2.Execute(function2, []vm.Value{})
	if err != nil {
		t.Fatalf("for版本执行失败: %v", err)
	}

	// 比较结果
	if len(results1) != len(results2) {
		t.Fatalf("结果数量不同: while=%d, for=%d", len(results1), len(results2))
	}

	if results1[0].ToString() != results2[0].ToString() {
		t.Errorf("结果不同: while=%s, for=%s", results1[0].ToString(), results2[0].ToString())
	}

	t.Logf("✅ while和for循环产生相同结果: %s", results1[0].ToString())
}
