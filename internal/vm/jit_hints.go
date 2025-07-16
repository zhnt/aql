package vm

// JIT提示系统：为将来的JIT编译提供性能提示

// JITHint JIT编译提示
type JITHint struct {
	Type   string                 // 提示类型
	Target string                 // 目标（函数名、闭包等）
	Data   map[string]interface{} // 提示数据
}

// JITHintCollector JIT提示收集器
type JITHintCollector struct {
	hints         []JITHint
	functionStats map[string]*FunctionStats
	closureStats  map[string]*ClosureStats
	hotSpots      []string
	coldSpots     []string
	enabled       bool
}

// FunctionStats 函数统计信息
type FunctionStats struct {
	CallCount       int64   // 调用次数
	TotalTime       float64 // 总执行时间（毫秒）
	AvgTime         float64 // 平均执行时间
	InlineCandidate bool    // 是否为内联候选
	JITCandidate    bool    // 是否为JIT候选
}

// ClosureStats 闭包统计信息
type ClosureStats struct {
	CreateCount    int64 // 创建次数
	CallCount      int64 // 调用次数
	CaptureCount   int   // 捕获变量数量
	StackAllocated bool  // 是否栈分配
	HotPath        bool  // 是否在热路径上
}

// NewJITHintCollector 创建新的JIT提示收集器
func NewJITHintCollector() *JITHintCollector {
	return &JITHintCollector{
		hints:         make([]JITHint, 0),
		functionStats: make(map[string]*FunctionStats),
		closureStats:  make(map[string]*ClosureStats),
		hotSpots:      make([]string, 0),
		coldSpots:     make([]string, 0),
		enabled:       false, // 默认禁用，避免性能开销
	}
}

// Enable 启用JIT提示收集
func (jhc *JITHintCollector) Enable() {
	jhc.enabled = true
}

// Disable 禁用JIT提示收集
func (jhc *JITHintCollector) Disable() {
	jhc.enabled = false
}

// AddHint 添加JIT提示
func (jhc *JITHintCollector) AddHint(hintType, target string, data map[string]interface{}) {
	if !jhc.enabled {
		return
	}

	hint := JITHint{
		Type:   hintType,
		Target: target,
		Data:   data,
	}
	jhc.hints = append(jhc.hints, hint)
}

// RecordFunctionCall 记录函数调用
func (jhc *JITHintCollector) RecordFunctionCall(functionName string, executionTime float64) {
	if !jhc.enabled {
		return
	}

	stats, exists := jhc.functionStats[functionName]
	if !exists {
		stats = &FunctionStats{
			CallCount:       0,
			TotalTime:       0,
			AvgTime:         0,
			InlineCandidate: false,
			JITCandidate:    false,
		}
		jhc.functionStats[functionName] = stats
	}

	stats.CallCount++
	stats.TotalTime += executionTime
	stats.AvgTime = stats.TotalTime / float64(stats.CallCount)

	// 热点检测
	if stats.CallCount > 100 && stats.AvgTime > 0.1 {
		stats.JITCandidate = true
		jhc.addToHotSpots(functionName)
	}

	// 内联候选检测
	if stats.CallCount > 50 && stats.AvgTime < 0.01 {
		stats.InlineCandidate = true
	}
}

// RecordClosureCreate 记录闭包创建
func (jhc *JITHintCollector) RecordClosureCreate(closureName string, captureCount int) {
	if !jhc.enabled {
		return
	}

	stats, exists := jhc.closureStats[closureName]
	if !exists {
		stats = &ClosureStats{
			CreateCount:    0,
			CallCount:      0,
			CaptureCount:   captureCount,
			StackAllocated: false,
			HotPath:        false,
		}
		jhc.closureStats[closureName] = stats
	}

	stats.CreateCount++

	// 栈分配提示
	if captureCount <= 3 && stats.CreateCount > 10 {
		stats.StackAllocated = true
		jhc.AddHint("stack_alloc", closureName, map[string]interface{}{
			"capture_count": captureCount,
			"create_count":  stats.CreateCount,
		})
	}
}

// RecordClosureCall 记录闭包调用
func (jhc *JITHintCollector) RecordClosureCall(closureName string) {
	if !jhc.enabled {
		return
	}

	stats, exists := jhc.closureStats[closureName]
	if !exists {
		return
	}

	stats.CallCount++

	// 热路径检测
	if stats.CallCount > 200 {
		stats.HotPath = true
		jhc.AddHint("hot_closure", closureName, map[string]interface{}{
			"call_count": stats.CallCount,
		})
	}
}

// addToHotSpots 添加到热点列表
func (jhc *JITHintCollector) addToHotSpots(functionName string) {
	// 避免重复添加
	for _, hotSpot := range jhc.hotSpots {
		if hotSpot == functionName {
			return
		}
	}
	jhc.hotSpots = append(jhc.hotSpots, functionName)
}

// GetHotSpots 获取热点函数列表
func (jhc *JITHintCollector) GetHotSpots() []string {
	return jhc.hotSpots
}

// GetInlineCandidates 获取内联候选函数
func (jhc *JITHintCollector) GetInlineCandidates() []string {
	candidates := make([]string, 0)
	for name, stats := range jhc.functionStats {
		if stats.InlineCandidate {
			candidates = append(candidates, name)
		}
	}
	return candidates
}

// GetJITCandidates 获取JIT编译候选函数
func (jhc *JITHintCollector) GetJITCandidates() []string {
	candidates := make([]string, 0)
	for name, stats := range jhc.functionStats {
		if stats.JITCandidate {
			candidates = append(candidates, name)
		}
	}
	return candidates
}

// GetStackAllocCandidates 获取栈分配候选闭包
func (jhc *JITHintCollector) GetStackAllocCandidates() []string {
	candidates := make([]string, 0)
	for name, stats := range jhc.closureStats {
		if stats.StackAllocated {
			candidates = append(candidates, name)
		}
	}
	return candidates
}

// GenerateOptimizationPlan 生成优化计划
func (jhc *JITHintCollector) GenerateOptimizationPlan() map[string]interface{} {
	plan := map[string]interface{}{
		"hot_spots":              jhc.GetHotSpots(),
		"inline_candidates":      jhc.GetInlineCandidates(),
		"jit_candidates":         jhc.GetJITCandidates(),
		"stack_alloc_candidates": jhc.GetStackAllocCandidates(),
		"total_hints":            len(jhc.hints),
	}

	return plan
}

// Reset 重置所有统计信息
func (jhc *JITHintCollector) Reset() {
	jhc.hints = make([]JITHint, 0)
	jhc.functionStats = make(map[string]*FunctionStats)
	jhc.closureStats = make(map[string]*ClosureStats)
	jhc.hotSpots = make([]string, 0)
	jhc.coldSpots = make([]string, 0)
}

// GetStats 获取统计摘要
func (jhc *JITHintCollector) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":        jhc.enabled,
		"total_hints":    len(jhc.hints),
		"function_count": len(jhc.functionStats),
		"closure_count":  len(jhc.closureStats),
		"hot_spot_count": len(jhc.hotSpots),
	}
}
