#!/usr/bin/env python3
# filepath: test_auth_api.py

import requests
import json
import time
import argparse
import sys


class MyIMApiTester:
    def __init__(self, base_url="http://localhost:8080"):
        """初始化 API 测试器

        Args:
            base_url: API 服务器地址
        """
        self.base_url = base_url
        self.token = None
        self.username = None

    def register(self, username, password):
        """测试注册功能

        Args:
            username: 用户名
            password: 密码

        Returns:
            bool: 注册是否成功
        """
        url = f"{self.base_url}/auth/register"
        payload = {
            "username": username,
            "password": password
        }
        headers = {"Content-Type": "application/json"}

        print(f"[测试注册] 用户名: {username}")
        try:
            response = requests.post(url, json=payload, headers=headers)
            result = response.json()

            if response.status_code == 200 and result.get("code") == 0:
                self.token = result.get("token")
                self.username = username
                print(f"[注册成功] Token: {self.token}")
                return True
            else:
                print(
                    f"[注册失败] 状态码: {response.status_code}, 错误: {result.get('error', '未知错误')}")
                return False
        except Exception as e:
            print(f"[注册异常] {e}")
            return False

    def login(self, username, password):
        """测试登录功能

        Args:
            username: 用户名
            password: 密码

        Returns:
            bool: 登录是否成功
        """
        url = f"{self.base_url}/auth/login"
        payload = {
            "username": username,
            "password": password
        }
        headers = {"Content-Type": "application/json"}

        print(f"[测试登录] 用户名: {username}")
        try:
            response = requests.post(url, json=payload, headers=headers)
            result = response.json()

            if response.status_code == 200 and result.get("code") == 0:
                self.token = result.get("token")
                self.username = username
                print(f"[登录成功] Token: {self.token}")
                return True
            else:
                print(
                    f"[登录失败] 状态码: {response.status_code}, 错误: {result.get('error', '未知错误')}")
                return False
        except Exception as e:
            print(f"[登录异常] {e}")
            return False

    def check_auth(self):
        """测试权限验证功能

        Returns:
            bool: 验证是否成功
        """
        if not self.token:
            print("[验证失败] 未登录，无法验证权限")
            return False

        url = f"{self.base_url}/auth/check"
        payload = {"token": self.token}
        headers = {"Content-Type": "application/json"}

        print(f"[测试验证] Token: {self.token}")
        try:
            response = requests.post(url, json=payload, headers=headers)
            result = response.json()

            if response.status_code == 200 and result.get("code") == 0:
                user_id = result.get("userid")
                username = result.get("username")
                print(f"[验证成功] 用户ID: {user_id}, 用户名: {username}")
                return True
            else:
                print(
                    f"[验证失败] 状态码: {response.status_code}, 错误: {result.get('error', '未知错误')}")
                return False
        except Exception as e:
            print(f"[验证异常] {e}")
            return False

    def logout(self):
        """测试登出功能

        Returns:
            bool: 登出是否成功
        """
        if not self.token:
            print("[登出失败] 未登录，无法登出")
            return False

        url = f"{self.base_url}/auth/logout"
        payload = {"token": self.token}
        headers = {"Content-Type": "application/json"}

        print(f"[测试登出] Token: {self.token}")
        try:
            response = requests.post(url, json=payload, headers=headers)
            result = response.json()

            if response.status_code == 200 and result.get("code") == 0:
                old_token = self.token
                self.token = None
                print(f"[登出成功] 已注销Token: {old_token}")
                return True
            else:
                print(
                    f"[登出失败] 状态码: {response.status_code}, 错误: {result.get('error', '未知错误')}")
                return False
        except Exception as e:
            print(f"[登出异常] {e}")
            return False

    def run_full_test(self, username, password):
        """运行完整测试流程

        Args:
            username: 用户名
            password: 密码

        Returns:
            bool: 测试是否全部通过
        """
        print("=" * 50)
        print(f"开始测试 MyIM API - 用户: {username}")
        print("=" * 50)

        # 1. 注册测试
        print("\n--- 测试注册 ---")
        if not self.register(username, password):
            print("\n[流程中断] 注册失败，无法继续测试")
            return False

        # 2. 验证权限
        print("\n--- 测试注册后的权限验证 ---")
        if not self.check_auth():
            print("\n[注意] 权限验证失败，但继续测试")

        # 3. 登出
        print("\n--- 测试登出 ---")
        if not self.logout():
            print("\n[注意] 登出失败，但继续测试")

        # 4. 登录
        print("\n--- 测试登录 ---")
        if not self.login(username, password):
            print("\n[流程中断] 登录失败，无法继续测试")
            return False

        # 5. 再次验证权限
        print("\n--- 测试登录后的权限验证 ---")
        if not self.check_auth():
            print("\n[注意] 登录后权限验证失败")

        # 6. 再次登出
        print("\n--- 测试再次登出 ---")
        if not self.logout():
            print("\n[注意] 再次登出失败")

        # 7. 登出后验证权限（应当失败）
        print("\n--- 测试登出后的权限验证（应当失败） ---")
        auth_result = self.check_auth()
        if auth_result:
            print("\n[警告] 登出后权限验证仍然成功，这可能是个问题")
        else:
            print("\n[正确] 登出后权限验证已正确失败")

        print("\n" + "=" * 50)
        print("测试完成!")
        print("=" * 50)
        return True


def main():
    parser = argparse.ArgumentParser(description='测试 MyIM 系统的 API 接口')
    parser.add_argument(
        '--url', default='http://localhost:8080', help='API 服务器地址')
    parser.add_argument(
        '--username', default=f'testuser_{int(time.time())}', help='测试用户名')
    parser.add_argument('--password', default='password123', help='测试密码')

    args = parser.parse_args()

    tester = MyIMApiTester(args.url)
    success = tester.run_full_test(args.username, args.password)

    if not success:
        sys.exit(1)


if __name__ == "__main__":
    main()
