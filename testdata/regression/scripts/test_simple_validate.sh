#!/bin/bash

# 简单验证脚本 - 不使用超时，用于验证基本功能

set -e

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTDATA_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$(dirname "$TESTDATA_DIR")")"

echo "🔍 AQL 简单验证测试"
echo "===================="

# 切换到项目根目录
cd "$PROJECT_ROOT"

# 测试基本的加法函数
echo "测试1: 基本加法函数"
result=$(./aql testdata/regression/basic/test_add_only.aql 2>/dev/null | tail -n 1)
if [[ "$result" == "结果: 8" ]]; then
    echo "✅ 测试1通过"
else
    echo "❌ 测试1失败，期望: 结果: 8，实际: $result"
fi

# 测试简单闭包
echo "测试2: 简单闭包"
if [[ -f "testdata/regression/closure/test_simple_closure.aql" ]]; then
    result=$(./aql testdata/regression/closure/test_simple_closure.aql 2>/dev/null | tail -n 1)
    if [[ "$result" == "结果: 42" ]]; then
        echo "✅ 测试2通过"
    else
        echo "❌ 测试2失败，期望: 结果: 42，实际: $result"
    fi
else
    echo "⚠️  测试2跳过（文件不存在）"
fi

# 测试GC系统
echo "测试3: GC系统"
if [[ -f "testdata/regression/gc/test_gc_simple.aql" ]]; then
    result=$(./aql testdata/regression/gc/test_gc_simple.aql 2>/dev/null | tail -n 1)
    if [[ "$result" == "结果: test" ]]; then
        echo "✅ 测试3通过"
    else
        echo "❌ 测试3失败，期望: 结果: test，实际: $result"
    fi
else
    echo "⚠️  测试3跳过（文件不存在）"
fi

echo ""
echo "🎉 基本验证完成" 