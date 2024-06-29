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
		newEnvironmentMap[v] = GetEnvironmentVariable(v)
	}

	return newEnvironmentMap
}

func PopulateFromEnvFile(envFilePath string) map[string]string {
	envFile, _ := godotenv.Read(envFilePath)
	return envFile
}
