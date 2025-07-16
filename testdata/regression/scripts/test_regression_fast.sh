#!/bin/bash

# AQL 快速回归测试脚本
# 执行时间目标：< 30秒
# 覆盖核心功能的关键测试

set -e

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTDATA_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$(dirname "$TESTDATA_DIR")")"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
FAILED_LIST=()

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 运行单个测试
run_test() {
    local test_file="$1"
    local test_name="$(basename "$test_file" .aql)"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    log_info "运行测试: $test_name"
    
    # 切换到项目根目录运行测试
    cd "$PROJECT_ROOT"
    
    if timeout 10s ./aql "$test_file" > /dev/null 2>&1; then
        log_success "测试通过: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "测试失败: $test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_LIST+=("$test_name")
        return 1
    fi
}

# 构建AQL解释器
build_aql() {
    log_info "构建AQL解释器..."
    cd "$PROJECT_ROOT"
    
    if make build-fast > /dev/null 2>&1; then
        log_success "AQL解释器构建成功"
        return 0
    else
        log_error "AQL解释器构建失败"
        return 1
    fi
}

# 主函数
main() {
    echo "🚀 AQL 快速回归测试"
    echo "==================="
    
    # 构建解释器
    if ! build_aql; then
        log_error "构建失败，测试终止"
        exit 1
    fi
    
    # 核心基础功能测试
    log_info "📋 基础功能测试"
    for test_file in "$TESTDATA_DIR/basic"/*.aql; do
        if [[ -f "$test_file" ]]; then
            run_test "$test_file"
        fi
    done
    
    # 关键闭包功能测试
    log_info "📋 闭包功能测试"
    key_closure_tests=(
        "test_closure_basic.aql"
        "test_closure_nested.aql"
        "test_closure_debug.aql"
        "test_simple_closure.aql"
    )
    
    for test_name in "${key_closure_tests[@]}"; do
        test_file="$TESTDATA_DIR/closure/$test_name"
        if [[ -f "$test_file" ]]; then
            run_test "$test_file"
        fi
    done
    
    # 关键GC功能测试
    log_info "📋 GC功能测试"
    key_gc_tests=(
        "test_gc_simple.aql"
    )
    
    for test_name in "${key_gc_tests[@]}"; do
        test_file="$TESTDATA_DIR/gc/$test_name"
        if [[ -f "$test_file" ]]; then
            run_test "$test_file"
        fi
    done
    
    # 输出测试结果
    echo ""
    echo "📊 测试结果汇总"
    echo "==============="
    echo "总测试数: $TOTAL_TESTS"
    echo "通过: $PASSED_TESTS"
    echo "失败: $FAILED_TESTS"
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        echo ""
        log_error "失败的测试:"
        for failed_test in "${FAILED_LIST[@]}"; do
            echo "  - $failed_test"
        done
        echo ""
        log_error "快速回归测试失败"
        exit 1
    else
        echo ""
        log_success "🎉 所有测试通过！"
        exit 0
    fi
}

# 运行主函数
main "$@" 