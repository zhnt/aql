# AQL双重执行架构 - 命令行使用指南

## 基本用法

### 1. 自动模式（推荐）
```bash
# 自动选择最佳执行模式
aql run script.aql

# 启用性能分析
aql run --profile script.aql

# 启用调试模式
aql run --debug script.aql
```

### 2. 指定执行模式
```bash
# 解释执行模式（开发阶段）
aql run --mode=interpret script.aql

# 编译执行模式（生产阶段）
aql run --mode=compile script.aql

# JIT编译模式（平衡性能）
aql run --mode=jit script.aql

# 混合模式（异步密集）
aql run --mode=hybrid script.aql
```

### 3. 配置文件使用
```bash
# 使用默认配置文件
aql run --config=aql_config.json script.aql

# 使用预设配置
aql run --profile=development script.aql
aql run --profile=production script.aql
aql run --profile=high_performance script.aql
```

## 开发场景示例

### 开发阶段
```bash
# 快速启动，易于调试
aql run --mode=interpret --debug \
       --breakpoint=main:10 \
       --watch=user_input \
       script.aql

# 实时代码重载
aql run --mode=interpret --watch \
       --auto-reload \
       script.aql
```

### 测试阶段
```bash
# 性能测试
aql run --mode=auto --profile \
       --benchmark=10 \
       script.aql

# 压力测试
aql run --mode=compile --stress \
       --concurrent=100 \
       script.aql
```

### 生产部署
```bash
# 编译为单一可执行文件
aql build --mode=compile --output=app script.aql

# 运行编译后的程序
./app

# 容器化部署
aql run --mode=compile --production \
       --config=production.json \
       script.aql
```

## AI服务相关示例

### AI服务调用
```bash
# 配置AI服务
aql run --ai-service=openai:gpt-4 \
       --ai-service=anthropic:claude-3 \
       ai_workflow.aql

# 并行AI服务调用
aql run --mode=hybrid --parallel-ai \
       --timeout=30s \
       ai_parallel.aql

# AI服务链式调用
aql run --mode=auto --ai-chain \
       --retry=3 \
       ai_chain.aql
```

### 异步AI工作流
```bash
# 异步执行模式
aql run --mode=hybrid --async \
       --coroutines=1000 \
       async_workflow.aql

# 事件循环调优
aql run --mode=hybrid --event-loop=4 \
       --promise-pool=100 \
       event_driven.aql
```

## 性能优化示例

### JIT编译优化
```bash
# 启用JIT编译
aql run --mode=jit --hotspot-threshold=50 \
       --compile-delay=100ms \
       compute_intensive.aql

# 渐进式优化
aql run --mode=jit --progressive \
       --optimization-level=O3 \
       long_running.aql
```

### 内存和GC优化
```bash
# 自定义GC设置
aql run --mode=compile --gc=hybrid \
       --heap-size=1GB \
       --max-pause=1ms \
       memory_intensive.aql

# 低延迟设置
aql run --mode=compile --gc=low-latency \
       --young-gen=64MB \
       --old-gen=256MB \
       realtime_app.aql
```

## 调试和分析

### 调试模式
```bash
# 步进调试
aql run --mode=interpret --debug \
       --step-mode \
       --trace \
       debug_script.aql

# 内存调试
aql run --mode=interpret --debug \
       --memory-debug \
       --gc-debug \
       memory_leak.aql
```

### 性能分析
```bash
# CPU分析
aql run --mode=auto --profile=cpu \
       --profile-duration=60s \
       performance_test.aql

# 内存分析
aql run --mode=auto --profile=memory \
       --memory-profile \
       memory_test.aql

# 全面性能分析
aql run --mode=auto --profile=all \
       --export-profile=profile.json \
       comprehensive_test.aql
```

## 高级功能

### 多文件项目
```bash
# 项目模式
aql run --project=. --main=main.aql

# 依赖管理
aql run --project=. --deps=install

# 模块化执行
aql run --mode=compile --modules \
       --optimize-modules \
       modular_app.aql
```

### 并发和分布式
```bash
# 并发执行
aql run --mode=hybrid --concurrent=10 \
       --worker-pool=4 \
       concurrent_app.aql

# 分布式执行
aql run --mode=compile --distributed \
       --nodes=cluster.yaml \
       distributed_app.aql
```

## 配置文件示例

### development.json
```json
{
  "execution": {
    "default_mode": "interpret",
    "auto_switch": false,
    "enable_profiling": false
  },
  "debugging": {
    "enabled": true,
    "trace_execution": true
  }
}
```

### production.json
```json
{
  "execution": {
    "default_mode": "compile",
    "auto_switch": true,
    "enable_profiling": true
  },
  "debugging": {
    "enabled": false
  },
  "performance": {
    "gc": {
      "mode": "production",
      "max_pause_time": "0.5ms"
    }
  }
}
```

## 环境变量配置

```bash
# 设置默认执行模式
export AQL_DEFAULT_MODE=auto

# 设置AI服务配置
export AQL_AI_SERVICES="openai:gpt-4,anthropic:claude-3"

# 设置性能配置
export AQL_HEAP_SIZE=1GB
export AQL_GC_MODE=hybrid

# 设置调试选项
export AQL_DEBUG=true
export AQL_TRACE=false
```

## 实际使用场景

### 1. 开发一个聊天机器人
```bash
# 开发阶段
aql run --mode=interpret --debug \
       --ai-service=openai:gpt-4 \
       --watch --auto-reload \
       chatbot.aql

# 测试阶段
aql run --mode=auto --profile \
       --ai-service=openai:gpt-4 \
       --concurrent=10 \
       chatbot.aql

# 生产部署
aql build --mode=compile --output=chatbot \
         --ai-service=openai:gpt-4 \
         --production \
         chatbot.aql
```

### 2. 开发数据分析管道
```bash
# 开发阶段
aql run --mode=interpret --debug \
       --memory-debug \
       data_pipeline.aql

# 性能优化
aql run --mode=jit --hotspot-threshold=20 \
       --optimization-level=O3 \
       data_pipeline.aql

# 生产运行
aql run --mode=compile --production \
       --heap-size=2GB \
       --gc=low-latency \
       data_pipeline.aql
```

### 3. 开发微服务
```bash
# 开发阶段
aql run --mode=interpret --debug \
       --port=8080 \
       --hot-reload \
       microservice.aql

# 容器化部署
aql build --mode=compile --output=service \
         --container \
         microservice.aql

# 运行容器
docker run -p 8080:8080 aql-service
```

## 最佳实践

1. **开发阶段**：使用 `--mode=interpret` 和 `--debug`
2. **测试阶段**：使用 `--mode=auto` 和 `--profile`
3. **生产阶段**：使用 `--mode=compile` 和优化配置
4. **性能调优**：使用 `--mode=jit` 和性能分析工具
5. **调试问题**：使用 `--debug` 和 `--trace` 选项

这个双重执行架构让AQL既保持了开发的灵活性，又提供了生产环境的高性能，是现代编程语言的理想选择。 