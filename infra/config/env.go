package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv は .env ファイルから環境変数を読み込みます。
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
}

// GetEnv は指定されたキーの環境変数を取得します。
func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
