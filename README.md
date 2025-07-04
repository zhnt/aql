# AQL (Agent Query Language)

AQL是一门专为AI Agent编排设计的现代编程语言。

## 项目概述

AQL旨在成为AI Agent生态的"SQL"，提供：
- 声明式Agent定义和编排
- 原生异步编程支持
- 基础设施集成能力
- 图灵完备的编程语言特性

## 项目结构

```
├── cmd/aql/           # 主程序入口
├── pkg/               # 公共库
│   ├── lexer/        # 词法分析器
│   ├── parser/       # 语法分析器
│   ├── ast/          # 抽象语法树
│   ├── runtime/      # 运行时系统
│   └── stdlib/       # 标准库
├── internal/          # 内部实现
│   ├── compiler/     # 编译器
│   └── vm/           # 虚拟机
├── examples/          # 示例代码
├── docs/             # 文档
└── test/             # 测试
```

## 开发状态

🚧 **开发中** - 当前正在设计语言核心和实现MVP

## 构建和运行

```bash
# 构建
go build ./cmd/aql

# 运行
./aql --help
```

## 语法预览

```aql
-- Agent定义
agent analyst {
    model: "gpt-4",
    skills: ["analysis", "reporting"]
}

-- 异步任务执行
async function analyze_market() {
    local data = await fetch_market_data()
    local result = await analyst.analyze(data)
    return result
}

-- 并行处理
for parallel item in dataset do
    await process_item(item)
end
```

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License 