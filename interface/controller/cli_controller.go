package controller

// TradingUsecase は分析ユースケースのインターフェースです。
type TradingUsecase interface {
	AnalyzeTrends()
	ExecuteTrade(symbol, side string, amountUSD float64, execute bool)
}

// CLIController はCLIからの入力を処理します。
type CLIController struct {
	usecase TradingUsecase
}

// NewCLIController は新しいCLIControllerを生成します。
func NewCLIController(usecase TradingUsecase) *CLIController {
	return &CLIController{
		usecase: usecase,
	}
}

// RunAnalysis は分析処理を開始します。
func (c *CLIController) RunAnalysis() {
	c.usecase.AnalyzeTrends()
}

// RunTrade は取引処理を開始します。
func (c *CLIController) RunTrade(symbol, side string, amountUSD float64, execute bool) {
	c.usecase.ExecuteTrade(symbol, side, amountUSD, execute)
}
