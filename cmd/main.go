package main

import (
	"crypto_trade_bot/infra/client"
	"crypto_trade_bot/infra/config"
	"crypto_trade_bot/interface/controller"
	"crypto_trade_bot/interface/gateway"
	"crypto_trade_bot/usecase"
	"flag"
	"log"
)

func main() {
	// コマンドラインフラグの定義
	tradeMode := flag.Bool("trade", false, "Enable trade mode")
	symbol := flag.String("symbol", "BTC-USDT", "Symbol to trade (e.g., BTC-USDT)")
	side := flag.String("side", "buy", "Trade side: 'buy' for long, 'sell' for short")
	amount := flag.Float64("amount", 10.0, "Amount in USD to trade")
	execute := flag.Bool("execute", false, "Set to true to execute the trade for real")

	flag.Parse()

	// 環境変数の読み込み
	config.LoadEnv()

	// 依存関係の注入 (DI)
	httpClient := client.NewHTTPClient()
	kucoinGateway := gateway.NewKuCoinGateway(httpClient)
	openaiGateway := gateway.NewOpenAIGateway(httpClient)
	tradingUsecase := usecase.NewTradingUsecase(kucoinGateway, openaiGateway)
	cliController := controller.NewCLIController(tradingUsecase)

	// モードに応じて処理を分岐
	if *tradeMode {
		log.Println("--- Trade Mode ---")
		cliController.RunTrade(*symbol, *side, *amount, *execute)
	} else {
		log.Println("--- Analysis Mode ---")
		cliController.RunAnalysis()
	}
}
