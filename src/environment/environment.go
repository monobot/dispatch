package environment

import (
	"os"

	"github.com/joho/godotenv"
)

func GetEnvironmentVariable(variableKey string) string {
	return os.Getenv(variableKey)
}

func PopulateVariables(environmentKeys []string) map[string]string {
	newEnvironmentMap := make(map[string]string)
	for _, v := range environmentKeys {
		envValue := GetEnvironmentVariable(v)
		if envValue != "" {
			newEnvironmentMap[v] = envValue
		}
	}

	return newEnvironmentMap
}

func PopulateFromEnvFile(envFilePath string) map[string]string {
	parsedEnvValues, _ := godotenv.Read(envFilePath)
	return parsedEnvValues
}
