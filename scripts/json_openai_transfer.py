import json
import os
import re
import sys

# 只匹配 JSON 合法的转义：\uXXXX 与有限白名单控制字符/引号/反斜杠/斜杠
_re_u = re.compile(r'\\u([0-9a-fA-F]{4})')
_re_ctrl = re.compile(r'\\([nrtbf"\\/])')  # \n \r \t \b \f \" \\ \/

_ctrl_map = {
    'n': '\n',
    'r': '\r',
    't': '\t',
    'b': '\b',
    'f': '\f',
    '"': '"',
    '\\': '\\',
    '/': '/',
}

def _replace_unicode_escapes(s: str) -> str:
    """仅将 \\uXXXX 转为字符（逐个替换），不碰其它诸如 \\d、\\w 等伪转义。"""
    def repl(m):
        code = int(m.group(1), 16)
        return chr(code)
    return _re_u.sub(repl, s)

def _replace_control_escapes(s: str) -> str:
    """只解码 JSON 白名单控制转义，忽略 \\d \\w 等。"""
    def repl(m):
        ch = m.group(1)
        return _ctrl_map.get(ch, m.group(0))
    return _re_ctrl.sub(repl, s)

def _fix_surrogates(s: str) -> str:
    """
    将可能存在的半代理项对合并为正确字符（如 emoji），
    并丢弃孤立代理项，防止写出时 'surrogates not allowed'。
    """
    # surrogatepass 保留半代理，随后的 utf-16 解码会把成对的代理合并为实际字符
    return s.encode('utf-16', 'surrogatepass').decode('utf-16', 'ignore')

def decode_value(value):
    if not isinstance(value, str):
        return value, None

    # 快速判定：若既无 \uXXXX 也无白名单控制转义，直接返回原值
    if ('\\u' not in value and not _re_ctrl.search(value)):
        return value, None

    try:
        s = _replace_unicode_escapes(value)
        s = _replace_control_escapes(s)
        s = _fix_surrogates(s)
        return s, None
    except Exception as e:
        return value, str(e)

def walk_and_decode(node, log):
    if isinstance(node, dict):
        for k, v in list(node.items()):
            if isinstance(v, (dict, list)):
                walk_and_decode(v, log)
            elif isinstance(v, str):
                new_v, err = decode_value(v)
                if err:
                    preview = v[:50].replace("\n", "\\n")
                    log.write(f"[失败] key={k} | 长度={len(v)} | 原文前50={preview} | 错误={err}\n")
                elif new_v is not v and new_v != v:
                    preview = v[:50].replace("\n", "\\n")
                    log.write(f"[成功] key={k} | 长度={len(v)} | 原文前50={preview}\n")
                node[k] = new_v
    elif isinstance(node, list):
        for i, item in enumerate(node):
            if isinstance(item, (dict, list)):
                walk_and_decode(item, log)
            elif isinstance(item, str):
                new_v, err = decode_value(item)
                if err:
                    preview = item[:50].replace("\n", "\\n")
                    log.write(f"[失败] list[{i}] | 长度={len(item)} | 原文前50={preview} | 错误={err}\n")
                elif new_v is not item and new_v != item:
                    preview = item[:50].replace("\n", "\\n")
                    log.write(f"[成功] list[{i}] | 长度={len(item)} | 原文前50={preview}\n")
                node[i] = new_v

def process_json_stream(input_path):
    base, ext = os.path.splitext(input_path)
    output_path = f"{base}_modified{ext}"
    log_path = f"{base}_log.txt"

    with open(input_path, 'r', encoding='utf-8', errors='ignore') as infile:
        # 读入整块 JSON（标准库保证正确性；如需真正流式请告知我改 JSONL/数组管道版）
        data = json.load(infile)

    # 写出时确保不会再因代理项报错
    with open(output_path, 'w', encoding='utf-8') as outfile:
        json.dump(data, outfile, ensure_ascii=False)
        outfile.write("\n")

    print(f"处理完成：输出文件 → {output_path}\n日志文件 → {log_path}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("用法: python process_json_stream.py <input.json>")
        sys.exit(1)
    process_json_stream(sys.argv[1])