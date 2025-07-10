package usecase

import (
	"crypto_trade_bot/domain"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/markcheno/go-talib"
)

// TradingUsecase は通貨選定やROIフィルタリングのユースケースを実装します。
type TradingUsecase struct {
	kucoinGateway KuCoinGateway
	openaiGateway OpenAIGateway
}

// KuCoinGateway は KuCoin API との通信のためのインターフェースです。
type KuCoinGateway interface {
	GetTop20USDTpairsByVolume() ([]string, error)
	GetCurrentPrice(symbol string) (float64, error)
	CreateOrder(symbol string, side string, orderType string, size string) (string, error)
	GetKlines(symbol string, granularity int, count int) ([][]string, error)
}

// OpenAIGateway は OpenAI API との通信のためのインターフェースです。
type OpenAIGateway interface {
	AskAboutAssets(assetSymbols []string, side string) (string, error)
}

// NewTradingUsecase は新しい TradingUsecase を生成します。
func NewTradingUsecase(kg KuCoinGateway, og OpenAIGateway) *TradingUsecase {
	return &TradingUsecase{
		kucoinGateway: kg,
		openaiGateway: og,
	}
}

// AnalyzeTrends は上昇・下降トレンドを分析する一連の処理を実行します。
func (uc *TradingUsecase) AnalyzeTrends() {
	log.Println("Fetching top 20 USDT pairs by volume...")
	pairs, err := uc.kucoinGateway.GetTop20USDTpairsByVolume()
	if err != nil {
		log.Fatalf("Error getting top pairs: %v", err)
	}
	log.Printf("Found pairs: %v\n", pairs)

	var longCandidates []domain.Asset
	var shortCandidates []domain.Asset
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, pair := range pairs {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			log.Printf("Analyzing %s...", p)

			klines, err := uc.kucoinGateway.GetKlines(p, 60, 100) // 60 minutes = 1 hour
			if err != nil {
				log.Printf("Could not get klines for %s: %v", p, err)
				return
			}

			closePrices := make([]float64, len(klines))
			for i, k := range klines {
				price, err := strconv.ParseFloat(k[2], 64)
				if err != nil {
					log.Printf("Could not parse close price for %s: %v", p, err)
					return
				}
				closePrices[i] = price
			}
			for i, j := 0, len(closePrices)-1; i < j; i, j = i+1, j-1 {
				closePrices[i], closePrices[j] = closePrices[j], closePrices[i]
			}

			if len(closePrices) < 26 { // MACD計算に最低限必要な期間
				log.Printf("Not enough data for MACD calculation on %s", p)
				return
			}

			macd, macdSignal, _ := talib.Macd(closePrices, 12, 26, 9)
			rsi := talib.Rsi(closePrices, 14)

			lastMacd := macd[len(macd)-1]
			lastMacdSignal := macdSignal[len(macdSignal)-1]
			prevMacd := macd[len(macd)-2]
			prevMacdSignal := macdSignal[len(macdSignal)-2]
			lastRsi := rsi[len(rsi)-1]

			// --- トレンド判断 ---
			// 上昇トレンド（ロング候補）
			isGoldenCross := prevMacd < prevMacdSignal && lastMacd > lastMacdSignal
			isRsiNotOverbought := lastRsi < 70.0
			if isGoldenCross && isRsiNotOverbought {
				asset := createAsset(p, closePrices, lastMacd, lastRsi)
				mu.Lock()
				longCandidates = append(longCandidates, asset)
				mu.Unlock()
				log.Printf("[LONG Candidate] %s: GoldenCross, RSI=%.2f", p, lastRsi)
			}

			// 下降トレンド（ショート候補）
			isDeadCross := prevMacd > prevMacdSignal && lastMacd < lastMacdSignal
			isRsiNotOversold := lastRsi > 30.0
			if isDeadCross && isRsiNotOversold {
				asset := createAsset(p, closePrices, lastMacd, lastRsi)
				mu.Lock()
				shortCandidates = append(shortCandidates, asset)
				mu.Unlock()
				log.Printf("[SHORT Candidate] %s: DeadCross, RSI=%.2f", p, lastRsi)
			}

		}(pair)
	}

	wg.Wait()

	// --- ロング候補の分析 ---
	if len(longCandidates) > 0 {
		log.Printf("\n--- Found %d LONG candidates ---", len(longCandidates))
		var assetInfo []string
		for _, asset := range longCandidates {
			info := fmt.Sprintf("%s (ROI: %.2f%%, MACD: %.4f, RSI: %.2f)", asset.Symbol, asset.CalculateROI(), asset.MACD, asset.RSI)
			log.Println("- " + info)
			assetInfo = append(assetInfo, info)
		}
		log.Println("Asking OpenAI for LONG analysis...")
		analysis, err := uc.openaiGateway.AskAboutAssets(assetInfo, "buy")
		if err != nil {
			log.Printf("Error getting LONG analysis from OpenAI: %v", err)
		} else {
			fmt.Println("\n--- OpenAI LONG Analysis ---")
			fmt.Println(analysis)
			fmt.Println("--------------------------")
		}
	} else {
		log.Println("\n--- No LONG candidates found ---")
	}

	// --- ショート候補の分析 ---
	if len(shortCandidates) > 0 {
		log.Printf("\n--- Found %d SHORT candidates ---", len(shortCandidates))
		var assetInfo []string
		for _, asset := range shortCandidates {
			info := fmt.Sprintf("%s (ROI: %.2f%%, MACD: %.4f, RSI: %.2f)", asset.Symbol, asset.CalculateROI(), asset.MACD, asset.RSI)
			log.Println("- " + info)
			assetInfo = append(assetInfo, info)
		}
		log.Println("Asking OpenAI for SHORT analysis...")
		analysis, err := uc.openaiGateway.AskAboutAssets(assetInfo, "sell")
		if err != nil {
			log.Printf("Error getting SHORT analysis from OpenAI: %v", err)
		} else {
			fmt.Println("\n--- OpenAI SHORT Analysis ---")
			fmt.Println(analysis)
			fmt.Println("---------------------------")
		}
	} else {
		log.Println("\n--- No SHORT candidates found ---")
	}
}

func createAsset(symbol string, closePrices []float64, macd, rsi float64) domain.Asset {
	return domain.Asset{
		Symbol:       symbol,
		CurrentPrice: closePrices[len(closePrices)-1],
		Price1H:      closePrices[len(closePrices)-2],
		MACD:         macd,
		RSI:          rsi,
	}
}

// ExecuteTrade は指定された条件で取引を実行します。
func (uc *TradingUsecase) ExecuteTrade(symbol, side string, amountUSD float64, execute bool) {
	if !execute {
		log.Println("Execute flag is not set. Exiting trade execution (Dry Run).")
		return
	}
	if side != "buy" && side != "sell" {
		log.Printf("Invalid side: %s. Must be 'buy' or 'sell'.", side)
		return
	}

	log.Printf("Starting trade execution for %s, side: %s, amount: %.2f USD", symbol, side, amountUSD)

	currentPrice, err := uc.kucoinGateway.GetCurrentPrice(symbol)
	if err != nil {
		log.Printf("Failed to get current price for %s: %v", symbol, err)
		return
	}
	log.Printf("Current price of %s is %.4f USD", symbol, currentPrice)

	size := amountUSD / currentPrice
	sizeStr := fmt.Sprintf("%f", size)

	log.Printf("Placing market %s order for %s with size %s", side, symbol, sizeStr)
	orderID, err := uc.kucoinGateway.CreateOrder(symbol, side, "market", sizeStr)
	if err != nil {
		log.Printf("Failed to create %s order: %v", side, err)
		return
	}
	log.Printf("%s order placed successfully. Order ID: %s", side, orderID)

	entryPrice := currentPrice
	log.Printf("Assumed entry price: %.4f", entryPrice)

	var targetPrice float64
	if side == "buy" { // ロングの場合
		targetPrice = entryPrice * 1.01 // 1%上昇で利益確定
		log.Printf("Will place SELL order when price reaches >= %.4f", targetPrice)
	} else { // ショートの場合
		targetPrice = entryPrice * 0.99 // 1%下落で利益確定
		log.Printf("Will place BUY order when price reaches <= %.4f", targetPrice)
	}

	for {
		time.Sleep(30 * time.Second)

		latestPrice, err := uc.kucoinGateway.GetCurrentPrice(symbol)
		if err != nil {
			log.Printf("Could not get latest price for %s: %v", symbol, err)
			continue
		}
		log.Printf("Latest price for %s: %.4f", symbol, latestPrice)

		// 利益確定条件のチェック
		if (side == "buy" && latestPrice >= targetPrice) || (side == "sell" && latestPrice <= targetPrice) {
			closeSide := "sell"
			if side == "sell" {
				closeSide = "buy"
			}
			log.Printf("Target price reached! Placing %s order to close position.", closeSide)
			closeSizeStr := fmt.Sprintf("%f", size)
			closeOrderID, err := uc.kucoinGateway.CreateOrder(symbol, closeSide, "market", closeSizeStr)
			if err != nil {
				log.Printf("Failed to create %s order: %v", closeSide, err)
				continue
			}
			log.Printf("%s order placed successfully. Order ID: %s. Exiting.", closeSide, closeOrderID)
			break
		}
	}
}
