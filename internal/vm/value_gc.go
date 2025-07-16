package vm

import (
	"fmt"
	"strconv"
	"unsafe"

	"github.com/zhnt/aql/internal/gc"
)

// =============================================================================
// Value 系统与 GC 集成
// =============================================================================

// ValueGC 集成 GC 的 Value 结构
// 保持 16 字节紧凑设计，添加 GC 支持
type ValueGC struct {
	typeAndFlags uint64 // 类型+标志+GC标记位 (8字节)
	data         uint64 // 数据存储 (8字节)
}

// 编译时检查：确保 ValueGC 大小为 16 字节
var _ = (*struct {
	a [16 - unsafe.Sizeof(ValueGC{})]byte
})(nil)

// =============================================================================
// 类型和标志位定义
// =============================================================================

// ValueTypeGC 优化的值类型枚举（与 GC 集成）
type ValueTypeGC uint8

const (
	ValueGCTypeNil ValueTypeGC = iota
	ValueGCTypeSmallInt
	ValueGCTypeDouble
	ValueGCTypeString // GC 管理
	ValueGCTypeBool
	ValueGCTypeFunction // GC 管理
	ValueGCTypeCallable // GC 管理（可调用对象：函数和闭包）
	ValueGCTypeClosure  // GC 管理（闭包）- 即将废弃
	ValueGCTypeArray    // GC 管理
	ValueGCTypeStruct   // GC 管理（预留）
)

// Value 标志位掩码和定义
const (
	VALUE_GC_TYPE_MASK     = 0x0F   // 低4位：类型（支持0-15）
	VALUE_GC_FLAG_MASK     = 0xF0   // 位4-7：标志位
	VALUE_GC_EXTENDED_MASK = 0xFF00 // 位8-15：扩展标志

	// 小整数边界
	VALUE_GC_SMALL_INT_MIN = -1073741824 // -2^30
	VALUE_GC_SMALL_INT_MAX = 1073741823  // 2^30-1
)

// Value 标志位定义
const (
	ValueGCFlagInline    = 1 << 4 // 内联存储标记
	ValueGCFlagGCManaged = 1 << 5 // GC 管理对象
	ValueGCFlagConst     = 1 << 6 // 常量标记
	ValueGCFlagWeak      = 1 << 7 // 弱引用标记
	ValueGCFlagImmutable = 1 << 8 // 不可变对象标记
)

// =============================================================================
// GC 管理的对象数据结构（存储在 GCObject.Data 中）
// =============================================================================

// GCStringData GC 管理的字符串数据
type GCStringData struct {
	Length uint32 // 字符串长度
	_      uint32 // 填充对齐到8字节
	// 字符串内容紧随其后
}

// GCArrayData GC 管理的数组数据
type GCArrayData struct {
	Length   uint32 // 数组长度
	Capacity uint32 // 数组容量
	// ValueGC 元素紧随其后
}

// GCFunctionData GC 管理的函数数据
type GCFunctionData struct {
	ParamCount   int32  // 参数数量
	MaxStackSize int32  // 最大栈大小
	IsVarArg     uint8  // 是否支持变参 (bool as uint8)
	IsAsync      uint8  // 是否为异步函数 (bool as uint8)
	_            uint16 // 填充对齐
	// 名称、指令、常量等数据紧随其后或通过偏移量访问
}

// GCClosureData 闭包的GC管理数据结构
type GCClosureData struct {
	FunctionPtr  uint64 // 指向Function对象的指针
	CaptureCount uint32 // 捕获变量数量
	_            uint32 // 填充对齐
	// 捕获的变量数据紧随其后: [(name_len, name_data, ValueGC), ...]
}

// =============================================================================
// ValueGC 全局管理器
// =============================================================================

// ValueGCManager Value 系统的 GC 管理器
type ValueGCManager struct {
	gcManager *gc.UnifiedGCManager // 底层 GC 管理器
}

// GlobalValueGCManager 全局 Value GC 管理器
var GlobalValueGCManager *ValueGCManager

// InitValueGCManager 初始化全局 Value GC 管理器
func InitValueGCManager(gcManager *gc.UnifiedGCManager) {
	GlobalValueGCManager = &ValueGCManager{
		gcManager: gcManager,
	}
}

// =============================================================================
// ValueGC 基础方法
// =============================================================================

// Type 获取值类型
func (v ValueGC) Type() ValueTypeGC {
	return ValueTypeGC(v.typeAndFlags & VALUE_GC_TYPE_MASK)
}

// IsInline 检查是否内联存储
func (v ValueGC) IsInline() bool {
	return (v.typeAndFlags & ValueGCFlagInline) != 0
}

// IsGCManaged 检查是否需要 GC 管理
func (v ValueGC) IsGCManaged() bool {
	return (v.typeAndFlags & ValueGCFlagGCManaged) != 0
}

// RequiresGC 检查是否需要 GC
func (v ValueGC) RequiresGC() bool {
	typ := v.Type()
	return (typ == ValueGCTypeString || typ == ValueGCTypeArray ||
		typ == ValueGCTypeFunction || typ == ValueGCTypeCallable ||
		typ == ValueGCTypeClosure || typ == ValueGCTypeStruct) &&
		!v.IsInline()
}

// =============================================================================
// ValueGC 构造函数
// =============================================================================

// NewNilValueGC 创建 nil 值
func NewNilValueGC() ValueGC {
	return ValueGC{typeAndFlags: uint64(ValueGCTypeNil), data: 0}
}

// NewSmallIntValueGC 创建小整数值
func NewSmallIntValueGC(i int32) ValueGC {
	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeSmallInt) | ValueGCFlagInline,
		data:         uint64(uint32(i)),
	}
}

// NewDoubleValueGC 创建双精度浮点值
func NewDoubleValueGC(f float64) ValueGC {
	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeDouble) | ValueGCFlagInline,
		data:         *(*uint64)(unsafe.Pointer(&f)),
	}
}

// NewBoolValueGC 创建布尔值
func NewBoolValueGC(b bool) ValueGC {
	data := uint64(0)
	if b {
		data = 1
	}
	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeBool) | ValueGCFlagInline,
		data:         data,
	}
}

// NewNumberValueGC 创建数值（自动选择最优存储方式）
func NewNumberValueGC(n float64) ValueGC {
	// 尝试作为小整数存储
	if n == float64(int64(n)) && n >= VALUE_GC_SMALL_INT_MIN && n <= VALUE_GC_SMALL_INT_MAX {
		return NewSmallIntValueGC(int32(n))
	}
	// 否则存储为 double
	return NewDoubleValueGC(n)
}

// NewStringValueGC 创建字符串值（智能存储：内联 vs GC）
func NewStringValueGC(s string) ValueGC {
	// 短字符串内联存储（≤7字节，为指针大小）
	if len(s) <= 7 {
		return newInlineStringValueGC(s)
	}
	// 长字符串 GC 管理
	return newGCStringValueGC(s)
}

// newInlineStringValueGC 创建内联字符串值
func newInlineStringValueGC(s string) ValueGC {
	if len(s) > 7 {
		panic("string too long for inline storage")
	}

	var data uint64
	// 存储长度在最低字节
	data = uint64(len(s))
	// 存储字符串内容
	for i, b := range []byte(s) {
		data |= uint64(b) << (8 + i*8)
	}

	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeString) | ValueGCFlagInline,
		data:         data,
	}
}

// newGCStringValueGC 创建 GC 管理的字符串值
func newGCStringValueGC(s string) ValueGC {
	if GlobalValueGCManager == nil {
		panic("ValueGCManager not initialized")
	}

	// 计算字符串对象大小：GCStringData + 字符串内容
	strData := []byte(s)
	objSize := int(unsafe.Sizeof(GCStringData{}) + uintptr(len(strData)))

	// 从 GC 分配对象
	gcObj := GlobalValueGCManager.gcManager.Allocate(int(objSize), uint8(gc.ObjectTypeString))
	if gcObj == nil {
		panic("failed to allocate string object")
	}

	// 初始化字符串数据
	strDataPtr := (*GCStringData)(gcObj.GetDataPtr())
	strDataPtr.Length = uint32(len(s))

	// 拷贝字符串内容到紧随结构体的内存中
	contentPtr := unsafe.Pointer(uintptr(unsafe.Pointer(strDataPtr)) + unsafe.Sizeof(GCStringData{}))
	copy((*[1024]byte)(contentPtr)[:len(strData)], strData)

	// 创建 Value，存储 GCObject 指针
	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeString) | ValueGCFlagGCManaged,
		data:         uint64(uintptr(unsafe.Pointer(gcObj))),
	}
}

// NewArrayValueGC 创建数组值（GC管理）- 动态大小支持
func NewArrayValueGC(elements []ValueGC) ValueGC {
	return NewArrayValueGCWithCapacity(elements, 0) // 0表示自动计算容量
}

// NewArrayValueGCWithCapacity 创建数组值并指定容量提示
func NewArrayValueGCWithCapacity(elements []ValueGC, hintCapacity int) ValueGC {
	if GlobalValueGCManager == nil {
		panic("ValueGCManager not initialized")
	}

	fmt.Printf("DEBUG [NewArrayValueGC] 开始创建数组，元素数量: %d, 容量提示: %d\n", len(elements), hintCapacity)

	// 计算所需容量
	requiredLength := len(elements)
	minCapacity := max(requiredLength, hintCapacity)

	// 使用智能容量计算
	actualCapacity := calculateExpandedCapacity(0, minCapacity)

	fmt.Printf("DEBUG [NewArrayValueGC] 容量计算: 最小=%d, 实际=%d\n", minCapacity, actualCapacity)

	// 计算内存大小
	headerSize := 16                    // GCObject Header
	arrayDataSize := 8                  // GCArrayData (简化为8字节)
	elementsSize := actualCapacity * 16 // Elements数据
	totalSize := headerSize + arrayDataSize + elementsSize

	fmt.Printf("DEBUG [NewArrayValueGC] 内存布局:\n")
	fmt.Printf("  - GCObject Header: %d字节\n", headerSize)
	fmt.Printf("  - GCArrayData: %d字节\n", arrayDataSize)
	fmt.Printf("  - 元素数量: %d, 容量: %d\n", requiredLength, actualCapacity)
	fmt.Printf("  - 元素数据大小: %d字节\n", elementsSize)
	fmt.Printf("  - 总大小: %d字节\n", totalSize)

	// 分配内存
	gcObj := GlobalValueGCManager.gcManager.AllocateIsolated(totalSize, uint8(gc.ObjectTypeArray))
	if gcObj == nil {
		fmt.Printf("DEBUG [NewArrayValueGC] 尝试普通分配作为后备\n")
		gcObj = GlobalValueGCManager.gcManager.Allocate(totalSize, uint8(gc.ObjectTypeArray))
	}

	if gcObj == nil {
		fmt.Printf("DEBUG [NewArrayValueGC] 错误: GC分配失败\n")
		panic("failed to allocate array object")
	}

	fmt.Printf("DEBUG [NewArrayValueGC] GC对象分配成功: %p\n", gcObj)

	// 初始化数组头
	arrData := (*GCArrayData)(gcObj.GetDataPtr())
	arrData.Length = uint32(requiredLength)
	arrData.Capacity = uint32(actualCapacity)

	fmt.Printf("DEBUG [NewArrayValueGC] 初始化数组头: Length=%d, Capacity=%d\n", arrData.Length, arrData.Capacity)

	// 拷贝元素
	for i, elem := range elements {
		elemPtr := getElementPtr(arrData, i)
		if elemPtr == nil {
			panic(fmt.Sprintf("failed to get element pointer for index %d", i))
		}
		*elemPtr = SafeCopyValueGC(elem)
		fmt.Printf("DEBUG [NewArrayValueGC] 拷贝元素[%d]: 类型=%s\n", i, elem.Type())
	}

	// 剩余位置填充nil
	nilValue := NewNilValueGC()
	for i := requiredLength; i < actualCapacity; i++ {
		elemPtr := getElementPtr(arrData, i)
		if elemPtr != nil {
			*elemPtr = nilValue
		}
	}

	fmt.Printf("DEBUG [NewArrayValueGC] 数组创建完成\n")

	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeArray) | ValueGCFlagGCManaged,
		data:         uint64(uintptr(unsafe.Pointer(gcObj))),
	}
}

// NewFunctionValueGC 创建函数值（旧版本，保持兼容性）
func NewFunctionValueGC(name string, paramCount int, maxStackSize int) ValueGC {
	if GlobalValueGCManager == nil {
		panic("ValueGCManager not initialized")
	}

	// 计算函数对象大小：基础结构 + 名称字符串
	nameBytes := []byte(name)
	objSize := int(unsafe.Sizeof(GCFunctionData{}) + uintptr(len(nameBytes)))

	// 从 GC 分配对象
	gcObj := GlobalValueGCManager.gcManager.Allocate(objSize, uint8(gc.ObjectTypeFunction))
	if gcObj == nil {
		panic("failed to allocate function object")
	}

	// 初始化函数数据
	funcDataPtr := (*GCFunctionData)(gcObj.GetDataPtr())
	funcDataPtr.ParamCount = int32(paramCount)
	funcDataPtr.MaxStackSize = int32(maxStackSize)
	funcDataPtr.IsVarArg = 0
	funcDataPtr.IsAsync = 0

	// 拷贝函数名称到紧随结构体的内存中
	namePtr := unsafe.Pointer(uintptr(unsafe.Pointer(funcDataPtr)) + unsafe.Sizeof(GCFunctionData{}))
	copy((*[256]byte)(namePtr)[:len(nameBytes)], nameBytes)

	// 创建 Value，存储 GCObject 指针
	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeFunction) | ValueGCFlagGCManaged,
		data:         uint64(uintptr(unsafe.Pointer(gcObj))),
	}
}

// NewFunctionValueGCFromID 创建函数值（使用Function ID）
func NewFunctionValueGCFromID(functionID int) ValueGC {
	// 简单地将Function ID存储在data字段中，使用内联存储
	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeFunction) | ValueGCFlagInline,
		data:         uint64(functionID),
	}
}

// NewCallableValueGC 创建Callable ValueGC（新的统一可调用对象）
func NewCallableValueGC(function *Function, upvalues []*Upvalue) ValueGC {
	fmt.Printf("DEBUG [NewCallableValueGC] 输入函数: %p\n", function)
	if function != nil {
		fmt.Printf("DEBUG [NewCallableValueGC] 函数名: %s\n", function.Name)
	} else {
		fmt.Printf("DEBUG [NewCallableValueGC] 函数为nil!\n")
	}
	fmt.Printf("DEBUG [NewCallableValueGC] upvalue数量: %d\n", len(upvalues))

	// 在堆上分配Callable对象，确保生命周期正确
	callable := &Callable{
		Function: function,
		Upvalues: upvalues,
	}

	fmt.Printf("DEBUG [NewCallableValueGC] 创建callable对象: %p\n", callable)
	fmt.Printf("DEBUG [NewCallableValueGC] callable.Function: %p\n", callable.Function)
	if callable.Function != nil {
		fmt.Printf("DEBUG [NewCallableValueGC] callable.Function.Name: %s\n", callable.Function.Name)
	}

	// 安全地存储堆指针
	result := ValueGC{
		typeAndFlags: uint64(ValueGCTypeCallable),
		data:         uint64(uintptr(unsafe.Pointer(callable))),
	}

	fmt.Printf("DEBUG [NewCallableValueGC] 存储的指针地址: 0x%x\n", result.data)
	fmt.Printf("DEBUG [NewCallableValueGC] 类型: %s\n", result.Type())

	return result
}

// NewClosureValueGC 创建闭包ValueGC（堆分配版本）- 即将废弃
func NewClosureValueGC(function *Function, captures map[string]ValueGC) ValueGC {
	fmt.Printf("DEBUG [NewClosureValueGC] 输入函数: %p\n", function)
	if function != nil {
		fmt.Printf("DEBUG [NewClosureValueGC] 函数名: %s\n", function.Name)
	}
	fmt.Printf("DEBUG [NewClosureValueGC] 捕获变量数量: %d\n", len(captures))

	// 在堆上分配闭包对象，确保生命周期正确
	closure := &Closure{
		Function: function,
		Captures: make(map[string]ValueGC),
	}

	fmt.Printf("DEBUG [NewClosureValueGC] 创建的闭包对象: %p\n", closure)
	fmt.Printf("DEBUG [NewClosureValueGC] 闭包.Function: %p\n", closure.Function)
	if closure.Function != nil {
		fmt.Printf("DEBUG [NewClosureValueGC] 闭包.Function.Name: %s\n", closure.Function.Name)
	}

	// 复制捕获变量到堆分配的map
	for name, value := range captures {
		closure.Captures[name] = value
		fmt.Printf("DEBUG [NewClosureValueGC] 复制捕获变量: %s -> %s\n", name, value.Type())
	}

	fmt.Printf("DEBUG [NewClosureValueGC] 最终闭包对象: %p, Function: %p\n", closure, closure.Function)

	// 安全地存储堆指针
	return ValueGC{
		typeAndFlags: uint64(ValueGCTypeClosure),
		data:         uint64(uintptr(unsafe.Pointer(closure))),
	}
}

// =============================================================================
// ValueGC 数据访问方法
// =============================================================================

// AsSmallInt 获取小整数值
func (v ValueGC) AsSmallInt() int32 {
	return int32(uint32(v.data))
}

// AsDouble 获取双精度浮点值
func (v ValueGC) AsDouble() float64 {
	return *(*float64)(unsafe.Pointer(&v.data))
}

// AsBool 获取布尔值
func (v ValueGC) AsBool() bool {
	return v.data != 0
}

// AsString 获取字符串值
func (v ValueGC) AsString() string {
	if v.IsInline() {
		// 内联字符串
		length := int(v.data & 0xFF)
		if length == 0 {
			return ""
		}
		data := make([]byte, length)
		for i := 0; i < length; i++ {
			data[i] = byte((v.data >> (8 + i*8)) & 0xFF)
		}
		return string(data)
	} else {
		// GC 管理的字符串
		gcObj := (*gc.GCObject)(unsafe.Pointer(uintptr(v.data)))
		strData := (*GCStringData)(gcObj.GetDataPtr())
		contentPtr := unsafe.Pointer(uintptr(unsafe.Pointer(strData)) + unsafe.Sizeof(GCStringData{}))
		return string((*[1024]byte)(contentPtr)[:strData.Length])
	}
}

// AsArrayData 获取数组数据和元素 - 动态大小支持
func (v ValueGC) AsArrayData() (*GCArrayData, []ValueGC, error) {
	if v.Type() != ValueGCTypeArray {
		return nil, nil, fmt.Errorf("not an array")
	}

	arrData, err := getArrayData(v)
	if err != nil {
		return nil, nil, err
	}

	// 使用动态slice创建，无大小限制
	elementsSlice := createArraySliceView(arrData)

	return arrData, elementsSlice, nil
}

// AsFunctionData 获取函数数据
func (v ValueGC) AsFunctionData() (*GCFunctionData, string, error) {
	if v.Type() != ValueGCTypeFunction {
		return nil, "", fmt.Errorf("not a function")
	}

	// 检查是否是内联存储的Function ID
	if v.IsInline() {
		// 对于Function ID，返回简单的占位符数据
		functionID := int(v.data)
		if function, err := GetFunction(functionID); err == nil {
			// 创建临时的GCFunctionData
			tempData := &GCFunctionData{
				ParamCount:   int32(function.ParamCount),
				MaxStackSize: int32(function.MaxStackSize),
				IsVarArg:     0,
				IsAsync:      0,
			}
			return tempData, function.Name, nil
		}
		return nil, "", fmt.Errorf("function with ID %d not found", functionID)
	}

	// 旧版本的GC管理的函数对象
	gcObj := (*gc.GCObject)(unsafe.Pointer(uintptr(v.data)))
	funcData := (*GCFunctionData)(gcObj.GetDataPtr())

	// 获取函数名称（这里简化处理，假设名称长度合理）
	namePtr := unsafe.Pointer(uintptr(unsafe.Pointer(funcData)) + unsafe.Sizeof(GCFunctionData{}))
	// 这里需要知道名称长度，暂时使用一个合理的估计
	name := string((*[256]byte)(namePtr)[:32]) // 简化处理

	return funcData, name, nil
}

// =============================================================================
// VM兼容性方法
// =============================================================================

// AsFunction 获取函数对象（VM兼容性）
func (v ValueGC) AsFunction() interface{} {
	if v.Type() != ValueGCTypeFunction {
		return nil
	}

	// 检查是否是内联存储的Function ID
	if v.IsInline() {
		// 从Function注册表获取完整的Function对象
		functionID := int(v.data)
		if function, err := GetFunction(functionID); err == nil {
			return function
		}
		return nil
	}

	// 旧版本的GC管理的函数对象，暂时返回ValueGC自身
	return v
}

// AsCallable 获取可调用对象（带安全检查）
func (v ValueGC) AsCallable() *Callable {
	fmt.Printf("DEBUG [AsCallable] 输入值类型: %s\n", v.Type())

	if v.Type() != ValueGCTypeCallable {
		fmt.Printf("DEBUG [AsCallable] 类型不匹配，期望: %s, 实际: %s\n", ValueGCTypeCallable, v.Type())
		return nil
	}

	// 检查指针有效性
	if v.data == 0 {
		fmt.Printf("DEBUG [AsCallable] 指针为空\n")
		return nil
	}

	fmt.Printf("DEBUG [AsCallable] 指针地址: 0x%x\n", v.data)

	// 从指针恢复可调用对象
	callable := (*Callable)(unsafe.Pointer(uintptr(v.data)))

	fmt.Printf("DEBUG [AsCallable] 恢复的callable指针: %p\n", callable)

	// 基本有效性检查
	if callable == nil {
		fmt.Printf("DEBUG [AsCallable] callable为nil\n")
		return nil
	}

	if callable.Function == nil {
		fmt.Printf("DEBUG [AsCallable] callable.Function为nil\n")
		return nil
	}

	fmt.Printf("DEBUG [AsCallable] 成功恢复callable，函数: %s\n", callable.Function.Name)
	return callable
}

// AsClosure 获取闭包对象（带安全检查）- 即将废弃
func (v ValueGC) AsClosure() *Closure {
	fmt.Printf("DEBUG [AsClosure] 检查闭包: Type=%s, data=%d\n", v.Type(), v.data)

	if v.Type() != ValueGCTypeClosure {
		fmt.Printf("DEBUG [AsClosure] 类型错误: 期望=%s, 实际=%s\n", ValueGCTypeClosure, v.Type())
		return nil
	}

	// 检查指针有效性
	if v.data == 0 {
		fmt.Printf("DEBUG [AsClosure] 数据指针为空\n")
		return nil
	}

	// 从指针恢复闭包对象
	closure := (*Closure)(unsafe.Pointer(uintptr(v.data)))
	fmt.Printf("DEBUG [AsClosure] 恢复闭包对象: %p\n", closure)

	// 基本有效性检查
	if closure == nil || closure.Function == nil {
		fmt.Printf("DEBUG [AsClosure] 闭包对象或函数为nil: closure=%p, function=%p\n", closure, closure.Function)
		return nil
	}

	fmt.Printf("DEBUG [AsClosure] 成功恢复闭包，函数: %s\n", closure.Function.Name)
	return closure
}

// IsTruthy 判断值是否为真（方法形式）
func (v ValueGC) IsTruthy() bool {
	return v.ToBool()
}

// =============================================================================
// 简化引用计数管理
// =============================================================================

// IncRef 增加引用计数（简化版本）
func (v ValueGC) IncRef() {
	if v.RequiresGC() {
		incrementValueRefCountSimple(v)
	}
}

// DecRef 减少引用计数（简化版本）
func (v ValueGC) DecRef() {
	if v.RequiresGC() {
		decrementValueRefCountSimple(v)
	}
}

// RefCount 获取引用计数（简化版本）
func (v ValueGC) RefCount() uint32 {
	if !v.RequiresGC() {
		return 0
	}
	return getValueRefCountSimple(v)
}

// 内部简化实现
func incrementValueRefCountSimple(v ValueGC) {
	objPtr := unsafe.Pointer(uintptr(v.data))
	header := (*gc.GCObjectHeader)(objPtr)

	oldRefCount := header.RefCount()
	header.IncRefCount()

	fmt.Printf("DEBUG [SimpleRefCount] IncRef: obj=%p, %d -> %d\n", objPtr, oldRefCount, header.RefCount())
}

func decrementValueRefCountSimple(v ValueGC) {
	objPtr := unsafe.Pointer(uintptr(v.data))
	header := (*gc.GCObjectHeader)(objPtr)

	oldRefCount := header.RefCount()
	newRefCount := header.DecRefCount()

	fmt.Printf("DEBUG [SimpleRefCount] DecRef: obj=%p, %d -> %d\n", objPtr, oldRefCount, newRefCount)

	if newRefCount == 0 {
		fmt.Printf("DEBUG [SimpleRefCount] 对象引用计数归零，开始清理: obj=%p\n", objPtr)
		handleZeroRefCountSimple(v)
	}
}

func getValueRefCountSimple(v ValueGC) uint32 {
	objPtr := unsafe.Pointer(uintptr(v.data))
	header := (*gc.GCObjectHeader)(objPtr)
	return header.RefCount()
}

// handleZeroRefCountSimple 简化的零引用计数处理
func handleZeroRefCountSimple(v ValueGC) {
	objPtr := unsafe.Pointer(uintptr(v.data))
	fmt.Printf("DEBUG [SimpleRefCount] 处理零引用计数: obj=%p, type=%s\n", objPtr, v.Type())

	// 根据类型处理子对象的引用计数
	switch v.Type() {
	case ValueGCTypeArray:
		handleArrayZeroRefSimple(v)
	case ValueGCTypeString:
		// 字符串没有子引用
	case ValueGCTypeFunction:
		// 函数对象处理
	case ValueGCTypeCallable:
		// 可调用对象处理
	case ValueGCTypeClosure:
		// 闭包对象处理
	}

	// 释放对象内存
	if GlobalValueGCManager != nil {
		gcObj := (*gc.GCObject)(objPtr)
		fmt.Printf("DEBUG [SimpleRefCount] 释放对象内存: obj=%p\n", objPtr)
		GlobalValueGCManager.gcManager.Deallocate(gcObj)
	}
}

// handleArrayZeroRefSimple 简化的数组零引用处理
func handleArrayZeroRefSimple(v ValueGC) {
	_, elements, err := v.AsArrayData()
	if err != nil {
		fmt.Printf("DEBUG [SimpleRefCount] 数组数据获取失败: %v\n", err)
		return
	}

	fmt.Printf("DEBUG [SimpleRefCount] 处理数组子元素引用计数: 长度=%d\n", len(elements))

	// 减少所有元素的引用计数
	for i, elem := range elements {
		if elem.RequiresGC() {
			fmt.Printf("DEBUG [SimpleRefCount] 减少元素[%d]引用计数: type=%s\n", i, elem.Type())
			elem.DecRef()
		}
	}
}

// =============================================================================
// ValueGC 赋值和拷贝（带引用计数管理）
// =============================================================================

// =============================================================================
// 兼容性保持：原有的复杂版本仍然保留
// =============================================================================

// CopyValueGC 安全拷贝值（自动管理引用计数）
func CopyValueGC(v ValueGC) ValueGC {
	if v.RequiresGC() {
		fmt.Printf("DEBUG [CopyValueGC] 拷贝GC对象: obj=%p, type=%s\n", unsafe.Pointer(uintptr(v.data)), v.Type())
		// 使用简化的引用计数管理
		v.IncRef()
	}
	return v
}

// AssignValueGC 安全赋值（自动管理引用计数）
func AssignValueGC(dst *ValueGC, src ValueGC) {
	if dst.RequiresGC() {
		fmt.Printf("DEBUG [AssignValueGC] 赋值前减少旧值引用: obj=%p, type=%s\n", unsafe.Pointer(uintptr(dst.data)), dst.Type())
		// 使用简化的引用计数管理
		dst.DecRef()
	}

	if src.RequiresGC() {
		fmt.Printf("DEBUG [AssignValueGC] 赋值时增加新值引用: obj=%p, type=%s\n", unsafe.Pointer(uintptr(src.data)), src.Type())
		// 使用简化的引用计数管理
		src.IncRef()
	}

	// 最后赋值
	*dst = src
}

// SafeCopyValueGC 安全拷贝值（自动管理引用计数，避免循环引用）
func SafeCopyValueGC(v ValueGC) ValueGC {
	return safeCopyValueGCWithDepth(v, 0, make(map[uintptr]bool))
}

// safeCopyValueGCWithDepth 带深度限制的安全拷贝（内部实现）
func safeCopyValueGCWithDepth(v ValueGC, depth int, visited map[uintptr]bool) ValueGC {
	// 深度限制，避免无限递归
	if depth > 10 {
		fmt.Printf("DEBUG [SafeCopyValueGC] 深度限制达到，返回nil: depth=%d\n", depth)
		return NewNilValueGC()
	}

	if v.IsGCManaged() {
		objPtr := uintptr(v.data)

		// 检查是否已经访问过（循环引用检测）
		if visited[objPtr] {
			fmt.Printf("DEBUG [SafeCopyValueGC] 检测到循环引用，返回nil: obj=%p\n", unsafe.Pointer(objPtr))
			return NewNilValueGC()
		}

		// 标记为已访问
		visited[objPtr] = true
		defer func() {
			delete(visited, objPtr)
		}()

		fmt.Printf("DEBUG [SafeCopyValueGC] 拷贝GC对象: obj=%p, type=%s, depth=%d\n", unsafe.Pointer(objPtr), v.Type(), depth)

		// 对于数组，进行深度拷贝以避免循环引用
		if v.Type() == ValueGCTypeArray {
			return safeCopyArrayValueGC(v, depth, visited)
		}

		// 对于其他GC对象，增加引用计数
		v.IncRef()
		return v
	}

	// 非GC对象直接返回
	return v
}

// safeCopyArrayValueGC 安全拷贝数组，避免循环引用
func safeCopyArrayValueGC(arrayValue ValueGC, depth int, visited map[uintptr]bool) ValueGC {
	arrData, elements, err := arrayValue.AsArrayData()
	if err != nil {
		fmt.Printf("DEBUG [SafeCopyArrayValueGC] 数组数据获取失败: %v\n", err)
		return NewNilValueGC()
	}

	fmt.Printf("DEBUG [SafeCopyArrayValueGC] 开始拷贝数组: 长度=%d, 深度=%d\n", arrData.Length, depth)

	// 创建新的元素数组
	newElements := make([]ValueGC, arrData.Length)

	// 递归拷贝每个元素
	for i := uint32(0); i < arrData.Length; i++ {
		element := elements[i]

		// 递归安全拷贝每个元素
		newElements[i] = safeCopyValueGCWithDepth(element, depth+1, visited)

		fmt.Printf("DEBUG [SafeCopyArrayValueGC] 拷贝元素[%d]: 原类型=%s, 新类型=%s\n", i, element.Type(), newElements[i].Type())
	}

	// 创建新的数组对象
	return NewArrayValueGC(newElements)
}

// =============================================================================
// ValueGC 类型检查方法
// =============================================================================

func (v ValueGC) IsNil() bool      { return v.Type() == ValueGCTypeNil }
func (v ValueGC) IsSmallInt() bool { return v.Type() == ValueGCTypeSmallInt }
func (v ValueGC) IsDouble() bool   { return v.Type() == ValueGCTypeDouble }
func (v ValueGC) IsString() bool   { return v.Type() == ValueGCTypeString }
func (v ValueGC) IsBool() bool     { return v.Type() == ValueGCTypeBool }
func (v ValueGC) IsFunction() bool { return v.Type() == ValueGCTypeFunction }
func (v ValueGC) IsCallable() bool { return v.Type() == ValueGCTypeCallable }
func (v ValueGC) IsClosure() bool  { return v.Type() == ValueGCTypeClosure }
func (v ValueGC) IsArray() bool    { return v.Type() == ValueGCTypeArray }

func (v ValueGC) IsNumber() bool {
	typ := v.Type()
	return typ == ValueGCTypeSmallInt || typ == ValueGCTypeDouble
}

// =============================================================================
// ValueGC 类型转换方法
// =============================================================================

// ToNumber 转换为数值
func (v ValueGC) ToNumber() (float64, error) {
	switch v.Type() {
	case ValueGCTypeSmallInt:
		return float64(v.AsSmallInt()), nil
	case ValueGCTypeDouble:
		return v.AsDouble(), nil
	case ValueGCTypeString:
		s := v.AsString()
		return strconv.ParseFloat(s, 64)
	default:
		return 0, fmt.Errorf("cannot convert %s to number", v.Type())
	}
}

// ToBool 转换为布尔值
func (v ValueGC) ToBool() bool {
	switch v.Type() {
	case ValueGCTypeNil:
		return false
	case ValueGCTypeBool:
		return v.AsBool()
	case ValueGCTypeSmallInt:
		return v.AsSmallInt() != 0
	case ValueGCTypeDouble:
		return v.AsDouble() != 0.0
	case ValueGCTypeString:
		return v.AsString() != ""
	default:
		return true // 函数等其他类型被认为是真值
	}
}

// ToString 转换为字符串
func (v ValueGC) ToString() string {
	return v.toStringWithDepth(0)
}

// toStringWithDepth 带递归深度限制的字符串转换
func (v ValueGC) toStringWithDepth(depth int) string {
	const maxDepth = 3 // 最大递归深度
	if depth > maxDepth {
		return "..."
	}

	switch v.Type() {
	case ValueGCTypeNil:
		return "nil"
	case ValueGCTypeSmallInt:
		return strconv.Itoa(int(v.AsSmallInt()))
	case ValueGCTypeDouble:
		return strconv.FormatFloat(v.AsDouble(), 'g', -1, 64)
	case ValueGCTypeString:
		return v.AsString()
	case ValueGCTypeBool:
		if v.AsBool() {
			return "true"
		}
		return "false"
	case ValueGCTypeFunction:
		if v.IsInline() {
			// 内联存储的Function ID
			functionID := int(v.data)
			if function, err := GetFunction(functionID); err == nil {
				return fmt.Sprintf("function:%s", function.Name)
			}
			return fmt.Sprintf("function:id=%d:invalid", functionID)
		} else {
			// 旧版本的GC管理函数对象
			if _, _, err := v.AsFunctionData(); err == nil {
				return "function"
			}
			return "function:invalid"
		}
	case ValueGCTypeClosure:
		if closure := v.AsClosure(); closure != nil {
			return fmt.Sprintf("closure:%s", closure.Function.Name)
		}
		return "closure:invalid"
	case ValueGCTypeArray:
		if _, elements, err := v.AsArrayData(); err == nil {
			result := "["
			for i, elem := range elements {
				if i > 0 {
					result += ", "
				}
				result += elem.toStringWithDepth(depth + 1)
			}
			result += "]"
			return result
		}
		return "array:invalid"
	default:
		return fmt.Sprintf("unknown:%d", v.Type())
	}
}

func (v ValueGC) String() string {
	return v.ToString()
}

// =============================================================================
// ValueTypeGC 字符串表示
// =============================================================================

func (vt ValueTypeGC) String() string {
	switch vt {
	case ValueGCTypeNil:
		return "nil"
	case ValueGCTypeSmallInt:
		return "smallint"
	case ValueGCTypeDouble:
		return "double"
	case ValueGCTypeString:
		return "string"
	case ValueGCTypeBool:
		return "bool"
	case ValueGCTypeFunction:
		return "function"
	case ValueGCTypeCallable:
		return "callable"
	case ValueGCTypeClosure:
		return "closure"
	case ValueGCTypeArray:
		return "array"
	case ValueGCTypeStruct:
		return "struct"
	default:
		return "unknown"
	}
}

// =============================================================================
// GC 指令支持函数
// =============================================================================

// WriteBarrierGC 执行GC写屏障
func WriteBarrierGC(objValue, fieldValue ValueGC) error {
	// 如果两个值都需要GC管理，记录引用关系
	if objValue.IsGCManaged() && fieldValue.IsGCManaged() {
		// 这里可以记录引用关系到GC管理器
		// 暂时简化实现
		if GlobalValueGCManager != nil && GlobalValueGCManager.gcManager != nil {
			// 记录对象间的引用关系
			return nil
		}
	}
	return nil
}

// IncRefGC 增加引用计数
func IncRefGC(value ValueGC) error {
	if !value.IsGCManaged() {
		return nil
	}

	// 如果是引用计数GC管理的对象，增加引用计数
	if GlobalValueGCManager != nil && GlobalValueGCManager.gcManager != nil {
		// 暂时简化实现，实际应该通过GC管理器操作
		return nil
	}
	return nil
}

// DecRefGC 减少引用计数
func DecRefGC(value ValueGC) error {
	if !value.IsGCManaged() {
		return nil
	}

	// 如果是引用计数GC管理的对象，减少引用计数
	if GlobalValueGCManager != nil && GlobalValueGCManager.gcManager != nil {
		// 暂时简化实现，实际应该通过GC管理器操作
		return nil
	}
	return nil
}

// TriggerGCCollection 触发垃圾回收
func TriggerGCCollection() error {
	if GlobalValueGCManager != nil && GlobalValueGCManager.gcManager != nil {
		GlobalValueGCManager.gcManager.ForceGC()
		return nil
	}
	return nil
}

// IsValidGCPointer 检查是否为有效的GC指针
func IsValidGCPointer(value ValueGC) bool {
	if !value.IsGCManaged() {
		return true // 非GC管理的值总是有效的
	}

	// 检查GC管理的对象是否有效
	if GlobalValueGCManager != nil && GlobalValueGCManager.gcManager != nil {
		// 暂时简化实现，总是返回true
		// 实际应该检查对象是否还存活、未被移动等
		return true
	}
	return false
}

// PinGCObject 固定GC对象防止移动
func PinGCObject(value ValueGC) error {
	if !value.IsGCManaged() {
		return nil
	}

	// 固定对象，防止标记清除过程中移动
	if GlobalValueGCManager != nil && GlobalValueGCManager.gcManager != nil {
		// 暂时简化实现
		// 实际应该在GC管理器中标记对象为固定状态
		return nil
	}
	return nil
}

// UnpinGCObject 取消固定GC对象
func UnpinGCObject(value ValueGC) error {
	if !value.IsGCManaged() {
		return nil
	}

	// 取消固定对象
	if GlobalValueGCManager != nil && GlobalValueGCManager.gcManager != nil {
		// 暂时简化实现
		// 实际应该在GC管理器中取消对象的固定状态
		return nil
	}
	return nil
}

// CreateWeakRefGC 创建弱引用
func CreateWeakRefGC(targetValue ValueGC) (ValueGC, error) {
	// 暂时简化实现，直接返回目标值
	// 实际应该创建一个特殊的弱引用对象
	if targetValue.IsGCManaged() {
		// 这里应该创建一个弱引用包装器
		// 暂时返回原值，添加弱引用标记
		weakRef := targetValue
		weakRef.typeAndFlags |= ValueGCFlagWeak
		return weakRef, nil
	}
	return targetValue, nil
}

// GetWeakRefTargetGC 获取弱引用的目标值
func GetWeakRefTargetGC(weakRefValue ValueGC) (ValueGC, error) {
	// 检查是否为弱引用
	if (weakRefValue.typeAndFlags & ValueGCFlagWeak) != 0 {
		// 检查目标对象是否还存活
		if IsValidGCPointer(weakRefValue) {
			// 返回目标值，移除弱引用标记
			targetValue := weakRefValue
			targetValue.typeAndFlags &^= ValueGCFlagWeak
			return targetValue, nil
		} else {
			// 目标对象已被回收，返回nil
			return NewNilValueGC(), nil
		}
	}

	// 不是弱引用，直接返回
	return weakRefValue, nil
}
