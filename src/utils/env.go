package utils

import (
	"os"
	"strings"
)

type EnvOptions struct {
	Debug bool
}

func getDebug() bool {
	value := os.Getenv("DEBUG")
	return value == "1" || strings.ToLower(value) == "true"
}

func GetEnvOptions() EnvOptions {
	return EnvOptions{
		Debug: getDebug(),
	}
}
