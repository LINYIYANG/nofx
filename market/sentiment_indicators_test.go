package market

import (
	"testing"
)

func TestGetSentimentIndicators(t *testing.T) {
	type args struct {
		symbol string
	}
	tests := []struct {
		name string
		args args
		want *SentimentData
	}{
		{
			name: "BTC",
			args: args{
				symbol: "BTC",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbols := []string{"BTCUSDT", "ETH", "SOL"}
			sentiments := GetSentimentIndicators(symbols)

			if sentiments != nil {
				for symbol, data := range sentiments {
					t.Logf("\n=== %s 情绪指标 ===\n", symbol)
					t.Logf("正面情绪比例: %.4f\n", data.PositiveRatio)
					t.Logf("负面情绪比例: %.4f\n", data.NegativeRatio)
					t.Logf("净情绪值: %.4f\n", data.NetSentiment)
					t.Logf("数据时间: %s\n", data.DataTime)
					t.Logf("数据延迟: %d分钟\n", data.DataDelayMinutes)
				}

				// 检查哪些symbol没有数据
				for _, symbol := range symbols {
					if _, exists := sentiments[symbol]; !exists {
						t.Logf("\n⚠️情绪指标：%s 没有获取到情绪数据\n", symbol)
					}
				}
			} else {
				t.Logf("情绪指标：未能获取任何情绪数据")
			}
		})
	}
}
