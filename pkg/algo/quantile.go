package algo

import (
	"math"
	"sort"

	"trend/pkg/models" // 引入 DataPoint 等结构
)

// CalculateEnvelope 基于历史数据计算动态包络线 (返回下界和上界)
func CalculateEnvelope(history []models.DataPoint) (float64, float64) {
	if len(history) == 0 {
		return 0, 0
	}

	values := make([]float64, len(history))
	for i, dp := range history {
		values[i] = dp.Value
	}

	// 排序以便计算分位数
	sort.Float64s(values)

	// 计算四分位数
	q1 := Quantile(values, 0.25)
	q3 := Quantile(values, 0.75)

	iqr := q3 - q1

	// 根据 Tukey's Fences 理论或者具体的动态缩放因子调整
	// 这里采用经典的 1.5 * IQR 设定上下界
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	return lowerBound, upperBound
}

// Quantile 计算排好序的数组中的分位数，p 为在 0.0 到 1.0 之间的概率
func Quantile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if p < 0 || p > 1 {
		return 0 // 非法值简要处理
	}

	pos := p * float64(n-1)
	idx := int(math.Floor(pos))
	remainder := pos - float64(idx)

	if idx+1 < n {
		return sorted[idx] + remainder*(sorted[idx+1]-sorted[idx])
	}
	return sorted[idx]
}
