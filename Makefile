# AQL (Advanced Query Language) Makefile
# 版本: 1.0.0-alpha

# 变量定义
APP_NAME = aql
VERSION = 1.0.0-alpha
MAIN_PATH = cmd/aql/main.go
BIN_DIR = bin
BUILD_DIR = build
DIST_DIR = dist
TESTDATA_DIR = testdata

# Go相关变量
GO = go
GOFMT = gofmt
GOTEST = go test
GOBUILD = go build
GOCLEAN = go clean
GOMOD = go mod

# 回归测试相关变量
REGRESSION_SCRIPTS_DIR = $(TESTDATA_DIR)/regression/scripts
FAST_REGRESSION_SCRIPT = $(REGRESSION_SCRIPTS_DIR)/test_regression_fast.sh
FULL_REGRESSION_SCRIPT = $(REGRESSION_SCRIPTS_DIR)/test_regression_full.sh
WATCH_REGRESSION_SCRIPT = $(REGRESSION_SCRIPTS_DIR)/test_regression_watch.sh

# 构建标志
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_FLAGS = -v $(LDFLAGS)

# 默认目标
.PHONY: all
all: clean fmt test build

# 格式化代码
.PHONY: fmt
fmt:
	@echo "🔧 格式化代码..."
	$(GOFMT) -s -w .
	$(GO) mod tidy

# 运行测试
.PHONY: test
test:
	@echo "🧪 运行测试..."
	$(GOTEST) -v ./...

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo "📊 运行测试并生成覆盖率报告..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "📈 覆盖率报告已生成: coverage.html"

# 快速回归测试
.PHONY: test-regression-fast
test-regression-fast: build-fast
	@echo "🚀 运行快速回归测试..."
	@if [ -f "$(FAST_REGRESSION_SCRIPT)" ]; then \
		chmod +x "$(FAST_REGRESSION_SCRIPT)"; \
		bash "$(FAST_REGRESSION_SCRIPT)"; \
	else \
		echo "❌ 快速回归测试脚本不存在: $(FAST_REGRESSION_SCRIPT)"; \
		exit 1; \
	fi

# 完整回归测试
.PHONY: test-regression-full
test-regression-full: build
	@echo "🚀 运行完整回归测试..."
	@if [ -f "$(FULL_REGRESSION_SCRIPT)" ]; then \
		chmod +x "$(FULL_REGRESSION_SCRIPT)"; \
		bash "$(FULL_REGRESSION_SCRIPT)"; \
	else \
		echo "❌ 完整回归测试脚本不存在: $(FULL_REGRESSION_SCRIPT)"; \
		exit 1; \
	fi

# 监控回归测试
.PHONY: test-regression-watch
test-regression-watch: build-fast
	@echo "👁️  启动监控回归测试..."
	@if [ -f "$(WATCH_REGRESSION_SCRIPT)" ]; then \
		chmod +x "$(WATCH_REGRESSION_SCRIPT)"; \
		bash "$(WATCH_REGRESSION_SCRIPT)"; \
	else \
		echo "❌ 监控回归测试脚本不存在: $(WATCH_REGRESSION_SCRIPT)"; \
		exit 1; \
	fi

# 监控回归测试（完整模式）
.PHONY: test-regression-watch-full
test-regression-watch-full: build
	@echo "👁️  启动监控回归测试（完整模式）..."
	@if [ -f "$(WATCH_REGRESSION_SCRIPT)" ]; then \
		chmod +x "$(WATCH_REGRESSION_SCRIPT)"; \
		bash "$(WATCH_REGRESSION_SCRIPT)" -m full; \
	else \
		echo "❌ 监控回归测试脚本不存在: $(WATCH_REGRESSION_SCRIPT)"; \
		exit 1; \
	fi

# 压力测试
.PHONY: test-regression-stress
test-regression-stress: build
	@echo "⚡ 运行压力测试..."
	@echo "🔄 运行100次快速回归测试..."
	@for i in {1..100}; do \
		echo "第$$i次测试..."; \
		if ! make test-regression-fast; then \
			echo "❌ 压力测试在第$$i次失败"; \
			exit 1; \
		fi; \
	done
	@echo "✅ 压力测试通过（100次测试）"

# 回归测试报告
.PHONY: test-regression-report
test-regression-report: test-regression-full
	@echo "📋 生成回归测试报告..."
	@if [ -f "test_report.txt" ]; then \
		cat test_report.txt; \
	else \
		echo "⚠️  测试报告文件不存在"; \
	fi

# 检查测试目录结构
.PHONY: test-check-structure
test-check-structure:
	@echo "🔍 检查测试目录结构..."
	@if [ -d "$(TESTDATA_DIR)" ]; then \
		echo "✅ 测试数据目录存在: $(TESTDATA_DIR)"; \
		tree $(TESTDATA_DIR) || ls -la $(TESTDATA_DIR); \
	else \
		echo "❌ 测试数据目录不存在: $(TESTDATA_DIR)"; \
		exit 1; \
	fi

# 设置测试脚本权限
.PHONY: test-setup
test-setup:
	@echo "🔧 设置测试脚本权限..."
	@if [ -d "$(REGRESSION_SCRIPTS_DIR)" ]; then \
		chmod +x $(REGRESSION_SCRIPTS_DIR)/*.sh; \
		echo "✅ 测试脚本权限设置完成"; \
	else \
		echo "❌ 回归测试脚本目录不存在: $(REGRESSION_SCRIPTS_DIR)"; \
		exit 1; \
	fi

# 构建主程序
.PHONY: build
build:
	@echo "🔨 构建 $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "✅ 构建完成: $(BIN_DIR)/$(APP_NAME)"

# 快速构建(不运行测试)
.PHONY: build-fast
build-fast:
	@echo "⚡ 快速构建 $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "✅ 快速构建完成: $(BIN_DIR)/$(APP_NAME)"

# 安装到系统
.PHONY: install
install: build
	@echo "📦 安装 $(APP_NAME) 到系统..."
	sudo cp $(BIN_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "✅ 安装完成: /usr/local/bin/$(APP_NAME)"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "🗑️  卸载 $(APP_NAME)..."
	sudo rm -f /usr/local/bin/$(APP_NAME)
	@echo "✅ 卸载完成"

# 运行程序
.PHONY: run
run: build
	@echo "🚀 运行 $(APP_NAME)..."
	./$(BIN_DIR)/$(APP_NAME)

# 开发模式运行(直接go run)
.PHONY: dev
dev:
	@echo "🔧 开发模式运行..."
	$(GO) run $(MAIN_PATH)

# 跨平台编译
.PHONY: build-all
build-all: clean
	@echo "🌍 跨平台编译..."
	@mkdir -p $(DIST_DIR)
	
	# Linux amd64
	@echo "📦 构建 Linux amd64..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_PATH)
	
	# Linux arm64
	@echo "📦 构建 Linux arm64..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 $(MAIN_PATH)
	
	# macOS amd64
	@echo "📦 构建 macOS amd64..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 $(MAIN_PATH)
	
	# macOS arm64 (Apple Silicon)
	@echo "📦 构建 macOS arm64..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN_PATH)
	
	# Windows amd64
	@echo "📦 构建 Windows amd64..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_PATH)
	
	@echo "✅ 跨平台编译完成，文件位于 $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

# 创建发布包
.PHONY: package
package: build-all
	@echo "📦 创建发布包..."
	@mkdir -p $(DIST_DIR)/packages
	
	# 创建tar.gz包
	cd $(DIST_DIR) && tar -czf packages/$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz $(APP_NAME)-linux-amd64
	cd $(DIST_DIR) && tar -czf packages/$(APP_NAME)-$(VERSION)-linux-arm64.tar.gz $(APP_NAME)-linux-arm64
	cd $(DIST_DIR) && tar -czf packages/$(APP_NAME)-$(VERSION)-darwin-amd64.tar.gz $(APP_NAME)-darwin-amd64
	cd $(DIST_DIR) && tar -czf packages/$(APP_NAME)-$(VERSION)-darwin-arm64.tar.gz $(APP_NAME)-darwin-arm64
	
	# 创建zip包(Windows)
	cd $(DIST_DIR) && zip packages/$(APP_NAME)-$(VERSION)-windows-amd64.zip $(APP_NAME)-windows-amd64.exe
	
	@echo "✅ 发布包创建完成:"
	@ls -la $(DIST_DIR)/packages/

# 基准测试
.PHONY: bench
bench:
	@echo "⚡ 运行基准测试..."
	$(GOTEST) -bench=. -benchmem ./...

# 代码检查
.PHONY: lint
lint:
	@echo "🔍 代码静态检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint 未安装，跳过静态检查"; \
		echo "💡 安装命令: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 清理
.PHONY: clean
clean:
	@echo "🧹 清理构建文件..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf $(DIST_DIR)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html test_report.txt
	@echo "✅ 清理完成"

# 深度清理(包括go mod cache)
.PHONY: clean-all
clean-all: clean
	@echo "🧹 深度清理..."
	$(GO) clean -modcache
	@echo "✅ 深度清理完成"

# 依赖管理
.PHONY: deps
deps:
	@echo "📦 更新依赖..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ 依赖更新完成"

# 查看依赖
.PHONY: deps-list
deps-list:
	@echo "📋 项目依赖列表:"
	$(GOMOD) list -m all

# 漏洞检查
.PHONY: security
security:
	@echo "🔒 安全漏洞检查..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "⚠️  govulncheck 未安装，跳过安全检查"; \
		echo "💡 安装命令: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

# 生成文档
.PHONY: docs
docs:
	@echo "📚 生成Go文档..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "🌐 启动文档服务器: http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "⚠️  godoc 未安装"; \
		echo "💡 安装命令: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# 项目信息
.PHONY: info
info:
	@echo "📋 AQL 项目信息:"
	@echo "  项目名称: $(APP_NAME)"
	@echo "  版本:     $(VERSION)"
	@echo "  Go版本:   $(shell $(GO) version)"
	@echo "  项目路径: $(PWD)"
	@echo "  主文件:   $(MAIN_PATH)"
	@echo "  测试目录: $(TESTDATA_DIR)"
	@echo ""
	@echo "📂 目录结构:"
	@tree -I 'bin|dist|build|.git' -L 2 || ls -la

# 帮助信息
.PHONY: help
help:
	@echo "🚀 AQL (Advanced Query Language) 构建工具"
	@echo ""
	@echo "📋 可用命令:"
	@echo "  make all          - 完整构建流程(fmt + test + build)"
	@echo "  make build        - 构建程序"
	@echo "  make build-fast   - 快速构建(跳过测试)"
	@echo "  make build-all    - 跨平台编译"
	@echo "  make package      - 创建发布包"
	@echo ""
	@echo "🧪 测试相关:"
	@echo "  make test         - 运行Go单元测试"
	@echo "  make test-coverage- 运行测试并生成覆盖率"
	@echo "  make bench        - 运行基准测试"
	@echo ""
	@echo "🔄 回归测试:"
	@echo "  make test-regression-fast   - 快速回归测试（< 30秒）"
	@echo "  make test-regression-full   - 完整回归测试（< 5分钟）"
	@echo "  make test-regression-watch  - 监控回归测试（自动）"
	@echo "  make test-regression-watch-full - 监控回归测试（完整模式）"
	@echo "  make test-regression-stress - 压力测试（100次）"
	@echo "  make test-regression-report - 生成测试报告"
	@echo ""
	@echo "🔧 测试工具:"
	@echo "  make test-setup     - 设置测试脚本权限"
	@echo "  make test-check-structure - 检查测试目录结构"
	@echo ""
	@echo "🔧 开发工具:"
	@echo "  make run          - 运行程序"
	@echo "  make dev          - 开发模式运行"
	@echo "  make fmt          - 格式化代码"
	@echo "  make lint         - 代码静态检查"
	@echo ""
	@echo "📦 部署相关:"
	@echo "  make install      - 安装到系统"
	@echo "  make uninstall    - 从系统卸载"
	@echo ""
	@echo "🧹 清理工具:"
	@echo "  make clean        - 清理构建文件"
	@echo "  make clean-all    - 深度清理"
	@echo ""
	@echo "📋 其他工具:"
	@echo "  make deps         - 更新依赖"
	@echo "  make security     - 安全检查"
	@echo "  make docs         - 生成文档"
	@echo "  make info         - 项目信息"
	@echo "  make help         - 显示此帮助"
	@echo ""
	@echo "💡 开发工作流建议:"
	@echo "  1. 开发时：make test-regression-watch"
	@echo "  2. 提交前：make test-regression-fast"
	@echo "  3. 发布前：make test-regression-full"

# 默认显示帮助
.DEFAULT_GOAL := help 