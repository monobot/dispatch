package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/exp/slices"

	"github.com/monobot/dispatch/src/models"
)

func readConfigurationFile(filePath string) *models.ConfigFile {
	myFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
	}
	defer myFile.Close()

	data, err := io.ReadAll(myFile)
	if err != nil {
		fmt.Println(err)
	}

	configFileStruct := models.ConfigFile{}
	err = json.Unmarshal(data, &configFileStruct)
	if err != nil {
		fmt.Println(err)
	}

	return &configFileStruct
}

func discoverConfigFilesPath(absolutePath string) *[]string {
	// Read all posible paths for dispatch configuration files so all available tasks are detected.
	dir, err := os.Open(absolutePath)
	if err != nil {
		fmt.Println(err)
	}

	files, err := dir.Readdir(0)
	if err != nil {
		fmt.Println(err)
	}

	configurationFileNames := []string{}

	re, err := regexp.Compile(".*.dispatch")
	if err != nil {
		fmt.Println("Error compiling regular expression")
	}

	for _, v := range files {
		fileName := v.Name()

		if re.MatchString(fileName) {
			configurationFileNames = append(configurationFileNames, filepath.Join(absolutePath, fileName))
		}
	}
	return &configurationFileNames
}

func getDiscoveryDirs() []string {
	currentDir, _ := os.Getwd()
	discoveryDirs := []string{currentDir}

	homeDir, _ := os.UserHomeDir()
	configuredDir := os.Getenv("DISPATCH_CONFIG_DIR")
	for _, dir := range []string{homeDir, configuredDir} {
		if !slices.Contains(discoveryDirs, dir) && dir != "" {
			discoveryDirs = append(discoveryDirs, dir)
		}
	}

	return discoveryDirs
}

func TaskDiscovery() []models.ConfigFile {
	configurationFileNames := []string{}

	discoveryDirs := getDiscoveryDirs()

	for _, filePath := range discoveryDirs {
		configurationFileNames = append(configurationFileNames, *discoverConfigFilesPath(filePath)...)
	}

	configurations := []models.ConfigFile{}
	for _, filePath := range configurationFileNames {
		configurations = append(configurations, *readConfigurationFile(filePath))
	}

	return configurations
}
