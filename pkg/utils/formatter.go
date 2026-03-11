package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseSizeToBytes 将形如 "100KB", "10MB", "1.5GB" 的字符串转换为 float64 字节数 (Bytes)
func ParseSizeToBytes(sizeStr string) (float64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, nil
	}

	sizeStr = strings.ToUpper(sizeStr)

	var i int
	for i = 0; i < len(sizeStr); i++ {
		// 寻找开始出现字母的地方，即单位的开始
		if sizeStr[i] >= 'A' && sizeStr[i] <= 'Z' {
			break
		}
	}

	numStr := strings.TrimSpace(sizeStr[:i])
	unitStr := strings.TrimSpace(sizeStr[i:])

	if numStr == "" {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, err
	}

	switch unitStr {
	case "", "B":
		return val, nil
	case "KB", "K":
		return val * 1024, nil
	case "MB", "M":
		return val * 1024 * 1024, nil
	case "GB", "G":
		return val * 1024 * 1024 * 1024, nil
	case "TB", "T":
		return val * 1024 * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", unitStr)
	}
}
