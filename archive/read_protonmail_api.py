#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
ProtonMail 邮件读取脚本 - 使用官方 proton-python-client API

解决 libcrypto 警告的方法：
强制使用纯 Python 的 SRP 实现，避免加载 ctypes 版本的 libssl.dylib
"""

import os
import sys
import time
from datetime import datetime

# ===== 重要：修复 libcrypto 兼容性问题 =====
# 在导入 proton 之前，阻止加载 _ctsrp (使用 ctypes 加载 libssl 的版本)
# 强制使用纯 Python 实现 _pysrp
sys.modules['proton.srp._ctsrp'] = None
# ==========================================

from proton.api import Session


def read_latest_email():
    """使用官方 proton-python-client 读取 ProtonMail 中的最新邮件"""
    # 从环境变量获取用户名和密码
    username = os.getenv('PM_USERNAME')
    password = os.getenv('PM_PASSWORD')

    if not username or not password:
        print('错误: 请设置环境变量 PM_USERNAME 和 PM_PASSWORD', file=sys.stderr)
        sys.exit(1)

    # 配置重试选项
    max_retries = 3
    retry_delay = 2  # 秒

    session = None

    # 尝试连接和认证
    for attempt in range(max_retries):
        try:
            if attempt > 0:
                print(f'\n第 {attempt + 1} 次尝试连接...')
                time.sleep(retry_delay)
            else:
                print('正在连接到 ProtonMail...')

            # 尝试不同的 API URL
            api_urls = [
                "https://mail.proton.me/api",
                "https://api.protonmail.ch"
            ]

            api_url = api_urls[attempt % len(api_urls)]
            print(f'使用 API 端点: {api_url}')

            # 创建 Proton Session
            # 使用实际的 Web Mail 客户端版本号
            appversion = "web-mail@5.0.88.4"
            print(f'  AppVersion: {appversion}')

            session = Session(
                api_url=api_url,
                appversion=appversion,
                user_agent="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
            )

            print('正在认证...')
            # 认证
            session.authenticate(username, password)

            print('✓ 登录成功！')
            break  # 成功则跳出循环

        except Exception as e:
            error_msg = str(e)

            if 'SSL' in error_msg or 'EOF' in error_msg or 'Connection' in error_msg:
                print(f'\n✗ 连接错误 (尝试 {attempt + 1}/{max_retries})')

                if attempt < max_retries - 1:
                    print('  可能原因: 网络不稳定、VPN干扰、防火墙')
                    print(f'  等待 {retry_delay} 秒后重试...')
                else:
                    print('\n所有重试均失败。')
                    print('\n诊断建议:')
                    print('1. 检查网络连接: ping mail.proton.me')
                    print('2. 检查系统时间是否正确（SSL 需要）')
                    print('3. 如果使用 VPN，尝试关闭或更换节点')
                    print('4. 检查防火墙设置')
                    print('5. 尝试使用其他网络（如手机热点）')
                    print('\n备选方案:')
                    print('- 使用 read_protonmail.py (需要 ProtonMail Bridge)')
                    print('- 直接在浏览器访问 https://mail.proton.me')
                    raise
            else:
                # 其他类型的错误
                print(f'\n认证错误: {error_msg}')
                raise

    if session is None:
        print('\n无法建立连接')
        sys.exit(1)

    # 获取邮件列表
    try:
        print('\n获取邮件信息...')

        # 调用 messages API 获取邮件列表
        # LabelID 0 = Inbox, Desc=1 表示降序（最新的在前）
        response = session.api_request(
            endpoint='/mail/v4/messages',
            method='get',
            additional_headers={},
            data={
                'LabelID': '0',  # 收件箱
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

            # 发件人信息
            sender = latest_message.get('Sender', {})
            print(f'\n发件人名: {sender.get("Name", "未知")}')
            print(f'发件人地址: {sender.get("Address", "未知")}')

            # 主题
            print(f'\n主题: {latest_message.get("Subject", "无主题")}')

            # 时间 (Unix 时间戳)
            time_stamp = latest_message.get('Time')
            if time_stamp:
                time_dt = datetime.fromtimestamp(time_stamp)
                print(f'时间: {time_dt.strftime("%Y-%m-%d %H:%M:%S")}')

            # 未读状态
            unread = latest_message.get('Unread', 0)
            print(f'未读: {"是" if unread == 1 else "否"}')

            # 邮件大小
            size = latest_message.get('Size', 0)
            size_kb = size / 1024
            print(f'大小: {size_kb:.2f} KB ({size} 字节)')

            # 收件人列表
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

            # 附件信息
            num_attachments = latest_message.get('NumAttachments', 0)
            if num_attachments > 0:
                print(f'\n附件数量: {num_attachments}')

            print('\n' + '='*60)

            # 注意事项
            print('\n💡 注意: 邮件正文已端到端加密')
            print('   API 只返回加密内容，需要 PGP 私钥解密')
            print('   建议使用 ProtonMail Bridge + IMAP 读取正文')

        else:
            print('\n✗ API 响应格式错误')
            print(f'响应内容: {response}')

    except Exception as e:
        print(f'\n✗ 获取邮件失败: {e}')
        import traceback
        traceback.print_exc()
        sys.exit(1)

    print('\n✓ 已断开连接')


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
