#!/bin/bash

# 设置颜色输出
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
NC="\033[0m" # No Color

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 设置输出目录
OUTPUT_DIR="$SCRIPT_DIR/bin"
mkdir -p "$OUTPUT_DIR"

# 获取模块名称
MODULE_NAME=$(grep -E '^module' go.mod | awk '{print $2}')
BINARY_NAME=$(basename "$MODULE_NAME")

# 默认编译所有平台
BUILD_ALL=true

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case $1 in
    --platform=*)
      PLATFORM="${1#*=}"
      BUILD_ALL=false
      shift
      ;;
    --os=*)
      TARGET_OS="${1#*=}"
      BUILD_ALL=false
      shift
      ;;
    --arch=*)
      TARGET_ARCH="${1#*=}"
      BUILD_ALL=false
      shift
      ;;
    --help)
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  --platform=<platform>  Build for specific platform (e.g., darwin-amd64, darwin-arm64, windows-amd64, linux-arm64)"
      echo "  --os=<os>            Build for specific OS (darwin, windows, linux)"
      echo "  --arch=<arch>        Build for specific architecture (amd64, arm64, 386)"
      echo "  --help               Show this help message"
      echo ""
      echo "Environment variables:"
      echo "  TARGET_OS            Same as --os"
      echo "  TARGET_ARCH          Same as --arch"
      echo "  TARGET_PLATFORM      Same as --platform"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# 检查环境变量
if [[ -n "$TARGET_PLATFORM" ]]; then
  PLATFORM="$TARGET_PLATFORM"
  BUILD_ALL=false
fi

if [[ -n "$TARGET_OS" ]]; then
  TARGET_OS="$TARGET_OS"
  BUILD_ALL=false
fi

if [[ -n "$TARGET_ARCH" ]]; then
  TARGET_ARCH="$TARGET_ARCH"
  BUILD_ALL=false
fi

# 定义要构建的平台
if [[ "$BUILD_ALL" = true ]]; then
  # 默认构建所有主流平台
  PLATFORMS=(
    "darwin/amd64"    # Mac Intel
    "darwin/arm64"    # Mac M1/M2
    "windows/amd64"   # Windows x64
    "windows/386"     # Windows x86
    "linux/amd64"     # Linux x64
    "linux/arm64"     # Linux ARM64
  )
else
  # 根据指定的平台参数构建
  if [[ -n "$PLATFORM" ]]; then
    IFS='-' read -r os arch <<< "$PLATFORM"
    PLATFORMS=("$os/$arch")
  elif [[ -n "$TARGET_OS" && -n "$TARGET_ARCH" ]]; then
    PLATFORMS=("$TARGET_OS/$TARGET_ARCH")
  elif [[ -n "$TARGET_OS" ]]; then
    # 如果只指定了OS，则构建该OS下的所有架构
    case "$TARGET_OS" in
      darwin)
        PLATFORMS=("darwin/amd64" "darwin/arm64")
        ;;
      windows)
        PLATFORMS=("windows/amd64" "windows/386")
        ;;
      linux)
        PLATFORMS=("linux/amd64" "linux/arm64")
        ;;
      *)
        echo "Unsupported OS: $TARGET_OS"
        exit 1
        ;;
    esac
  elif [[ -n "$TARGET_ARCH" ]]; then
    # 如果只指定了架构，则构建所有OS下的该架构
    case "$TARGET_ARCH" in
      amd64)
        PLATFORMS=("darwin/amd64" "windows/amd64" "linux/amd64")
        ;;
      arm64)
        PLATFORMS=("darwin/arm64" "linux/arm64")
        ;;
      386)
        PLATFORMS=("windows/386")
        ;;
      *)
        echo "Unsupported architecture: $TARGET_ARCH"
        exit 1
        ;;
    esac
  fi
fi

echo -e "${YELLOW}Building $BINARY_NAME for the following platforms:${NC}"
for platform in "${PLATFORMS[@]}"; do
  echo "  - $platform"
done
echo ""

# 编译函数
build() {
  local os=$1
  local arch=$2
  local output_name=$BINARY_NAME
  
  # Windows平台添加.exe后缀
  if [[ "$os" == "windows" ]]; then
    output_name="${output_name}.exe"
  fi
  
  echo -e "${YELLOW}Building for $os/$arch...${NC}"
  
  # 设置环境变量并编译
  output_file="$OUTPUT_DIR/${BINARY_NAME}-${os}-${arch}"
  if [[ "$os" == "windows" ]]; then
    output_file="${output_file}.exe"
  fi

  if [[ "$os" == "linux" ]]; then
    export CGO_ENABLED=0
  fi
  
  GOOS=$os GOARCH=$arch go build -o "$output_file" .
  
  if [[ $? -eq 0 ]]; then
    echo -e "${GREEN}Successfully built $output_file${NC}"
  else
    echo -e "\033[0;31mFailed to build for $os/$arch\033[0m"
  fi
}

# 开始编译
for platform in "${PLATFORMS[@]}"; do
  IFS='/' read -r os arch <<< "$platform"
  build "$os" "$arch"
done

echo -e "\n${GREEN}Build process completed!${NC}"
echo -e "Binaries are available in: $OUTPUT_DIR"