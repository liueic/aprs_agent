# APRS Agent Makefile

# 变量定义
BINARY_NAME=aprs_agent
BUILD_DIR=build
CONFIG_FILE=app.conf

# Go相关变量
GO=go
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

# 版本信息
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

.PHONY: all build clean run test install uninstall help

# 默认目标
all: clean build

# 构建项目
build:
	@echo "构建 APRS Agent..."
	@mkdir -p ${BUILD_DIR}
	${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} .
	@echo "构建完成: ${BUILD_DIR}/${BINARY_NAME}"

# 构建特定平台
build-linux:
	@echo "构建 Linux 版本..."
	@mkdir -p ${BUILD_DIR}
	GOOS=linux GOARCH=amd64 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 .
	@echo "构建完成: ${BUILD_DIR}/${BINARY_NAME}-linux-amd64"

build-windows:
	@echo "构建 Windows 版本..."
	@mkdir -p ${BUILD_DIR}
	GOOS=windows GOARCH=amd64 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe .
	@echo "构建完成: ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe"

build-macos:
	@echo "构建 macOS 版本..."
	@mkdir -p ${BUILD_DIR}
	GOOS=darwin GOARCH=amd64 ${GO} build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 .
	@echo "构建完成: ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64"

# 构建所有平台
build-all: build-linux build-windows build-macos

# 运行项目
run:
	@echo "运行 APRS Agent..."
	${GO} run main.go

# 运行构建后的二进制文件
run-binary:
	@echo "运行构建后的二进制文件..."
	@if [ ! -f "${BUILD_DIR}/${BINARY_NAME}" ]; then \
		echo "请先构建项目: make build"; \
		exit 1; \
	fi
	@if [ ! -f "${CONFIG_FILE}" ]; then \
		echo "配置文件 ${CONFIG_FILE} 不存在"; \
		exit 1; \
	fi
	cd ${BUILD_DIR} && ./${BINARY_NAME}

# 安装依赖
deps:
	@echo "安装 Go 依赖..."
	${GO} mod download
	${GO} mod tidy

# 测试
test:
	@echo "运行测试..."
	${GO} test -v ./...

# 测试覆盖率
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	${GO} test -v -coverprofile=coverage.out ./...
	${GO} tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 代码格式化
fmt:
	@echo "格式化代码..."
	${GO} fmt ./...

# 代码检查
lint:
	@echo "检查代码..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装，跳过代码检查"; \
	fi

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf ${BUILD_DIR}
	@rm -f coverage.out coverage.html
	@echo "清理完成"

# 安装到系统
install:
	@echo "安装 APRS Agent..."
	@if [ ! -f "${BUILD_DIR}/${BINARY_NAME}" ]; then \
		echo "请先构建项目: make build"; \
		exit 1; \
	fi
	@sudo cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/
	@sudo chmod +x /usr/local/bin/${BINARY_NAME}
	@echo "安装完成: /usr/local/bin/${BINARY_NAME}"

# 从系统卸载
uninstall:
	@echo "卸载 APRS Agent..."
	@sudo rm -f /usr/local/bin/${BINARY_NAME}
	@echo "卸载完成"

# 创建发布包
package:
	@echo "创建发布包..."
	@mkdir -p ${BUILD_DIR}/release
	@cp ${CONFIG_FILE} ${BUILD_DIR}/release/
	@cp README.md ${BUILD_DIR}/release/
	@cp LICENSE ${BUILD_DIR}/release/ 2>/dev/null || echo "LICENSE 文件不存在"
	@cd ${BUILD_DIR} && tar -czf release/aprs_agent-${VERSION}-${GOOS}-${GOARCH}.tar.gz release/ ${BINARY_NAME}
	@echo "发布包已创建: ${BUILD_DIR}/release/aprs_agent-${VERSION}-${GOOS}-${GOARCH}.tar.gz"

# 开发模式运行
dev:
	@echo "开发模式运行..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air 未安装，使用标准模式运行"; \
		${GO} run main.go; \
	fi

# 显示帮助信息
help:
	@echo "APRS Agent Makefile 帮助"
	@echo ""
	@echo "可用目标:"
	@echo "  build          - 构建项目"
	@echo "  build-linux    - 构建 Linux 版本"
	@echo "  build-windows  - 构建 Windows 版本"
	@echo "  build-macos    - 构建 macOS 版本"
	@echo "  build-all      - 构建所有平台版本"
	@echo "  run            - 运行项目 (go run)"
	@echo "  run-binary     - 运行构建后的二进制文件"
	@echo "  deps           - 安装依赖"
	@echo "  test           - 运行测试"
	@echo "  test-coverage  - 运行测试并生成覆盖率报告"
	@echo "  fmt            - 格式化代码"
	@echo "  lint           - 代码检查"
	@echo "  clean          - 清理构建文件"
	@echo "  install        - 安装到系统"
	@echo "  uninstall      - 从系统卸载"
	@echo "  package        - 创建发布包"
	@echo "  dev            - 开发模式运行"
	@echo "  help           - 显示此帮助信息"
	@echo ""
	@echo "示例:"
	@echo "  make build     # 构建项目"
	@echo "  make run       # 运行项目"
	@echo "  make clean     # 清理文件"

# 显示版本信息
version:
	@echo "APRS Agent 版本: ${VERSION}"
	@echo "构建时间: ${BUILD_TIME}"
	@echo "目标平台: ${GOOS}/${GOARCH}"
