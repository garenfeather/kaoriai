#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
ProtonMail 邮件读取脚本 - 支持手动 CAPTCHA

工作流程:
1. 脚本尝试登录
2. 如果遇到 CAPTCHA，指导你在浏览器中完成
3. 从浏览器开发者工具复制认证 token
4. 脚本使用 token 继续访问邮件
"""

import os
import sys
import time
import json
from datetime import datetime

# ===== 修复 libcrypto 兼容性问题 =====
sys.modules['proton.srp._ctsrp'] = None
# ==========================================

from proton.api import Session
from proton.exceptions import ProtonError


def save_session(session, filename='proton_session.json'):
    """保存会话到文件"""
    try:
        session_data = session.dump()
        with open(filename, 'w') as f:
            json.dump(session_data, f, indent=2)
        print(f'✓ 会话已保存到: {filename}')
        return True
    except Exception as e:
        print(f'✗ 保存会话失败: {e}')
        return False


def load_session(filename='proton_session.json'):
    """从文件加载会话"""
    try:
        if not os.path.exists(filename):
            return None

        with open(filename, 'r') as f:
            session_data = json.load(f)

        print(f'✓ 从文件加载会话: {filename}')
        session = Session.load(session_data)

        # 测试会话是否有效
        try:
            session.api_request('/tests/ping', method='get')
            print('✓ 会话有效')
            return session
        except:
            print('✗ 会话已过期')
            return None

    except Exception as e:
        print(f'✗ 加载会话失败: {e}')
        return None


def manual_captcha_guide():
    """显示手动完成 CAPTCHA 的指导"""
    print('\n' + '='*70)
    print('需要完成 CAPTCHA 验证')
    print('='*70)
    print('\n请按照以下步骤操作:\n')

    print('步骤 1: 在浏览器中登录 ProtonMail')
    print('  1. 打开浏览器访问: https://account.proton.me/login')
    print('  2. 输入用户名和密码')
    print('  3. 完成 CAPTCHA 验证')
    print('  4. 成功登录到 ProtonMail\n')

    print('步骤 2: 提取访问令牌 (Access Token)')
    print('  1. 在浏览器中按 F12 打开开发者工具')
    print('  2. 切换到 "Application" (或 "应用程序") 标签页')
    print('  3. 左侧选择 "Session Storage" -> "https://account.proton.me"')
    print('  4. 找到名为 "AUTHORIZATION" 或类似的键')
    print('  5. 或者切换到 "Console" 标签，输入并执行:')
    print('     sessionStorage.getItem("AUTHORIZATION")')
    print('  6. 复制整个令牌值\n')

    print('步骤 3: 输入令牌')
    print('  将令牌粘贴到下面的提示中\n')
    print('='*70 + '\n')

    token = input('请粘贴访问令牌 (Access Token): ').strip()

    if not token:
        print('\n✗ 未输入令牌')
        return None

    return token


def create_session_with_token(token, api_url="https://mail.proton.me/api"):
    """使用访问令牌创建会话"""
    try:
        session = Session(
            api_url=api_url,
            appversion="web-mail@5.0.88.4",
            user_agent="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
        )

        # 手动设置会话数据
        session._session_data = {
            'AccessToken': token,
            'RefreshToken': '',  # 暂时为空
            'UID': ''  # 暂时为空
        }

        # 设置认证头
        session.s.headers['Authorization'] = f'Bearer {token}'

        # 测试令牌是否有效
        try:
            response = session.api_request('/core/v4/users', method='get')
            if response:
                print('✓ 令牌有效，已成功创建会话')
                return session
        except Exception as e:
            print(f'✗ 令牌无效或已过期: {e}')
            return None

    except Exception as e:
        print(f'✗ 创建会话失败: {e}')
        return None


def read_latest_email():
    """读取最新邮件"""
    username = os.getenv('PM_USERNAME')
    password = os.getenv('PM_PASSWORD')

    if not username or not password:
        print('错误: 请设置环境变量 PM_USERNAME 和 PM_PASSWORD', file=sys.stderr)
        sys.exit(1)

    session = None
    session_file = 'proton_session.json'

    # 尝试加载已保存的会话
    print('检查已保存的会话...')
    session = load_session(session_file)

    if session is None:
        print('\n未找到有效会话，尝试登录...')

        # 尝试正常登录
        try:
            print('正在连接到 ProtonMail...')

            session = Session(
                api_url="https://mail.proton.me/api",
                appversion="web-mail@5.0.88.4",
                user_agent="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
            )

            print('正在认证...')
            session.authenticate(username, password)

            print('✓ 登录成功！')

            # 保存会话
            save_session(session, session_file)

        except ProtonError as e:
            error_msg = str(e)

            if 'CAPTCHA' in error_msg or 'captcha' in error_msg.lower():
                print('\n' + '='*70)
                print('⚠️  需要 CAPTCHA 验证')
                print('='*70)

                # 显示手动完成 CAPTCHA 的指导
                token = manual_captcha_guide()

                if token:
                    session = create_session_with_token(token)

                    if session:
                        # 保存会话供下次使用
                        save_session(session, session_file)
                    else:
                        print('\n无法创建会话，请重试')
                        sys.exit(1)
                else:
                    print('\n操作取消')
                    sys.exit(1)
            else:
                print(f'\n✗ 认证失败: {error_msg}')
                raise

    if session is None:
        print('\n无法建立会话')
        sys.exit(1)

    # 获取邮件列表
    try:
        print('\n获取邮件信息...')

        response = session.api_request(
            endpoint='/mail/v4/messages',
            method='get',
            jsondata={
                'LabelID': '0',
                'Page': 0,
                'PageSize': 10,
                'Limit': 10,
                'Sort': 'Time',
                'Desc': 1
            }
        )

        if response and 'Messages' in response:
            messages = response['Messages']
            total = response.get('Total', 0)

            print(f'✓ 收件箱共有 {total} 封邮件')
            print(f'✓ 获取到 {len(messages)} 封邮件')

            if not messages or len(messages) == 0:
                print('\n收件箱为空')
                return

            # 获取最新的邮件
            latest_message = messages[0]

            # 输出邮件信息
            print('\n' + '='*60)
            print('最新邮件信息')
            print('='*60)

            sender = latest_message.get('Sender', {})
            print(f'\n发件人名: {sender.get("Name", "未知")}')
            print(f'发件人地址: {sender.get("Address", "未知")}')

            print(f'\n主题: {latest_message.get("Subject", "无主题")}')

            time_stamp = latest_message.get('Time')
            if time_stamp:
                time_dt = datetime.fromtimestamp(time_stamp)
                print(f'时间: {time_dt.strftime("%Y-%m-%d %H:%M:%S")}')

            unread = latest_message.get('Unread', 0)
            print(f'未读: {"是" if unread == 1 else "否"}')

            size = latest_message.get('Size', 0)
            size_kb = size / 1024
            print(f'大小: {size_kb:.2f} KB')

            to_list = latest_message.get('ToList', [])
            if to_list:
                print(f'\n收件人:')
                for recipient in to_list:
                    name = recipient.get('Name', '')
                    address = recipient.get('Address', '')
                    if name:
                        print(f'  • {name} <{address}>')
                    else:
                        print(f'  • {address}')

            num_attachments = latest_message.get('NumAttachments', 0)
            if num_attachments > 0:
                print(f'\n附件数量: {num_attachments}')

            print('\n' + '='*60)

            print('\n💡 提示: 会话已保存，下次运行将自动使用')
            print(f'   会话文件: {session_file}')

        else:
            print('\n✗ API 响应格式错误')
            print(f'响应: {response}')

    except Exception as e:
        print(f'\n✗ 获取邮件失败: {e}')
        import traceback
        traceback.print_exc()
        sys.exit(1)

    print('\n✓ 完成')


if __name__ == '__main__':
    try:
        read_latest_email()
    except KeyboardInterrupt:
        print('\n\n用户中断')
        sys.exit(0)
    except Exception as error:
        import traceback
        print(f'\n✗ 发生错误: {str(error)}', file=sys.stderr)
        print('\n详细错误信息:', file=sys.stderr)
        traceback.print_exc()
        sys.exit(1)
