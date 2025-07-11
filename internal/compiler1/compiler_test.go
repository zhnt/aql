package compiler1

import (
	"testing"

	"github.com/zhnt/aql/internal/lexer1"
	"github.com/zhnt/aql/internal/parser1"
	"github.com/zhnt/aql/internal/vm"
)

func TestIntegerLiteralCompilation(t *testing.T) {
	input := "5;"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	compiler := New()
	function, err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}

	if len(function.Instructions) != 2 {
		t.Fatalf("期望2个指令，得到%d个", len(function.Instructions))
	}

	// 检查第一个指令: LOADK
	if function.Instructions[0].OpCode != vm.OP_LOADK {
		t.Errorf("期望指令OP_LOADK，得到%v", function.Instructions[0].OpCode)
	}

	// 检查第二个指令: POP
	if function.Instructions[1].OpCode != vm.OP_POP {
		t.Errorf("期望指令OP_POP，得到%v", function.Instructions[1].OpCode)
	}

	// 检查常量池
	if len(function.Constants) != 1 {
		t.Fatalf("期望1个常量，得到%d个", len(function.Constants))
	}

	constant := function.Constants[0]
	if !constant.IsNumber() {
		t.Errorf("期望常量类型为数字，得到%v", constant.Type())
	}

	if num, _ := constant.ToNumber(); num != 5.0 {
		t.Errorf("期望常量值5.0，得到%f", num)
	}
}

func TestBooleanLiteralCompilation(t *testing.T) {
	input := "true;"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	compiler := New()
	function, err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}

	if len(function.Instructions) != 2 {
		t.Fatalf("期望2个指令，得到%d个", len(function.Instructions))
	}

	// 检查常量池
	if len(function.Constants) != 1 {
		t.Fatalf("期望1个常量，得到%d个", len(function.Constants))
	}

	constant := function.Constants[0]
	if !constant.IsBool() {
		t.Errorf("期望常量类型为布尔值，得到%v", constant.Type())
	}

	if constant.AsBool() != true {
		t.Errorf("期望常量值true，得到%v", constant.AsBool())
	}
}

func TestStringLiteralCompilation(t *testing.T) {
	input := "\"hello world\";"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	compiler := New()
	function, err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}

	if len(function.Instructions) != 2 {
		t.Fatalf("期望2个指令，得到%d个", len(function.Instructions))
	}

	// 检查常量池
	if len(function.Constants) != 1 {
		t.Fatalf("期望1个常量，得到%d个", len(function.Constants))
	}

	constant := function.Constants[0]
	if !constant.IsString() {
		t.Errorf("期望常量类型为字符串，得到%v", constant.Type())
	}

	if constant.AsString() != "hello world" {
		t.Errorf("期望常量值\"hello world\"，得到%s", constant.AsString())
	}
}

func TestInfixExpressionCompilation(t *testing.T) {
	input := "1 + 2;"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	compiler := New()
	function, err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}

	// 期望指令：
	// LOADK 0 0  ; 加载常量1到R0
	// LOADK 0 1  ; 加载常量2到R0（覆盖）
	// ADD 0 0 1  ; R0 := R0 + R1
	// POP        ; 丢弃结果

	if len(function.Instructions) != 4 {
		t.Fatalf("期望4个指令，得到%d个", len(function.Instructions))
	}

	// 检查指令
	expectedOps := []vm.OpCode{vm.OP_LOADK, vm.OP_LOADK, vm.OP_ADD, vm.OP_POP}
	for i, expectedOp := range expectedOps {
		if function.Instructions[i].OpCode != expectedOp {
			t.Errorf("指令[%d]: 期望%v，得到%v", i, expectedOp, function.Instructions[i].OpCode)
		}
	}

	// 检查常量池
	if len(function.Constants) != 2 {
		t.Fatalf("期望2个常量，得到%d个", len(function.Constants))
	}

	if num, _ := function.Constants[0].ToNumber(); num != 1.0 {
		t.Errorf("期望常量[0]值1.0，得到%f", num)
	}

	if num, _ := function.Constants[1].ToNumber(); num != 2.0 {
		t.Errorf("期望常量[1]值2.0，得到%f", num)
	}
}

func TestLetStatementCompilation(t *testing.T) {
	input := "let x = 10;"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	compiler := New()
	function, err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}

	// 期望指令：
	// LOADK 0 0      ; 加载常量10到R0
	// SET_GLOBAL 0 0 ; 设置全局变量x = R0

	if len(function.Instructions) != 2 {
		t.Fatalf("期望2个指令，得到%d个", len(function.Instructions))
	}

	// 检查指令
	expectedOps := []vm.OpCode{vm.OP_LOADK, vm.OP_SET_GLOBAL}
	for i, expectedOp := range expectedOps {
		if function.Instructions[i].OpCode != expectedOp {
			t.Errorf("指令[%d]: 期望%v，得到%v", i, expectedOp, function.Instructions[i].OpCode)
		}
	}

	// 检查常量池
	if len(function.Constants) != 1 {
		t.Fatalf("期望1个常量，得到%d个", len(function.Constants))
	}

	if num, _ := function.Constants[0].ToNumber(); num != 10.0 {
		t.Errorf("期望常量值10.0，得到%f", num)
	}

	// 检查符号表
	symbol, ok := compiler.symbolTable.Resolve("x")
	if !ok {
		t.Fatalf("变量x未在符号表中定义")
	}

	if symbol.Name != "x" {
		t.Errorf("期望符号名x，得到%s", symbol.Name)
	}

	if symbol.Scope != GLOBAL_SCOPE {
		t.Errorf("期望符号作用域GLOBAL_SCOPE，得到%v", symbol.Scope)
	}

	if symbol.Index != 0 {
		t.Errorf("期望符号索引0，得到%d", symbol.Index)
	}
}

func TestIdentifierCompilation(t *testing.T) {
	input := "let x = 10; x;"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	compiler := New()
	function, err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}

	// 期望指令：
	// LOADK 0 0      ; 加载常量10到R0
	// SET_GLOBAL 0 0 ; 设置全局变量x = R0
	// GET_GLOBAL 0 0 ; 获取全局变量x到R0
	// POP            ; 丢弃结果

	if len(function.Instructions) != 4 {
		t.Fatalf("期望4个指令，得到%d个", len(function.Instructions))
	}

	// 检查指令
	expectedOps := []vm.OpCode{vm.OP_LOADK, vm.OP_SET_GLOBAL, vm.OP_GET_GLOBAL, vm.OP_POP}
	for i, expectedOp := range expectedOps {
		if function.Instructions[i].OpCode != expectedOp {
			t.Errorf("指令[%d]: 期望%v，得到%v", i, expectedOp, function.Instructions[i].OpCode)
		}
	}
}

func TestPrefixExpressionCompilation(t *testing.T) {
	input := "!true;"

	l := lexer1.New(input)
	p := parser1.New(l)
	program := p.ParseProgram()

	compiler := New()
	function, err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}

	// 期望指令：
	// LOADK 0 0  ; 加载常量true到R0
	// NOT 0 0 0  ; R0 := !R0
	// POP        ; 丢弃结果

	if len(function.Instructions) != 3 {
		t.Fatalf("期望3个指令，得到%d个", len(function.Instructions))
	}

	// 检查指令
	expectedOps := []vm.OpCode{vm.OP_LOADK, vm.OP_NOT, vm.OP_POP}
	for i, expectedOp := range expectedOps {
		if function.Instructions[i].OpCode != expectedOp {
			t.Errorf("指令[%d]: 期望%v，得到%v", i, expectedOp, function.Instructions[i].OpCode)
		}
	}
}

func TestCompilerErrors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x;", "undefined variable: x"},
		{"if (true) { 1 }", "if expressions not yet implemented"},
		{"function() { 1 }", "function literals not yet implemented"},
		{"add(1, 2)", "function calls not yet implemented"},
		{"[1, 2, 3]", "array literals not yet implemented"},
		{"arr[0]", "index expressions not yet implemented"},
	}

	for _, tt := range tests {
		l := lexer1.New(tt.input)
		p := parser1.New(l)
		program := p.ParseProgram()

		compiler := New()
		_, err := compiler.Compile(program)
		if err == nil {
			t.Errorf("期望编译错误，但没有出错。输入：%s", tt.input)
			continue
		}

		if err.Error() != "编译错误: "+tt.expected {
			t.Errorf("期望错误消息：%s，得到：%s", tt.expected, err.Error())
		}
	}
}
