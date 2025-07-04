# AQL 技术设计分析

## 1. 循环语句设计对比

### 1.1 Lua vs Python 循环语句分析

#### **Lua 循环语句**
```lua
-- 数值for循环
for i = 1, 10 do
    print(i)
end

for i = 1, 10, 2 do  -- 步长为2
    print(i)
end

-- 迭代器for循环
for key, value in pairs(table) do
    print(key, value)
end

for index, value in ipairs(array) do
    print(index, value)
end

-- while循环
while condition do
    -- body
end

-- repeat-until循环（类似do-while）
repeat
    -- body
until condition

-- Lua没有原生switch语句
```

#### **Python 循环语句**
```python
# for循环
for i in range(10):
    print(i)

for i in range(1, 11, 2):  # 起始、结束、步长
    print(i)

# 字典迭代
for key, value in dict.items():
    print(key, value)

# 列表迭代
for index, value in enumerate(list):
    print(index, value)

# while循环
while condition:
    # body

# Python 3.10+ match语句（类似switch）
match value:
    case "option1":
        handle_option1()
    case "option2":
        handle_option2()
    case _:  # default
        handle_default()
```

### 1.2 AQL 循环语句设计（推荐方案）

**选择原则**：**采用Lua风格 + Python增强**，理由：
1. **Lua语法更简洁**：`do...end`比Python的缩进更明确
2. **兼容性更好**：括号语法在任何编辑器都能正确显示
3. **扩展性强**：易于添加AQL特有的Agent循环特性

```aql
-- ===== 基础循环（借鉴Lua） =====
-- 数值循环
for i = 1, 10 do
    process(i)
end

for i = 1, 100, 5 do  -- 步长5
    batch_process(i)
end

-- 迭代循环
for key, value in pairs(config) do
    set_parameter(key, value)
end

for index, item in ipairs(task_list) do
    execute_task(index, item)
end

-- while循环
while agents_available() do
    local task = get_next_task()
    assign_task(task)
end

-- ===== AQL增强特性 =====
-- 异步for循环（AQL原创）
for await result in async_generator(data_stream) do
    process_realtime(result)
end

-- 并行for循环（AQL原创）
for parallel item in large_dataset do
    await heavy_computation(item)
end

-- Agent专用循环（AQL原创）
for agent in agent_pool where agent.status == "idle" do
    assign_task(agent, next_task())
end

-- 模式匹配（借鉴Python 3.10）
match task.type
case "analysis" then
    handle_analysis(task)
case "report" then
    handle_report(task)
case t if t.startswith("ml_") then
    handle_ml_task(task)
else
    handle_unknown(task)
end

-- ===== 错误处理循环 =====
-- 重试循环
retry max_attempts = 3, backoff = "exponential" do
    result = await risky_operation()
    if result.success then break end
end

-- 直到成功循环
until success do
    result = await attempt_operation()
    success = result.status == "ok"
    if not success then
        await delay(1000)  -- 等待1秒重试
    end
end
```

### 1.3 循环语法优势分析

| 特性 | Lua风格 | Python风格 | AQL选择 | 理由 |
|------|---------|-------------|---------|------|
| **可读性** | `do...end`明确 | 缩进依赖 | ✅ Lua风格 | 括号更明确，避免缩进问题 |
| **编辑器兼容** | 任何编辑器 | 需要Python支持 | ✅ Lua风格 | 更好的工具兼容性 |
| **语法简洁** | 相对简洁 | 极简 | ⚖️ 平衡 | 保持简洁但不牺牲明确性 |
| **迭代器支持** | `pairs/ipairs` | `items/enumerate` | ✅ 两者融合 | 提供多种迭代方式 |
| **模式匹配** | 无 | `match/case` | ✅ 借鉴Python | 现代语言必备特性 |

## 2. 模块系统设计对比

### 2.1 Lua vs Python 模块系统分析

#### **Lua 模块系统**
```lua
-- 模块定义 (math_utils.lua)
local M = {}

function M.add(a, b)
    return a + b
end

local function private_helper(x)
    return x * 2
end

function M.multiply_by_two(x)
    return private_helper(x)
end

return M

-- 模块使用
local math_utils = require("math_utils")
local result = math_utils.add(1, 2)

-- 或者
local add = require("math_utils").add
local result = add(1, 2)
```

#### **Python 模块系统**
```python
# 模块定义 (math_utils.py)
def add(a, b):
    return a + b

def _private_helper(x):  # 下划线表示私有
    return x * 2

def multiply_by_two(x):
    return _private_helper(x)

__all__ = ['add', 'multiply_by_two']  # 显式导出

# 模块使用
import math_utils
result = math_utils.add(1, 2)

# 或者
from math_utils import add, multiply_by_two
result = add(1, 2)

# 或者
from math_utils import add as math_add
result = math_add(1, 2)
```

### 2.2 AQL 模块系统设计（推荐Python风格）

**选择Python风格的核心理由**：

1. **表达力更强**：支持选择性导入、别名导入
2. **名空间管理更好**：避免全局污染
3. **现代化特性**：符合现代编程语言趋势
4. **生态兼容性**：更容易与Python/JS生态集成
5. **企业级特性**：更好的依赖管理和版本控制

```aql
-- ===== AQL模块系统（基于Python增强） =====

-- 模块定义 (agent_utils.aql)
module agent_utils

-- 私有函数（不导出）
local function validate_agent_config(config)
    assert(config.model ~= nil, "Model is required")
    assert(config.skills ~= nil, "Skills are required")
    return true
end

-- 公有函数
export function create_agent(config)
    validate_agent_config(config)
    
    return {
        id = generate_id(),
        model = config.model,
        skills = config.skills,
        status = "idle"
    }
end

export function deploy_agent(agent, infrastructure)
    assert(agent ~= nil, "Agent is required")
    assert(infrastructure ~= nil, "Infrastructure is required")
    
    local deployment = infrastructure.deploy(agent)
    agent.deployment_id = deployment.id
    
    return deployment
end

-- 类定义也可以导出
export class AgentManager {
    function new(pool_size)
        self.pool_size = pool_size
        self.agents = []
    end
    
    function add_agent(agent)
        if #self.agents >= self.pool_size then
            error("Pool is full")
        end
        self.agents.push(agent)
    end
}

-- 常量导出
export MAX_AGENTS = 100
export DEFAULT_TIMEOUT = "5m"

-- 默认导出
export default {
    create_agent = create_agent,
    deploy_agent = deploy_agent,
    AgentManager = AgentManager
}

-- ===== 模块使用 =====

-- 1. 完整导入
import agent_utils
local agent = agent_utils.create_agent({model: "gpt-4", skills: ["analysis"]})

-- 2. 选择性导入
import {create_agent, deploy_agent} from agent_utils
local agent = create_agent({model: "gpt-4", skills: ["analysis"]})

-- 3. 别名导入
import agent_utils as au
import {create_agent as new_agent} from agent_utils
local agent = new_agent({model: "gpt-4", skills: ["analysis"]})

-- 4. 默认导入
import agent_utils_default from agent_utils
local agent = agent_utils_default.create_agent(config)

-- 5. 通配符导入（不推荐，但支持）
import * from agent_utils  -- 导入所有导出项

-- ===== 包管理系统 =====

-- package.aql
package {
    name = "my_agent_system",
    version = "1.0.0",
    description = "AI Agent orchestration system",
    
    dependencies = {
        "aql-std" = "^1.0.0",        -- 标准库
        "aql-http" = "^0.5.0",       -- HTTP客户端
        "aql-agents" = "^2.1.0",     -- Agent框架
        "openai-api" = "^1.2.0"      -- OpenAI集成
    },
    
    dev_dependencies = {
        "aql-test" = "^1.0.0",       -- 测试框架
        "aql-lint" = "^0.3.0"        -- 代码检查
    },
    
    repositories = [
        "https://packages.aql-lang.org",
        "https://private-repo.company.com/aql"
    ]
}

-- ===== 条件导入（AQL原创特性） =====

-- 基于环境的条件导入
import {MockAgent} from "test_utils" if ENV == "test"
import {ProductionAgent} from "agent_core" if ENV == "production"

-- 基于特性的条件导入
import "gpu_acceleration" if FEATURE_GPU_ENABLED
import "cpu_fallback" if not FEATURE_GPU_ENABLED

-- 异步模块加载
async function load_ml_modules()
    local {TensorFlowAgent} = await import("tensorflow_agents")
    local {PyTorchAgent} = await import("pytorch_agents")
    
    return {
        tensorflow = TensorFlowAgent,
        pytorch = PyTorchAgent
    }
end
```

### 2.3 模块系统优势对比

| 特性 | Lua方式 | Python方式 | AQL选择 | 优势 |
|------|---------|-------------|---------|------|
| **语法简洁性** | ✅ 极简 | ⚖️ 中等 | Python风格 | 可读性 > 简洁性 |
| **选择性导入** | ❌ 不支持 | ✅ 强大 | ✅ Python风格 | 减少名空间污染 |
| **别名支持** | ❌ 基础 | ✅ 完整 | ✅ Python风格 | 避免命名冲突 |
| **包管理** | ❌ 弱 | ✅ 成熟 | ✅ Python风格 | 企业级需求 |
| **循环依赖检测** | ❌ 无 | ✅ 有 | ✅ 增强 | 避免运行时错误 |
| **热重载** | ❌ 困难 | ⚖️ 可行 | ✅ 原生支持 | 开发效率 |

## 3. MVP Stackframe 设计

### 3.1 为什么从Stackframe开始？

**Stackframe是语言运行时的核心**，包含：
1. **执行上下文**：当前函数的局部变量、参数
2. **控制流信息**：返回地址、异常处理
3. **作用域链**：变量查找、闭包支持
4. **调试信息**：行号、文件名、调用栈

对于AQL，还需要额外支持：
- **Agent上下文**：当前执行的Agent信息
- **异步状态**：Promise、协程状态
- **资源跟踪**：内存、网络、计算资源

### 3.2 AQL Stackframe 结构定义

```aql
-- ===== Stackframe 核心结构 =====
struct AQLStackFrame {
    -- ===== 基础执行信息 =====
    frame_id: string,              -- 唯一标识符
    frame_type: FrameType,         -- 帧类型枚举
    parent_frame: *AQLStackFrame,  -- 父帧指针
    
    -- ===== 函数执行信息 =====
    function_name: string,         -- 函数名
    function_signature: string,    -- 函数签名
    instruction_pointer: usize,    -- 当前指令位置
    return_address: usize,         -- 返回地址
    
    -- ===== 变量和作用域 =====
    local_variables: HashMap<string, Value>,    -- 局部变量
    parameters: Array<Value>,                   -- 函数参数
    upvalues: Array<*Upvalue>,                 -- 闭包捕获的变量
    
    -- ===== 控制流状态 =====
    exception_handlers: Array<ExceptionHandler>,  -- 异常处理器
    loop_stack: Array<LoopInfo>,                  -- 循环嵌套信息
    
    -- ===== AQL特有：Agent上下文 =====
    agent_context: ?AgentContext,              -- 当前Agent上下文
    task_context: ?TaskContext,                -- 当前任务上下文
    
    -- ===== AQL特有：异步状态 =====
    async_state: ?AsyncState,                  -- 异步执行状态
    await_point: ?AwaitPoint,                  -- await挂起点
    
    -- ===== 调试信息 =====
    debug_info: DebugInfo,                     -- 源码位置等
    
    -- ===== 资源跟踪 =====
    resource_tracker: ResourceTracker,        -- 资源使用追踪
}

-- ===== 相关枚举和结构 =====
enum FrameType {
    Function,           -- 普通函数调用
    AsyncFunction,      -- 异步函数调用
    AgentMethod,        -- Agent方法调用
    CoroutineFrame,     -- 协程帧
    ModuleInit,         -- 模块初始化
    ExceptionHandler,   -- 异常处理
}

struct AgentContext {
    agent_id: string,
    agent_type: string,
    model_config: ModelConfig,
    current_skills: Array<string>,
    memory_state: MemoryState,
}

struct TaskContext {
    task_id: string,
    task_type: string,
    priority: Priority,
    deadline: ?Timestamp,
    dependencies: Array<string>,
}

struct AsyncState {
    promise_id: string,
    state: PromiseState,  -- Pending/Fulfilled/Rejected
    result: ?Value,
    continuation: ?Continuation,
}

struct AwaitPoint {
    await_id: string,
    awaited_promise: string,
    resume_instruction: usize,
}

struct DebugInfo {
    file_name: string,
    line_number: usize,
    column_number: usize,
    source_snippet: string,
}

struct ResourceTracker {
    memory_usage: usize,
    cpu_time: Duration,
    network_calls: usize,
    agent_interactions: usize,
}
```

### 3.3 MVP Stackframe 实现策略

#### **阶段1：基础Stackframe（Week 1-2）**
```rust
// 简化版本，只包含核心功能
struct MVPStackFrame {
    // 基础执行信息
    frame_id: String,
    function_name: String,
    parent: Option<Box<MVPStackFrame>>,
    
    // 变量存储
    locals: HashMap<String, Value>,
    
    // 调试信息
    line_number: usize,
    file_name: String,
}

// 基础Value类型
enum Value {
    Nil,
    Boolean(bool),
    Number(f64),
    String(String),
    Array(Vec<Value>),
    Object(HashMap<String, Value>),
    Function(Function),
}
```

#### **阶段2：异步支持（Week 3-4）**
```rust
// 添加异步支持
struct AsyncFrame {
    base: MVPStackFrame,
    
    // 异步状态
    async_state: AsyncState,
    continuation: Option<Continuation>,
}

enum AsyncState {
    Running,
    Suspended(AwaitInfo),
    Completed(Value),
    Failed(Error),
}
```

#### **阶段3：Agent支持（Week 5-6）**
```rust
// 添加Agent上下文
struct AgentFrame {
    base: AsyncFrame,
    
    // Agent信息
    agent_id: String,
    agent_config: AgentConfig,
    current_task: Option<TaskInfo>,
}
```

### 3.4 Stackframe 操作接口

```aql
-- ===== Stackframe 管理接口 =====
class StackFrameManager {
    -- 创建新帧
    function push_frame(frame_type, function_name, debug_info)
        local frame = AQLStackFrame.new {
            frame_id = generate_uuid(),
            frame_type = frame_type,
            function_name = function_name,
            parent_frame = self.current_frame,
            debug_info = debug_info,
            local_variables = HashMap.new(),
            resource_tracker = ResourceTracker.new()
        }
        
        self.frame_stack.push(frame)
        self.current_frame = frame
        
        return frame
    end
    
    -- 弹出帧
    function pop_frame()
        assert(#self.frame_stack > 0, "Cannot pop empty stack")
        
        local frame = self.frame_stack.pop()
        self.current_frame = self.frame_stack.top()
        
        -- 清理资源
        frame.resource_tracker.cleanup()
        
        return frame
    end
    
    -- 变量操作
    function set_local(name, value)
        self.current_frame.local_variables[name] = value
    end
    
    function get_local(name)
        return self.current_frame.local_variables[name]
    end
    
    function get_variable(name)
        -- 从当前帧开始向上查找
        local frame = self.current_frame
        while frame ~= nil do
            if frame.local_variables[name] ~= nil then
                return frame.local_variables[name]
            end
            frame = frame.parent_frame
        end
        
        error("Undefined variable: " .. name)
    end
    
    -- 异步支持
    function suspend_frame(await_point)
        assert(self.current_frame.frame_type == FrameType.AsyncFunction)
        
        self.current_frame.async_state.state = PromiseState.Pending
        self.current_frame.await_point = await_point
        
        -- 保存continuation
        self.save_continuation()
    end
    
    function resume_frame(result)
        assert(self.current_frame.await_point ~= nil)
        
        -- 恢复执行状态
        self.current_frame.async_state.state = PromiseState.Fulfilled
        self.current_frame.async_state.result = result
        
        -- 继续执行
        self.restore_continuation()
    end
}
```

### 3.5 MVP开发计划

```aql
-- ===== MVP 开发里程碑 =====
milestone mvp_phase1 {
    duration = "2 weeks"
    deliverables = [
        "基础Stackframe结构",
        "简单函数调用支持", 
        "局部变量管理",
        "基础调试信息"
    ]
    
    success_criteria = {
        "能执行简单的AQL函数",
        "支持递归调用",
        "变量作用域正确",
        "调用栈可追踪"
    }
}

milestone mvp_phase2 {
    duration = "2 weeks"
    depends_on = [mvp_phase1]
    deliverables = [
        "异步函数支持",
        "Promise/Future实现",
        "基础协程支持"
    ]
    
    success_criteria = {
        "支持async/await语法",
        "异步调用栈正确",
        "Promise状态管理"
    }
}

milestone mvp_phase3 {
    duration = "2 weeks" 
    depends_on = [mvp_phase2]
    deliverables = [
        "Agent上下文集成",
        "任务调度基础",
        "基础资源跟踪"
    ]
    
    success_criteria = {
        "Agent函数调用正常",
        "任务上下文传递",
        "资源使用监控"
    }
}
```

**总结**：从Stackframe开始是正确的MVP策略，因为它是语言运行时的基础。通过渐进式实现，可以快速验证核心概念，然后逐步添加AQL的特有特性（Agent、异步、资源管理）。 