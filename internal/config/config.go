package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv          string
	Port            string
	MySQLDSN        string
	JWTSecret       string
	JWTIssuer       string
	JWTTTL          time.Duration
	Timezone        string
	AutoMigrate     bool
	AutoSeedDemoData bool
	WechatAppID     string
	WechatAppSecret string
	WechatMock      bool
}

func Load() Config {
	return Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		Port:            getEnv("PORT", "8080"),
		MySQLDSN:        getEnv("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/baby_record?charset=utf8mb4&parseTime=True&loc=Local"),
		JWTSecret:       getEnv("JWT_SECRET", "baby-record-secret"),
		JWTIssuer:       getEnv("JWT_ISSUER", "baby-record-server"),
		JWTTTL:          time.Duration(getInt("JWT_TTL_MINUTES", 60*24*7)) * time.Minute,
		Timezone:        getEnv("TIMEZONE", "Asia/Shanghai"),
		AutoMigrate:     getBool("AUTO_MIGRATE", true),
		AutoSeedDemoData: getBool("AUTO_SEED_DEMO_DATA", true),
		WechatAppID:     os.Getenv("WECHAT_APP_ID"),
		WechatAppSecret: os.Getenv("WECHAT_APP_SECRET"),
		WechatMock:      getBool("WECHAT_MOCK", true),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
