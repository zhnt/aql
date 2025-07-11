package vm

import (
	"fmt"
	"strconv"
	"unsafe"
)

// ValueType 优化的值类型枚举
type ValueType uint8

const (
	ValueTypeNil ValueType = iota
	ValueTypeSmallInt
	ValueTypeDouble
	ValueTypeString
	ValueTypeBool
	ValueTypeFunction
	ValueTypeArray
)

// Value类型常量和掩码
const (
	VALUE_TYPE_MASK     = 0x07        // 类型字段掩码
	VALUE_SMALL_INT_MIN = -1073741824 // -2^30，小整数最小值
	VALUE_SMALL_INT_MAX = 1073741823  // 2^30-1，小整数最大值
)

// Value 优化的16字节紧凑Value表示
//
// 内存布局：
// - typeAndFlags: 8字节，包含类型信息和标志位
// - data: 8字节，存储实际数据
//
// 性能优化：
// - 小整数(-2^30 到 2^30-1)直接内联存储，零heap分配
// - double使用位级复制避免类型断言
// - 快速类型检查和算术运算路径
type Value struct {
	typeAndFlags uint64
	data         uint64
}

// String 返回ValueType的字符串表示
func (vt ValueType) String() string {
	switch vt {
	case ValueTypeNil:
		return "nil"
	case ValueTypeSmallInt:
		return "smallint"
	case ValueTypeDouble:
		return "double"
	case ValueTypeString:
		return "string"
	case ValueTypeBool:
		return "bool"
	case ValueTypeFunction:
		return "function"
	case ValueTypeArray:
		return "array"
	default:
		return "unknown"
	}
}

// 构造函数

func NewNilValue() Value {
	return Value{typeAndFlags: uint64(ValueTypeNil), data: 0}
}

func NewNumberValue(n float64) Value {
	// 尝试作为小整数存储
	if n == float64(int64(n)) && n >= VALUE_SMALL_INT_MIN && n <= VALUE_SMALL_INT_MAX {
		return NewSmallIntValue(int32(n))
	}
	// 否则存储为double
	return NewDoubleValue(n)
}

func NewSmallIntValue(i int32) Value {
	return Value{
		typeAndFlags: uint64(ValueTypeSmallInt),
		data:         uint64(uint32(i)), // 使用无符号转换处理负数
	}
}

func NewDoubleValue(f float64) Value {
	return Value{
		typeAndFlags: uint64(ValueTypeDouble),
		data:         *(*uint64)(unsafe.Pointer(&f)),
	}
}

func NewStringValue(s string) Value {
	return Value{
		typeAndFlags: uint64(ValueTypeString),
		data:         uint64(uintptr(unsafe.Pointer(&s))),
	}
}

func NewBoolValue(b bool) Value {
	data := uint64(0) // 明确初始化，提高代码可读性
	if b {
		data = 1
	}
	return Value{
		typeAndFlags: uint64(ValueTypeBool),
		data:         data,
	}
}

func NewFunctionValue(f *Function) Value {
	return Value{
		typeAndFlags: uint64(ValueTypeFunction),
		data:         uint64(uintptr(unsafe.Pointer(f))),
	}
}

// NewArrayValue 创建数组Value (使用新的ArrayManager)
func NewArrayValue(elements []Value) Value {
	id := GlobalArrayManager.CreateArray(elements)
	return Value{
		typeAndFlags: uint64(ValueTypeArray),
		data:         id, // 直接存储64位ID
	}
}

// NewEmptyArrayValue 创建空数组Value
func NewEmptyArrayValue() Value {
	return NewArrayValue([]Value{})
}

// 类型检查方法

func (v Value) Type() ValueType {
	return ValueType(v.typeAndFlags & VALUE_TYPE_MASK)
}

func (v Value) IsNil() bool {
	return v.Type() == ValueTypeNil
}

func (v Value) IsSmallInt() bool {
	return v.Type() == ValueTypeSmallInt
}

func (v Value) IsDouble() bool {
	return v.Type() == ValueTypeDouble
}

func (v Value) IsNumber() bool {
	typ := v.Type()
	return typ == ValueTypeSmallInt || typ == ValueTypeDouble
}

func (v Value) IsString() bool {
	return v.Type() == ValueTypeString
}

func (v Value) IsBool() bool {
	return v.Type() == ValueTypeBool
}

func (v Value) IsFunction() bool {
	return v.Type() == ValueTypeFunction
}

func (v Value) IsArray() bool {
	return v.Type() == ValueTypeArray
}

// 数据访问方法（高性能）

func (v Value) AsSmallInt() int32 {
	return int32(uint32(v.data))
}

func (v Value) AsDouble() float64 {
	return *(*float64)(unsafe.Pointer(&v.data))
}

func (v Value) AsString() string {
	return *(*string)(unsafe.Pointer(uintptr(v.data)))
}

func (v Value) AsBool() bool {
	return v.data != 0
}

func (v Value) AsFunction() *Function {
	return (*Function)(unsafe.Pointer(uintptr(v.data)))
}

// AsArray方法现在定义在array.go中

// 类型转换方法

func (v Value) ToNumber() (float64, error) {
	switch v.Type() {
	case ValueTypeSmallInt:
		return float64(v.AsSmallInt()), nil
	case ValueTypeDouble:
		return v.AsDouble(), nil
	case ValueTypeString:
		s := v.AsString()
		return strconv.ParseFloat(s, 64)
	default:
		return 0, fmt.Errorf("cannot convert %s to number", v.Type())
	}
}

func (v Value) ToBool() bool {
	switch v.Type() {
	case ValueTypeNil:
		return false
	case ValueTypeBool:
		return v.AsBool()
	case ValueTypeSmallInt:
		return v.AsSmallInt() != 0
	case ValueTypeDouble:
		return v.AsDouble() != 0.0
	case ValueTypeString:
		return v.AsString() != ""
	default:
		return true // 函数等其他类型被认为是真值
	}
}

func (v Value) ToString() string {
	switch v.Type() {
	case ValueTypeNil:
		return "nil"
	case ValueTypeSmallInt:
		return strconv.Itoa(int(v.AsSmallInt()))
	case ValueTypeDouble:
		return strconv.FormatFloat(v.AsDouble(), 'g', -1, 64)
	case ValueTypeString:
		return v.AsString()
	case ValueTypeBool:
		if v.AsBool() {
			return "true"
		}
		return "false"
	case ValueTypeFunction:
		return fmt.Sprintf("function:%p", v.AsFunction())
	case ValueTypeArray:
		array, err := v.AsArray()
		if err != nil {
			return "invalid_array"
		}
		result := "["
		for i, elem := range array.Elements {
			if i > 0 {
				result += ", "
			}
			result += elem.ToString()
		}
		result += "]"
		return result
	default:
		return fmt.Sprintf("unknown:%d", v.Type())
	}
}

func (v Value) String() string {
	return v.ToString()
}

// 优化的算术运算

func AddValues(a, b Value) (Value, error) {
	// 快速路径：小整数加法（最常见的情况）
	if a.IsSmallInt() && b.IsSmallInt() {
		aInt := a.AsSmallInt()
		bInt := b.AsSmallInt()

		// 使用64位算术检查溢出
		result64 := int64(aInt) + int64(bInt)

		// 如果结果在小整数范围内，直接返回小整数
		if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
			return NewSmallIntValue(int32(result64)), nil
		}

		// 溢出时转为double
		return NewDoubleValue(float64(aInt) + float64(bInt)), nil
	}

	// 字符串连接
	if a.IsString() || b.IsString() {
		return NewStringValue(a.ToString() + b.ToString()), nil
	}

	// 通用数值加法
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValue(), fmt.Errorf("invalid number operands")
		}
		return NewNumberValue(aNum + bNum), nil
	}

	return NewNilValue(), fmt.Errorf("cannot add %s and %s", a.Type(), b.Type())
}

func SubtractValues(a, b Value) (Value, error) {
	// 快速路径：小整数减法
	if a.IsSmallInt() && b.IsSmallInt() {
		aInt := a.AsSmallInt()
		bInt := b.AsSmallInt()

		// 检查溢出
		result64 := int64(aInt) - int64(bInt)
		if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
			return NewSmallIntValue(int32(result64)), nil
		}

		// 溢出时转为double
		return NewDoubleValue(float64(aInt) - float64(bInt)), nil
	}

	// 通用数值减法
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValue(), fmt.Errorf("invalid number operands")
		}
		return NewNumberValue(aNum - bNum), nil
	}

	return NewNilValue(), fmt.Errorf("cannot subtract %s and %s", a.Type(), b.Type())
}

func MultiplyValues(a, b Value) (Value, error) {
	// 快速路径：小整数乘法
	if a.IsSmallInt() && b.IsSmallInt() {
		aInt := a.AsSmallInt()
		bInt := b.AsSmallInt()

		// 检查溢出（简化检查）
		result64 := int64(aInt) * int64(bInt)
		if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
			return NewSmallIntValue(int32(result64)), nil
		}

		// 溢出时转为double
		return NewDoubleValue(float64(aInt) * float64(bInt)), nil
	}

	// 通用数值乘法
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValue(), fmt.Errorf("invalid number operands")
		}
		return NewNumberValue(aNum * bNum), nil
	}

	return NewNilValue(), fmt.Errorf("cannot multiply %s and %s", a.Type(), b.Type())
}

// 特殊优化：批量运算
// 对于连续的小整数运算，提供专门的批量接口

func AddSmallInts(a, b int32) (Value, bool) {
	result64 := int64(a) + int64(b)
	if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
		return NewSmallIntValue(int32(result64)), true
	}
	return NewDoubleValue(float64(a) + float64(b)), true
}

func SubtractSmallInts(a, b int32) (Value, bool) {
	result64 := int64(a) - int64(b)
	if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
		return NewSmallIntValue(int32(result64)), true
	}
	return NewDoubleValue(float64(a) - float64(b)), true
}

func MultiplySmallInts(a, b int32) (Value, bool) {
	result64 := int64(a) * int64(b)
	if result64 >= VALUE_SMALL_INT_MIN && result64 <= VALUE_SMALL_INT_MAX {
		return NewSmallIntValue(int32(result64)), true
	}
	return NewDoubleValue(float64(a) * float64(b)), true
}

// 比较运算方法

// Equal 判断两个值是否相等
func (v Value) Equal(other Value) bool {
	// 类型不同直接返回false
	if v.Type() != other.Type() {
		return false
	}

	switch v.Type() {
	case ValueTypeNil:
		return true
	case ValueTypeSmallInt:
		return v.AsSmallInt() == other.AsSmallInt()
	case ValueTypeDouble:
		return v.AsDouble() == other.AsDouble()
	case ValueTypeString:
		return v.AsString() == other.AsString()
	case ValueTypeBool:
		return v.AsBool() == other.AsBool()
	case ValueTypeFunction:
		return v.AsFunction() == other.AsFunction()
	case ValueTypeArray:
		// 数组比较：比较长度和每个元素
		arr1, err1 := v.AsArray()
		arr2, err2 := other.AsArray()
		if err1 != nil || err2 != nil {
			return false
		}
		if arr1.Length != arr2.Length {
			return false
		}
		for i := 0; i < arr1.Length; i++ {
			if !arr1.Elements[i].Equal(arr2.Elements[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// IsTruthy 判断值是否为真值
func (v Value) IsTruthy() bool {
	return v.ToBool()
}

// LessThan 比较两个值的大小关系 (<)
func LessThan(a, b Value) (Value, error) {
	// 数值比较
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValue(), fmt.Errorf("invalid number operands")
		}
		return NewBoolValue(aNum < bNum), nil
	}

	// 字符串比较
	if a.IsString() && b.IsString() {
		return NewBoolValue(a.AsString() < b.AsString()), nil
	}

	return NewNilValue(), fmt.Errorf("cannot compare %s and %s", a.Type(), b.Type())
}

// GreaterThan 比较两个值的大小关系 (>)
func GreaterThan(a, b Value) (Value, error) {
	// 数值比较
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValue(), fmt.Errorf("invalid number operands")
		}
		return NewBoolValue(aNum > bNum), nil
	}

	// 字符串比较
	if a.IsString() && b.IsString() {
		return NewBoolValue(a.AsString() > b.AsString()), nil
	}

	return NewNilValue(), fmt.Errorf("cannot compare %s and %s", a.Type(), b.Type())
}

// NegateValue 对值取负
func NegateValue(v Value) (Value, error) {
	switch v.Type() {
	case ValueTypeSmallInt:
		i := v.AsSmallInt()
		// 检查溢出
		if i == VALUE_SMALL_INT_MIN {
			return NewDoubleValue(-float64(i)), nil
		}
		return NewSmallIntValue(-i), nil
	case ValueTypeDouble:
		return NewDoubleValue(-v.AsDouble()), nil
	default:
		return NewNilValue(), fmt.Errorf("cannot negate %s", v.Type())
	}
}

// 数组操作方法现在定义在array.go中
// ArrayGetValue, ArraySetValue, ArrayLengthValue 等函数
// 已经在array.go中重新实现以使用新的ArrayManager系统
