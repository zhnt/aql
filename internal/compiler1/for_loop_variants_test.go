package compiler1

import (
	"testing"

	"github.com/zhnt/aql/internal/lexer1"
	"github.com/zhnt/aql/internal/parser1"
)

func TestAllForLoopVariants(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "完整for循环",
			input:       "for (let i = 0; i < 3; i = i + 1) { i; }",
			description: "标准C风格for循环",
		},
		{
			name:        "无初始化",
			input:       "let i = 0; for (; i < 3; i = i + 1) { i; }",
			description: "省略初始化语句（变量预定义）",
		},
		{
			name:        "无条件",
			input:       "for (let i = 0; ; i = i + 1) { i; }",
			description: "省略条件表达式（无限循环）",
		},
		{
			name:        "无更新",
			input:       "for (let i = 0; i < 3; ) { i = i + 1; }",
			description: "省略更新表达式",
		},
		{
			name:        "只有条件",
			input:       "let i = 0; for (; i < 3; ) { i = i + 1; }",
			description: "只有条件表达式（变量预定义）",
		},
		{
			name:        "两个分号",
			input:       "for (;;) { x = 1; }",
			description: "经典无限循环",
		},
		{
			name:        "三个分号",
			input:       "for (;;;) { x = 1; }",
			description: "带额外分号的无限循环",
		},
		{
			name:        "赋值初始化",
			input:       "for (x = 0; x < 2; x = x + 1) { x; }",
			description: "使用赋值而非let声明",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("测试: %s", tt.description)

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

func TestForLoopSyntaxComparison(t *testing.T) {
	t.Log("=== AQL C风格For循环语法对比 ===")

	syntaxExamples := []struct {
		syntax      string
		description string
		aqlCode     string
	}{
		{
			syntax:      "for (init; cond; update)",
			description: "标准三部分for循环",
			aqlCode:     "for (let i = 0; i < 5; i = i + 1) { print(i); }",
		},
		{
			syntax:      "for (; cond; update)",
			description: "省略初始化",
			aqlCode:     "for (; running; step = step + 1) { process(); }",
		},
		{
			syntax:      "for (init; ; update)",
			description: "省略条件（无限循环）",
			aqlCode:     "for (let x = 0; ; x = x + 1) { if (x > 10) break; }",
		},
		{
			syntax:      "for (init; cond; )",
			description: "省略更新",
			aqlCode:     "for (let y = 0; y < 10; ) { y = y + 2; }",
		},
		{
			syntax:      "for (;;)",
			description: "经典无限循环",
			aqlCode:     "for (;;) { doWork(); if (done) break; }",
		},
		{
			syntax:      "for (;;;)",
			description: "容错：额外分号",
			aqlCode:     "for (;;;) { keepRunning(); }",
		},
	}

	for _, example := range syntaxExamples {
		t.Logf("语法: %-25s | %s", example.syntax, example.description)
		t.Logf("示例: %s", example.aqlCode)
		t.Log("")
	}
}

func TestForLoopExecutionDemo(t *testing.T) {
	// 简单的可执行for循环例子
	input := "let sum = 0; for (let i = 1; i <= 3; i = i + 1) { sum = sum + i; } sum"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("解析失败: %v", p.Errors())
	}

	compiler := New()
	function, err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}

	t.Logf("✅ for循环计算示例编译成功")
	t.Logf("   代码: %s", input)
	t.Logf("   生成指令: %d条", len(function.Instructions))
	t.Logf("   常量: %d个", len(function.Constants))
}
