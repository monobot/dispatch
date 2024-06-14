package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type ConfigConditionsStruct struct {
	Variable  string `json:"variable"`
	Value     string `json:"value"`
	Allowance bool   `json:"allowance"`
}

type ConfigCommandsStruct struct {
	Command    string                   `json:"command"`
	Conditions []ConfigConditionsStruct `json:"conditions,omitempty"`
}

type ConfigParamsStruct struct {
	Name      string `json:"name"`
	Mandatory bool   `json:"mandatory"`
	Default   string `json:"default"`
}

type ConfigTasksStruct struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Commands    []ConfigCommandsStruct `json:"commands"`
	Envs        []string               `json:"envs,omitempty"`
	Params      []ConfigParamsStruct   `json:"params,omitempty"`
}

type ConfigGroupStruct struct {
	Name  string              `json:"name"`
	Tasks []ConfigTasksStruct `json:"tasks"`
}

type ConfigFileStruct struct {
	Envs   []string            `json:"envs"`
	Groups []ConfigGroupStruct `json:"groups"`
}

func readConfigurationFile(filePath string) *ConfigFileStruct {
	myFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
	}
	defer myFile.Close()

	data, err := io.ReadAll(myFile)
	if err != nil {
		fmt.Println(err)
	}

	configFileStruct := ConfigFileStruct{}
	err = json.Unmarshal(data, &configFileStruct)
	if err != nil {
		fmt.Println(err)
	}

	return &configFileStruct
}

func discoverConfigFilesPath(absolutePath string) *[]string {
	// Read all posible paths for dispatch configuration files so all available tasks are detected.
	f, err := os.Open(absolutePath)
	if err != nil {
		fmt.Println(err)
	}

	files, err := f.Readdir(0)
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

func main() {
	configurationFileNames := []string{}

	currentDir, _ := os.Getwd()
	checkDirs := []string{"/home/monobot/", currentDir}
	for _, filePath := range checkDirs {
		configurationFileNames = append(configurationFileNames, *discoverConfigFilesPath(filePath)...)
	}

	configurations := []ConfigFileStruct{}
	for _, filePath := range configurationFileNames {
		configurations = append(configurations, *readConfigurationFile(filePath))
	}
	fmt.Printf("%v", configurations)

}
