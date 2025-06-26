#!/usr/bin/env python3
"""
测试 ggcode 中的 go-judge 集成
"""

import requests
import json

# 配置
BASE_URL = "http://localhost:8080"
GO_JUDGE_URL = "http://localhost:5050"

def test_go_judge_direct():
    """直接测试 go-judge 服务"""
    print("=== 测试 go-judge 服务 ===")
    
    # 测试版本信息
    try:
        response = requests.get(f"{GO_JUDGE_URL}/version")
        print(f"go-judge 版本: {response.status_code}")
        if response.status_code == 200:
            print(f"版本信息: {response.text}")
    except Exception as e:
        print(f"go-judge 服务连接失败: {e}")
        return False
    
    # 测试简单的 C++ 程序
    test_request = {
        "cmd": [{
            "args": ["/usr/bin/bash", "-c", "echo 'Hello World'"],
            "env": ["PATH=/usr/bin:/bin"],
            "cpuLimit": 5000000000,  # 5秒
            "clockLimit": 5000000000,
            "memoryLimit": 134217728,  # 128MB
            "procLimit": 50,
            "copyOut": ["stdout", "stderr"]
        }]
    }
    
    try:
        response = requests.post(f"{GO_JUDGE_URL}/run", json=test_request)
        print(f"go-judge 执行测试: {response.status_code}")
        if response.status_code == 200:
            result = response.json()
            print(f"执行结果: {json.dumps(result, indent=2)}")
    except Exception as e:
        print(f"go-judge 执行测试失败: {e}")
        return False
    
    return True

def test_ggcode_go_judge_api():
    """测试 ggcode 的 go-judge API"""
    print("\n=== 测试 ggcode go-judge API ===")
    
    # 测试健康检查
    try:
        response = requests.get(f"{BASE_URL}/api/go-judge/health")
        print(f"健康检查: {response.status_code}")
        if response.status_code == 200:
            print(f"健康状态: {response.json()}")
    except Exception as e:
        print(f"健康检查失败: {e}")
        return False
    
    # 测试支持的语言
    try:
        response = requests.get(f"{BASE_URL}/api/go-judge/languages")
        print(f"支持的语言: {response.status_code}")
        if response.status_code == 200:
            print(f"语言列表: {response.json()}")
    except Exception as e:
        print(f"获取语言列表失败: {e}")
    
    # 测试简单的代码执行
    test_code = {
        "language": "python",
        "code": "print('Hello, World!')",
        "input": "",
        "timeLimit": 5000,
        "memoryLimit": 131072
    }
    
    try:
        response = requests.post(f"{BASE_URL}/api/go-judge/execute", json=test_code)
        print(f"代码执行测试: {response.status_code}")
        if response.status_code == 200:
            result = response.json()
            print(f"执行结果: {json.dumps(result, indent=2)}")
        else:
            print(f"执行失败: {response.text}")
    except Exception as e:
        print(f"代码执行测试失败: {e}")
        return False
    
    return True

def test_level_4_factorial():
    """测试关卡4的阶乘问题"""
    print("\n=== 测试关卡4阶乘问题 ===")
    
    # 正确的阶乘代码
    correct_code = """
def factorial(n):
    if n <= 1:
        return 1
    return n * factorial(n-1)

n = int(input())
print(factorial(n))
"""
    
    # 错误的代码
    wrong_code = """
n = int(input())
print(n * 2)  # 错误的计算
"""
    
    test_cases = [
        ("正确代码", correct_code, True),
        ("错误代码", wrong_code, False)
    ]
    
    for name, code, should_pass in test_cases:
        print(f"\n测试 {name}:")
        test_request = {
            "code": code,
            "language": "python"
        }
        
        try:
            response = requests.post(f"{BASE_URL}/api/go-judge/level/4/test", json=test_request)
            print(f"测试状态: {response.status_code}")
            if response.status_code == 200:
                result = response.json()
                print(f"测试结果: {json.dumps(result, indent=2)}")
                
                # 检查结果是否符合预期
                if 'result' in result:
                    status = result['result'].get('status', '')
                    if should_pass and status == 'Accepted':
                        print("✅ 测试通过，结果正确")
                    elif not should_pass and status != 'Accepted':
                        print("✅ 测试通过，正确识别错误")
                    else:
                        print(f"❌ 测试结果不符合预期: {status}")
            else:
                print(f"测试失败: {response.text}")
        except Exception as e:
            print(f"测试请求失败: {e}")

if __name__ == "__main__":
    print("开始测试 go-judge 集成...")
    
    # 测试 go-judge 服务
    if not test_go_judge_direct():
        print("❌ go-judge 服务测试失败，请确保服务正在运行")
        exit(1)
    
    # 测试 ggcode API
    if not test_ggcode_go_judge_api():
        print("❌ ggcode go-judge API 测试失败")
        exit(1)
    
    # 测试具体关卡
    test_level_4_factorial()
    
    print("\n🎉 所有测试完成！") 