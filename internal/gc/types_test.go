package gc

import (
	"sync"
	"testing"
	"unsafe"
)

// =============================================================================
// GCObjectHeader测试
// =============================================================================

func TestGCObjectHeaderSize(t *testing.T) {
	// 验证GCObjectHeader确实是16字节
	size := unsafe.Sizeof(GCObjectHeader{})
	if size != 16 {
		t.Errorf("GCObjectHeader size should be 16 bytes, got %d", size)
	}

	// 验证对齐
	align := unsafe.Alignof(GCObjectHeader{})
	if align < 8 {
		t.Errorf("GCObjectHeader alignment should be at least 8 bytes, got %d", align)
	}
}

func TestGCObjectHeaderRefCount(t *testing.T) {
	header := NewGCObjectHeader(ObjectTypeString, 64)

	// 测试初始引用计数
	if header.GetRefCount() != 1 {
		t.Errorf("Initial ref count should be 1, got %d", header.GetRefCount())
	}

	// 测试递增
	newCount := header.IncRefCount()
	if newCount != 2 {
		t.Errorf("After increment, ref count should be 2, got %d", newCount)
	}

	// 测试递减
	newCount = header.DecRefCount()
	if newCount != 1 {
		t.Errorf("After decrement, ref count should be 1, got %d", newCount)
	}

	// 测试减到0
	newCount = header.DecRefCount()
	if newCount != 0 {
		t.Errorf("After final decrement, ref count should be 0, got %d", newCount)
	}
}

func TestGCObjectHeaderFlags(t *testing.T) {
	header := NewGCObjectHeader(ObjectTypeArray, 128)

	// 测试初始状态
	if header.IsMarked() {
		t.Error("Object should not be marked initially")
	}

	if header.MightHaveCycles() {
		t.Error("Object should not have cycles flag initially")
	}

	// 测试设置标记位
	header.SetMarked()
	if !header.IsMarked() {
		t.Error("Object should be marked after SetMarked()")
	}

	// 测试设置循环引用标志
	header.SetCyclic()
	if !header.MightHaveCycles() {
		t.Error("Object should have cycles flag after SetCyclic()")
	}

	// 测试清除标记位
	header.ClearMarked()
	if header.IsMarked() {
		t.Error("Object should not be marked after ClearMarked()")
	}

	// 循环引用标志应该仍然存在
	if !header.MightHaveCycles() {
		t.Error("Object should still have cycles flag")
	}
}

func TestGCObjectHeaderConcurrency(t *testing.T) {
	header := NewGCObjectHeader(ObjectTypeStruct, 256)

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // 增加和减少引用计数的goroutines

	// 并发递增引用计数
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				header.IncRefCount()
			}
		}()
	}

	// 并发递减引用计数
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				header.DecRefCount()
			}
		}()
	}

	wg.Wait()

	// 最终引用计数应该是1（初始值）
	finalCount := header.GetRefCount()
	if finalCount != 1 {
		t.Errorf("Final ref count should be 1, got %d", finalCount)
	}
}

// =============================================================================
// GCObject测试
// =============================================================================

func TestGCObjectCreation(t *testing.T) {
	obj := NewGCObject(ObjectTypeString, 64)

	if obj == nil {
		t.Fatal("NewGCObject returned nil")
	}

	if obj.Type() != ObjectTypeString {
		t.Errorf("Object type should be ObjectTypeString, got %v", obj.Type())
	}

	if obj.Size() != 64 {
		t.Errorf("Object size should be 64, got %d", obj.Size())
	}

	if len(obj.Data) != 64 {
		t.Errorf("Object data length should be 64, got %d", len(obj.Data))
	}

	// 测试对象ID唯一性
	obj2 := NewGCObject(ObjectTypeArray, 128)
	if obj.ID() == obj2.ID() {
		t.Error("Different objects should have different IDs")
	}
}

func TestGCObjectExtendedType(t *testing.T) {
	obj := NewGCObject(ObjectTypeFunction, 32)

	// 测试初始扩展类型
	if obj.ExtendedType() != 0 {
		t.Errorf("Initial extended type should be 0, got %d", obj.ExtendedType())
	}

	// 设置扩展类型
	obj.SetExtendedType(1234)
	if obj.ExtendedType() != 1234 {
		t.Errorf("Extended type should be 1234, got %d", obj.ExtendedType())
	}
}

func TestGCObjectTypeConversion(t *testing.T) {
	// 测试字符串对象转换
	strObj := NewGCObject(ObjectTypeString, 48)
	if strObj.AsStringObject() == nil {
		t.Error("String object conversion should succeed")
	}

	// 错误的类型转换应该返回nil
	if strObj.AsArrayObject() != nil {
		t.Error("Array object conversion should fail for string object")
	}

	// 测试数组对象转换
	arrObj := NewGCObject(ObjectTypeArray, 48)
	if arrObj.AsArrayObject() == nil {
		t.Error("Array object conversion should succeed")
	}

	if arrObj.AsStructObject() != nil {
		t.Error("Struct object conversion should fail for array object")
	}
}

// =============================================================================
// 工具函数测试
// =============================================================================

func TestAlignmentFunctions(t *testing.T) {
	testCases := []struct {
		input   uint32
		align8  uint32
		align16 uint32
	}{
		{1, 8, 16},
		{8, 8, 16},
		{9, 16, 16},
		{16, 16, 16},
		{17, 24, 32},
		{32, 32, 32},
		{33, 40, 48},
	}

	for _, tc := range testCases {
		if result := Align8(tc.input); result != tc.align8 {
			t.Errorf("Align8(%d) = %d, want %d", tc.input, result, tc.align8)
		}

		if result := Align16(tc.input); result != tc.align16 {
			t.Errorf("Align16(%d) = %d, want %d", tc.input, result, tc.align16)
		}
	}
}

func TestSizeClass(t *testing.T) {
	testCases := []struct {
		size  uint32
		class int
	}{
		{1, 0},
		{16, 0},
		{17, 1},
		{32, 1},
		{33, 2},
		{48, 2},
		{49, 3},
		{64, 3},
		{65, 4},
		{96, 4},
		{97, 5},
		{128, 5},
		{129, 6},
		{192, 6},
		{193, 7},
		{256, 7},
		{257, -1}, // 大对象
	}

	for _, tc := range testCases {
		if result := SizeClass(tc.size); result != tc.class {
			t.Errorf("SizeClass(%d) = %d, want %d", tc.size, result, tc.class)
		}
	}
}

// =============================================================================
// 内存布局测试
// =============================================================================

func TestSpecializedObjectSizes(t *testing.T) {
	// 测试特殊对象的大小
	stringObjSize := unsafe.Sizeof(StringObject{})
	arrayObjSize := unsafe.Sizeof(ArrayObject{})
	structObjSize := unsafe.Sizeof(StructObject{})
	smallObjSize := unsafe.Sizeof(SmallObject{})

	t.Logf("StringObject size: %d bytes", stringObjSize)
	t.Logf("ArrayObject size: %d bytes", arrayObjSize)
	t.Logf("StructObject size: %d bytes", structObjSize)
	t.Logf("SmallObject size: %d bytes", smallObjSize)

	// SmallObject应该是64字节（1个缓存行）
	if smallObjSize != 64 {
		t.Errorf("SmallObject size should be 64 bytes, got %d", smallObjSize)
	}

	// 其他对象应该是48字节或更小
	if stringObjSize > 48 {
		t.Errorf("StringObject size should be <= 48 bytes, got %d", stringObjSize)
	}

	if arrayObjSize > 48 {
		t.Errorf("ArrayObject size should be <= 48 bytes, got %d", arrayObjSize)
	}

	if structObjSize > 48 {
		t.Errorf("StructObject size should be <= 48 bytes, got %d", structObjSize)
	}
}

// =============================================================================
// 性能基准测试
// =============================================================================

func BenchmarkGCObjectHeaderIncRef(b *testing.B) {
	header := NewGCObjectHeader(ObjectTypeString, 64)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		header.IncRefCount()
	}
}

func BenchmarkGCObjectHeaderDecRef(b *testing.B) {
	header := NewGCObjectHeader(ObjectTypeString, 64)

	// 先增加很多引用计数
	for i := 0; i < b.N; i++ {
		header.IncRefCount()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		header.DecRefCount()
	}
}

func BenchmarkGCObjectCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := NewGCObject(ObjectTypeArray, 128)
		_ = obj // 避免优化掉
	}
}

func BenchmarkGCObjectHeaderFlags(b *testing.B) {
	header := NewGCObjectHeader(ObjectTypeStruct, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		header.SetMarked()
		_ = header.IsMarked()
		header.ClearMarked()
	}
}
