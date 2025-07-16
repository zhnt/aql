package vm

import (
	"fmt"
	"unsafe"

	"github.com/zhnt/aql/internal/gc"
)

// =============================================================================
// ValueGC 算术运算（GC 安全）
// =============================================================================

// AddValuesGC 加法运算（GC 安全）
func AddValuesGC(a, b ValueGC) (ValueGC, error) {
	// 快速路径：小整数加法（最常见的情况）
	if a.IsSmallInt() && b.IsSmallInt() {
		aInt := a.AsSmallInt()
		bInt := b.AsSmallInt()

		// 使用64位算术检查溢出
		result64 := int64(aInt) + int64(bInt)

		// 如果结果在小整数范围内，直接返回小整数
		if result64 >= VALUE_GC_SMALL_INT_MIN && result64 <= VALUE_GC_SMALL_INT_MAX {
			return NewSmallIntValueGC(int32(result64)), nil
		}

		// 溢出时转为double
		return NewDoubleValueGC(float64(aInt) + float64(bInt)), nil
	}

	// 字符串连接
	if a.IsString() || b.IsString() {
		resultStr := a.ToString() + b.ToString()
		return NewStringValueGC(resultStr), nil
	}

	// 通用数值加法
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValueGC(), fmt.Errorf("invalid number operands")
		}
		return NewNumberValueGC(aNum + bNum), nil
	}

	return NewNilValueGC(), fmt.Errorf("cannot add %s and %s", a.Type(), b.Type())
}

// SubtractValuesGC 减法运算（GC 安全）
func SubtractValuesGC(a, b ValueGC) (ValueGC, error) {
	// 快速路径：小整数减法
	if a.IsSmallInt() && b.IsSmallInt() {
		aInt := a.AsSmallInt()
		bInt := b.AsSmallInt()

		// 检查溢出
		result64 := int64(aInt) - int64(bInt)
		if result64 >= VALUE_GC_SMALL_INT_MIN && result64 <= VALUE_GC_SMALL_INT_MAX {
			return NewSmallIntValueGC(int32(result64)), nil
		}

		// 溢出时转为double
		return NewDoubleValueGC(float64(aInt) - float64(bInt)), nil
	}

	// 通用数值减法
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValueGC(), fmt.Errorf("invalid number operands")
		}
		return NewNumberValueGC(aNum - bNum), nil
	}

	return NewNilValueGC(), fmt.Errorf("cannot subtract %s and %s", a.Type(), b.Type())
}

// MultiplyValuesGC 乘法运算（GC 安全）
func MultiplyValuesGC(a, b ValueGC) (ValueGC, error) {
	// 快速路径：小整数乘法
	if a.IsSmallInt() && b.IsSmallInt() {
		aInt := a.AsSmallInt()
		bInt := b.AsSmallInt()

		// 检查溢出（简化检查）
		result64 := int64(aInt) * int64(bInt)
		if result64 >= VALUE_GC_SMALL_INT_MIN && result64 <= VALUE_GC_SMALL_INT_MAX {
			return NewSmallIntValueGC(int32(result64)), nil
		}

		// 溢出时转为double
		return NewDoubleValueGC(float64(aInt) * float64(bInt)), nil
	}

	// 通用数值乘法
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValueGC(), fmt.Errorf("invalid number operands")
		}
		return NewNumberValueGC(aNum * bNum), nil
	}

	return NewNilValueGC(), fmt.Errorf("cannot multiply %s and %s", a.Type(), b.Type())
}

// DivideValuesGC 除法运算（GC 安全）
func DivideValuesGC(a, b ValueGC) (ValueGC, error) {
	// 通用数值除法
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValueGC(), fmt.Errorf("invalid number operands")
		}
		if bNum == 0 {
			return NewNilValueGC(), fmt.Errorf("division by zero")
		}
		return NewNumberValueGC(aNum / bNum), nil
	}

	return NewNilValueGC(), fmt.Errorf("cannot divide %s and %s", a.Type(), b.Type())
}

// ModuloValuesGC 取模运算（GC 安全）
func ModuloValuesGC(a, b ValueGC) (ValueGC, error) {
	// 仅支持整数取模
	if a.IsSmallInt() && b.IsSmallInt() {
		aInt := a.AsSmallInt()
		bInt := b.AsSmallInt()
		if bInt == 0 {
			return NewNilValueGC(), fmt.Errorf("modulo by zero")
		}
		return NewSmallIntValueGC(aInt % bInt), nil
	}

	return NewNilValueGC(), fmt.Errorf("modulo operation requires integers")
}

// NegateValueGC 取负运算（GC 安全）
func NegateValueGC(v ValueGC) (ValueGC, error) {
	switch v.Type() {
	case ValueGCTypeSmallInt:
		i := v.AsSmallInt()
		// 检查溢出
		if i == VALUE_GC_SMALL_INT_MIN {
			return NewDoubleValueGC(-float64(i)), nil
		}
		return NewSmallIntValueGC(-i), nil
	case ValueGCTypeDouble:
		return NewDoubleValueGC(-v.AsDouble()), nil
	default:
		return NewNilValueGC(), fmt.Errorf("cannot negate %s", v.Type())
	}
}

// =============================================================================
// ValueGC 比较运算（GC 安全）
// =============================================================================

// EqualValuesGC 相等比较（GC 安全）
func EqualValuesGC(a, b ValueGC) (ValueGC, error) {
	result := a.Equal(b)
	return NewBoolValueGC(result), nil
}

// Equal 判断两个值是否相等
func (v ValueGC) Equal(other ValueGC) bool {
	// 类型不同直接返回false
	if v.Type() != other.Type() {
		return false
	}

	switch v.Type() {
	case ValueGCTypeNil:
		return true
	case ValueGCTypeSmallInt:
		return v.AsSmallInt() == other.AsSmallInt()
	case ValueGCTypeDouble:
		return v.AsDouble() == other.AsDouble()
	case ValueGCTypeString:
		return v.AsString() == other.AsString()
	case ValueGCTypeBool:
		return v.AsBool() == other.AsBool()
	case ValueGCTypeFunction:
		// 函数引用相等比较
		return v.data == other.data
	case ValueGCTypeArray:
		return v.equalArray(other)
	default:
		return false
	}
}

// equalArray 数组相等比较（深度比较）
func (v ValueGC) equalArray(other ValueGC) bool {
	arrData1, elements1, err1 := v.AsArrayData()
	arrData2, elements2, err2 := other.AsArrayData()
	if err1 != nil || err2 != nil {
		return false
	}

	// 引用相等
	if v.data == other.data {
		return true
	}

	// 长度不同
	if arrData1.Length != arrData2.Length {
		return false
	}

	// 逐元素比较
	for i := uint32(0); i < arrData1.Length; i++ {
		if !elements1[i].Equal(elements2[i]) {
			return false
		}
	}

	return true
}

// LessThanValuesGC 小于比较（GC 安全）
func LessThanValuesGC(a, b ValueGC) (ValueGC, error) {
	// 数值比较
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValueGC(), fmt.Errorf("invalid number operands")
		}
		return NewBoolValueGC(aNum < bNum), nil
	}

	// 字符串比较
	if a.IsString() && b.IsString() {
		return NewBoolValueGC(a.AsString() < b.AsString()), nil
	}

	return NewNilValueGC(), fmt.Errorf("cannot compare %s and %s", a.Type(), b.Type())
}

// LessEqualValuesGC 小于等于比较（GC 安全）
func LessEqualValuesGC(a, b ValueGC) (ValueGC, error) {
	// 数值比较
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValueGC(), fmt.Errorf("invalid number operands")
		}
		return NewBoolValueGC(aNum <= bNum), nil
	}

	// 字符串比较
	if a.IsString() && b.IsString() {
		return NewBoolValueGC(a.AsString() <= b.AsString()), nil
	}

	return NewNilValueGC(), fmt.Errorf("cannot compare %s and %s", a.Type(), b.Type())
}

// GreaterThanValuesGC 大于比较（GC 安全）
func GreaterThanValuesGC(a, b ValueGC) (ValueGC, error) {
	// 数值比较
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValueGC(), fmt.Errorf("invalid number operands")
		}
		return NewBoolValueGC(aNum > bNum), nil
	}

	// 字符串比较
	if a.IsString() && b.IsString() {
		return NewBoolValueGC(a.AsString() > b.AsString()), nil
	}

	return NewNilValueGC(), fmt.Errorf("cannot compare %s and %s", a.Type(), b.Type())
}

// GreaterEqualValuesGC 大于等于比较（GC 安全）
func GreaterEqualValuesGC(a, b ValueGC) (ValueGC, error) {
	// 数值比较
	if a.IsNumber() && b.IsNumber() {
		aNum, aErr := a.ToNumber()
		bNum, bErr := b.ToNumber()
		if aErr != nil || bErr != nil {
			return NewNilValueGC(), fmt.Errorf("invalid number operands")
		}
		return NewBoolValueGC(aNum >= bNum), nil
	}

	// 字符串比较
	if a.IsString() && b.IsString() {
		return NewBoolValueGC(a.AsString() >= b.AsString()), nil
	}

	return NewNilValueGC(), fmt.Errorf("cannot compare %s and %s", a.Type(), b.Type())
}

// NotEqualValuesGC 不等比较（GC 安全）
func NotEqualValuesGC(a, b ValueGC) (ValueGC, error) {
	result := !a.Equal(b)
	return NewBoolValueGC(result), nil
}

// =============================================================================
// ValueGC 逻辑运算（GC 安全）
// =============================================================================

// LogicalAndValuesGC 逻辑与运算（GC 安全）
func LogicalAndValuesGC(a, b ValueGC) (ValueGC, error) {
	if !a.ToBool() {
		return CopyValueGC(a), nil // 返回第一个假值
	}
	return CopyValueGC(b), nil // 返回第二个值
}

// LogicalOrValuesGC 逻辑或运算（GC 安全）
func LogicalOrValuesGC(a, b ValueGC) (ValueGC, error) {
	if a.ToBool() {
		return CopyValueGC(a), nil // 返回第一个真值
	}
	return CopyValueGC(b), nil // 返回第二个值
}

// LogicalNotValueGC 逻辑非运算（GC 安全）
func LogicalNotValueGC(v ValueGC) (ValueGC, error) {
	return NewBoolValueGC(!v.ToBool()), nil
}

// =============================================================================
// ValueGC 数组操作（GC 安全）
// =============================================================================

// ArrayGetValueGC 获取数组元素（GC 安全，支持多维数组）- 动态访问
func ArrayGetValueGC(arrayValue ValueGC, index int) (ValueGC, error) {
	// 类型验证
	if !arrayValue.IsArray() {
		return NewNilValueGC(), fmt.Errorf("not an array: got %s", arrayValue.Type())
	}

	arrData, err := getArrayData(arrayValue)
	if err != nil {
		return NewNilValueGC(), fmt.Errorf("failed to get array data: %v", err)
	}

	// 边界检查
	if index < 0 || index >= int(arrData.Length) {
		return NewNilValueGC(), fmt.Errorf("array index out of bounds: %d (length: %d)", index, arrData.Length)
	}

	fmt.Printf("DEBUG [ArrayGetValueGC] 数组长度: %d, 索引: %d\n", arrData.Length, index)

	// 获取元素指针
	elemPtr := getElementPtr(arrData, index)
	if elemPtr == nil {
		return NewNilValueGC(), fmt.Errorf("failed to get element pointer for index %d", index)
	}

	element := *elemPtr
	fmt.Printf("DEBUG [ArrayGetValueGC] 元素类型: %s\n", element.Type())

	// 类型特定的调试信息
	switch element.Type() {
	case ValueGCTypeSmallInt:
		fmt.Printf("DEBUG [ArrayGetValueGC] 小整数值: %d\n", element.AsSmallInt())
	case ValueGCTypeDouble:
		fmt.Printf("DEBUG [ArrayGetValueGC] 双精度值: %f\n", element.AsDouble())
	case ValueGCTypeString:
		fmt.Printf("DEBUG [ArrayGetValueGC] 字符串值: %s\n", element.AsString())
	case ValueGCTypeArray:
		if elemArrData, elemErr := getArrayData(element); elemErr == nil {
			fmt.Printf("DEBUG [ArrayGetValueGC] 嵌套数组长度: %d\n", elemArrData.Length)
		}
	case ValueGCTypeNil:
		fmt.Printf("DEBUG [ArrayGetValueGC] nil值\n")
	}

	// 使用SafeCopyValueGC进行安全拷贝，避免循环引用
	result := SafeCopyValueGC(element)
	fmt.Printf("DEBUG [ArrayGetValueGC] 安全拷贝后类型: %s\n", result.Type())

	return result, nil
}

// ArraySetValueGCWithExpansion 支持扩容的数组设置方法
func ArraySetValueGCWithExpansion(arrayValue ValueGC, index int, value ValueGC) (ValueGC, error) {
	if index < 0 {
		return NewNilValueGC(), fmt.Errorf("negative index: %d", index)
	}

	arrData, err := getArrayData(arrayValue)
	if err != nil {
		return NewNilValueGC(), err
	}

	fmt.Printf("DEBUG [ArraySetValueGCWithExpansion] 设置元素: index=%d, 当前容量=%d\n", index, arrData.Capacity)

	// 检查是否需要扩容
	if index >= int(arrData.Capacity) {
		fmt.Printf("DEBUG [ArraySetValueGCWithExpansion] 需要扩容: index=%d >= capacity=%d\n", index, arrData.Capacity)

		// 扩容并返回新的ValueGC
		newArrayValue, err := expandArrayForIndex(arrayValue, index)
		if err != nil {
			return NewNilValueGC(), err
		}

		// 在新数组上设置值
		err = ArraySetValueGC(newArrayValue, index, value)
		if err != nil {
			return NewNilValueGC(), err
		}

		return newArrayValue, nil
	}

	// 容量足够，直接设置
	err = ArraySetValueGC(arrayValue, index, value)
	return arrayValue, err
}

// ArraySetValueGC 设置数组元素（容量内操作）
func ArraySetValueGC(arrayValue ValueGC, index int, value ValueGC) error {
	if index < 0 {
		return fmt.Errorf("negative index: %d", index)
	}

	arrData, err := getArrayData(arrayValue)
	if err != nil {
		return err
	}

	// 检查容量
	if index >= int(arrData.Capacity) {
		return fmt.Errorf("index %d exceeds capacity %d, use ArraySetValueGCWithExpansion",
			index, arrData.Capacity)
	}

	// 扩展长度（如果需要）
	if index >= int(arrData.Length) {
		// 填充中间元素为nil
		nilValue := NewNilValueGC()
		for i := int(arrData.Length); i < index; i++ {
			elemPtr := getElementPtr(arrData, i)
			if elemPtr != nil {
				*elemPtr = nilValue
			}
		}
		arrData.Length = uint32(index + 1)
	}

	// 设置元素
	elemPtr := getElementPtr(arrData, index)
	if elemPtr == nil {
		return fmt.Errorf("invalid element pointer for index %d", index)
	}

	oldValue := *elemPtr

	// 管理引用计数
	if oldValue.RequiresGC() {
		oldValue.DecRef()
	}

	*elemPtr = SafeCopyValueGC(value)
	fmt.Printf("DEBUG [ArraySetValueGC] 设置元素[%d]成功, 类型=%s\n", index, value.Type())
	return nil
}

// ArrayLengthValueGC 获取数组长度（GC 安全）
func ArrayLengthValueGC(arrayValue ValueGC) (ValueGC, error) {
	arrData, _, err := arrayValue.AsArrayData()
	if err != nil {
		return NewNilValueGC(), err
	}

	return NewSmallIntValueGC(int32(arrData.Length)), nil
}

// ArrayAppendValueGC 向数组追加元素（GC 安全）
func ArrayAppendValueGC(arrayValue ValueGC, value ValueGC) error {
	arrData, elements, err := arrayValue.AsArrayData()
	if err != nil {
		return err
	}

	// 检查容量
	if arrData.Length >= arrData.Capacity {
		// 需要扩容
		newCapacity := arrData.Capacity * 2
		if newCapacity == 0 {
			newCapacity = 4
		}

		newElements := make([]ValueGC, arrData.Length, newCapacity)
		copy(newElements, elements)
		elements = newElements
		arrData.Capacity = newCapacity
	}

	// 追加元素（安全拷贝）
	elements = append(elements, CopyValueGC(value))
	arrData.Length++

	return nil
}

// ArrayConcatValuesGC 连接两个数组（GC 安全）
func ArrayConcatValuesGC(a, b ValueGC) (ValueGC, error) {
	arrData1, elements1, err1 := a.AsArrayData()
	arrData2, elements2, err2 := b.AsArrayData()
	if err1 != nil || err2 != nil {
		return NewNilValueGC(), fmt.Errorf("both operands must be arrays")
	}

	// 创建新数组
	newElements := make([]ValueGC, 0, arrData1.Length+arrData2.Length)

	// 拷贝第一个数组的元素
	for i := uint32(0); i < arrData1.Length; i++ {
		newElements = append(newElements, CopyValueGC(elements1[i]))
	}

	// 拷贝第二个数组的元素
	for i := uint32(0); i < arrData2.Length; i++ {
		newElements = append(newElements, CopyValueGC(elements2[i]))
	}

	return NewArrayValueGC(newElements), nil
}

// =============================================================================
// 矩阵操作辅助函数
// =============================================================================

// NewMatrixValueGC 创建二维矩阵（GC安全）
func NewMatrixValueGC(rows, cols int) (ValueGC, error) {
	if rows <= 0 || cols <= 0 {
		return NewNilValueGC(), fmt.Errorf("matrix dimensions must be positive: rows=%d, cols=%d", rows, cols)
	}

	fmt.Printf("DEBUG [NewMatrixValueGC] 创建矩阵: %dx%d\n", rows, cols)

	// 创建矩阵的行数组
	matrix := make([]ValueGC, rows)

	// 为每一行创建列数组
	for i := 0; i < rows; i++ {
		rowElements := make([]ValueGC, cols)
		nilValue := NewNilValueGC()

		// 初始化行中的所有元素为nil
		for j := 0; j < cols; j++ {
			rowElements[j] = nilValue
		}

		// 创建行数组
		matrix[i] = NewArrayValueGC(rowElements)
		fmt.Printf("DEBUG [NewMatrixValueGC] 创建行 %d: 长度=%d\n", i, cols)
	}

	// 创建矩阵（二维数组）
	result := NewArrayValueGC(matrix)
	fmt.Printf("DEBUG [NewMatrixValueGC] 矩阵创建完成: %dx%d\n", rows, cols)

	return result, nil
}

// GetMatrixElementValueGC 获取矩阵元素 matrix[row][col]（GC安全）
func GetMatrixElementValueGC(matrix ValueGC, row, col int) (ValueGC, error) {
	fmt.Printf("DEBUG [GetMatrixElementValueGC] 获取矩阵元素: [%d][%d]\n", row, col)

	// 检查矩阵类型
	if !matrix.IsArray() {
		return NewNilValueGC(), fmt.Errorf("not a matrix (not an array): got %s", matrix.Type())
	}

	// 获取行
	rowArray, err := ArrayGetValueGC(matrix, row)
	if err != nil {
		return NewNilValueGC(), fmt.Errorf("failed to get row %d: %v", row, err)
	}

	// 检查行是否为数组
	if !rowArray.IsArray() {
		return NewNilValueGC(), fmt.Errorf("row %d is not an array: got %s", row, rowArray.Type())
	}

	// 获取列元素
	element, err := ArrayGetValueGC(rowArray, col)
	if err != nil {
		return NewNilValueGC(), fmt.Errorf("failed to get column %d from row %d: %v", col, row, err)
	}

	fmt.Printf("DEBUG [GetMatrixElementValueGC] 获取元素成功: [%d][%d] = %s\n", row, col, element.Type())
	return element, nil
}

// SetMatrixElementValueGC 设置矩阵元素 matrix[row][col] = value（GC安全）
func SetMatrixElementValueGC(matrix ValueGC, row, col int, value ValueGC) error {
	fmt.Printf("DEBUG [SetMatrixElementValueGC] 设置矩阵元素: [%d][%d] = %s\n", row, col, value.Type())

	// 检查矩阵类型
	if !matrix.IsArray() {
		return fmt.Errorf("not a matrix (not an array): got %s", matrix.Type())
	}

	// 直接访问矩阵数据和元素，不进行拷贝
	matrixData, matrixElements, err := matrix.AsArrayData()
	if err != nil {
		return fmt.Errorf("failed to get matrix data: %v", err)
	}

	// 边界检查行索引
	if row < 0 || row >= int(matrixData.Length) {
		return fmt.Errorf("row index out of bounds: %d (matrix rows: %d)", row, matrixData.Length)
	}

	// 获取行的原始引用（不拷贝）
	rowArray := matrixElements[row]
	fmt.Printf("DEBUG [SetMatrixElementValueGC] 获取行[%d]: 类型=%s\n", row, rowArray.Type())

	// 检查行是否为数组
	if !rowArray.IsArray() {
		return fmt.Errorf("row %d is not an array: got %s", row, rowArray.Type())
	}

	// 设置列元素
	err = ArraySetValueGC(rowArray, col, value)
	if err != nil {
		return fmt.Errorf("failed to set column %d in row %d: %v", col, row, err)
	}

	fmt.Printf("DEBUG [SetMatrixElementValueGC] 设置元素成功: [%d][%d] = %s\n", row, col, value.Type())
	return nil
}

// GetMatrixDimensionsValueGC 获取矩阵维度 (rows, cols)（GC安全）
func GetMatrixDimensionsValueGC(matrix ValueGC) (int, int, error) {
	if !matrix.IsArray() {
		return 0, 0, fmt.Errorf("not a matrix (not an array): got %s", matrix.Type())
	}

	// 获取行数
	matrixData, elements, err := matrix.AsArrayData()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get matrix data: %v", err)
	}

	rows := int(matrixData.Length)
	if rows == 0 {
		return 0, 0, nil
	}

	// 获取列数（从第一行）
	firstRow := elements[0]
	if !firstRow.IsArray() {
		return 0, 0, fmt.Errorf("first row is not an array: got %s", firstRow.Type())
	}

	firstRowData, _, err := firstRow.AsArrayData()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get first row data: %v", err)
	}

	cols := int(firstRowData.Length)
	fmt.Printf("DEBUG [GetMatrixDimensionsValueGC] 矩阵维度: %dx%d\n", rows, cols)

	return rows, cols, nil
}

// IsValidMatrixValueGC 检查是否为有效矩阵（所有行长度相同）
func IsValidMatrixValueGC(matrix ValueGC) bool {
	if !matrix.IsArray() {
		return false
	}

	matrixData, elements, err := matrix.AsArrayData()
	if err != nil {
		return false
	}

	if matrixData.Length == 0 {
		return true // 空矩阵是有效的
	}

	// 检查第一行以确定列数
	firstRow := elements[0]
	if !firstRow.IsArray() {
		return false
	}

	firstRowData, _, err := firstRow.AsArrayData()
	if err != nil {
		return false
	}

	expectedCols := firstRowData.Length

	// 检查所有行的长度是否相同
	for i := uint32(0); i < matrixData.Length; i++ {
		row := elements[i]
		if !row.IsArray() {
			return false
		}

		rowData, _, err := row.AsArrayData()
		if err != nil {
			return false
		}

		if rowData.Length != expectedCols {
			return false
		}
	}

	return true
}

// FillMatrixValueGC 用指定值填充矩阵（GC安全）
func FillMatrixValueGC(matrix ValueGC, value ValueGC) error {
	if !matrix.IsArray() {
		return fmt.Errorf("not a matrix (not an array): got %s", matrix.Type())
	}

	rows, cols, err := GetMatrixDimensionsValueGC(matrix)
	if err != nil {
		return fmt.Errorf("failed to get matrix dimensions: %v", err)
	}

	fmt.Printf("DEBUG [FillMatrixValueGC] 填充矩阵: %dx%d, 值类型=%s\n", rows, cols, value.Type())

	// 填充每个元素
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			err = SetMatrixElementValueGC(matrix, i, j, value)
			if err != nil {
				return fmt.Errorf("failed to fill element [%d][%d]: %v", i, j, err)
			}
		}
	}

	fmt.Printf("DEBUG [FillMatrixValueGC] 矩阵填充完成\n")
	return nil
}

// =============================================================================
// ValueGC 实用工具方法
// =============================================================================

// IsTruthyValueGC 判断值是否为真值
func IsTruthyValueGC(v ValueGC) bool {
	return v.ToBool()
}

// IsNilValueGC 判断值是否为 nil
func IsNilValueGC(v ValueGC) bool {
	return v.IsNil()
}

// CloneValueGC 深度克隆值（GC 安全）
func CloneValueGC(v ValueGC) ValueGC {
	switch v.Type() {
	case ValueGCTypeArray:
		// 深度克隆数组
		if arrData, elements, err := v.AsArrayData(); err == nil {
			newElements := make([]ValueGC, arrData.Length)
			for i := uint32(0); i < arrData.Length; i++ {
				newElements[i] = CloneValueGC(elements[i])
			}
			return NewArrayValueGC(newElements)
		}
		return NewNilValueGC()
	case ValueGCTypeString, ValueGCTypeFunction:
		// 字符串和函数的引用拷贝
		return CopyValueGC(v)
	default:
		// 基础类型直接拷贝
		return v
	}
}

// =============================================================================
// 数组扩容辅助函数
// =============================================================================

// expandArrayForIndex 扩容数组以支持指定索引的访问
// 返回新的 ValueGC，调用者需要更新引用
func expandArrayForIndex(arrayValue ValueGC, targetIndex int) (ValueGC, error) {
	fmt.Printf("DEBUG [expandArrayForIndex] 开始扩容数组: targetIndex=%d\n", targetIndex)

	// 获取原数组数据
	oldArrData, err := getArrayData(arrayValue)
	if err != nil {
		return NewNilValueGC(), fmt.Errorf("failed to get array data: %v", err)
	}

	fmt.Printf("DEBUG [expandArrayForIndex] 原数组: 长度=%d, 容量=%d\n", oldArrData.Length, oldArrData.Capacity)

	// 计算新容量
	requiredCapacity := targetIndex + 1
	newCapacity := calculateExpandedCapacity(int(oldArrData.Capacity), requiredCapacity)

	fmt.Printf("DEBUG [expandArrayForIndex] 新容量计算: 需要=%d, 实际=%d\n", requiredCapacity, newCapacity)

	// 分配新的更大的数组
	headerSize := 16
	arrayDataSize := 8
	elementsSize := newCapacity * 16
	totalSize := headerSize + arrayDataSize + elementsSize

	newGcObj := GlobalValueGCManager.gcManager.AllocateIsolated(totalSize, uint8(gc.ObjectTypeArray))
	if newGcObj == nil {
		fmt.Printf("DEBUG [expandArrayForIndex] 尝试普通分配作为后备\n")
		newGcObj = GlobalValueGCManager.gcManager.Allocate(totalSize, uint8(gc.ObjectTypeArray))
	}

	if newGcObj == nil {
		return NewNilValueGC(), fmt.Errorf("failed to allocate expanded array")
	}

	fmt.Printf("DEBUG [expandArrayForIndex] 新数组分配成功: %p\n", newGcObj)

	// 初始化新数组头
	newArrData := (*GCArrayData)(newGcObj.GetDataPtr())
	newArrData.Length = oldArrData.Length
	newArrData.Capacity = uint32(newCapacity) // 容量变化！

	fmt.Printf("DEBUG [expandArrayForIndex] 新数组头: 长度=%d, 容量=%d\n", newArrData.Length, newArrData.Capacity)

	// 拷贝现有元素
	for i := uint32(0); i < oldArrData.Length; i++ {
		oldElemPtr := getElementPtr(oldArrData, int(i))
		newElemPtr := getElementPtr(newArrData, int(i))
		if oldElemPtr != nil && newElemPtr != nil {
			*newElemPtr = SafeCopyValueGC(*oldElemPtr)
		}
	}

	// 填充剩余位置为nil
	nilValue := NewNilValueGC()
	for i := int(oldArrData.Length); i < newCapacity; i++ {
		newElemPtr := getElementPtr(newArrData, i)
		if newElemPtr != nil {
			*newElemPtr = nilValue
		}
	}

	fmt.Printf("DEBUG [expandArrayForIndex] 扩容完成\n")

	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeArray) | ValueGCFlagGCManaged,
		data:         uint64(uintptr(unsafe.Pointer(newGcObj))),
	}, nil
}

// =============================================================================
// 动态扩容策略函数
// =============================================================================

// calculateExpandedCapacity 智能容量计算：结合逻辑+物理优化
func calculateExpandedCapacity(currentCap, requiredMin int) int {
	if requiredMin <= currentCap {
		return currentCap // 无需扩容
	}

	// 逻辑层：计算理想容量
	logicalCap := currentCap
	switch {
	case currentCap <= 16:
		// 小数组：2倍增长，快速达到实用大小
		for logicalCap < requiredMin {
			if logicalCap == 0 {
				logicalCap = 4
			} else {
				logicalCap *= 2
			}
		}

	case currentCap <= 512:
		// 中等数组：1.5倍增长，平衡性能和内存
		for logicalCap < requiredMin {
			logicalCap += logicalCap >> 1 // *= 1.5
		}

	case currentCap <= 4096:
		// 大数组：1.25倍增长，节省内存
		for logicalCap < requiredMin {
			logicalCap += logicalCap >> 2 // *= 1.25
		}

	default:
		// 超大数组：固定增长，最小化浪费
		increment := max(currentCap>>3, 1024) // 至少1024增长
		logicalCap = ((requiredMin + increment - 1) / increment) * increment
	}

	// 物理层：对齐到合适的边界
	return alignToMemoryBoundary(logicalCap)
}

// alignToMemoryBoundary 内存边界对齐：减少分配器碎片
func alignToMemoryBoundary(capacity int) int {
	const (
		CACHE_LINE = 64   // CPU缓存行
		PAGE_SIZE  = 4096 // 内存页
	)

	elementSize := 16 // sizeof(ValueGC)
	headerSize := 24  // GCObject + GCArrayData
	totalSize := headerSize + capacity*elementSize

	switch {
	case totalSize <= CACHE_LINE:
		// 小对象：缓存行对齐
		alignedSize := ((totalSize + CACHE_LINE - 1) / CACHE_LINE) * CACHE_LINE
		return (alignedSize - headerSize) / elementSize

	case totalSize <= PAGE_SIZE:
		// 中对象：512字节对齐
		alignedSize := ((totalSize + 511) / 512) * 512
		return (alignedSize - headerSize) / elementSize

	default:
		// 大对象：页对齐
		alignedSize := ((totalSize + PAGE_SIZE - 1) / PAGE_SIZE) * PAGE_SIZE
		return (alignedSize - headerSize) / elementSize
	}
}

// max 返回两个整数的最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getElementPtr 动态获取元素指针 - 无硬编码限制
func getElementPtr(arrData *GCArrayData, index int) *ValueGC {
	if index < 0 || index >= int(arrData.Capacity) {
		return nil
	}

	basePtr := unsafe.Pointer(uintptr(unsafe.Pointer(arrData)) + 8) // 跳过8字节头
	elementPtr := unsafe.Pointer(uintptr(basePtr) + uintptr(index)*16)
	return (*ValueGC)(elementPtr)
}

// getArrayData 从ValueGC获取数组数据
func getArrayData(arrayValue ValueGC) (*GCArrayData, error) {
	if !arrayValue.IsArray() {
		return nil, fmt.Errorf("not an array")
	}

	gcObj := (*gc.GCObject)(unsafe.Pointer(uintptr(arrayValue.data)))
	return (*GCArrayData)(gcObj.GetDataPtr()), nil
}

// createArraySliceView 动态创建slice view，无大小限制
func createArraySliceView(arrData *GCArrayData) []ValueGC {
	if arrData.Length == 0 {
		return nil
	}

	basePtr := unsafe.Pointer(uintptr(unsafe.Pointer(arrData)) + 8)

	// 使用unsafe.Slice创建动态大小的slice
	return unsafe.Slice((*ValueGC)(basePtr), arrData.Length)
}
