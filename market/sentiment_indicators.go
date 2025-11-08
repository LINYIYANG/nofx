package market

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SentimentData 情绪指标数据结构
type SentimentData struct {
	Symbol           string  `json:"symbol"`
	PositiveRatio    float64 `json:"positive_ratio"`
	NegativeRatio    float64 `json:"negative_ratio"`
	NetSentiment     float64 `json:"net_sentiment"`
	DataTime         string  `json:"data_time"`
	DataDelayMinutes int     `json:"data_delay_minutes"`
}

// API请求结构体
type sentimentRequest struct {
	APIKey    string   `json:"apiKey"`
	Endpoints []string `json:"endpoints"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
	TimeType  string   `json:"timeType"`
	Token     []string `json:"token"`
}

// API响应结构体
type sentimentResponse struct {
	Code int `json:"code"`
	Data []struct {
		Token       string `json:"token"`
		TimePeriods []struct {
			StartTime string `json:"startTime"`
			Data      []struct {
				Endpoint string `json:"endpoint"`
				Value    string `json:"value"`
			} `json:"data"`
		} `json:"timePeriods"`
	} `json:"data"`
}

// GetSentimentIndicators 获取多个symbol的情绪指标
func GetSentimentIndicators(symbols []string) map[string]*SentimentData {
	const (
		apiURL = "https://service.cryptoracle.network/openapi/v2/endpoint"
		apiKey = "2b144650-4a16-4eb5-bbcd-70824577687b"
	)

	// 如果symbols为空，返回空map
	if len(symbols) == 0 {
		return make(map[string]*SentimentData)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ 情绪指标获取失败: %v\n", r)
		}
	}()

	// 提取基础资产
	baseSymbols := make([]string, len(symbols))
	symbolMapping := make(map[string]string) // 基础资产 -> 原始交易对

	for i, symbol := range symbols {
		baseAsset := ExtractBaseAsset(symbol)
		baseSymbols[i] = baseAsset
		symbolMapping[baseAsset] = symbol
	}

	// 获取最近4小时数据
	endTime := time.Now()
	startTime := endTime.Add(-4 * time.Hour)

	requestBody := sentimentRequest{
		APIKey:    apiKey,
		Endpoints: []string{"CO-A-02-01", "CO-A-02-02"},
		StartTime: startTime.Format("2006-01-02 15:04:05"),
		EndTime:   endTime.Format("2006-01-02 15:04:05"),
		TimeType:  "15m",
		Token:     baseSymbols,
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("❌ 情绪指标，请求体序列化失败: %v\n", err)
		return nil
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("❌ 情绪指标，创建请求失败: %v\n", err)
		return nil
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", apiKey)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ 情绪指标，请求发送失败: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ 情绪指标，读取响应失败: %v\n", err)
		return nil
	}

	if resp.StatusCode != 200 {
		log.Printf("❌ 情绪指标，API返回状态码: %d\n", resp.StatusCode)
		return nil
	}

	// 解析响应
	var apiResp sentimentResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("❌ 情绪指标，响应解析失败: %v\n", err)
		return nil
	}

	// 检查API响应状态
	if apiResp.Code != 200 || len(apiResp.Data) == 0 {
		log.Println("❌ 情绪指标，API返回数据为空或状态码错误")
		return nil
	}

	// 创建结果map
	result := make(map[string]*SentimentData)

	// 处理每个symbol的数据
	for _, symbolData := range apiResp.Data {
		baseSymbol := symbolData.Token
		symbol := symbolMapping[baseSymbol]

		// 查找第一个有有效数据的时间段
		for _, period := range symbolData.TimePeriods {
			sentiment := make(map[string]float64)
			validDataFound := false

			for _, item := range period.Data {
				value := strings.TrimSpace(item.Value)
				if value == "" {
					continue // 跳过空值
				}

				// 只处理目标endpoint
				if item.Endpoint == "CO-A-02-01" || item.Endpoint == "CO-A-02-02" {
					floatValue, err := strconv.ParseFloat(value, 64)
					if err != nil {
						continue // 转换失败则跳过
					}
					sentiment[item.Endpoint] = floatValue
					validDataFound = true
				}
			}

			// 如果找到有效数据
			if validDataFound {
				positive, hasPositive := sentiment["CO-A-02-01"]
				negative, hasNegative := sentiment["CO-A-02-02"]

				if hasPositive && hasNegative {
					netSentiment := positive - negative

					// 计算数据延迟
					dataTime, err := time.Parse("2006-01-02 15:04:05", period.StartTime)
					if err != nil {
						log.Printf("❌ 情绪指标，时间解析失败: %v\n", err)
						continue
					}
					dataDelay := int(time.Since(dataTime).Minutes())

					log.Printf("✅ 情绪指标：%s 使用情绪数据时间: %s (延迟: %d分钟)\n", symbol, period.StartTime, dataDelay)

					result[symbol] = &SentimentData{
						Symbol:           symbol,
						PositiveRatio:    positive,
						NegativeRatio:    negative,
						NetSentiment:     netSentiment,
						DataTime:         period.StartTime,
						DataDelayMinutes: dataDelay,
					}
					break // 找到有效数据后跳出时间段循环，处理下一个symbol
				}
			}
		}

		// 如果没有找到有效数据
		if result[symbol] == nil {
			log.Printf("❌ 情绪指标：%s 所有时间段数据都为空\n", symbol)
		}
	}

	return result
}

// ExtractBaseAsset 使用正则表达式从交易对中提取基础资产
func ExtractBaseAsset(symbol string) string {
	// 匹配以字母开头，后面跟着数字或字母，然后以USDT、USD、BUSD等结尾
	re := regexp.MustCompile(`^([A-Z]+)(USDT|USD|BUSD|USDC|ETH|BTC|BNB)$`)
	matches := re.FindStringSubmatch(symbol)

	if len(matches) > 1 {
		return matches[1] // 返回基础资产部分
	}

	return symbol // 如果不匹配，返回原字符串
}
