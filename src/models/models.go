package models

import (
	"fmt"

	"github.com/monobot/dispatch/src/environment"
	"golang.org/x/exp/slices"
)

type ConfigCondition struct {
	Variable  string `json:"variable"`
	Value     string `json:"value"`
	Allowance bool   `json:"allowance"`
}

type ConfigCommand struct {
	Command    string            `json:"command"`
	Conditions []ConfigCondition `json:"conditions,omitempty"`
}

type ConfigParam struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // choices: string, int, bool
	Default     string `json:"default"`
	Mandatory   bool   `json:"mandatory"`
}

type ConfigTask struct {
	Name        string          `json:"name"`
	Group       string          `group:"name"`
	Description string          `json:"description,omitempty"`
	Commands    []ConfigCommand `json:"commands"`
	Envs        []string        `json:"envs,omitempty"`
	Params      []ConfigParam   `json:"params,omitempty"`
}

func (task ConfigTask) Help() {
	fmt.Printf("%s:\n", task.Name)
	fmt.Printf("  description: %s\n", task.Description)
	fmt.Println("  params accepted:\n", task.Description)
	for _, param := range task.Params {
		fmt.Printf("    %s\n", param.Name)
	}
}

type ConfigFile struct {
	Envs  []string     `json:"envs"`
	Tasks []ConfigTask `json:"tasks"`
}

func (config *ConfigFile) TaskNames() []string {
	taskNames := []string{}
	for _, task := range config.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	return taskNames
}

func (config *ConfigFile) Combine(added ConfigFile) ConfigFile {
	combinedEnvironments := config.Envs
	for _, env := range added.Envs {
		if !slices.Contains(combinedEnvironments, env) {
			combinedEnvironments = append(combinedEnvironments, env)
		}
	}

	newConfigTasks := config.Tasks
	currentTaskNames := config.TaskNames()
	for _, task := range added.Tasks {
		if slices.Contains(currentTaskNames, task.Name) {
			panic(fmt.Sprintf("Task names should not be duplicated!\n    Task %s is duplicated", task.Name))
		}
		newConfigTasks = append(newConfigTasks, task)
	}

	return ConfigFile{
		Envs:  combinedEnvironments,
		Tasks: newConfigTasks,
	}
}

type Configuration struct {
	ConfigFile ConfigFile
	Envs       map[string]string
	Params     map[string]ConfigParam
	Tasks      map[string]ConfigTask
	Groups     map[string][]string
}

func (configuration *Configuration) AddParam(param string, value ConfigParam) *Configuration {
	configuration.Params[param] = value

	return configuration
}

func BuildConfiguration(configFiles []ConfigFile) Configuration {
	configFile := configFiles[0]
	for _, innerConfig := range configFiles[1:] {
		configFile = configFile.Combine(innerConfig)
	}

	groups := make(map[string][]string)
	tasks := make(map[string]ConfigTask)
	for _, task := range configFile.Tasks {
		tasks[task.Name] = task
		taskGroup := task.Group
		if taskGroup == "" {
			taskGroup = "default"
		}
		currentGroupTasks, ok := groups[taskGroup]
		if !ok {
			currentGroupTasks = []string{}
		}

		groups[taskGroup] = append(currentGroupTasks, task.Name)
	}

	return Configuration{
		ConfigFile: configFile,
		Envs:       environment.PopulateVariables(configFile.Envs),
		Params:     make(map[string]ConfigParam),
		Tasks:      tasks,
		Groups:     groups,
	}
}
