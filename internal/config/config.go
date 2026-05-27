package config

import (
	"flag"
	"os"
)

// Config задаёт параметры запуска сервиса.
// Каждый параметр задаётся флагом или одноимённой переменной окружения (флаг имеет приоритет).
type Config struct {
	RunAddress           string // -a / RUN_ADDRESS
	DatabaseURI          string // -d / DATABASE_URI
	AccrualSystemAddress string // -r / ACCRUAL_SYSTEM_ADDRESS
}

func Load() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.RunAddress, "a", "", "address and port to run service")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "database connection URI")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "accrual system address")

	flag.Parse()

	cfg.RunAddress = valueFromFlagOrEnv(cfg.RunAddress, "RUN_ADDRESS", "localhost:8080")
	cfg.DatabaseURI = valueFromFlagOrEnv(cfg.DatabaseURI, "DATABASE_URI", "")
	cfg.AccrualSystemAddress = valueFromFlagOrEnv(cfg.AccrualSystemAddress, "ACCRUAL_SYSTEM_ADDRESS", "")

	return cfg
}

func valueFromFlagOrEnv(flagValue, envKey, defaultValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultValue
}
