package vm

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ImprovedArrayManager 改进的数组管理器，解决内存泄漏问题
type ImprovedArrayManager struct {
	mutex       sync.RWMutex
	arrays      map[uint64]*ArrayObject
	nextID      uint64
	lastCleanup int64 // 上次清理时间戳
}

// NewImprovedArrayManager 创建改进的数组管理器
func NewImprovedArrayManager() *ImprovedArrayManager {
	am := &ImprovedArrayManager{
		arrays:      make(map[uint64]*ArrayObject),
		nextID:      1,
		lastCleanup: time.Now().Unix(),
	}

	// 启动后台清理goroutine
	go am.backgroundCleanup()

	return am
}

// CreateArrayWithCleanup 创建数组（带自动清理）
func (am *ImprovedArrayManager) CreateArrayWithCleanup(elements []Value) uint64 {
	// 定期触发清理
	if time.Now().Unix()-atomic.LoadInt64(&am.lastCleanup) > 60 { // 每60秒
		go am.cleanup()
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	id := am.nextID
	am.nextID++

	array := &ArrayObject{
		Elements: make([]Value, len(elements)),
		Length:   len(elements),
		ID:       id,
	}
	copy(array.Elements, elements)

	am.arrays[id] = array

	// 设置finalizer，当ArrayObject被GC时自动清理
	runtime.SetFinalizer(array, func(obj *ArrayObject) {
		am.removeArray(obj.ID)
	})

	return id
}

// removeArray 从管理器中移除数组
func (am *ImprovedArrayManager) removeArray(id uint64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	delete(am.arrays, id)
}

// backgroundCleanup 后台清理goroutine
func (am *ImprovedArrayManager) backgroundCleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		am.cleanup()
	}
}

// cleanup 清理不再被引用的数组
func (am *ImprovedArrayManager) cleanup() {
	atomic.StoreInt64(&am.lastCleanup, time.Now().Unix())

	// 强制GC以确保finalizer运行
	runtime.GC()
	runtime.GC() // 两次确保完整回收

	// 等待finalizer执行
	time.Sleep(10 * time.Millisecond)
}

// GetArrayImproved 获取数组（改进版）
func (am *ImprovedArrayManager) GetArrayImproved(id uint64) (*ArrayObject, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	array, exists := am.arrays[id]
	if !exists {
		return nil, fmt.Errorf("array not found or has been garbage collected: ID=%d", id)
	}

	return array, nil
}

// 简化的引用相等比较
func (v Value) EqualByReference(other Value) bool {
	if v.Type() != other.Type() {
		return false
	}

	if v.Type() == ValueTypeArray {
		return v.ArrayID() == other.ArrayID()
	}

	return v.Equal(other) // 其他类型使用原来的比较
}

// 优化的数组深度比较
func (v Value) EqualArrayOptimized(other Value) bool {
	if v.Type() != ValueTypeArray || other.Type() != ValueTypeArray {
		return false
	}

	// 快速路径：引用相等
	if v.ArrayID() == other.ArrayID() {
		return true
	}

	arr1, err1 := v.AsArray()
	arr2, err2 := other.AsArray()
	if err1 != nil || err2 != nil {
		return false
	}

	// 长度不同直接返回false
	if arr1.Length != arr2.Length {
		return false
	}

	// 空数组
	if arr1.Length == 0 {
		return true
	}

	// 优化：先比较首尾元素
	if !arr1.Elements[0].Equal(arr2.Elements[0]) {
		return false
	}

	if arr1.Length > 1 {
		if !arr1.Elements[arr1.Length-1].Equal(arr2.Elements[arr1.Length-1]) {
			return false
		}
	}

	// 最后比较中间元素
	for i := 1; i < arr1.Length-1; i++ {
		if !arr1.Elements[i].Equal(arr2.Elements[i]) {
			return false
		}
	}

	return true
}
