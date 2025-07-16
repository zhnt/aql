package vm

import (
	"fmt"
	"sync"
)

// FunctionRegistry 全局函数注册表
// 存储所有编译的Function对象，为ValueGC函数提供完整的Function数据
type FunctionRegistry struct {
	mu        sync.RWMutex
	functions map[int]*Function
	nextID    int
}

// 全局函数注册表实例
var GlobalFunctionRegistry *FunctionRegistry

// InitFunctionRegistry 初始化全局函数注册表
func InitFunctionRegistry() {
	GlobalFunctionRegistry = &FunctionRegistry{
		functions: make(map[int]*Function),
		nextID:    1, // 从1开始，0保留为无效ID
	}
}

// RegisterFunction 注册函数并返回ID
func (fr *FunctionRegistry) RegisterFunction(function *Function) int {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	id := fr.nextID
	fr.functions[id] = function
	fr.nextID++

	return id
}

// GetFunction 根据ID获取函数
func (fr *FunctionRegistry) GetFunction(id int) (*Function, error) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	function, exists := fr.functions[id]
	if !exists {
		return nil, fmt.Errorf("function with ID %d not found", id)
	}

	return function, nil
}

// GetFunctionCount 获取注册的函数数量
func (fr *FunctionRegistry) GetFunctionCount() int {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	return len(fr.functions)
}

// Clear 清空注册表（用于测试）
func (fr *FunctionRegistry) Clear() {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	fr.functions = make(map[int]*Function)
	fr.nextID = 1
}

// 便利方法：全局注册函数
func RegisterFunction(function *Function) int {
	if GlobalFunctionRegistry == nil {
		InitFunctionRegistry()
	}
	return GlobalFunctionRegistry.RegisterFunction(function)
}

// 便利方法：全局获取函数
func GetFunction(id int) (*Function, error) {
	if GlobalFunctionRegistry == nil {
		return nil, fmt.Errorf("function registry not initialized")
	}
	return GlobalFunctionRegistry.GetFunction(id)
}
