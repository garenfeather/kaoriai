package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"strings"
)

// Difference 表示一个差异
type Difference struct {
	Path      string // JSON 路径
	Type      string // 差异类型: "key_missing", "key_extra", "value_diff", "type_diff"
	Value1    string // 文件1的值
	Value2    string // 文件2的值
	Detail    string // 详细说明
}

// 全局差异列表
var differences []Difference
var maxDiffs = 50

// compareJSON 比较两个 JSON 结构
func compareJSON(path string, obj1, obj2 interface{}) {
	if len(differences) >= maxDiffs {
		return
	}

	// 类型检查
	type1 := reflect.TypeOf(obj1)
	type2 := reflect.TypeOf(obj2)

	if type1 != type2 {
		differences = append(differences, Difference{
			Path:   path,
			Type:   "type_diff",
			Value1: formatValue(obj1),
			Value2: formatValue(obj2),
			Detail: fmt.Sprintf("类型不同: %v vs %v", type1, type2),
		})
		return
	}

	switch v1 := obj1.(type) {
	case map[string]interface{}:
		v2 := obj2.(map[string]interface{})
		compareMaps(path, v1, v2)

	case []interface{}:
		v2 := obj2.([]interface{})
		compareArrays(path, v1, v2)

	default:
		// 基本类型比较
		if !reflect.DeepEqual(obj1, obj2) {
			differences = append(differences, Difference{
				Path:   path,
				Type:   "value_diff",
				Value1: formatValue(obj1),
				Value2: formatValue(obj2),
				Detail: "值不同",
			})
		}
	}
}

// compareMaps 比较两个 map
func compareMaps(path string, map1, map2 map[string]interface{}) {
	// 获取所有键
	allKeys := make(map[string]bool)
	for k := range map1 {
		allKeys[k] = true
	}
	for k := range map2 {
		allKeys[k] = true
	}

	// 按字母排序键
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 比较每个键
	for _, key := range keys {
		if len(differences) >= maxDiffs {
			return
		}

		val1, exists1 := map1[key]
		val2, exists2 := map2[key]

		newPath := path
		if newPath == "" {
			newPath = key
		} else {
			newPath = path + "." + key
		}

		if !exists1 && exists2 {
			// 键只在文件2中存在
			differences = append(differences, Difference{
				Path:   newPath,
				Type:   "key_extra",
				Value1: "<不存在>",
				Value2: formatValue(val2),
				Detail: "键只在文件2中存在",
			})
		} else if exists1 && !exists2 {
			// 键只在文件1中存在
			differences = append(differences, Difference{
				Path:   newPath,
				Type:   "key_missing",
				Value1: formatValue(val1),
				Value2: "<不存在>",
				Detail: "键只在文件1中存在",
			})
		} else {
			// 键在两个文件中都存在,比较值
			compareJSON(newPath, val1, val2)
		}
	}
}

// compareArrays 比较两个数组
func compareArrays(path string, arr1, arr2 []interface{}) {
	len1 := len(arr1)
	len2 := len(arr2)

	if len1 != len2 {
		differences = append(differences, Difference{
			Path:   path,
			Type:   "value_diff",
			Value1: fmt.Sprintf("数组长度: %d", len1),
			Value2: fmt.Sprintf("数组长度: %d", len2),
			Detail: fmt.Sprintf("数组长度不同: %d vs %d", len1, len2),
		})
		return
	}

	// 比较每个元素
	for i := 0; i < len1; i++ {
		if len(differences) >= maxDiffs {
			return
		}
		newPath := fmt.Sprintf("%s[%d]", path, i)
		compareJSON(newPath, arr1[i], arr2[i])
	}
}

// formatValue 格式化值用于显示
func formatValue(val interface{}) string {
	if val == nil {
		return "<null>"
	}

	switch v := val.(type) {
	case string:
		// 长文本只显示前100个字符
		runes := []rune(v)
		if len(runes) > 100 {
			return string(runes[:100]) + "..."
		}
		return v

	case map[string]interface{}:
		return fmt.Sprintf("<对象,包含%d个键>", len(v))

	case []interface{}:
		return fmt.Sprintf("<数组,长度%d>", len(v))

	default:
		str := fmt.Sprintf("%v", v)
		runes := []rune(str)
		if len(runes) > 100 {
			return string(runes[:100]) + "..."
		}
		return str
	}
}

// printDifferences 打印差异
func printDifferences() {
	if len(differences) == 0 {
		fmt.Println("✅ 两个 JSON 文件完全一致!")
		return
	}

	fmt.Printf("发现 %d 个差异", len(differences))
	if len(differences) >= maxDiffs {
		fmt.Printf(" (已达到最大显示数量 %d,可能还有更多)", maxDiffs)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, diff := range differences {
		fmt.Printf("[%d] 路径: %s\n", i+1, diff.Path)

		switch diff.Type {
		case "key_missing":
			fmt.Printf("    类型: 键缺失\n")
			fmt.Printf("    说明: 该键只在文件1中存在,文件2中不存在\n")
			fmt.Printf("    文件1值: %s\n", diff.Value1)

		case "key_extra":
			fmt.Printf("    类型: 额外的键\n")
			fmt.Printf("    说明: 该键只在文件2中存在,文件1中不存在\n")
			fmt.Printf("    文件2值: %s\n", diff.Value2)

		case "value_diff":
			fmt.Printf("    类型: 值不同\n")
			fmt.Printf("    文件1: %s\n", diff.Value1)
			fmt.Printf("    文件2: %s\n", diff.Value2)

		case "type_diff":
			fmt.Printf("    类型: 数据类型不同\n")
			fmt.Printf("    说明: %s\n", diff.Detail)
			fmt.Printf("    文件1: %s\n", diff.Value1)
			fmt.Printf("    文件2: %s\n", diff.Value2)
		}

		fmt.Println(strings.Repeat("-", 80))
	}
}

func main() {
	// 解析命令行参数
	file1 := flag.String("file1", "", "第一个JSON文件路径")
	file2 := flag.String("file2", "", "第二个JSON文件路径")
	limit := flag.Int("limit", 50, "最多显示的差异数量")
	flag.Parse()

	// 处理位置参数
	if *file1 == "" && flag.NArg() >= 2 {
		*file1 = flag.Arg(0)
		*file2 = flag.Arg(1)
	}

	if *file1 == "" || *file2 == "" {
		fmt.Println("用法: compare_json <file1.json> <file2.json> [-limit N]")
		fmt.Println("或者: compare_json -file1 <file1.json> -file2 <file2.json> [-limit N]")
		fmt.Println()
		fmt.Println("参数:")
		fmt.Println("  file1, file2  - 要比较的两个JSON文件")
		fmt.Println("  -limit N      - 最多显示N个差异 (默认: 50)")
		os.Exit(1)
	}

	maxDiffs = *limit

	// 读取文件1
	data1, err := ioutil.ReadFile(*file1)
	if err != nil {
		fmt.Printf("读取文件1失败: %v\n", err)
		os.Exit(1)
	}

	// 读取文件2
	data2, err := ioutil.ReadFile(*file2)
	if err != nil {
		fmt.Printf("读取文件2失败: %v\n", err)
		os.Exit(1)
	}

	// 解析 JSON
	var json1, json2 interface{}

	if err := json.Unmarshal(data1, &json1); err != nil {
		fmt.Printf("解析文件1失败: %v\n", err)
		os.Exit(1)
	}

	if err := json.Unmarshal(data2, &json2); err != nil {
		fmt.Printf("解析文件2失败: %v\n", err)
		os.Exit(1)
	}

	// 显示文件信息
	fmt.Println("比较 JSON 文件")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("文件1: %s (大小: %d 字节)\n", *file1, len(data1))
	fmt.Printf("文件2: %s (大小: %d 字节)\n", *file2, len(data2))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// 比较 JSON
	compareJSON("", json1, json2)

	// 打印结果
	printDifferences()
}
