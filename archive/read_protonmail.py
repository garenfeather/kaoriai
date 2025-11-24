#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
ProtonMail 邮件读取脚本

注意: 由于 ProtonMail 的端到端加密特性，直接通过 API 访问需要复杂的加密处理。
推荐使用 ProtonMail Bridge (官方IMAP/SMTP桥接服务) 来访问邮件。

安装 ProtonMail Bridge:
1. 访问 https://proton.me/mail/bridge
2. 下载并安装 ProtonMail Bridge
3. 运行 Bridge 并登录你的账户
4. Bridge 会提供本地 IMAP 服务器地址和密码

使用本脚本前需要:
1. 安装并运行 ProtonMail Bridge
2. 从 Bridge 获取 IMAP 密码
3. 设置环境变量:
   - PM_USERNAME: ProtonMail 邮箱地址
   - PM_BRIDGE_PASSWORD: Bridge 生成的 IMAP 密码 (不是账户密码!)
"""

import os
import sys
import imaplib
import email
from email.header import decode_header
from datetime import datetime


def decode_str(s):
    """解码邮件头部字符串"""
    if s is None:
        return ""

    value, encoding = decode_header(s)[0]
    if isinstance(value, bytes):
        value = value.decode(encoding or 'utf-8', errors='replace')
    return value


def read_latest_email():
    """通过 ProtonMail Bridge IMAP 读取最新邮件"""
    # 从环境变量获取用户名和密码
    username = os.getenv('PM_USERNAME')
    password = os.getenv('PM_BRIDGE_PASSWORD')

    if not username or not password:
        print('错误: 请设置环境变量 PM_USERNAME 和 PM_BRIDGE_PASSWORD', file=sys.stderr)
        print('', file=sys.stderr)
        print('PM_USERNAME: 你的 ProtonMail 邮箱地址', file=sys.stderr)
        print('PM_BRIDGE_PASSWORD: ProtonMail Bridge 生成的 IMAP 密码', file=sys.stderr)
        print('', file=sys.stderr)
        print('如何获取 Bridge 密码:', file=sys.stderr)
        print('1. 下载并安装 ProtonMail Bridge: https://proton.me/mail/bridge', file=sys.stderr)
        print('2. 运行 Bridge 应用并登录', file=sys.stderr)
        print('3. 在 Bridge 中查看你的 IMAP 设置', file=sys.stderr)
        sys.exit(1)

    try:
        print('正在连接到 ProtonMail Bridge...')

        # ProtonMail Bridge 默认监听在本地端口
        imap_server = '127.0.0.1'
        imap_port = 1143  # Bridge 默认 IMAP 端口

        # 连接到 IMAP 服务器
        mail = imaplib.IMAP4(imap_server, imap_port)

        print('正在认证...')
        mail.login(username, password)

        print('登录成功！')

        # 选择收件箱
        mail.select('INBOX')

        # 搜索所有邮件
        status, messages = mail.search(None, 'ALL')

        if status != 'OK':
            print('无法搜索邮件')
            return

        # 获取邮件 ID 列表
        email_ids = messages[0].split()

        if not email_ids:
            print('收件箱为空')
            return

        print(f'收件箱共有 {len(email_ids)} 封邮件')

        # 获取最新的邮件（列表中的最后一个）
        latest_email_id = email_ids[-1]

        # 获取邮件
        status, msg_data = mail.fetch(latest_email_id, '(RFC822)')

        if status != 'OK':
            print('无法获取邮件')
            return

        # 解析邮件
        for response_part in msg_data:
            if isinstance(response_part, tuple):
                msg = email.message_from_bytes(response_part[1])

                # 输出邮件信息
                print('\n' + '='*50)
                print('最新邮件信息:')
                print('='*50)

                # 发件人
                from_header = decode_str(msg.get('From'))
                print(f'发件人: {from_header}')

                # 收件人
                to_header = decode_str(msg.get('To'))
                print(f'收件人: {to_header}')

                # 主题
                subject = decode_str(msg.get('Subject'))
                print(f'主题: {subject or "无主题"}')

                # 日期
                date_header = msg.get('Date')
                print(f'时间: {date_header}')

                # 邮件正文预览
                if msg.is_multipart():
                    for part in msg.walk():
                        content_type = part.get_content_type()
                        if content_type == 'text/plain':
                            try:
                                body = part.get_payload(decode=True)
                                if body:
                                    body_text = body.decode('utf-8', errors='replace')
                                    preview = body_text[:200].replace('\n', ' ').strip()
                                    print(f'\n正文预览: {preview}...')
                                    break
                            except:
                                pass
                else:
                    try:
                        body = msg.get_payload(decode=True)
                        if body:
                            body_text = body.decode('utf-8', errors='replace')
                            preview = body_text[:200].replace('\n', ' ').strip()
                            print(f'\n正文预览: {preview}...')
                    except:
                        pass

                print('='*50)

        # 关闭连接
        mail.close()
        mail.logout()

        print('\n已断开连接')

    except imaplib.IMAP4.error as e:
        print(f'IMAP 错误: {e}', file=sys.stderr)
        print('\n请确认:', file=sys.stderr)
        print('1. ProtonMail Bridge 已安装并正在运行', file=sys.stderr)
        print('2. 使用的是 Bridge 生成的 IMAP 密码，而不是账户密码', file=sys.stderr)
        print('3. Bridge 监听在 127.0.0.1:1143', file=sys.stderr)
        sys.exit(1)
    except ConnectionRefusedError:
        print('连接被拒绝', file=sys.stderr)
        print('\nProtonMail Bridge 可能未运行', file=sys.stderr)
        print('请启动 ProtonMail Bridge 应用后重试', file=sys.stderr)
        sys.exit(1)
    except Exception as error:
        import traceback
        print(f'发生错误: {str(error)}', file=sys.stderr)
        print('\n详细错误信息:', file=sys.stderr)
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    read_latest_email()
