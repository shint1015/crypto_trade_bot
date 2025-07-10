package gateway

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto_trade_bot/infra/client"
	"crypto_trade_bot/infra/config"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// KuCoinGateway は KuCoin API との通信を抽象化します。
type KuCoinGateway struct {
	httpClient *client.HTTPClient
	baseURL    string
	apiKey     string
	apiSecret  string
	passphrase string
}

// NewKuCoinGateway は新しい KuCoinGateway を生成します。
func NewKuCoinGateway(httpClient *client.HTTPClient) *KuCoinGateway {
	// 先物取引APIのエンドポイントに変更
	return &KuCoinGateway{
		httpClient: httpClient,
		baseURL:    "https://api-futures.kucoin.com",
		apiKey:     config.GetEnv("KUCOIN_API_KEY", ""),
		apiSecret:  config.GetEnv("KUCOIN_API_SECRET", ""),
		passphrase: config.GetEnv("KUCOIN_API_PASSPHRASE", ""),
	}
}

// FuturesContractsResponse は先物APIの契約リストのレスポンス構造体です。
type FuturesContractsResponse struct {
	Code string `json:"code"`
	Data []struct {
		Symbol        string  `json:"symbol"`
		Volume24h     float64 `json:"volume24h"`
		IsDele        bool    `json:"isDele"`
		Status        string  `json:"status"`
		QuoteCurrency string  `json:"quoteCurrency"`
	} `json:"data"`
}

// GetTop20USDTpairsByVolume は24時間の取引量が多いUSDTペアを上位20件取得します。
func (g *KuCoinGateway) GetTop20USDTpairsByVolume() ([]string, error) {
	// 先物APIのエンドポイントに変更
	url := fmt.Sprintf("%s/api/v1/contracts/active", g.baseURL)
	respBody, err := g.httpClient.Get(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get active contracts: %w", err)
	}

	var contractsResp FuturesContractsResponse
	if err := json.Unmarshal(respBody, &contractsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contracts response: %w", err)
	}

	if contractsResp.Code != "200000" {
		return nil, fmt.Errorf("KuCoin API error: %s", string(respBody))
	}

	// USDT建ての契約のみをフィルタリングし、取引量でソート
	type contract struct {
		symbol string
		volume float64
	}
	var usdtContracts []contract
	for _, c := range contractsResp.Data {
		if c.Status == "Open" && !c.IsDele && c.QuoteCurrency == "USDT" {
			usdtContracts = append(usdtContracts, contract{symbol: c.Symbol, volume: c.Volume24h})
		}
	}

	// volumeで降順ソート
	sort.Slice(usdtContracts, func(i, j int) bool {
		return usdtContracts[i].volume > usdtContracts[j].volume
	})

	var pairs []string
	for i, c := range usdtContracts {
		if i >= 20 {
			break
		}
		pairs = append(pairs, c.symbol)
	}
	return pairs, nil
}

// GetCurrentPrice は現在の価格を取得します。
func (g *KuCoinGateway) GetCurrentPrice(symbol string) (float64, error) {
	// 先物APIのエンドポイントに変更
	url := fmt.Sprintf("%s/api/v1/ticker?symbol=%s", g.baseURL, symbol)
	respBody, err := g.httpClient.Get(url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get current price for %s: %w", symbol, err)
	}

	var priceResp struct {
		Code string `json:"code"`
		Data struct {
			Price float64 `json:"price"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &priceResp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal price response for %s: %w", symbol, err)
	}

	if priceResp.Code != "200000" {
		return 0, fmt.Errorf("KuCoin API error for price %s: %s", symbol, string(respBody))
	}

	return priceResp.Data.Price, nil
}

// --- Private Methods for Authentication ---

func (g *KuCoinGateway) getAuthHeaders(method, endpoint, body string) map[string]string {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	strToSign := timestamp + method + endpoint + body

	h := hmac.New(sha256.New, []byte(g.apiSecret))
	h.Write([]byte(strToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	passphraseHash := hmac.New(sha256.New, []byte(g.apiSecret))
	passphraseHash.Write([]byte(g.passphrase))
	passphraseSignature := base64.StdEncoding.EncodeToString(passphraseHash.Sum(nil))

	return map[string]string{
		"KC-API-KEY":         g.apiKey,
		"KC-API-SIGN":        signature,
		"KC-API-TIMESTAMP":   timestamp,
		"KC-API-PASSPHRASE":  passphraseSignature,
		"KC-API-KEY-VERSION": "2",
		"Content-Type":       "application/json",
	}
}

// CreateOrder は新しい注文を作成します。
func (g *KuCoinGateway) CreateOrder(symbol string, side string, orderType string, size string) (string, error) {
	endpoint := "/api/v1/orders"
	url := g.baseURL + endpoint

	reqBodyMap := map[string]string{
		"clientOid": fmt.Sprintf("%d", time.Now().UnixNano()),
		"symbol":    symbol,
		"side":      side,      // "buy" or "sell"
		"type":      orderType, // "market" or "limit"
		"leverage":  "1",       // レバレッジを1に固定
		"size":      size,      // for market order
		// "price":     price,  // for limit order
	}
	reqBodyBytes, err := json.Marshal(reqBodyMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal order request body: %w", err)
	}
	reqBody := string(reqBodyBytes)

	headers := g.getAuthHeaders("POST", endpoint, reqBody)

	respBody, err := g.httpClient.Post(url, headers, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create order: %w", err)
	}

	var orderResp struct {
		Code string `json:"code"`
		Data struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal order response: %s", string(respBody))
	}

	if orderResp.Code != "200000" {
		return "", fmt.Errorf("failed to create order on KuCoin: %s", orderResp.Msg)
	}

	return orderResp.Data.OrderID, nil
}

// GetKlines は指定した期間のローソク足データを取得します。
func (g *KuCoinGateway) GetKlines(symbol string, granularity int, count int) ([][]string, error) {
	// 先物APIのK-Lineエンドポイントとパラメータに変更
	// granularity: 1, 5, 15, 30, 60, 120, 240, 480, 720, 1440, 10080 (minutes)
	// 先物APIではstartAt/endAtは使わない（直近のデータを取得する）
	endpoint := fmt.Sprintf("/api/v1/kline/query?symbol=%s&granularity=%d", symbol, granularity)
	url := g.baseURL + endpoint

	respBody, err := g.httpClient.Get(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines for %s: %w", symbol, err)
	}

	var klineResp struct {
		Code string     `json:"code"`
		Data [][]string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &klineResp); err != nil {
		// KuCoinのデータ形式は [[string, string, ...]] なので、一度interface{}で受けてから変換する
		var rawKlineResp struct {
			Code string          `json:"code"`
			Data [][]interface{} `json:"data"`
		}
		if err2 := json.Unmarshal(respBody, &rawKlineResp); err2 != nil {
			return nil, fmt.Errorf("failed to unmarshal kline response for %s: %w", symbol, err2)
		}
		if rawKlineResp.Code != "200000" {
			return nil, fmt.Errorf("KuCoin API error for kline %s: %s", symbol, string(respBody))
		}
		// interface{}をstringに変換
		klineResp.Data = make([][]string, len(rawKlineResp.Data))
		for i, d := range rawKlineResp.Data {
			klineResp.Data[i] = make([]string, len(d))
			for j, v := range d {
				klineResp.Data[i][j] = fmt.Sprintf("%v", v)
			}
		}
	}

	if klineResp.Code != "200000" && klineResp.Code != "" {
		return nil, fmt.Errorf("KuCoin API error for kline %s: %s", symbol, string(respBody))
	}

	if len(klineResp.Data) == 0 {
		return nil, fmt.Errorf("no kline data returned for %s", symbol)
	}

	return klineResp.Data, nil
}
