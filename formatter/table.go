package formatter

import (
	"fmt"
	mongodb "hyper-notify-bot/db"
	"math"
	"strconv"
	"strings"
)

// FormatTableAsText 将表格数据格式化为纯文本
func FormatTableAsText(data []mongodb.TableRow) string {
	// 创建表头
	header := "编号      正          负"
	separator := "-----------------------------"

	// 创建表格行
	table := header + "\n" + separator + "\n"
	for _, row := range data {
		table += fmt.Sprintf("%-4.1f    %9.2f    %9.2f\n", row.ID, row.Pos, row.Neg)
	}
	return table
}

// FormatTableAsHTML 将表格数据格式化为HTML
func FormatTableAsHTML(data []mongodb.PositionResult, coin, oraclePrice string, longSz, shortSz float64) string {
	// 解析 oraclePrice 为浮点数以便比较
	targetPrice, err := strconv.ParseFloat(oraclePrice, 64)
	if err != nil {
		// 处理解析错误，例如使用默认值或返回错误信息
		targetPrice = 0
	}
	// 初始化最接近值的索引和最小差值
	closestIndex := -1
	minDiff := math.MaxFloat64

	// 1. 遍历 data，找到最接近 oraclePrice 的 binF
	for i, row := range data {
		binF, err := strconv.ParseFloat(row.Bin.String(), 64)
		if err != nil {
			continue // 跳过无法解析的项
		}
		diff := math.Abs(binF - targetPrice)
		if diff < minDiff {
			minDiff = diff
			closestIndex = i
		}
	}

	percentLong := longSz / (math.Abs(shortSz) + longSz)
	percentShort := 1 - percentLong

	percentLongStr := fmt.Sprintf("%.2f%%", percentLong*100)
	percentShortStr := fmt.Sprintf("%.2f%%", percentShort*100)
	// 创建表头
	table := `<b>📊 Position Data</b>`
	// 添加 Oracle 价格
	if oraclePrice != "" {
		table += fmt.Sprintf("\n\n<b>当前 "+coin+" Oracle 价格: %s</b>", formatStringNumber(oraclePrice))
		table += fmt.Sprintf("\n\n<b>统计 "+coin+" Long 总数: %9s</b>", formatStringNumber(fmt.Sprintf("%.2f", longSz)))
	}
	table += `
<pre>
💰Price   🟢Long(` + percentLongStr + `)
------------------------------

`
	var showData []mongodb.PositionResult
	if len(data) > 30 {
		showData = data[closestIndex-10 : closestIndex+10]
		closestIndex = 10
	} else {
		showData = data
	}

	maxLong := 0.0
	maxShort := 0.0
	maxLongVauleIndex := 0
	maxShortVauleIndex := 0
	for i, row := range showData {
		longTmp, _ := strconv.ParseFloat(row.Long.String(), 64)
		shortTmp, _ := strconv.ParseFloat(row.Short.String(), 64)
		//maxBinF, _ := strconv.ParseFloat(row.Bin.String(), 64)
		if longTmp > maxLong {
			maxLong = longTmp
			maxLongVauleIndex = i
		}
		if shortTmp < maxShort {
			maxShort = shortTmp
			maxShortVauleIndex = i
		}
	}

	percentMaxL := maxLong / longSz
	percentMaxS := math.Abs(maxShort) / math.Abs(shortSz)

	curMaxPercentLongStr := fmt.Sprintf("%.2f%%", percentMaxL*100)
	curMaxPercentShortStr := fmt.Sprintf("%.2f%%", percentMaxS*100)

	maxLongBinF, _ := strconv.ParseFloat(showData[maxLongVauleIndex].Bin.String(), 64)
	maxShortBinF, _ := strconv.ParseFloat(showData[maxShortVauleIndex].Bin.String(), 64)

	//tmpMaxL := fmt.Sprintf("%9.2f(%s)", maxLong, curMaxPercentLongStr)
	//tmpMaxS := fmt.Sprintf("%9.2f(%s)", maxShort, curMaxPercentShortStr)

	tmpMaxL := fmt.Sprintf("🔸%-4.2f  %9.2f(%s)", maxLongBinF, maxLong, curMaxPercentLongStr)
	tmpMaxS := fmt.Sprintf("🔸%-4.2f  %9.2f(%s)", maxShortBinF, maxShort, curMaxPercentShortStr)

	maxLengthL := len(tmpMaxL)
	maxLengthS := len(tmpMaxS)

	// 创建表格行
	tableLong := ``
	tableShort := ``
	for i, row := range showData {
		binF, _ := strconv.ParseFloat(row.Bin.String(), 64)
		longF, _ := strconv.ParseFloat(row.Long.String(), 64)
		shortF, _ := strconv.ParseFloat(row.Short.String(), 64)

		percentL := longF / longSz
		percentS := math.Abs(shortF) / math.Abs(shortSz)

		curPercentLongStr := fmt.Sprintf("%.2f%%", percentL*100)
		curPercentShortStr := fmt.Sprintf("%.2f%%", percentS*100)

		//barsL := formatPercentWithBars(longF / maxLong)
		//barsS := formatPercentWithBars(math.Abs(shortF) / math.Abs(maxShort))

		n := "2"
		if binF > 99999 {
			n = "1"
		}

		curStrL := fmt.Sprintf("%9.2f(%s)", longF, curPercentLongStr)
		curStrS := fmt.Sprintf("%9.2f(%s)", shortF, curPercentShortStr)
		//为了列对齐，补充空格
		spacesNumL := "  "
		spacesNumS := "  "
		for n := 0; n < (maxLengthL - len(curStrL)); n++ {
			spacesNumL += " "
		}
		for n := 0; n < (maxLengthS - len(curStrS)); n++ {
			spacesNumS += " "
		}

		// 2. 判断是否为最接近的行，如果是则加粗
		if i == closestIndex {
			tableLong += fmt.Sprintf("🔸%-4."+n+"f  %9s(%s)\n", binF, formatStringNumber(fmt.Sprintf("%.2f ", longF)), curPercentLongStr)
			tableShort += fmt.Sprintf("🔸%-4."+n+"f  %9s(%s)\n", binF, formatStringNumber(fmt.Sprintf("%.2f ", shortF)), curPercentShortStr)
		} else {
			tableLong += fmt.Sprintf("🔹%-4."+n+"f  %9s(%s)\n", binF, formatStringNumber(fmt.Sprintf("%.2f ", longF)), curPercentLongStr)
			tableShort += fmt.Sprintf("🔹%-4."+n+"f  %9s(%s)\n", binF, formatStringNumber(fmt.Sprintf("%.2f ", shortF)), curPercentShortStr)
		}

		////为了列对齐，补充空格
		//spacesBeginNumL := ""
		//spacesBeginNumS := ""
		//for n := 0; n < (maxLengthL - len(barsL) - 3); n++ {
		//	spacesBeginNumL += " "
		//}
		//for n := 0; n < (maxLengthS - len(barsS) - 3); n++ {
		//	spacesBeginNumS += " "
		//}
		//
		//tableLong += fmt.Sprintf("%s%s\n", spacesBeginNumL, barsL)
		//tableLong += "------------------------------\n"
		//
		//tableShort += fmt.Sprintf("%s%s\n", spacesBeginNumS, barsS)
		//tableShort += "------------------------------\n"
	}

	table += tableLong + "</pre>\n\n"
	table += fmt.Sprintf("<b>统计 "+coin+" Short 总数: %9s</b>", formatStringNumber(fmt.Sprintf("%.2f ", shortSz)))
	table += `
<pre>
💰Price     🔴Short(` + percentShortStr + `)
------------------------------
`
	table += tableShort + "</pre>"
	// 创建交易页面链接
	if oraclePrice != "N/A" {
		tradeURL := fmt.Sprintf("https://app.hyperliquid.xyz/trade/%s/USDC", strings.ToUpper(coin))
		table += fmt.Sprintf("\n\n<a href=\"%s\">📈 查看更多 %s 交易数据</a>", tradeURL, strings.ToUpper(coin))
	}

	return table
}

func formatPercentWithBars(percent float64) string {
	// 确保百分比值在0到1之间
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	// 计算需要显示的竖线数量：任何大于0的比例都至少显示1个"|"
	var numBars int
	if percent == 0 {
		numBars = 0
	} else {
		numBars = int(math.Ceil(percent * 15)) // 关键修改：10 -> 15
	}

	// 构建竖线字符串
	bars := strings.Repeat("|", numBars)
	fmt.Sprintf("%s (%.1f%%)", bars, percent*100)
	// 为了直观，也返回原始的百分比数值
	//return fmt.Sprintf("%s (%.1f%%)", bars, percent*100)
	return bars
}

func formatStringNumber(s string) string {
	// 处理负号
	sign := ""
	if s != "" && s[0] == '-' {
		sign = "-"
		s = s[1:]
	}

	// 分离整数部分与小数部分
	parts := strings.Split(s, ".")
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) > 1 {
		decimalPart = "." + parts[1]
	}

	// 去除整数部分的前导零
	integerPart = strings.TrimLeft(integerPart, "0")
	if integerPart == "" {
		integerPart = "0"
	}

	fmt.Println("------------------", integerPart, len(integerPart))
	if len(integerPart) <= 3 {
		return sign + integerPart + decimalPart
	}

	// 为整数部分添加千分位逗号
	var formattedInteger strings.Builder
	n := len(integerPart)
	for i, runeValue := range integerPart {
		formattedInteger.WriteRune(runeValue)
		// 当前位之后还剩多少位？如果能被3整除且不是最后一位，则添加逗号
		if (n-i-1)%3 == 0 && i != n-1 {
			formattedInteger.WriteString(",")
		}
	}

	return sign + formattedInteger.String() + decimalPart
}
