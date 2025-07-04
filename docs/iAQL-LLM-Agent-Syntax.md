# iAQL: 意图驱动AI编程语言
# Intent-driven AI Query Language

## 语言设计理念

iAQL采用声明式、图形化思维的语法设计：

```
设计原则:
📝 声明式优先 - 描述想要什么，而非如何实现
🔀 流程图思维 - 用连接符表达逻辑关系
🏗️ 分层结构 - 清晰的语法层次和作用域
🎯 语义明确 - 关键字直接对应AI领域概念
🔧 IPDSL兼容 - 兼容IPDSL(星际领域专用语言)基础设施
```

## 1. 关键字体系 (Keywords)

### 1.1 顶层关键字 (Top-level Keywords)

```iaql
# AI实体定义
agent           # 定义智能代理 (解析为@agent类型的块)
llm             # 定义语言模型 (解析为@llm类型的块)
rag             # 定义检索增强生成 (解析为@rag类型的块)
workflow        # 定义工作流 (解析为@workflow类型的块)
conversation    # 定义对话 (解析为@conversation类型的块)

# 执行控制  
start_execution # 开始执行
end_execution   # 结束执行
loop_block      # 循环执行
if_block        # 条件执行
parallel_block  # 并行执行

# 声明性配置
config          # 全局配置
import          # 导入模块
export          # 导出接口
```

### 1.2 行为关键字 (Action Keywords)

```iaql
# LLM操作
ask             # 询问LLM
generate        # 生成内容
complete        # 补全文本
translate       # 翻译
summarize       # 摘要

# Agent行为
think           # 思考分析
plan            # 制定计划
execute         # 执行任务
reflect         # 反思总结
learn           # 学习更新

# 数据操作
retrieve        # 检索信息
filter          # 过滤数据
transform       # 转换数据
validate        # 验证数据
store           # 存储数据
```

### 1.3 修饰关键字 (Modifier Keywords)

```iaql
# 执行策略
competitive     # 竞争式执行
ensemble        # 集成式执行
sequential      # 顺序执行
adaptive        # 自适应执行

# 质量控制
with_validation # 带验证
with_fallback   # 带回退
with_retry      # 带重试
with_timeout    # 带超时

# 上下文管理
with_memory     # 带记忆
with_context    # 带上下文
with_history    # 带历史
```

## 2. 操作符体系 (Operators)

### 2.1 流程操作符 (Flow Operators)

借鉴PlantUML的箭头设计，确保HCL兼容：

```iaql
# 基础流程 (Basic Flow)
->              # 数据流转 (data flow)
->>             # 异步流转 (async flow)  
-->             # 弱依赖流转 (weak dependency)
-*>             # 广播流转 (broadcast)

# 条件流程 (Conditional Flow)
-?>             # 条件流转 (conditional)
-!>             # 异常流转 (exception)
-@>             # 循环流转 (loop back)

# 分组操作符 (Grouping)
|||             # 并行分组 (parallel group)
===             # 顺序分组 (sequential group)  
~~~             # 循环分组 (loop group)
```

### 2.2 关系操作符 (Relationship Operators)

```iaql
# 层次关系（HCL兼容模式）
is              # 身份关系 (仅在flow中使用): agent is "data_analyst"
has             # 拥有关系: agent has skills
contains        # 包含关系: workflow contains steps
uses            # 使用关系: agent uses llm

# 配置赋值（HCL标准模式）
=               # 配置赋值: role = "data_analyst"

# 逻辑关系  
and             # 逻辑与
or              # 逻辑或
not             # 逻辑非
when            # 条件判断
unless          # 反向条件

# 数量关系
all             # 全部
any             # 任意
none            # 无
some            # 一些
most            # 大部分
```

### 2.3 执行操作符 (Execution Operators)

```iaql
# 策略选择 (基于你的原创语法)
N->K            # N选K策略: 5->1 (5个选1个)
N->K->merge     # N选K合并: 3->2->merge
N->all->X       # N全选操作: 5->all->evaluate

# 控制流
repeat          # 重复执行
until           # 直到条件
while           # 当条件时
foreach         # 遍历执行
```

## 3. 语法层次结构

### 3.1 文档结构（HCL兼容）

```iaql
# 全局配置
config {
    default_llm = "gpt-4"
    debug_mode = true
}

# 模块导入
import {
    modules = ["./agents", "./workflows"]
}

# AI实体定义区
agent "analyst" { ... }
llm "gpt4" { ... }
workflow "analysis" { ... }

# 执行流程区
start_execution "main" {
    # 主要逻辑
}
```

### 3.2 实体定义语法（HCL兼容）

```iaql
# Agent定义模板
agent "analyst" {
    # 基础配置（HCL标准语法）
    role = "数据分析师"
    model = "gpt-4"
    temperature = 0.7
    
    # iAQL标签支持
    tags = {
        domain = "finance"
        expertise = "statistics"
        team = "data_science"
    }
    
    capabilities {
        skills = ["analysis", "visualization", "reporting"]
    }
    
    llm_config {
        primary = "gpt-4"
        fallback = "claude-3"
        max_tokens = 2000
    }
    
    # 行为定义（仍保持iAQL特色）
    behavior {
        flow = "input -> think -> plan -> execute -> reflect"
    }
}

# LLM定义模板（HCL兼容）
llm "gpt4" {
    model = "gpt-4"
    provider = "openai"
    
    parameters {
        temperature = 0.7
        max_tokens = 2000
    }
    
    strategies {
        competitive = "3->1->auto"
        ensemble = "3->all->merge"
    }
    
    tags = {
        cost = "high"
        quality = "premium"
        speed = "medium"
    }
}
```

## 4. 核心语法模式

### 4.1 简单交互模式

```iaql
# 最简LLM调用
ask "gpt4" "解释量子计算"

# 带策略的调用
ask {
    type = "competitive"
    models = ["gpt4", "claude", "gemini"]
    strategy = "3->1->auto"
    question = "什么是区块链？"
}

# Agent交互
analyst {
    action = "think about '市场趋势' then generate report"
}
```

### 4.2 工作流模式（HCL兼容）

```iaql
workflow "content_creation" {
    participants {
        researcher = "research_agent"
        writer = "writer_agent"
        editor = "editor_agent"
    }
    
    tags = {
        category = "content"
        priority = "high"
    }
    
    flow {
        steps = [
            "researcher -> gather_information -> data",
            "data -> writer -> create_draft -> draft", 
            "draft -> editor -> review_edit -> final_content"
        ]
    }
}
```

### 4.3 并行协作模式（HCL兼容）

```iaql
parallel_block "market_analysis" {
    concurrent_tasks = [
        "technical_analyst -> analyze_charts",
        "fundamental_analyst -> analyze_financials", 
        "sentiment_analyst -> analyze_news"
    ]
    
    merge_strategy = "weighted_combination"
    
    tags = {
        analysis_type = "comprehensive"
        time_sensitive = true
    }
}
```

### 4.4 循环优化模式（HCL兼容）

```iaql
loop_block "strategy_optimization" {
    initial_action = "execute_strategy"
    
    condition {
        expression = "performance < target"
        max_iterations = 10
    }
    
    optimization {
        variants = 5
        selection = "best_performer"
    }
    
    tags = {
        optimization_type = "performance"
        domain = "strategy"
    }
}
```

## 5. 高级语法特性

### 5.1 条件分支（HCL兼容）

```iaql
if_block "market_condition" {
    condition = "market_volatility > threshold"
    
    then_action {
        strategy = "conservative_approach"
    }
    
    else_action {
        strategy = "aggressive_approach"
    }
    
    tags = {
        decision_type = "risk_management"
    }
}

# 多条件分支
switch_block "user_expertise" {
    cases = {
        beginner = "simple_explanation"
        intermediate = "detailed_analysis"
        expert = "technical_details"
    }
    default = "adaptive_response"
    
    tags = {
        personalization = true
    }
}
```

### 5.2 异常处理（HCL兼容）

```iaql
try_block "risky_operation" {
    main_action = "high_risk_operation"
    
    catch_rules = {
        network_error = "fallback_method"
        timeout_error = "retry_with_backoff"
    }
    
    finally_action = "cleanup_resources"
    
    tags = {
        error_handling = "comprehensive"
        reliability = "high"
    }
}
```

### 5.3 记忆和上下文（HCL兼容）

```iaql
conversation "customer_support" {
    memory = "persistent"
    context_window = 10
    
    participants = {
        user = "customer"
        agent = "support_agent"
    }
    
    agent_config {
        memory_enabled = true
        context_aware = true
    }
    
    tags = {
        conversation_type = "support"
        priority = "customer_satisfaction"
    }
}
```

## 6. RAG专用语法

### 6.1 基础RAG流程（HCL兼容）

```iaql
rag "knowledge_qa" {
    knowledge_base = "company_docs"
    
    pipeline {
        steps = [
            "query -> expand_keywords",
            "search_docs -> rank_results",
            "select_top_5 -> summarize_context",
            "generate_answer -> with_citations"
        ]
    }
    
    tags = {
        rag_type = "qa"
        data_source = "internal"
    }
}
```

### 6.2 多源RAG（HCL兼容）

```iaql
rag "multi_source_research" {
    sources = {
        technical_docs = 0.4
        user_manuals = 0.3
        community_qa = 0.2
        expert_opinions = 0.1
    }
    
    fusion_strategy = "weighted_rank_merge"
    
    tags = {
        rag_type = "multi_source"
        complexity = "high"
    }
}
```

## 7. Agent编排语法

### 7.1 团队协作（HCL兼容）

```iaql
team "development_team" {
    members = {
        product_manager = "pm_agent"
        architect = "arch_agent"
        developer = "dev_agent"
        tester = "test_agent"
    }
    
    collaboration {
        workflow = [
            "product_manager -> define_requirements -> requirements",
            "requirements -> architect -> design_system -> architecture",
            "architecture -> developer -> implement_code -> code",
            "code -> tester -> validate_quality -> feedback"
        ]
    }
    
    tags = {
        team_type = "development"
        methodology = "agile"
    }
}
```

### 7.2 智能调度（HCL兼容）

```iaql
scheduler "task_distributor" {
    queue = "incoming_tasks"
    
    allocation_strategy = {
        method = "workload_balancing"
        criteria = "expertise_matching"
    }
    
    rules = {
        high_priority = "senior_agent"
        complex_task = "collaborative_solve"
        routine_task = "junior_agent"
    }
    
    tags = {
        scheduler_type = "intelligent"
        optimization = "load_balancing"
    }
}
```

## 8. 系统工程模式

### 8.1 完整开发流程（HCL兼容）

```iaql
workflow "system_development" {
    intent = "构建交易系统"
    
    phases = {
        requirements = {
            stakeholders = ["customer", "analyst", "architect"]
            output = "requirements_doc"
        }
        
        design = {
            input = "requirements_doc"
            participants = ["architect", "tech_lead"]
            strategy = "5_designs -> select_best_2 -> hybrid_approach"
            output = "system_design"
        }
        
        implementation = {
            input = "system_design"
            teams = ["backend_team", "frontend_team", "ai_team"]
            mode = "parallel_development"
            output = "system_components"
        }
        
        integration = {
            input = "system_components"
            strategy = "incremental_integration with_continuous_testing"
            output = "integrated_system"
        }
        
        deployment = {
            input = "integrated_system"
            stages = ["staging", "production"]
            strategy = "blue_green_deployment with_monitoring"
        }
    }
    
    quality_assurance {
        process = "each_phase -> multi_agent_review -> 3_validators -> consensus"
        feedback_loop = true
    }
    
    tags = {
        project_type = "system_development"
        complexity = "high"
        methodology = "iterative"
    }
}
```

## 9. 配置和管理

### 9.1 全局配置（HCL兼容）

```iaql
config {
    # LLM配置
    default_llm = "gpt-4"
    fallback_llm = "claude-3"
    
    # 策略配置
    default_strategies = {
        competitive = "5->1->auto"
        ensemble = "3->all->merge"
        consensus = "3->majority->decide"
    }
    
    # 质量配置
    quality_thresholds = {
        response_time = "10s"
        accuracy = 0.85
        user_satisfaction = 0.8
    }
    
    tags = {
        config_type = "global"
        environment = "production"
    }
}
```

### 9.2 监控和指标（HCL兼容）

```iaql
monitor "system_health" {
    metrics = {
        performance = ["response_time", "throughput", "accuracy"]
        resource = ["cpu_usage", "memory_usage", "api_calls"]
        business = ["user_satisfaction", "task_completion_rate"]
    }
    
    alerts = {
        high_latency = "response_time > 10s"
        low_accuracy = "accuracy < 0.8"
        system_overload = "cpu_usage > 90%"
    }
    
    auto_scaling = {
        scale_up = "load > 80%"
        scale_down = "load < 20%"
    }
    
    tags = {
        monitor_type = "comprehensive"
        alert_enabled = true
    }
}
```

## 10. 极简表达模式

### 10.1 一行表达（函数调用形式）

```iaql
# 快捷PPT生成
ppt_generate {
    topic = "AI革命"
    slides = 25
    style = "consultant"
    auto_generate = true
}

# 快捷系统开发
system_develop {
    type = "交易系统"
    phases = 3
    mode = "parallel"
    target = "production"
}

# 快捷Agent对话  
agent_task {
    agent = "trader"
    task = "分析BTC走势"
    output_format = "report_with_charts"
}
```

### 10.2 语法糖（HCL兼容）

```iaql
# 快捷定义
quick_agent "data_analyst" {
    skills = ["analysis", "visualization"]
    auto_ready = true
    
    tags = {
        setup = "quick"
    }
}

quick_llm "gpt4" {
    temperature = 0.7
    auto_ready = true
    
    tags = {
        model_type = "premium"
    }
}

quick_workflow "content" {
    agents = ["researcher", "writer"]
    mode = "sequential"
    
    tags = {
        workflow_type = "content_creation"
    }
}
```

## 11. 完整示例

### 11.1 智能交易系统（HCL兼容）

```iaql
# 全局配置
config {
    default_llm = "gpt-4"
    risk_tolerance = 0.05
    
    tags = {
        system = "trading"
        environment = "production"
    }
}

# LLM定义
llm "gpt4" {
    model = "gpt-4"
    provider = "openai"
    temperature = 0.3
    
    tags = {
        usage = "analysis"
        cost = "high"
    }
}

# Agent定义
agent "market_analyst" {
    role = "市场分析师"
    model = "gpt4"
    
    skills = ["technical_analysis", "fundamental_analysis", "sentiment_analysis"]
    
    behavior {
        flow = "market_data -> analyze -> 3_perspectives -> merge -> insights"
    }
    
    tags = {
        expertise = "market_analysis"
        team = "trading"
    }
}

agent "risk_manager" {
    role = "风险管理师"
    model = "gpt4"
    
    constraints = {
        max_position = 0.05
        stop_loss = 0.03
    }
    
    behavior {
        flow = "strategy -> evaluate_risk -> approve_or_reject"
    }
    
    tags = {
        expertise = "risk_management"
        critical = true
    }
}

# 工作流定义
workflow "trading_decision" {
    participants = {
        analyst = "market_analyst"
        risk_manager = "risk_manager_agent"
        trader = "execution_agent"
    }
    
    flow {
        steps = [
            "analyst -> gather_market_data -> analysis",
            "analysis -> risk_manager -> evaluate_risk -> assessment"
        ]
        
        conditions = {
            low_risk = "assessment.risk_level <= var.risk_level"
            action_on_low_risk = "execute_trade"
            action_on_high_risk = "adjust_strategy"
        }
    }
    
    tags = {
        workflow_type = "trading_decision"
        automation_level = "semi_automated"
    }
}

# 优化循环
loop_block "continuous_optimization" {
    condition = "market_open == true"
    
    process = {
        execute = "trading_decision"
        evaluate = "performance_assessment"
        optimize = "strategy_refinement"
    }
    
    optimization {
        variants = 5
        selection = "best_performer"
        update_frequency = "daily"
    }
    
    tags = {
        optimization_type = "continuous"
        learning_enabled = true
    }
}
```

## 12. iAQL与IPDSL深度集成设计

### 12.1 @符号块类型解析

在IPDSL中，@符号不是HCL的原生语法，而是通过BlockHandler扩展机制实现的：

```go
// iAQL扩展的块类型映射
var iAQLBlockTypes = map[string]string{
    "@agent":        "agent",
    "@llm":          "llm", 
    "@rag":          "rag",
    "@workflow":     "workflow",
    "@conversation": "conversation",
    "@start":        "start_execution",
    "@end":          "end_execution",
    "@loop":         "loop_block",
    "@if":           "if_block",
    "@parallel":     "parallel_block",
}

// 在预处理阶段转换@符号
func (p *IAQLPreProcessor) TransformAtSymbols(content []byte) []byte {
    for atSymbol, blockType := range iAQLBlockTypes {
        content = bytes.ReplaceAll(content, []byte(atSymbol), []byte(blockType))
    }
    return content
}
```

### 12.2 Tags系统集成

iAQL充分利用IPDSL的tags系统：

```iaql
agent "analyst" {
    role = "数据分析师"
    
    # 利用IPDSL tags进行元数据管理
    tags = {
        # 分类标签
        domain = "finance"
        skill_level = "expert"
        team = "quantitative_research"
        
        # 运维标签  
        environment = "production"
        cost_center = "trading_desk"
        compliance_level = "high"
        
        # 功能标签
        capabilities = ["analysis", "visualization", "reporting"]
        languages = ["python", "sql", "r"]
        
        # 生命周期标签
        version = "1.2.0"
        status = "active"
        last_updated = "2024-01-15"
        
        # 自定义iAQL标签
        agent_type = "analytical"
        learning_enabled = true
        memory_persistent = true
    }
}

# Tags可用于查询和过滤
resource "agent_pool" "finance_team" {
    filter_tags = {
        domain = "finance"
        team = "quantitative_research"
        status = "active"
    }
}

# Tags支持继承和组合
workflow "risk_analysis" {
    inherit_tags_from = ["market_analyst", "risk_manager"]
    
    additional_tags = {
        workflow_type = "risk_assessment"
        criticality = "high"
    }
}
```

### 12.3 IPDSL扩展机制实现

基于IPDSL的BlockHandler机制扩展iAQL：

```go
// iAQL扩展包注册
func NewIAQLExtension() *DSLExtension {
    return &DSLExtension{
        name:    "ipdsl-iaql",
        version: "1.0.0",
        domain:  "ai_programming",
        
        blockHandlers: []BlockHandlerIface{
            NewAgentBlockHandler(),
            NewLLMBlockHandler(), 
            NewRAGBlockHandler(),
            NewWorkflowBlockHandler(),
        },
        
        functions: []CustomFunction{
            NewAskFunction(),           // ask(model, query)
            NewGenerateFunction(),      // generate(template, data)
            NewCompetitiveFunction(),   // competitive(models, strategy)
        },
        
        validators: []DomainValidatorIface{
            NewAgentValidator(),
            NewLLMConfigValidator(),
            NewWorkflowValidator(),
        },
        
        tagSchemas: []TagSchemaIface{
            NewAgentTagSchema(),        // agent特定的tags验证
            NewWorkflowTagSchema(),     // workflow特定的tags验证
        },
    }
}

// Agent块处理器实现
type AgentBlockHandler struct {
    llmRegistry  LLMRegistryIface
    tagValidator TagValidatorIface
}

func (h *AgentBlockHandler) GetSupportedBlockType() string {
    return "agent"
}

func (h *AgentBlockHandler) ParseBlock(block *hcl.Block, ctx ModuleIface) (BlockIface, diag.Diags) {
    agent := &IAQLAgent{
        BaseBlock: NewBaseBlock(block),
        Name:      block.Labels[0],
    }
    
    content, diags := block.Body.Content(&hcl.BodySchema{
        Attributes: []hcl.AttributeSchema{
            {Name: "role", Required: false},
            {Name: "model", Required: false},
            {Name: "temperature", Required: false},
            {Name: "tags", Required: false},        // Tags支持
        },
        Blocks: []hcl.BlockHeaderSchema{
            {Type: "skills", LabelNames: []string{}},
            {Type: "behavior", LabelNames: []string{}},
            {Type: "constraints", LabelNames: []string{}},
        },
    })
    
    // 处理tags
    if tagsAttr := content.Attributes["tags"]; tagsAttr != nil {
        agent.TagsExpr = tagsAttr.Expr
    }
    
    return agent, diags
}
```

### 12.4 完整的混合语法示例

```hcl
# 文件：intelligent_trading.iaql

# 标准IPDSL配置
variable "environment" {
  type    = string
  default = "production"
  
  tags = {
    category = "config"
    scope = "global"
  }
}

variable "risk_level" {
  type    = number
  default = 0.05
  
  tags = {
    category = "config"
    domain = "risk_management"
  }
}

# iAQL AI实体定义（完全HCL兼容）
llm "gpt4" {
  model = "gpt-4-turbo"
  provider = "openai"
  
  parameters {
    temperature = 0.3
    max_tokens = 2000
  }
  
  strategies = {
    competitive = "5->1->auto"
    ensemble = "3->all->merge"
  }
  
  tags = {
    llm_type = "premium"
    cost_tier = "high"
    reliability = "production"
    iaql_version = "1.0"
  }
}

rag "market_knowledge" {
  knowledge_base = "financial_data"
  embedding_model = "text-embedding-ada-002"
  
  retrieval {
    strategy = ["semantic", "keyword", "hybrid"]
    top_k = 10
  }
  
  tags = {
    rag_type = "financial"
    data_classification = "internal"
    update_frequency = "daily"
  }
}

agent "market_analyst" {
  # HCL兼容配置
  model = "gpt4"
  timeout = 30
  role = "高级市场分析师"
  
  # 技能和约束配置
  expertise = ["technical_analysis", "sentiment_analysis"]
  
  constraints = {
    max_position = var.risk_level
    stop_loss = 0.03
    confidence_threshold = 0.8
  }
  
  # iAQL行为定义
  behavior {
    flow = "market_data -> analyze -> 3_perspectives -> merge -> insights"
  }
  
  # 丰富的tags支持
  tags = {
    # 功能分类
    agent_type = "analytical"
    domain = "finance"
    expertise_level = "senior"
    
    # 技术标签
    model_backend = "openai"
    response_time_sla = "5s"
    accuracy_target = "0.9"
    
    # 运维标签
    environment = var.environment
    cost_center = "trading_desk"
    owner = "quant_team"
    
    # 合规标签
    data_access_level = "restricted"
    audit_required = true
    compliance_framework = "sox"
    
    # iAQL特定标签
    iaql_agent_version = "1.0"
    memory_enabled = true
    learning_mode = "adaptive"
  }
}

workflow "trading_decision" {
  participants = {
    analyst = "market_analyst"
    risk_manager = "risk_manager_agent"
  }
  
  flow {
    steps = [
      "analyst -> gather_market_data -> analysis",
      "analysis -> risk_manager -> evaluate_risk -> assessment"
    ]
  }
  
  conditions = {
    low_risk = "assessment.risk_level <= var.risk_level"
    action_on_low_risk = "execute_trade"
    action_on_high_risk = "adjust_strategy"
  }
  
  tags = {
    workflow_type = "trading_decision"
    automation_level = "semi_automated"
    business_criticality = "high"
    sla_requirement = "real_time"
    
    # 合规相关
    regulatory_approval = "cftc_compliant"
    risk_model_version = "v2.1"
    
    # 性能相关
    max_execution_time = "30s"
    success_rate_target = "0.95"
  }
}

# 标准IPDSL输出（与iAQL完美集成）
output "trading_system_endpoint" {
  value = workflow.trading_decision.api_endpoint
  
  tags = {
    output_type = "api_endpoint"
    consumer = "trading_interface"
  }
}

# 混合使用IPDSL资源
resource "monitoring" "trading_monitor" {
  target = workflow.trading_decision
  
  metrics = ["latency", "accuracy", "profit_loss"]
  
  alert_config = {
    high_risk = var.risk_level * 1.5
    low_performance = 0.6
  }
  
  tags = {
    monitor_type = "trading_system"
    alert_channels = ["slack", "email", "dashboard"]
    iaql_integration = true
  }
}
```

## 13. 关键技术问题解答

### 13.1 问题1：冒号vs等号的HCL兼容性

**✅ 解决方案**：完全采用HCL标准`=`语法
- **对象属性**: `{key = value}` ✅ 
- **块属性**: `attribute = value` ✅
- **禁用冒号**: `{key: value}` ❌ (HCL不支持)

### 13.2 问题2：is操作符的语义化处理

**✅ 解决方案**：上下文分离使用
- **配置上下文**: 使用`=`进行属性赋值
- **流程上下文**: 保留`is`进行语义表达
- **系统内部**: 统一处理为赋值关系

### 13.3 问题3：tags系统的深度利用

**✅ 强大的tags支持**：
- **分类管理**: 按domain、team、expertise分类
- **运维集成**: environment、cost_center、owner
- **合规追溯**: compliance_level、audit_required
- **性能监控**: sla_requirement、success_rate_target
- **版本管理**: iaql_version、model_version

### 13.4 问题4：@符号的解析机制

**✅ 预处理转换**：
- **语法糖**: `@agent` → `agent` (块类型)
- **预处理器**: 在HCL解析前进行符号转换
- **扩展机制**: 通过IPDSL BlockHandler注册处理器
- **完全兼容**: 转换后的语法100% HCL兼容

## 总结

这套重新设计的iAQL语法体系在保持AI领域语义特色的同时，确保了与IPDSL/HCL的完全兼容：

### 🎯 **兼容性保证**
- **HCL标准语法**: 所有对象使用`{key = value}`格式
- **块语法兼容**: 完全遵循HCL块定义规范
- **@符号处理**: 通过预处理器实现语法糖转换
- **Tags深度集成**: 充分利用IPDSL的tags基础设施

### 🔧 **技术优势**  
- **无缝集成**: 与IPDSL基础设施零摩擦集成
- **扩展性**: 基于BlockHandler的标准扩展机制
- **可维护性**: 清晰的语法规则和转换逻辑
- **向后兼容**: 标准HCL工具链完全支持

这真正实现了**"AI时代的PlantUML"** - 让AI编程像画图一样直观，同时保持工程化的严谨性！ 