// Package vm - 简化的数组系统设计
//
// 设计目标：
// 1. 保持Value的16字节紧凑设计
// 2. 使用Go GC进行内存管理
// 3. 单一64位ID，无版本号复杂性
// 4. 简化的API和实现
package vm

import (
	"fmt"
	"sync"
)

// =============================================================================
// Go GC + 大ID方案
// =============================================================================

// 设计原理：
// - 使用单一的64位ID作为数组标识符
// - 依赖Go GC进行自动内存管理
// - 简化的ArrayObject结构，只包含核心数据
// - 全局ArrayManager统一管理所有数组

// ArrayObject 数组对象，由Go GC管理
type ArrayObject struct {
	Elements []Value // 数组元素
	Length   int     // 数组长度
	ID       uint64  // 数组ID，用于调试
}

// ArrayManager 全局数组管理器
type ArrayManager struct {
	mutex  sync.RWMutex
	arrays map[uint64]*ArrayObject // ID -> Array映射
	nextID uint64                  // 下一个分配的ID（只增不减）
}

// 全局单例
var GlobalArrayManager = &ArrayManager{
	arrays: make(map[uint64]*ArrayObject),
	nextID: 1, // 从1开始，0表示无效
}

// =============================================================================
// Array管理器实现
// =============================================================================

// CreateArray 创建新数组，返回唯一ID
func (am *ArrayManager) CreateArray(elements []Value) uint64 {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// 分配唯一ID（永不重用）
	id := am.nextID
	am.nextID++

	// 创建Array对象
	array := &ArrayObject{
		Elements: make([]Value, len(elements)),
		Length:   len(elements),
		ID:       id,
	}
	copy(array.Elements, elements)

	// 注册到管理器
	am.arrays[id] = array

	return id
}

// GetArray 获取数组对象
func (am *ArrayManager) GetArray(id uint64) (*ArrayObject, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	array, exists := am.arrays[id]
	if !exists {
		return nil, fmt.Errorf("array not found: ID=%d", id)
	}

	return array, nil
}

// CloneArray 克隆数组
func (am *ArrayManager) CloneArray(id uint64) (uint64, error) {
	array, err := am.GetArray(id)
	if err != nil {
		return 0, err
	}

	// 深拷贝元素
	newElements := make([]Value, len(array.Elements))
	copy(newElements, array.Elements)

	return am.CreateArray(newElements), nil
}

// GetStats 获取管理器统计信息
func (am *ArrayManager) GetStats() (activeArrays int, nextID uint64) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return len(am.arrays), am.nextID
}

// =============================================================================
// Value系统集成
// =============================================================================

// AsArray 获取数组对象
func (v Value) AsArray() (*ArrayObject, error) {
	if !v.IsArray() {
		return nil, fmt.Errorf("not an array")
	}

	id := v.data
	return GlobalArrayManager.GetArray(id)
}

// ArrayID 获取数组ID
func (v Value) ArrayID() uint64 {
	if !v.IsArray() {
		return 0
	}
	return v.data
}

// =============================================================================
// 数组操作实现
// =============================================================================

// ArrayGetValue 获取数组元素
func ArrayGetValue(arrayValue Value, index int) (Value, error) {
	array, err := arrayValue.AsArray()
	if err != nil {
		return NewNilValue(), err
	}

	if index < 0 || index >= array.Length {
		return NewNilValue(), fmt.Errorf("array index out of bounds: %d", index)
	}

	return array.Elements[index], nil
}

// ArraySetValue 设置数组元素
func ArraySetValue(arrayValue Value, index int, value Value) error {
	array, err := arrayValue.AsArray()
	if err != nil {
		return err
	}

	if index < 0 || index >= array.Length {
		return fmt.Errorf("array index out of bounds: %d", index)
	}

	array.Elements[index] = value
	return nil
}

// ArrayLengthValue 获取数组长度
func ArrayLengthValue(arrayValue Value) (Value, error) {
	array, err := arrayValue.AsArray()
	if err != nil {
		return NewNilValue(), err
	}

	return NewSmallIntValue(int32(array.Length)), nil
}

// ArrayCopyValue 深拷贝数组
func ArrayCopyValue(arrayValue Value) (Value, error) {
	if !arrayValue.IsArray() {
		return NewNilValue(), fmt.Errorf("not an array")
	}

	id := arrayValue.ArrayID()
	newID, err := GlobalArrayManager.CloneArray(id)
	if err != nil {
		return NewNilValue(), err
	}

	return Value{
		typeAndFlags: uint64(ValueGCTypeArray),
		data:         newID,
	}, nil
}

// ArraySliceValue 数组切片
func ArraySliceValue(arrayValue Value, start, end int) (Value, error) {
	array, err := arrayValue.AsArray()
	if err != nil {
		return NewNilValue(), err
	}

	if start < 0 || end > array.Length || start > end {
		return NewNilValue(), fmt.Errorf("invalid slice range: [%d:%d]", start, end)
	}

	// 创建切片
	sliceElements := make([]ValueGC, end-start)
	for i, elem := range array.Elements[start:end] {
		sliceElements[i] = elem
	}

	return NewArrayValueGC(sliceElements), nil
}

// =============================================================================
// 扩展的数组指令
// =============================================================================

// 数组相关的VM指令
const (
	OP_ARRAY_COPY   OpCode = iota + 100 // ARRAY_COPY A B : R(A) := copy(R(B))
	OP_ARRAY_SLICE                      // ARRAY_SLICE A B C D : R(A) := R(B)[R(C):R(D)]
	OP_ARRAY_APPEND                     // ARRAY_APPEND A B : R(A).append(R(B))
	OP_ARRAY_LENGTH                     // ARRAY_LENGTH A B : R(A) := len(R(B))
	OP_ARRAY_CONCAT                     // ARRAY_CONCAT A B C : R(A) := R(B) + R(C)
)

// =============================================================================
// 内存管理说明
// =============================================================================

// 内存管理策略：
// 1. 依赖Go GC进行自动内存管理
// 2. ArrayObject由Go GC负责回收
// 3. 长期运行的程序可以实现清理机制：
//    - 定期清理不再使用的数组
//    - 在特定生命周期点进行清理
//    - 根据内存使用情况动态清理

// 潜在的内存泄漏：
// - arrays map会持续增长
// - 适用于短期运行的程序
// - 长期运行需要额外的清理策略

// 未来可能的优化：
// 1. 实现定期清理机制
// 2. 使用弱引用或finalizer
// 3. 实现引用计数（如有需要）
// 4. 内存压力时的自动清理

// =============================================================================
// 性能特点
// =============================================================================

// 优点：
// - 简化的API，无版本号复杂性
// - 更快的访问速度（无版本检查）
// - 更少的内存开销（每个数组节省4字节）
// - 代码更简洁，易于维护

// 缺点：
// - 潜在的内存泄漏（长期运行）
// - 无法检测失效引用（但Go GC保证内存安全）
// - map会持续增长（需要清理策略）

// 适用场景：
// - 短期运行的脚本
// - 内存使用不太敏感的应用
// - 优先考虑简洁性和性能的场景
