#!/bin/bash

# AQL 闭包回归测试脚本
# 用法: ./run_regression_tests.sh [--verbose]

set -e

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../../" && pwd)"
CONFIG_FILE="${SCRIPT_DIR}/test_config.json"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 命令行参数
VERBOSE=false
if [[ "$1" == "--verbose" ]]; then
    VERBOSE=true
fi

# 统计变量
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo -e "${BLUE}=== AQL 闭包回归测试 ===${NC}"
echo

# 检查 AQL 可执行文件
AQL_CMD="${PROJECT_ROOT}/cmd/aql"
if [[ ! -d "$AQL_CMD" ]]; then
    echo -e "${RED}错误: 找不到 AQL 命令目录 $AQL_CMD${NC}"
    exit 1
fi

# 读取并解析配置文件
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo -e "${RED}错误: 找不到配置文件 $CONFIG_FILE${NC}"
    exit 1
fi

# 提取结果的函数
extract_result() {
    local output="$1"
    # 查找"结果:"行并提取
    echo "$output" | grep "^结果:" | tail -1 || echo ""
}

# 运行单个测试
run_test() {
    local test_file="$1"
    local expected_output="$2"
    local test_name="$3"
    local description="$4"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -e "${BLUE}测试 $TOTAL_TESTS: $test_name${NC}"
    if [[ "$VERBOSE" == "true" ]]; then
        echo "  文件: $test_file"
        echo "  描述: $description"
        echo "  预期: $expected_output"
    fi
    
    # 运行 AQL
    local full_path="${SCRIPT_DIR}/${test_file}"
    if [[ ! -f "$full_path" ]]; then
        echo -e "  ${RED}✗ 失败: 测试文件不存在${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
    
    # 运行测试并捕获输出
    local raw_output
    local exit_code
    
    cd "$PROJECT_ROOT"
    raw_output=$(go run "$AQL_CMD" "$full_path" 2>&1) || exit_code=$?
    
    if [[ $exit_code -ne 0 ]]; then
        echo -e "  ${RED}✗ 失败: 执行错误 (退出码: $exit_code)${NC}"
        if [[ "$VERBOSE" == "true" ]]; then
            echo "  错误输出:"
            echo "$raw_output" | sed 's/^/    /'
        fi
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
    
    # 提取结果
    local actual_output
    actual_output=$(extract_result "$raw_output")
    
    if [[ -z "$actual_output" ]]; then
        echo -e "  ${RED}✗ 失败: 未找到结果输出${NC}"
        if [[ "$VERBOSE" == "true" ]]; then
            echo "  完整输出:"
            echo "$raw_output" | sed 's/^/    /'
        fi
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
    
    # 比较结果
    if [[ "$actual_output" == "$expected_output" ]]; then
        echo -e "  ${GREEN}✓ 通过${NC}"
        if [[ "$VERBOSE" == "true" ]]; then
            echo "  实际: $actual_output"
        fi
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "  ${RED}✗ 失败: 结果不匹配${NC}"
        echo -e "  预期: ${YELLOW}$expected_output${NC}"
        echo -e "  实际: ${YELLOW}$actual_output${NC}"
        if [[ "$VERBOSE" == "true" ]]; then
            echo "  完整输出:"
            echo "$raw_output" | sed 's/^/    /'
        fi
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 使用 jq 解析 JSON 配置文件（如果可用），否则手动解析
if command -v jq >/dev/null 2>&1; then
    # 使用 jq 解析
    while IFS= read -r line; do
        if [[ -z "$line" ]]; then continue; fi
        
        eval "$line"
        run_test "$file" "$expected_output" "$name" "$description"
        echo
    done < <(jq -r '.tests[] | "name=\(.name | @sh); file=\(.file | @sh); expected_output=\(.expected_output | @sh); description=\(.description | @sh)"' "$CONFIG_FILE")
else
    # 手动解析 JSON（简化版本）
    echo -e "${YELLOW}警告: 未找到 jq，使用简化的 JSON 解析${NC}"
    
    # 硬编码测试用例（从配置文件复制）
    run_test "test_simple_non_closure.aql" "结果: 99" "非闭包数组操作" "测试非闭包环境下的数组修改和访问，验证基础寄存器分配"
    echo
    
    run_test "test_closure_parameter_debug.aql" "结果: 52" "简单闭包参数传递" "测试简单闭包的参数传递和upvalue访问"
    echo
    
    run_test "test_complex_register_conflict.aql" "结果: 99" "复杂闭包数组操作" "测试闭包中的数组修改和访问，验证寄存器冲突修复"
    echo
    
    run_test "test_deep_nesting.aql" "结果: 75" "三级嵌套闭包" "测试三级嵌套闭包的变量捕获和访问"
    echo
    
    run_test "test_multi_param_closures.aql" "结果: 20" "多参数闭包" "测试多参数闭包的复杂场景"
    echo
    
    run_test "test_array_closure.aql" "结果: 3" "数组作为upvalue" "测试数组作为自由变量在闭包中的使用"
    echo
fi

# 输出测试摘要
echo -e "${BLUE}=== 测试摘要 ===${NC}"
echo "总计测试: $TOTAL_TESTS"
echo -e "通过: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败: ${RED}$FAILED_TESTS${NC}"

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "${GREEN}所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}有 $FAILED_TESTS 个测试失败${NC}"
    exit 1
fi 