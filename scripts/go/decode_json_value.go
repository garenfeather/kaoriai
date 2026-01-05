package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

var (
	// 匹配 \uXXXX Unicode 转义序列
	reUnicode = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	// 匹配 JSON 合法的控制字符转义
	reCtrl = regexp.MustCompile(`\\([nrtbf"\\/])`)
)

// 控制字符映射表
var ctrlMap = map[byte]rune{
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'b':  '\b',
	'f':  '\f',
	'"':  '"',
	'\\': '\\',
	'/':  '/',
}

// Logger 日志记录器
type Logger struct {
	file *os.File
}

func NewLogger(path string) (*Logger, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

func (l *Logger) Write(format string, args ...interface{}) {
	if l.file != nil {
		fmt.Fprintf(l.file, format, args...)
	}
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// replaceUnicodeEscapes 将 \uXXXX 转换为实际字符
func replaceUnicodeEscapes(s string) string {
	return reUnicode.ReplaceAllStringFunc(s, func(match string) string {
		// 提取十六进制码
		hexCode := match[2:] // 跳过 \u
		code, err := strconv.ParseInt(hexCode, 16, 32)
		if err != nil {
			return match // 解析失败,保留原文
		}
		return string(rune(code))
	})
}

// replaceControlEscapes 解码 JSON 白名单控制转义
func replaceControlEscapes(s string) string {
	return reCtrl.ReplaceAllStringFunc(s, func(match string) string {
		ch := match[1] // 跳过反斜杠
		if r, ok := ctrlMap[ch]; ok {
			return string(r)
		}
		return match // 不在白名单中,保留原文
	})
}

// fixSurrogates 修复 UTF-16 代理对
// 将可能存在的半代理项对合并为正确字符(如 emoji)
func fixSurrogates(s string) string {
	var result strings.Builder
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// 检查是否是高代理项 (0xD800-0xDBFF)
		if r >= 0xD800 && r <= 0xDBFF {
			// 检查下一个字符是否是低代理项 (0xDC00-0xDFFF)
			if i+1 < len(runes) {
				next := runes[i+1]
				if next >= 0xDC00 && next <= 0xDFFF {
					// 合并代理对
					combined := utf16.DecodeRune(r, next)
					if combined != utf8.RuneError {
						result.WriteRune(combined)
						i++ // 跳过低代理项
						continue
					}
				}
			}
			// 孤立的高代理项,跳过
			continue
		}

		// 检查是否是孤立的低代理项,跳过
		if r >= 0xDC00 && r <= 0xDFFF {
			continue
		}

		// 普通字符,直接添加
		result.WriteRune(r)
	}

	return result.String()
}

// decodeValue 解码单个字符串值
func decodeValue(value string) (string, error) {
	// 快速判定: 若既无 \uXXXX 也无白名单控制转义,直接返回原值
	if !strings.Contains(value, `\u`) && !reCtrl.MatchString(value) {
		return value, nil
	}

	defer func() {
		if r := recover(); r != nil {
			// 捕获panic,转换为错误
		}
	}()

	s := replaceUnicodeEscapes(value)
	s = replaceControlEscapes(s)
	s = fixSurrogates(s)

	return s, nil
}

// walkAndDecode 递归遍历 JSON 结构并解码字符串
func walkAndDecode(node interface{}, log *Logger) interface{} {
	switch v := node.(type) {
	case map[string]interface{}:
		for key, val := range v {
			switch item := val.(type) {
			case map[string]interface{}, []interface{}:
				v[key] = walkAndDecode(item, log)
			case string:
				newVal, err := decodeValue(item)
				if err != nil {
					preview := item
					if len(preview) > 50 {
						preview = preview[:50]
					}
					preview = strings.ReplaceAll(preview, "\n", "\\n")
					log.Write("[失败] key=%s | 长度=%d | 原文前50=%s | 错误=%v\n",
						key, len(item), preview, err)
				} else if newVal != item {
					preview := item
					if len(preview) > 50 {
						preview = preview[:50]
					}
					preview = strings.ReplaceAll(preview, "\n", "\\n")
					log.Write("[成功] key=%s | 长度=%d | 原文前50=%s\n",
						key, len(item), preview)
				}
				v[key] = newVal
			}
		}
		return v

	case []interface{}:
		for i, item := range v {
			switch val := item.(type) {
			case map[string]interface{}, []interface{}:
				v[i] = walkAndDecode(val, log)
			case string:
				newVal, err := decodeValue(val)
				if err != nil {
					preview := val
					if len(preview) > 50 {
						preview = preview[:50]
					}
					preview = strings.ReplaceAll(preview, "\n", "\\n")
					log.Write("[失败] list[%d] | 长度=%d | 原文前50=%s | 错误=%v\n",
						i, len(val), preview, err)
				} else if newVal != val {
					preview := val
					if len(preview) > 50 {
						preview = preview[:50]
					}
					preview = strings.ReplaceAll(preview, "\n", "\\n")
					log.Write("[成功] list[%d] | 长度=%d | 原文前50=%s\n",
						i, len(val), preview)
				}
				v[i] = newVal
			}
		}
		return v
	}

	return node
}

// processJSONFile 处理 JSON 文件
func processJSONFile(inputPath string) error {
	// 读取输入文件
	data, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	// 解析 JSON
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("解析JSON失败: %v", err)
	}

	// 生成输出文件名和日志文件名
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	outputPath := base + "_modified" + ext
	logPath := base + "_log.txt"

	// 创建日志记录器
	log, err := NewLogger(logPath)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}
	defer log.Close()

	// 递归处理 JSON 数据
	jsonData = walkAndDecode(jsonData, log)

	// 创建输出文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %v", err)
	}
	defer outFile.Close()

	// 使用 Encoder 写入,禁用 HTML 转义
	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // 关键: 禁用HTML转义,防止 > 被转为 \u003e

	if err := encoder.Encode(jsonData); err != nil {
		return fmt.Errorf("序列化JSON失败: %v", err)
	}

	fmt.Printf("处理完成:\n")
	fmt.Printf("  输出文件 → %s\n", outputPath)
	fmt.Printf("  日志文件 → %s\n", logPath)

	return nil
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("用法: decode_json_value <input.json>")
		fmt.Println("示例: decode_json_value data/conversations.json")
		os.Exit(1)
	}

	inputPath := flag.Arg(0)

	if err := processJSONFile(inputPath); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
