package environment

import (
	"os"
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
