package vm

// =============================================================================
// Value 兼容性适配层
// 为旧的编译器模块提供兼容接口
// =============================================================================

// Value 兼容性类型，映射到新的ValueGC
type Value = ValueGC

// 兼容性构造函数
func NewNilValue() Value {
	return NewNilValueGC()
}

func NewNumberValue(n float64) Value {
	return NewNumberValueGC(n)
}

func NewStringValue(s string) Value {
	return NewStringValueGC(s)
}

func NewBoolValue(b bool) Value {
	return NewBoolValueGC(b)
}

func NewSmallIntValue(i int32) Value {
	return NewSmallIntValueGC(i)
}
