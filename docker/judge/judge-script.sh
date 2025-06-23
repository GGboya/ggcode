#!/bin/bash

# 代码评测脚本
# 用法: ./judge-script.sh <language> <time_limit> <memory_limit> <input_file> <expected_output_file>

set -e

LANGUAGE=$1
TIME_LIMIT=$2          # 秒
MEMORY_LIMIT=$3        # MB
INPUT_FILE=$4
EXPECTED_FILE=$5
SOURCE_FILE="Main"

# 检查参数
if [ $# -ne 5 ]; then
    echo "错误: 参数不正确"
    echo "用法: $0 <language> <time_limit> <memory_limit> <input_file> <expected_output_file>"
    exit 1
fi

# 结果文件
ACTUAL_OUTPUT="/tmp/actual_output.txt"
COMPILE_LOG="/tmp/compile.log"
RUNTIME_LOG="/tmp/runtime.log"

# 清理之前的文件
rm -f "$ACTUAL_OUTPUT" "$COMPILE_LOG" "$RUNTIME_LOG"

echo "=== 开始评测 ==="
echo "语言: $LANGUAGE"
echo "时间限制: ${TIME_LIMIT}s"
echo "内存限制: ${MEMORY_LIMIT}MB"

case $LANGUAGE in
    "cpp")
        echo "编译 C++ 代码..."
        if ! g++ -std=c++17 -O2 -Wall -Wextra -static -DONLINE_JUDGE -o "${SOURCE_FILE}" "${SOURCE_FILE}.cpp" 2>"$COMPILE_LOG"; then
            echo "编译失败:"
            cat "$COMPILE_LOG"
            exit 2
        fi
        
        echo "运行 C++ 程序..."
        timeout "${TIME_LIMIT}s" ./"${SOURCE_FILE}" < "$INPUT_FILE" > "$ACTUAL_OUTPUT" 2>"$RUNTIME_LOG"
        RESULT=$?
        ;;
        
    "python")
        echo "运行 Python 代码..."
        timeout "${TIME_LIMIT}s" python3 "${SOURCE_FILE}.py" < "$INPUT_FILE" > "$ACTUAL_OUTPUT" 2>"$RUNTIME_LOG"
        RESULT=$?
        ;;
        
    "java")
        echo "编译 Java 代码..."
        if ! javac -encoding UTF-8 "${SOURCE_FILE}.java" 2>"$COMPILE_LOG"; then
            echo "编译失败:"
            cat "$COMPILE_LOG"
            exit 2
        fi
        
        echo "运行 Java 程序..."
        timeout "${TIME_LIMIT}s" java -Xmx${MEMORY_LIMIT}m -Dfile.encoding=UTF-8 -Djava.security.manager -Djava.security.policy=/opt/java.policy Main < "$INPUT_FILE" > "$ACTUAL_OUTPUT" 2>"$RUNTIME_LOG"
        RESULT=$?
        ;;
        
    "go")
        echo "编译 Go 代码..."
        if ! go build -o "${SOURCE_FILE}" "${SOURCE_FILE}.go" 2>"$COMPILE_LOG"; then
            echo "编译失败:"
            cat "$COMPILE_LOG"
            exit 2
        fi
        
        echo "运行 Go 程序..."
        timeout "${TIME_LIMIT}s" ./"${SOURCE_FILE}" < "$INPUT_FILE" > "$ACTUAL_OUTPUT" 2>"$RUNTIME_LOG"
        RESULT=$?
        ;;
        
    *)
        echo "不支持的语言: $LANGUAGE"
        exit 1
        ;;
esac

# 检查运行结果
if [ $RESULT -eq 124 ]; then
    echo "TLE: 时间超限"
    exit 3
elif [ $RESULT -ne 0 ]; then
    echo "RE: 运行时错误 (退出代码: $RESULT)"
    if [ -s "$RUNTIME_LOG" ]; then
        echo "错误信息:"
        cat "$RUNTIME_LOG"
    fi
    exit 4
fi

# 比较输出
echo "比较输出结果..."
if cmp -s "$ACTUAL_OUTPUT" "$EXPECTED_FILE"; then
    echo "AC: 答案正确"
    exit 0
else
    echo "WA: 答案错误"
    echo "期望输出:"
    cat "$EXPECTED_FILE"
    echo "实际输出:"
    cat "$ACTUAL_OUTPUT"
    exit 5
fi 