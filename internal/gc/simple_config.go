package gc

// UnifiedAllocatorConfig 统一分配器配置
type UnifiedAllocatorConfig struct {
	EnableDebug       bool   // 启用调试输出
	DefaultRegionSize uint32 // 默认内存区域大小
}

// DefaultUnifiedConfig 默认统一配置
var DefaultUnifiedConfig = UnifiedAllocatorConfig{
	EnableDebug:       true,      // 暂时启用调试
	DefaultRegionSize: 64 * 1024, // 64KB
}
