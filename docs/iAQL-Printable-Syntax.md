# iAQL可打印字符语法设计 - 集成IPDSL对象引用

## 1. 可打印字符操作符设计

### 基本操作符（以-开头）
```
--> : 流转操作符 (替代→)
-<  : 分解操作符 (替代▽，向左分解)
->  : 汇总操作符 (替代△，向右汇总)
-|  : 并行操作符
-@  : 数量标识符
-#  : 确认点标识符
-*  : 通配符/全部
-o  : 单元标识符
```

### 组合操作符
```
-@8     : 数量8
-@*     : 全部数量
-<@8    : 分解为8个
->@1    : 汇总为1个
-|@3    : 并行3个
-#wait  : 等待确认
-#auto  : 自动执行
-o1     : 单个单元
```

## 2. 与IPDSL对象集成

### IPDSL变量引用语法
```hcl
// IPDSL基础对象定义
context "ppt_generation" {
  variable "topic" {
    type = "string"
    description = "PPT主题"
  }
  
  variable "storylines" {
    type = "list(object)"
    description = "故事线集合"
  }
  
  variable "selected_storyline" {
    type = "object"
    description = "选中的故事线"
  }
  
  variable "sections" {
    type = "list(object)"
    description = "章节列表"
  }
}

// iAQL中引用IPDSL变量
usecase "PPT生成" {
  execute {
    // 引用IPDSL变量
    var.topic -<@8 storyline_options -> var.storylines
    var.storylines ->@3 analysis_options -<@1 -> var.selected_storyline
    var.selected_storyline -<@${var.slide_count} -> var.sections
    var.sections --> content_generation -|@* --> images --> ppt_output
  }
}
```

### 对象属性访问
```hcl
// IPDSL对象定义
resource "storyline" {
  title = "string"
  sections = "list"
  theme = "string"
  target_audience = "string"
}

// iAQL中访问对象属性
usecase "内容优化" {
  execute {
    storyline.title -<@4 title_variants ->@1 final_title
    storyline.sections -< section_content -|@* --> 
      [text_content, image_prompts] -> layout_design
    storyline.theme --> style_consistency -#auto
  }
}
```

## 3. 完整语法重构

### 角色定义（保持IPDSL风格）
```hcl
role "content_creator" {
  skills = ["writing", "storytelling", "audience_analysis"]
  capacity = 5
  
  variables {
    current_workload = 0
    quality_score = 0.95
  }
}

role "visual_designer" {
  skills = ["design", "image_generation", "layout"]
  capacity = 3
  
  variables {
    style_preference = "modern"
    resolution_default = "1024x1024"
  }
}
```

### 产品分解结构（集成IPDSL对象）
```hcl
product "business_ppt" {
  metadata {
    version = "1.0"
    created_by = var.user_id
    created_at = timestamp()
  }
  
  component "storyline_layer" {
    depends_on = [var.topic, var.audience]
    
    module "narrative_structure" {
      properties = {
        coherence_score = 0.9
        engagement_level = "high"
      }
    }
    
    module "transition_logic" {
      properties = {
        flow_smoothness = 0.95
      }
    }
  }
  
  component "content_layer" {
    depends_on = [product.business_ppt.component.storyline_layer]
    
    module "text_content" {
      count = var.slide_count
    }
    
    module "visual_content" {
      count = var.slide_count
      properties = {
        image_style = var.brand_style
      }
    }
  }
}
```

### 工作流程（可打印字符版本）
```hcl
method "intelligent_ppt_generation" {
  input {
    topic = var.topic
    audience = var.audience  
    slide_count = var.slide_count
    style = var.presentation_style
  }
  
  phase "storyline_creation" {
    workflow "ideation_convergence" {
      job "storyline_generation" -> role.content_creator {
        task "generate_options" {
          execute = "var.topic -<@8 storyline_options"
          output = var.storyline_candidates
        }
        
        task "analyze_and_select" {
          execute = "var.storyline_candidates ->@3 detailed_analysis -<@1"
          output = var.selected_storyline
          control = "-#wait"  // 需要人工确认
        }
      }
    }
  }
  
  phase "content_development" {
    workflow "parallel_creation" {
      job "content_breakdown" -> role.content_creator {
        task "section_decomposition" {
          execute = "var.selected_storyline -<@${var.slide_count} sections"
          output = var.slide_sections
        }
        
        task "content_generation" {
          execute = "var.slide_sections -< detailed_content -|@*"
          output = var.slide_contents
        }
      }
      
      job "visual_creation" -> role.visual_designer {
        task "prompt_generation" {
          execute = "var.slide_contents -< image_prompts -|@*"
          output = var.image_prompts
        }
        
        task "image_generation" {
          execute = "var.image_prompts --> images -|@* -#auto"
          output = var.slide_images
          depends_on = [var.brand_style]
        }
      }
    }
  }
  
  phase "final_assembly" {
    workflow "integration" {
      job "ppt_assembly" -> role.visual_designer {
        task "layout_integration" {
          execute = "[var.slide_contents, var.slide_images] -> layout_design"
          output = var.formatted_slides
        }
        
        task "output_generation" {
          execute = "var.formatted_slides --> -|@* [pptx, pdf]"
          output = [var.pptx_file, var.pdf_file]
        }
      }
    }
  }
}
```

## 4. 复杂场景示例

### 多版本PPT生成
```hcl
usecase "multi_version_ppt" {
  variables {
    base_topic = "AI智能城市解决方案"
    audiences = ["投资人", "政府官员", "技术团队"]
    versions = length(var.audiences)
  }
  
  execute {
    // 为每个受众生成专门版本
    var.base_topic -<@${var.versions} audience_specific_storylines
    
    for_each audience in var.audiences {
      storyline_for[audience] -<@12 sections_for[audience]
      sections_for[audience] -< content_for[audience] -|@*
      content_for[audience] --> images_for[audience] -|@*
      [content_for[audience], images_for[audience]] -> ppt_for[audience]
    }
    
    // 汇总生成报告
    -@* ppt_versions -> generation_report -#auto
  }
}
```

### 条件分支执行
```hcl
usecase "adaptive_ppt_generation" {
  execute {
    var.topic -<@8 initial_storylines
    
    if var.complexity_level == "high" {
      initial_storylines ->@5 detailed_analysis -<@2
    } else {
      initial_storylines ->@3 simple_analysis -<@1  
    }
    
    selected_storyline -<@${var.slide_count} sections
    
    // 根据时间限制选择生成策略
    case var.urgency {
      "high" -> sections -< quick_content -|@* --> simple_images
      "medium" -> sections -< standard_content -|@* --> quality_images  
      "low" -> sections -< premium_content -|@* --> custom_images -#wait
    }
    
    [content, images] -> final_ppt
  }
}
```

## 5. 语法规则汇总

### 操作符优先级
```
1. -@n (数量标识)     : 最高优先级
2. -< (分解)         : 高优先级  
3. -> (汇总)         : 高优先级
4. -| (并行)         : 中优先级
5. --> (流转)        : 低优先级
6. -# (控制)         : 最低优先级
```

### IPDSL集成点
```
- var.* : 变量引用
- resource.* : 资源引用  
- role.* : 角色引用
- product.* : 产品引用
- ${expression} : 表达式插值
- depends_on = [] : 依赖关系
- for_each, if, case : 控制结构
```

### 类型系统
```
- 数值类型: -@8, -@*, -@${var.count}
- 字符串类型: "storyline", var.topic
- 数组类型: [item1, item2], var.list  
- 对象类型: storyline.title, resource.attr
- 函数类型: length(), timestamp()
```

## 6. 实际应用

### 简化调用
```
# 最简版本
topic -<@8 -> @1 --> @12 -< --> images --> ppt

# 带确认版本
topic -<@8 ->@3 -<@1 -#wait --> @12 -< --> images --> ppt -#auto

# 完整版本  
var.topic -<@8 storylines ->@3 analysis -<@1 selected -#wait -->
var.selected -<@${var.slides} sections -< content -|@* -->
content -< prompts --> images -|@* -#auto -->
[content, images] -> layout --> -|@* [pptx, pdf]
```

这样的设计既保持了语法的简洁性，又完全使用可打印字符，同时深度集成了IPDSL的对象系统和变量引用机制。 