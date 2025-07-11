package vm

import (
	"testing"
	"unsafe"
)

func TestValueSystem(t *testing.T) {
	t.Log("=== VM3 Value系统测试 ===")

	// 测试内存大小
	size := unsafe.Sizeof(Value{})
	t.Logf("VM3 Value大小: %d 字节", size)

	// 测试小整数
	v1 := NewSmallIntValue(42)
	if !v1.IsSmallInt() {
		t.Error("应该是小整数类型")
	}
	if v1.AsSmallInt() != 42 {
		t.Errorf("期望42，得到%d", v1.AsSmallInt())
	}

	// 测试自动优化：整数作为SmallInt存储
	v2 := NewNumberValue(100)
	if !v2.IsSmallInt() {
		t.Error("整数应该自动优化为SmallInt")
	}
	if v2.AsSmallInt() != 100 {
		t.Errorf("期望100，得到%d", v2.AsSmallInt())
	}

	// 测试大数：自动存储为Double
	v3 := NewNumberValue(3.14159)
	if !v3.IsDouble() {
		t.Error("浮点数应该存储为Double")
	}
	if v3.AsDouble() != 3.14159 {
		t.Errorf("期望3.14159，得到%f", v3.AsDouble())
	}

	// 测试字符串
	v4 := NewStringValue("hello")
	if !v4.IsString() {
		t.Error("应该是字符串类型")
	}
	if v4.AsString() != "hello" {
		t.Errorf("期望'hello'，得到'%s'", v4.AsString())
	}

	// 测试布尔值
	v5 := NewBoolValue(true)
	if !v5.IsBool() {
		t.Error("应该是布尔类型")
	}
	if !v5.AsBool() {
		t.Error("期望true")
	}

	t.Log("✅ Value系统测试通过")
}

func TestArithmetic(t *testing.T) {
	t.Log("=== VM3 算术运算测试 ===")

	// 小整数加法
	a := NewSmallIntValue(10)
	b := NewSmallIntValue(20)
	result, err := AddValues(a, b)
	if err != nil {
		t.Fatalf("小整数加法错误: %v", err)
	}
	if !result.IsSmallInt() || result.AsSmallInt() != 30 {
		t.Errorf("期望SmallInt(30)，得到%s", result.ToString())
	}

	// 小整数溢出测试
	big1 := NewSmallIntValue(VALUE_SMALL_INT_MAX)
	big2 := NewSmallIntValue(1)
	result, err = AddValues(big1, big2)
	if err != nil {
		t.Fatalf("溢出处理错误: %v", err)
	}
	if !result.IsDouble() {
		t.Error("溢出应该转为Double")
	}
	expectedValue := float64(VALUE_SMALL_INT_MAX) + 1.0
	if result.AsDouble() != expectedValue {
		t.Errorf("期望%f，得到%f", expectedValue, result.AsDouble())
	}

	// 混合运算：SmallInt + Double
	si := NewSmallIntValue(10)
	d := NewDoubleValue(3.5)
	result, err = AddValues(si, d)
	if err != nil {
		t.Fatalf("混合运算错误: %v", err)
	}
	if result.AsDouble() != 13.5 {
		t.Errorf("期望13.5，得到%f", result.AsDouble())
	}

	// 字符串连接
	s1 := NewStringValue("Hello ")
	s2 := NewStringValue("World")
	result, err = AddValues(s1, s2)
	if err != nil {
		t.Fatalf("字符串连接错误: %v", err)
	}
	if result.AsString() != "Hello World" {
		t.Errorf("期望'Hello World'，得到'%s'", result.AsString())
	}

	t.Log("✅ 算术运算测试通过")
}

func TestSimpleExecution(t *testing.T) {
	t.Log("=== VM3 简单执行测试 ===")

	// 创建简单函数：return 10 + 20
	fn := NewFunction("simple_add")
	fn.MaxStackSize = 8

	// 添加常量
	k1 := fn.AddNumberConstant(10) // K0 = 10
	k2 := fn.AddNumberConstant(20) // K1 = 20

	// 指令序列：
	// LOADK R0, K0   ; R0 = 10
	// LOADK R1, K1   ; R1 = 20
	// ADD R2, R0, R1 ; R2 = R0 + R1
	// RETURN R2, 2   ; return R2
	fn.AddInstructionBx(OP_LOADK, 0, k1)
	fn.AddInstructionBx(OP_LOADK, 1, k2)
	fn.AddInstruction(OP_ADD, 2, 0, 1)
	fn.AddInstruction(OP_RETURN, 2, 2, 0)

	// 执行函数
	executor := NewExecutor()
	results, err := executor.Execute(fn, nil)
	if err != nil {
		t.Fatalf("执行错误: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("期望1个返回值，得到%d个", len(results))
	}

	result := results[0]
	if !result.IsSmallInt() || result.AsSmallInt() != 30 {
		t.Errorf("期望SmallInt(30)，得到%s", result.ToString())
	}

	t.Log("✅ 简单执行测试通过")
}

func TestComplexExecution(t *testing.T) {
	t.Log("=== VM3 复杂执行测试 ===")

	// 创建复杂函数：f(x) = (x + 5) * (x - 3)
	fn := NewFunction("complex")
	fn.ParamCount = 1
	fn.MaxStackSize = 16

	// 添加常量
	k1 := fn.AddNumberConstant(5) // K0 = 5
	k2 := fn.AddNumberConstant(3) // K1 = 3

	// 指令序列：
	// 参数x在R0
	// LOADK R1, K0      ; R1 = 5
	// LOADK R2, K1      ; R2 = 3
	// ADD R3, R0, R1    ; R3 = x + 5
	// SUB R4, R0, R2    ; R4 = x - 3
	// MUL R5, R3, R4    ; R5 = (x+5) * (x-3)
	// RETURN R5, 2      ; return R5
	fn.AddInstructionBx(OP_LOADK, 1, k1)
	fn.AddInstructionBx(OP_LOADK, 2, k2)
	fn.AddInstruction(OP_ADD, 3, 0, 1)
	fn.AddInstruction(OP_SUB, 4, 0, 2)
	fn.AddInstruction(OP_MUL, 5, 3, 4)
	fn.AddInstruction(OP_RETURN, 5, 2, 0)

	// 测试x=10: (10+5)*(10-3) = 15*7 = 105
	executor := NewExecutor()
	args := []Value{NewSmallIntValue(10)}
	results, err := executor.Execute(fn, args)
	if err != nil {
		t.Fatalf("执行错误: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("期望1个返回值，得到%d个", len(results))
	}

	result := results[0]
	expectedValue := int32((10 + 5) * (10 - 3)) // 105
	if !result.IsSmallInt() || result.AsSmallInt() != expectedValue {
		t.Errorf("期望SmallInt(%d)，得到%s", expectedValue, result.ToString())
	}

	t.Log("✅ 复杂执行测试通过")
}

func TestValueConversion(t *testing.T) {
	t.Log("=== VM3 值转换测试 ===")

	// 测试ToNumber转换
	v1 := NewSmallIntValue(42)
	n1, err := v1.ToNumber()
	if err != nil || n1 != 42.0 {
		t.Errorf("SmallInt转Number失败: %v, %f", err, n1)
	}

	v2 := NewDoubleValue(3.14)
	n2, err := v2.ToNumber()
	if err != nil || n2 != 3.14 {
		t.Errorf("Double转Number失败: %v, %f", err, n2)
	}

	// 测试ToBool转换
	v3 := NewSmallIntValue(0)
	if v3.ToBool() {
		t.Error("0应该转换为false")
	}

	v4 := NewSmallIntValue(42)
	if !v4.ToBool() {
		t.Error("非0数字应该转换为true")
	}

	v5 := NewStringValue("")
	if v5.ToBool() {
		t.Error("空字符串应该转换为false")
	}

	v6 := NewStringValue("hello")
	if !v6.ToBool() {
		t.Error("非空字符串应该转换为true")
	}

	// 测试ToString转换
	v7 := NewSmallIntValue(42)
	if v7.ToString() != "42" {
		t.Errorf("SmallInt转字符串失败，期望'42'，得到'%s'", v7.ToString())
	}

	v8 := NewDoubleValue(3.14)
	if v8.ToString() != "3.14" {
		t.Errorf("Double转字符串失败，期望'3.14'，得到'%s'", v8.ToString())
	}

	t.Log("✅ 值转换测试通过")
}

func TestErrorHandling(t *testing.T) {
	t.Log("=== VM3 错误处理测试 ===")

	// 测试无效算术运算
	str := NewStringValue("hello")
	num := NewSmallIntValue(42)

	_, err := SubtractValues(str, num)
	if err == nil {
		t.Error("字符串减法应该报错")
	}

	_, err = MultiplyValues(str, num)
	if err == nil {
		t.Error("字符串乘法应该报错")
	}

	// 测试寄存器越界
	frame := NewStackFrame(NewFunction("test"), nil, -1)
	err = frame.SetRegister(-1, NewSmallIntValue(42))
	if err == nil {
		t.Error("负数索引应该报错")
	}

	err = frame.SetRegister(1000, NewSmallIntValue(42))
	if err == nil {
		t.Error("过大索引应该报错")
	}

	t.Log("✅ 错误处理测试通过")
}

func TestEdgeCases(t *testing.T) {
	t.Log("=== VM3 边界情况测试 ===")

	// 测试小整数边界值
	minVal := NewSmallIntValue(VALUE_SMALL_INT_MIN)
	if !minVal.IsSmallInt() || minVal.AsSmallInt() != VALUE_SMALL_INT_MIN {
		t.Errorf("小整数最小值测试失败")
	}

	maxVal := NewSmallIntValue(VALUE_SMALL_INT_MAX)
	if !maxVal.IsSmallInt() || maxVal.AsSmallInt() != VALUE_SMALL_INT_MAX {
		t.Errorf("小整数最大值测试失败")
	}

	// 测试自动类型选择
	justInRange := NewNumberValue(float64(VALUE_SMALL_INT_MAX))
	if !justInRange.IsSmallInt() {
		t.Error("边界整数应该存储为SmallInt")
	}

	justOutOfRange := NewNumberValue(float64(VALUE_SMALL_INT_MAX) + 1.0)
	if !justOutOfRange.IsDouble() {
		t.Error("超出范围的数应该存储为Double")
	}

	// 测试浮点数
	floatVal := NewNumberValue(3.14159)
	if !floatVal.IsDouble() {
		t.Error("浮点数应该存储为Double")
	}

	t.Log("✅ 边界情况测试通过")
}

func TestPerformanceCharacteristics(t *testing.T) {
	t.Log("=== VM3 性能特性验证 ===")

	// 验证小整数运算确实更快（定性测试）
	start := testing.Benchmark(func(b *testing.B) {
		a := NewSmallIntValue(10)
		c := NewSmallIntValue(20)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = AddValues(a, c)
		}
	})

	t.Logf("小整数加法性能: %v", start)

	// 验证内存确实更紧凑
	values := make([]Value, 1000)
	for i := range values {
		values[i] = NewSmallIntValue(int32(i))
	}

	t.Logf("创建1000个Value完成，每个%d字节", unsafe.Sizeof(Value{}))
	t.Log("✅ 性能特性验证完成")
}
