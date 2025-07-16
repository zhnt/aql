#!/bin/bash

# AQL 完整回归测试脚本
# 执行时间目标：< 5分钟
# 覆盖全部功能测试

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
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0
FAILED_LIST=()

# 测试类别统计
declare -A CATEGORY_STATS
CATEGORY_STATS=(
    ["basic_passed"]=0
    ["basic_failed"]=0
    ["closure_passed"]=0
    ["closure_failed"]=0
    ["gc_passed"]=0
    ["gc_failed"]=0
    ["integration_passed"]=0
    ["integration_failed"]=0
)

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

log_category() {
    echo -e "${CYAN}📂 $1${NC}"
}

# 运行单个测试
run_test() {
    local test_file="$1"
    local category="$2"
    local test_name="$(basename "$test_file" .aql)"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    log_info "运行测试: $test_name"
    
    # 切换到项目根目录运行测试
    cd "$PROJECT_ROOT"
    
    if timeout 30s ./aql "$test_file" > /dev/null 2>&1; then
        log_success "测试通过: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        CATEGORY_STATS["${category}_passed"]=$((CATEGORY_STATS["${category}_passed"] + 1))
        return 0
    else
        log_error "测试失败: $test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        CATEGORY_STATS["${category}_failed"]=$((CATEGORY_STATS["${category}_failed"] + 1))
        FAILED_LIST+=("$category/$test_name")
        return 1
    fi
}

# 运行测试类别
run_category() {
    local category="$1"
    local category_dir="$2"
    local category_name="$3"
    
    log_category "$category_name"
    
    local category_tests=0
    local category_passed=0
    
    if [[ -d "$category_dir" ]]; then
        for test_file in "$category_dir"/*.aql; do
            if [[ -f "$test_file" ]]; then
                category_tests=$((category_tests + 1))
                if run_test "$test_file" "$category"; then
                    category_passed=$((category_passed + 1))
                fi
            fi
        done
    fi
    
    if [[ $category_tests -eq 0 ]]; then
        log_warning "没有找到 $category_name 测试文件"
    else
        log_info "$category_name: $category_passed/$category_tests 通过"
    fi
}

# 构建AQL解释器
build_aql() {
    log_info "构建AQL解释器..."
    cd "$PROJECT_ROOT"
    
    if make build > /dev/null 2>&1; then
        log_success "AQL解释器构建成功"
        return 0
    else
        log_error "AQL解释器构建失败"
        return 1
    fi
}

# 检查环境
check_environment() {
    log_info "检查测试环境..."
    
    # 检查Go环境
    if ! command -v go &> /dev/null; then
        log_error "Go环境未安装"
        return 1
    fi
    
    # 检查项目结构
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]]; then
        log_error "项目根目录结构错误"
        return 1
    fi
    
    # 检查测试目录
    if [[ ! -d "$TESTDATA_DIR" ]]; then
        log_error "测试数据目录不存在: $TESTDATA_DIR"
        return 1
    fi
    
    log_success "环境检查通过"
    return 0
}

# 生成测试报告
generate_report() {
    local report_file="$PROJECT_ROOT/test_report.txt"
    
    {
        echo "AQL 完整回归测试报告"
        echo "====================="
        echo "测试时间: $(date)"
        echo "项目路径: $PROJECT_ROOT"
        echo ""
        echo "测试结果汇总:"
        echo "  总测试数: $TOTAL_TESTS"
        echo "  通过: $PASSED_TESTS"
        echo "  失败: $FAILED_TESTS"
        echo "  跳过: $SKIPPED_TESTS"
        echo ""
        echo "各类别测试结果:"
        echo "  基础功能: ${CATEGORY_STATS[basic_passed]}/${CATEGORY_STATS[basic_passed]+CATEGORY_STATS[basic_failed]} 通过"
        echo "  闭包系统: ${CATEGORY_STATS[closure_passed]}/${CATEGORY_STATS[closure_passed]+CATEGORY_STATS[closure_failed]} 通过"
        echo "  GC系统: ${CATEGORY_STATS[gc_passed]}/${CATEGORY_STATS[gc_passed]+CATEGORY_STATS[gc_failed]} 通过"
        echo "  集成测试: ${CATEGORY_STATS[integration_passed]}/${CATEGORY_STATS[integration_passed]+CATEGORY_STATS[integration_failed]} 通过"
        echo ""
        
        if [[ $FAILED_TESTS -gt 0 ]]; then
            echo "失败的测试:"
            for failed_test in "${FAILED_LIST[@]}"; do
                echo "  - $failed_test"
            done
            echo ""
        fi
        
        echo "报告生成时间: $(date)"
    } > "$report_file"
    
    log_info "测试报告已生成: $report_file"
}

# 主函数
main() {
    echo "🚀 AQL 完整回归测试"
    echo "==================="
    
    # 检查环境
    if ! check_environment; then
        log_error "环境检查失败，测试终止"
        exit 1
    fi
    
    # 构建解释器
    if ! build_aql; then
        log_error "构建失败，测试终止"
        exit 1
    fi
    
    # 运行各类别测试
    run_category "basic" "$TESTDATA_DIR/basic" "基础功能测试"
    run_category "closure" "$TESTDATA_DIR/closure" "闭包系统测试"
    run_category "gc" "$TESTDATA_DIR/gc" "GC系统测试"
    run_category "integration" "$TESTDATA_DIR/integration" "集成测试"
    
    # 生成测试报告
    generate_report
    
    # 输出测试结果
    echo ""
    echo "📊 测试结果汇总"
    echo "==============="
    echo "总测试数: $TOTAL_TESTS"
    echo "通过: $PASSED_TESTS"
    echo "失败: $FAILED_TESTS"
    echo "跳过: $SKIPPED_TESTS"
    
    # 各类别统计
    echo ""
    echo "📈 各类别结果:"
    echo "  基础功能: ${CATEGORY_STATS[basic_passed]} 通过, ${CATEGORY_STATS[basic_failed]} 失败"
    echo "  闭包系统: ${CATEGORY_STATS[closure_passed]} 通过, ${CATEGORY_STATS[closure_failed]} 失败"
    echo "  GC系统: ${CATEGORY_STATS[gc_passed]} 通过, ${CATEGORY_STATS[gc_failed]} 失败"
    echo "  集成测试: ${CATEGORY_STATS[integration_passed]} 通过, ${CATEGORY_STATS[integration_failed]} 失败"
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        echo ""
        log_error "失败的测试:"
        for failed_test in "${FAILED_LIST[@]}"; do
            echo "  - $failed_test"
        done
        echo ""
        log_error "完整回归测试失败"
        exit 1
    else
        echo ""
        log_success "🎉 所有测试通过！"
        exit 0
    fi
}

# 运行主函数
main "$@" 