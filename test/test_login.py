#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# 测试 GoChat API 用户认证功能

import requests
import json
import time
import random
import string

# 配置 API 基础 URL
BASE_URL = "http://localhost:8080"  # 根据实际API部署修改此URL


def generate_random_username(length=8):
    """生成随机用户名，用于测试注册"""
    return ''.join(random.choice(string.ascii_lowercase) for _ in range(length))


def print_response(response):
    """打印响应内容和状态码"""
    print(f"Status Code: {response.json().get("code")}")
    print(
        f"Response: {json.dumps(response.json(), indent=2, ensure_ascii=False)}")
    print("-" * 50)


def test_register(username, password):
    """测试用户注册功能"""
    print("\n===== 测试用户注册 =====")
    url = f"{BASE_URL}/user/register"
    payload = {
        "username": username,
        "password": password
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)
        return response.json().get("code") == 200
    except Exception as e:
        print(f"注册请求异常: {e}")
        return False


def test_register_duplicate(username, password):
    """测试重复注册"""
    print("\n===== 测试重复注册（预期失败）=====")
    url = f"{BASE_URL}/user/register"
    payload = {
        "username": username,
        "password": password
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)
        # 预期返回400，因为用户名已存在
        return response.json().get("code") == 400
    except Exception as e:
        print(f"重复注册请求异常: {e}")
        return False


def test_login(username, password):
    """测试用户登录功能"""
    print("\n===== 测试用户登录 =====")
    url = f"{BASE_URL}/user/login"
    payload = {
        "username": username,
        "password": password
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)

        if response.json().get("code") == 200:
            data = response.json()
            return data.get("token"), data.get("user_id")
        return None, None
    except Exception as e:
        print(f"登录请求异常: {e}")
        return None, None


def test_login_invalid(username, wrong_password):
    """测试无效登录"""
    print("\n===== 测试无效登录（预期失败）=====")
    url = f"{BASE_URL}/user/login"
    payload = {
        "username": username,
        "password": wrong_password
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)
        # 预期返回400，因为密码错误
        return response.json().get("code") == 400
    except Exception as e:
        print(f"无效登录请求异常: {e}")
        return False


def test_check_auth(token):
    """测试身份验证检查"""
    print("\n===== 测试身份验证检查 =====")
    url = f"{BASE_URL}/user/checkAuth"
    payload = {
        "token": token
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)
        return response.json().get("code") == 200
    except Exception as e:
        print(f"验证检查请求异常: {e}")
        return False


def test_check_auth_invalid(invalid_token):
    """测试无效身份验证检查"""
    print("\n===== 测试无效身份验证检查（预期失败）=====")
    url = f"{BASE_URL}/user/checkAuth"
    payload = {
        "token": invalid_token
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)
        # 预期返回400，因为令牌无效
        return response.json().get("code") == 400
    except Exception as e:
        print(f"无效验证检查请求异常: {e}")
        return False


def test_logout(token):
    """测试用户登出功能"""
    print("\n===== 测试用户登出 =====")
    url = f"{BASE_URL}/user/logout"
    payload = {
        "token": token
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)
        return response.json().get("code") == 200
    except Exception as e:
        print(f"登出请求异常: {e}")
        return False


def test_logout_invalid(invalid_token):
    """测试无效登出"""
    print("\n===== 测试无效登出（预期失败）=====")
    url = f"{BASE_URL}/user/logout"
    payload = {
        "token": invalid_token
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)
        # 预期返回400，因为令牌无效或已过期
        return response.json().get("code") == 400
    except Exception as e:
        print(f"无效登出请求异常: {e}")
        return False


def test_auth_after_logout(token):
    """测试登出后尝试使用令牌"""
    print("\n===== 测试登出后尝试使用令牌（预期失败）=====")
    url = f"{BASE_URL}/user/checkAuth"
    payload = {
        "token": token
    }

    try:
        response = requests.post(url, json=payload)
        print_response(response)
        # 预期返回400，因为令牌已经被注销
        return response.json().get("code") == 400
    except Exception as e:
        print(f"登出后验证请求异常: {e}")
        return False


def run_all_tests():
    """运行所有测试用例"""
    test_results = {}

    # 生成随机用户名和密码用于测试
    username = generate_random_username()
    password = "TestPassword123"
    wrong_password = "WrongPassword123"
    invalid_token = "invalid_token_123456789"

    print(f"使用测试用户名: {username}")

    # 测试注册
    test_results["注册"] = test_register(username, password)

    # 测试重复注册
    test_results["重复注册"] = test_register_duplicate(username, password)

    # 测试无效登录
    test_results["无效登录"] = test_login_invalid(username, wrong_password)

    # 测试有效登录
    token, user_id = test_login(username, password)
    test_results["登录"] = token is not None and user_id is not None

    if token:
        # 测试身份验证检查
        test_results["身份验证检查"] = test_check_auth(token)

        # 测试无效身份验证检查
        test_results["无效身份验证检查"] = test_check_auth_invalid(invalid_token)

        # 测试有效登出
        test_results["登出"] = test_logout(token)

        # 测试登出后使用令牌
        test_results["登出后验证"] = test_auth_after_logout(token)

    # 测试无效登出
    test_results["无效登出"] = test_logout_invalid(invalid_token)

    # 打印测试结果摘要
    print("\n===== 测试结果摘要 =====")
    for test_name, result in test_results.items():
        status = "通过" if result else "失败"
        print(f"{test_name}: {status}")

    # 计算通过率
    passed = sum(1 for result in test_results.values() if result)
    total = len(test_results)
    pass_rate = passed / total * 100 if total > 0 else 0

    print(f"\n通过率: {pass_rate:.2f}% ({passed}/{total})")


if __name__ == "__main__":
    run_all_tests()
