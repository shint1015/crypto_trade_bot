package gateway

import (
	"bytes"
	"crypto_trade_bot/infra/client"
	"crypto_trade_bot/infra/config"
	"encoding/json"
	"fmt"
	"strings"
)

// OpenAIGateway は OpenAI API との通信を抽象化します。
type OpenAIGateway struct {
	httpClient *client.HTTPClient
	apiKey     string
	baseURL    string
}

// NewOpenAIGateway は新しい OpenAIGateway を生成します。
func NewOpenAIGateway(httpClient *client.HTTPClient) *OpenAIGateway {
	return &OpenAIGateway{
		httpClient: httpClient,
		apiKey:     config.GetEnv("OPENAI_API_KEY", ""),
		baseURL:    "https://api.openai.com/v1",
	}
}

// OpenAIRequest は OpenAI API へのリクエストボディです。
type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message はプロンプトのメッセージ構造体です。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse は OpenAI API からのレスポンスボディです。
type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// AskAboutAssets は指定された通貨ペアについて、OpenAIに分析を依頼します。
func (g *OpenAIGateway) AskAboutAssets(assetSymbols []string, side string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set. Please create a .env file and add your API key")
	}

	var prompt string
	if side == "buy" {
		prompt = fmt.Sprintf(
			"以下の通貨ペアが、MACDのゴールデンクロスとRSIが70未満という条件で抽出されました。これらは上昇トレンドの可能性があります。これらのテクニカル指標を考慮した上で、今後さらに上昇が期待できる通貨はどれですか？理由も添えて、最も有望なものを1つか2つに絞って教えてください。\n\n%s",
			strings.Join(assetSymbols, "\n"),
		)
	} else if side == "sell" {
		prompt = fmt.Sprintf(
			"以下の通貨ペアが、MACDのデッドクロスとRSIが30以上という条件で抽出されました。これらは下降トレンドの可能性があります。これらのテクニカル指標を考慮した上で、今後さらに下落が期待できる（ショートポジションが有効な）通貨はどれですか？理由も添えて、最も有望なものを1つか2つに絞って教えてください。\n\n%s",
			strings.Join(assetSymbols, "\n"),
		)
	} else {
		return "", fmt.Errorf("invalid side for OpenAI prompt: %s", side)
	}

	requestBody, err := json.Marshal(OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal openai request: %w", err)
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + g.apiKey,
	}

	url := fmt.Sprintf("%s/chat/completions", g.baseURL)
	respBody, err := g.httpClient.Post(url, headers, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to post to openai: %w", err)
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal openai response: %w", err)
	}

	if len(openAIResp.Choices) > 0 {
		return openAIResp.Choices[0].Message.Content, nil
	}

	return "No response from OpenAI.", nil
}
