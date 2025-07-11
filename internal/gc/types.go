package gc

import (
	"sync/atomic"
	"unsafe"
)

// =============================================================================
// GC对象头设计 - 16字节，8字节对齐，缓存友好
// =============================================================================

// GCObjectHeader 16字节GC对象头，8字节对齐，缓存友好
// 使用两个uint64字段确保8字节对齐
type GCObjectHeader struct {
	// 第一个8字节：引用计数和大小信息
	RefCountAndSize uint64 // 低32位：RefCount, 高32位：Size

	// 第二个8字节：类型和标志信息
	TypesAndFlags uint64 // 位布局：ExtendedType(16位) | ObjectType(8位) | Flags(8位) | Reserved(32位)
} // 总计16字节，强制8字节对齐

// 编译时检查：确保GCObjectHeader大小为16字节
var _ = (*struct {
	a [16 - unsafe.Sizeof(GCObjectHeader{})]byte
})(nil)

// =============================================================================
// GC标志位定义
// =============================================================================

// GC标志位定义
const (
	GCFlagMarked     = 1 << 0 // 标记-清除标记
	GCFlagCyclic     = 1 << 1 // 可能循环引用
	GCFlagFinalizer  = 1 << 2 // 需要finalizer
	GCFlagWeakRef    = 1 << 3 // 被弱引用指向
	GCFlagGenOld     = 1 << 4 // 老年代对象
	GCFlagPinned     = 1 << 5 // 不可移动对象
	GCFlagLargeObj   = 1 << 6 // 大对象标记
	GCFlagThreadSafe = 1 << 7 // 线程安全对象
)

// =============================================================================
// 扩展类型ID范围分配
// =============================================================================

// 扩展类型ID范围（支持65536种类型）
const (
	ExtTypeSystemStart = 1   // 系统类型 (1-100)
	ExtTypeUserFunc    = 101 // 用户函数 (101-500)
	ExtTypeUserClass   = 501 // 用户类 (501-65535)
)

// =============================================================================
// 基础对象类型枚举
// =============================================================================

// ObjectType 基础对象类型（与vm.ValueType保持兼容）
type ObjectType uint8

const (
	ObjectTypeString   ObjectType = iota // 字符串对象
	ObjectTypeArray                      // 数组对象
	ObjectTypeStruct                     // 结构体对象
	ObjectTypeFunction                   // 函数对象
	ObjectTypeClosure                    // 闭包对象
	ObjectTypeModule                     // 模块对象
	ObjectTypeUserData                   // 用户数据对象
	ObjectTypeWeakRef                    // 弱引用对象
)

// =============================================================================
// GC对象定义
// =============================================================================

// GCObject GC管理的对象
type GCObject struct {
	Header GCObjectHeader // 16字节对象头
	Data   []byte         // 实际数据，柔性数组
}

// ObjectID 对象唯一标识符
type ObjectID uint64

// =============================================================================
// GCObjectHeader操作方法
// =============================================================================

// NewGCObjectHeader 创建新的GC对象头
func NewGCObjectHeader(objType ObjectType, size uint32) GCObjectHeader {
	return GCObjectHeader{
		RefCountAndSize: uint64(1) | (uint64(size) << 32), // RefCount=1, Size=size
		TypesAndFlags:   uint64(objType) << 16,            // ObjectType在16位位置
	}
}

// IncRefCount 原子递增引用计数，返回新值
func (h *GCObjectHeader) IncRefCount() uint32 {
	for {
		current := atomic.LoadUint64(&h.RefCountAndSize)
		refCount := uint32(current) + 1
		size := uint32(current >> 32)
		newValue := uint64(refCount) | (uint64(size) << 32)
		if atomic.CompareAndSwapUint64(&h.RefCountAndSize, current, newValue) {
			return refCount
		}
	}
}

// DecRefCount 原子递减引用计数，返回新值
func (h *GCObjectHeader) DecRefCount() uint32 {
	for {
		current := atomic.LoadUint64(&h.RefCountAndSize)
		refCount := uint32(current) - 1
		size := uint32(current >> 32)
		newValue := uint64(refCount) | (uint64(size) << 32)
		if atomic.CompareAndSwapUint64(&h.RefCountAndSize, current, newValue) {
			return refCount
		}
	}
}

// GetRefCount 获取当前引用计数
func (h *GCObjectHeader) GetRefCount() uint32 {
	return uint32(atomic.LoadUint64(&h.RefCountAndSize))
}

// GetSize 获取对象大小
func (h *GCObjectHeader) GetSize() uint32 {
	return uint32(atomic.LoadUint64(&h.RefCountAndSize) >> 32)
}

// SetFlag 设置标志位
func (h *GCObjectHeader) SetFlag(flag uint8) {
	for {
		current := atomic.LoadUint64(&h.TypesAndFlags)
		newValue := current | (uint64(flag) << 8)
		if atomic.CompareAndSwapUint64(&h.TypesAndFlags, current, newValue) {
			return
		}
	}
}

// ClearFlag 清除标志位
func (h *GCObjectHeader) ClearFlag(flag uint8) {
	for {
		current := atomic.LoadUint64(&h.TypesAndFlags)
		newValue := current &^ (uint64(flag) << 8)
		if atomic.CompareAndSwapUint64(&h.TypesAndFlags, current, newValue) {
			return
		}
	}
}

// HasFlag 检查是否设置了标志位
func (h *GCObjectHeader) HasFlag(flag uint8) bool {
	current := atomic.LoadUint64(&h.TypesAndFlags)
	flags := uint8(current >> 8)
	return (flags & flag) != 0
}

// GetObjectType 获取对象类型
func (h *GCObjectHeader) GetObjectType() ObjectType {
	current := atomic.LoadUint64(&h.TypesAndFlags)
	return ObjectType(current >> 16)
}

// GetExtendedType 获取扩展类型ID
func (h *GCObjectHeader) GetExtendedType() uint16 {
	current := atomic.LoadUint64(&h.TypesAndFlags)
	return uint16(current)
}

// SetExtendedType 设置扩展类型ID
func (h *GCObjectHeader) SetExtendedType(extType uint16) {
	for {
		current := atomic.LoadUint64(&h.TypesAndFlags)
		newValue := (current &^ 0xFFFF) | uint64(extType)
		if atomic.CompareAndSwapUint64(&h.TypesAndFlags, current, newValue) {
			return
		}
	}
}

// IsMarked 检查是否被标记（用于标记清除GC）
func (h *GCObjectHeader) IsMarked() bool {
	return h.HasFlag(GCFlagMarked)
}

// SetMarked 设置标记位
func (h *GCObjectHeader) SetMarked() {
	h.SetFlag(GCFlagMarked)
}

// ClearMarked 清除标记位
func (h *GCObjectHeader) ClearMarked() {
	h.ClearFlag(GCFlagMarked)
}

// MightHaveCycles 检查对象是否可能包含循环引用
func (h *GCObjectHeader) MightHaveCycles() bool {
	return h.HasFlag(GCFlagCyclic)
}

// SetCyclic 标记对象可能包含循环引用
func (h *GCObjectHeader) SetCyclic() {
	h.SetFlag(GCFlagCyclic)
}

// IncRef 原子增加引用计数 (简化包装)
func (h *GCObjectHeader) IncRef() uint32 {
	return h.IncRefCount()
}

// DecRef 原子减少引用计数 (简化包装)
func (h *GCObjectHeader) DecRef() uint32 {
	return h.DecRefCount()
}

// RefCount 获取当前引用计数 (简化包装)
func (h *GCObjectHeader) RefCount() uint32 {
	return h.GetRefCount()
}

// IsCyclic 检查对象是否可能有循环引用 (简化包装)
func (h *GCObjectHeader) IsCyclic() bool {
	return h.MightHaveCycles()
}

// ClearCyclic 清除循环引用标志
func (h *GCObjectHeader) ClearCyclic() {
	for {
		old := atomic.LoadUint64(&h.TypesAndFlags)
		new := old &^ (uint64(GCFlagCyclic) << 8)
		if atomic.CompareAndSwapUint64(&h.TypesAndFlags, old, new) {
			break
		}
	}
}

// =============================================================================
// GCObject操作方法
// =============================================================================

// NewGCObject 创建新的GC对象
func NewGCObject(objType ObjectType, size uint32) *GCObject {
	obj := &GCObject{
		Header: NewGCObjectHeader(objType, size),
		Data:   make([]byte, size),
	}
	return obj
}

// ID 获取对象ID（基于地址）
func (obj *GCObject) ID() ObjectID {
	return ObjectID(uintptr(unsafe.Pointer(obj)))
}

// Type 获取对象类型
func (obj *GCObject) Type() ObjectType {
	return obj.Header.GetObjectType()
}

// Size 获取对象大小
func (obj *GCObject) Size() uint32 {
	return obj.Header.GetSize()
}

// ExtendedType 获取扩展类型ID
func (obj *GCObject) ExtendedType() uint16 {
	return obj.Header.GetExtendedType()
}

// SetExtendedType 设置扩展类型ID
func (obj *GCObject) SetExtendedType(extType uint16) {
	obj.Header.SetExtendedType(extType)
}

// =============================================================================
// 内存布局优化的专用对象类型
// =============================================================================

// SmallObject 小对象内联优化（<64字节）
type SmallObject struct {
	Header GCObjectHeader // 16字节
	Fields [3]uint64      // 24字节数据，总共40字节
	_      [24]byte       // 24字节填充，总共64字节（1缓存行）
}

// StringObject 字符串对象优化
type StringObject struct {
	Header GCObjectHeader // 16字节
	Length uint32         // 4字节，字符串长度
	_      uint32         // 4字节填充
	Data   []byte         // 字符串数据（slice header 24字节）
} // 总计48字节，缓存友好

// ArrayObject 数组对象优化
type ArrayObject struct {
	Header   GCObjectHeader // 16字节
	Length   uint32         // 4字节，数组长度
	Capacity uint32         // 4字节，数组容量
	Elements unsafe.Pointer // 8字节，指向元素数组
	_        [16]byte       // 16字节填充，总共48字节
} // 总计48字节，缓存友好

// StructObject 结构体对象
type StructObject struct {
	Header     GCObjectHeader // 16字节
	FieldCount uint32         // 4字节，字段数量
	_          uint32         // 4字节填充
	Fields     unsafe.Pointer // 8字节，指向字段数组
	_          [16]byte       // 16字节填充，总共48字节
} // 总计48字节，缓存友好

// =============================================================================
// 工具函数
// =============================================================================

// AlignSize 对齐大小到指定边界
func AlignSize(size, align uint32) uint32 {
	return (size + align - 1) &^ (align - 1)
}

// Align8 对齐到8字节边界
func Align8(size uint32) uint32 {
	return AlignSize(size, 8)
}

// Align16 对齐到16字节边界
func Align16(size uint32) uint32 {
	return AlignSize(size, 16)
}

// SizeClass 计算大小类别（用于分级分配）
func SizeClass(size uint32) int {
	if size <= 16 {
		return 0
	}
	if size <= 32 {
		return 1
	}
	if size <= 48 {
		return 2
	}
	if size <= 64 {
		return 3
	}
	if size <= 96 {
		return 4
	}
	if size <= 128 {
		return 5
	}
	if size <= 192 {
		return 6
	}
	if size <= 256 {
		return 7
	}
	// 大于256字节的对象使用不同的分配策略
	return -1
}

// =============================================================================
// 类型转换安全检查
// =============================================================================

// AsStringObject 安全转换为字符串对象
func (obj *GCObject) AsStringObject() *StringObject {
	if obj.Type() != ObjectTypeString {
		return nil
	}
	return (*StringObject)(unsafe.Pointer(obj))
}

// AsArrayObject 安全转换为数组对象
func (obj *GCObject) AsArrayObject() *ArrayObject {
	if obj.Type() != ObjectTypeArray {
		return nil
	}
	return (*ArrayObject)(unsafe.Pointer(obj))
}

// AsStructObject 安全转换为结构体对象
func (obj *GCObject) AsStructObject() *StructObject {
	if obj.Type() != ObjectTypeStruct {
		return nil
	}
	return (*StructObject)(unsafe.Pointer(obj))
}
