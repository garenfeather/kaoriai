import json
import sys
import os

def jsonl_to_json_array(input_path):
    base, ext = os.path.splitext(input_path)
    output_path = f"{base}_array.json"

    with open(input_path, "r", encoding="utf-8") as infile, \
         open(output_path, "w", encoding="utf-8") as outfile:
        outfile.write("[\n")
        first = True
        for line in infile:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                # 如果是转义错误或不完整行，原样写入字符串形式
                obj = line
            if not first:
                outfile.write(",\n")
            else:
                first = False
            json.dump(obj, outfile, ensure_ascii=False)
        outfile.write("\n]\n")

    print(f"✅ 转换完成：{output_path}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("用法: python jsonl_to_json_array.py <input.jsonl>")
        sys.exit(1)
    jsonl_to_json_array(sys.argv[1])