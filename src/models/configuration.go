package models

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/color"
	"github.com/monobot/dispatch/src/environment"
	"golang.org/x/exp/slices"
)

type ContextData struct {
	Flags []string
	Data  map[string]string
}

func (contextData *ContextData) AddFlag(flag string) {
	alreadyContains := slices.Contains(contextData.Flags, flag)
	if !alreadyContains {
		contextData.Flags = append(contextData.Flags, flag)
	}
}

func (contextData *ContextData) HasFlag(flag string) bool {
	return slices.Contains(contextData.Flags, flag)
}

func (contextData *ContextData) UpdateDatum(key string, value string) {
	currentGroupTasks, ok := contextData.Data[key]
	if !ok || currentGroupTasks == "" {
		contextData.Data[key] = value
	}
}
func (contextData *ContextData) UpdateData(data map[string]string) {
	for key, value := range data {
		contextData.UpdateDatum(key, value)
	}
}

type Configuration struct {
	Params map[string]Parameter
	Envs   []string

	Tasks       map[string]Task
	TaskGroups  map[string][]string
	ContextData ContextData
}

func (configuration *Configuration) HasFlag(flag string) bool {
	return configuration.ContextData.HasFlag(flag)
}

func BuildConfiguration(configFiles []ConfigFile, contextData ContextData) *Configuration {
	// configure default tasks
	configFile := ConfigFile{
		Envs: []string{},
		Tasks: []Task{
			{
				Name:        "help",
				Description: "Show this help",
				Commands:    []Command{},
			},
		},
	}
	for _, innerConfig := range configFiles {
		configFile = configFile.Combine(innerConfig)
	}

	groups := make(map[string][]string)
	tasks := make(map[string]Task)
	envsValues := map[string]string{}
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

		for _, param := range task.Params {
			value, hasKey := envsValues[param.Name]
			if !hasKey {
				envsValues[param.Name] = value
			}
		}
	}

	configuration := Configuration{
		Tasks:      tasks,
		Envs:       configFile.Envs,
		TaskGroups: groups,
	}
	contextData.UpdateData(environment.PopulateVariables(configFile.Envs))
	contextData.UpdateData(envsValues)

	configuration.ContextData = contextData
	return &configuration
}

func PrintHelpGroupTasks(groupTasks []string, configuration *Configuration, indentCount int) {
	for _, taskName := range groupTasks {
		task := configuration.Tasks[taskName]

		task.Help(indentCount, configuration.HasFlag("verbose"))
	}
}

func Help(configuration *Configuration) {
	// Print help message
	fmt.Println("")
	title := color.New(color.FgRed).Add(color.Bold)
	title.Println("THIS IS 'dispatch' HELP.")
	fmt.Println("You can find more information on how to build and configure your own dispatch tasks, here:")
	fmt.Println("    TODO")
	fmt.Println("")

	// environments
	color.Yellow("Environments:\n")
	environments := strings.Join(configuration.Envs, ", ")
	fmt.Printf("    %s\n\n", environments)

	// tasks
	indentCount := 0
	if len(configuration.TaskGroups) > 1 {
		indentCount += 1
	}
	groupNames := reflect.ValueOf(configuration.TaskGroups).MapKeys()
	for _, groupName := range groupNames {
		groupTasks := configuration.TaskGroups[groupName.String()]
		if len(configuration.TaskGroups) > 1 {
			color.Yellow("%s:\n", groupName)
		}
		PrintHelpGroupTasks(groupTasks, configuration, indentCount)
	}
}
