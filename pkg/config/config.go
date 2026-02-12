package config

import (
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	App        AppConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Blockchain BlockchainConfig
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string
	Version string
	Port    int
	Env     string // development, staging, production
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxIdleConns int
	MaxOpenConns int
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string
	ExpireTime time.Duration
}

// BlockchainConfig 区块链配置
type BlockchainConfig struct {
	Ethereum EthereumConfig
	Bitcoin  BitcoinConfig
	Tron     TronConfig
	BSC      EthereumConfig
	Polygon  EthereumConfig
}

// EthereumConfig 以太坊兼容链配置
type EthereumConfig struct {
	RPCURL             string
	ChainID            int64
	Confirmations      int
	GasLimitMultiplier float64
}

// BitcoinConfig 比特币配置
type BitcoinConfig struct {
	RPCURL        string
	RPCUser       string
	RPCPassword   string
	Network       string // mainnet, testnet
	Confirmations int
}

// TronConfig 波场配置
type TronConfig struct {
	RPCURL        string
	APIKey        string
	Network       string
	Confirmations int
}

// Load 加载配置
func Load() *Config {
	return &Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "custodial-wallet"),
			Version: getEnv("APP_VERSION", "1.0.0"),
			Port:    getEnvInt("APP_PORT", 8080),
			Env:     getEnv("APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnvInt("DB_PORT", 5432),
			User:         getEnv("DB_USER", "postgres"),
			Password:     getEnv("DB_PASSWORD", "postgres"),
			DBName:       getEnv("DB_NAME", "custodial_wallet"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 100),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			ExpireTime: time.Duration(getEnvInt("JWT_EXPIRE_HOURS", 24)) * time.Hour,
		},
		Blockchain: BlockchainConfig{
			Ethereum: EthereumConfig{
				RPCURL:             getEnv("ETH_RPC_URL", "http://localhost:8545"),
				ChainID:            int64(getEnvInt("ETH_CHAIN_ID", 1)),
				Confirmations:      getEnvInt("ETH_CONFIRMATIONS", 12),
				GasLimitMultiplier: 1.2,
			},
			Bitcoin: BitcoinConfig{
				RPCURL:        getEnv("BTC_RPC_URL", "http://localhost:8332"),
				RPCUser:       getEnv("BTC_RPC_USER", "bitcoin"),
				RPCPassword:   getEnv("BTC_RPC_PASSWORD", "bitcoin"),
				Network:       getEnv("BTC_NETWORK", "mainnet"),
				Confirmations: getEnvInt("BTC_CONFIRMATIONS", 6),
			},
			Tron: TronConfig{
				RPCURL:        getEnv("TRON_RPC_URL", "https://api.trongrid.io"),
				APIKey:        getEnv("TRON_API_KEY", ""),
				Network:       getEnv("TRON_NETWORK", "mainnet"),
				Confirmations: getEnvInt("TRON_CONFIRMATIONS", 19),
			},
			BSC: EthereumConfig{
				RPCURL:             getEnv("BSC_RPC_URL", "https://bsc-dataseed.binance.org/"),
				ChainID:            int64(getEnvInt("BSC_CHAIN_ID", 56)),
				Confirmations:      getEnvInt("BSC_CONFIRMATIONS", 15),
				GasLimitMultiplier: 1.2,
			},
			Polygon: EthereumConfig{
				RPCURL:             getEnv("POLYGON_RPC_URL", "https://polygon-rpc.com/"),
				ChainID:            int64(getEnvInt("POLYGON_CHAIN_ID", 137)),
				Confirmations:      getEnvInt("POLYGON_CONFIRMATIONS", 128),
				GasLimitMultiplier: 1.2,
			},
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
