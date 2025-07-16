#!/bin/bash

# AQL 监控回归测试脚本
# 自动监控文件变化并运行回归测试

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
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# 配置
WATCH_INTERVAL=2  # 监控间隔（秒）
DEBOUNCE_TIME=1   # 防抖时间（秒）
TEST_MODE="fast"  # 默认测试模式：fast/full

# 监控的文件模式
WATCH_PATTERNS=(
    "*.go"
    "*.aql"
    "Makefile"
    "go.mod"
    "go.sum"
)

# 监控的目录
WATCH_DIRS=(
    "$PROJECT_ROOT/internal"
    "$PROJECT_ROOT/cmd"
    "$PROJECT_ROOT/pkg"
    "$PROJECT_ROOT/testdata"
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

log_watch() {
    echo -e "${PURPLE}👁️  $1${NC}"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    # 检查fswatch（推荐）
    if command -v fswatch &> /dev/null; then
        WATCH_TOOL="fswatch"
        log_success "使用 fswatch 监控文件变化"
        return 0
    fi
    
    # 检查inotifywait（Linux）
    if command -v inotifywait &> /dev/null; then
        WATCH_TOOL="inotifywait"
        log_success "使用 inotifywait 监控文件变化"
        return 0
    fi
    
    # 降级到轮询模式
    WATCH_TOOL="polling"
    log_warning "未找到文件监控工具，使用轮询模式"
    log_info "建议安装 fswatch: brew install fswatch (macOS) 或 apt-get install inotify-tools (Linux)"
    return 0
}

# 获取文件修改时间
get_file_mtime() {
    local file="$1"
    if [[ -f "$file" ]]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            stat -f "%m" "$file"
        else
            stat -c "%Y" "$file"
        fi
    else
        echo "0"
    fi
}

# 轮询监控
polling_watch() {
    local last_check_time=$(date +%s)
    
    while true; do
        local current_time=$(date +%s)
        local changed=false
        
        # 检查监控目录中的文件
        for watch_dir in "${WATCH_DIRS[@]}"; do
            if [[ -d "$watch_dir" ]]; then
                for pattern in "${WATCH_PATTERNS[@]}"; do
                    while IFS= read -r -d '' file; do
                        local mtime=$(get_file_mtime "$file")
                        if [[ $mtime -gt $last_check_time ]]; then
                            log_watch "文件变化: $file"
                            changed=true
                            break 2
                        fi
                    done < <(find "$watch_dir" -name "$pattern" -type f -print0 2>/dev/null)
                done
            fi
        done
        
        if [[ "$changed" == true ]]; then
            last_check_time=$current_time
            sleep $DEBOUNCE_TIME  # 防抖
            run_regression_test
        fi
        
        sleep $WATCH_INTERVAL
    done
}

# fswatch监控
fswatch_watch() {
    local watch_paths=()
    
    # 构建监控路径
    for watch_dir in "${WATCH_DIRS[@]}"; do
        if [[ -d "$watch_dir" ]]; then
            watch_paths+=("$watch_dir")
        fi
    done
    
    if [[ ${#watch_paths[@]} -eq 0 ]]; then
        log_error "没有找到可监控的目录"
        return 1
    fi
    
    log_info "开始监控目录: ${watch_paths[*]}"
    
    # 构建文件过滤器
    local include_filters=()
    for pattern in "${WATCH_PATTERNS[@]}"; do
        include_filters+=("-i" "$pattern")
    done
    
    # 启动fswatch监控
    fswatch -r "${include_filters[@]}" --event Created --event Updated --event Moved --event Renamed "${watch_paths[@]}" | while read -r changed_file; do
        log_watch "文件变化: $changed_file"
        sleep $DEBOUNCE_TIME  # 防抖
        run_regression_test
    done
}

# inotifywait监控
inotifywait_watch() {
    local watch_paths=()
    
    # 构建监控路径
    for watch_dir in "${WATCH_DIRS[@]}"; do
        if [[ -d "$watch_dir" ]]; then
            watch_paths+=("$watch_dir")
        fi
    done
    
    if [[ ${#watch_paths[@]} -eq 0 ]]; then
        log_error "没有找到可监控的目录"
        return 1
    fi
    
    log_info "开始监控目录: ${watch_paths[*]}"
    
    # 启动inotifywait监控
    inotifywait -m -r --format '%w%f' -e create,modify,move,delete "${watch_paths[@]}" | while read -r changed_file; do
        # 检查文件是否匹配模式
        local matches=false
        for pattern in "${WATCH_PATTERNS[@]}"; do
            if [[ "$changed_file" == $pattern ]]; then
                matches=true
                break
            fi
        done
        
        if [[ "$matches" == true ]]; then
            log_watch "文件变化: $changed_file"
            sleep $DEBOUNCE_TIME  # 防抖
            run_regression_test
        fi
    done
}

# 运行回归测试
run_regression_test() {
    log_info "运行回归测试 ($TEST_MODE)..."
    
    local test_script
    if [[ "$TEST_MODE" == "fast" ]]; then
        test_script="$SCRIPT_DIR/test_regression_fast.sh"
    else
        test_script="$SCRIPT_DIR/test_regression_full.sh"
    fi
    
    if [[ -f "$test_script" ]]; then
        if bash "$test_script"; then
            log_success "回归测试通过"
        else
            log_error "回归测试失败"
        fi
    else
        log_error "测试脚本不存在: $test_script"
    fi
    
    echo ""
    log_info "继续监控文件变化..."
}

# 显示帮助
show_help() {
    cat << EOF
AQL 监控回归测试脚本

用法: $0 [选项]

选项:
  -m, --mode MODE     测试模式 (fast|full)，默认: fast
  -i, --interval SEC  监控间隔秒数，默认: 2
  -d, --debounce SEC  防抖时间秒数，默认: 1
  -h, --help          显示此帮助信息

示例:
  $0                  # 使用默认设置开始监控
  $0 -m full          # 使用完整测试模式
  $0 -i 1 -d 0.5      # 更高频率监控

支持的文件监控工具:
  1. fswatch (推荐) - 跨平台，功能强大
  2. inotifywait - Linux原生工具
  3. polling - 轮询模式，兼容性最好

EOF
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -m|--mode)
                TEST_MODE="$2"
                shift 2
                ;;
            -i|--interval)
                WATCH_INTERVAL="$2"
                shift 2
                ;;
            -d|--debounce)
                DEBOUNCE_TIME="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 验证测试模式
    if [[ "$TEST_MODE" != "fast" && "$TEST_MODE" != "full" ]]; then
        log_error "无效的测试模式: $TEST_MODE"
        exit 1
    fi
}

# 主函数
main() {
    echo "👁️  AQL 监控回归测试"
    echo "==================="
    
    parse_args "$@"
    
    log_info "配置信息:"
    log_info "  测试模式: $TEST_MODE"
    log_info "  监控间隔: ${WATCH_INTERVAL}s"
    log_info "  防抖时间: ${DEBOUNCE_TIME}s"
    log_info "  项目根目录: $PROJECT_ROOT"
    
    # 检查依赖
    if ! check_dependencies; then
        log_error "依赖检查失败"
        exit 1
    fi
    
    # 初始运行一次测试
    log_info "初始运行回归测试..."
    run_regression_test
    
    # 开始监控
    log_info "开始监控文件变化..."
    log_info "按 Ctrl+C 停止监控"
    
    case $WATCH_TOOL in
        fswatch)
            fswatch_watch
            ;;
        inotifywait)
            inotifywait_watch
            ;;
        polling)
            polling_watch
            ;;
        *)
            log_error "未知的监控工具: $WATCH_TOOL"
            exit 1
            ;;
    esac
}

# 信号处理
cleanup() {
    log_info "停止监控..."
    exit 0
}

trap cleanup SIGINT SIGTERM

# 运行主函数
main "$@" 