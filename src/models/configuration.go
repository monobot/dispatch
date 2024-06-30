package models

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/color"
	"github.com/monobot/dispatch/src/environment"
	"golang.org/x/exp/slices"
)

type ConfigurationData struct {
	Flags       []string
	ContextData map[string]string
}

func (configurationData *ConfigurationData) AddFlag(flag string) {
	alreadyContains := slices.Contains(configurationData.Flags, flag)
	if !alreadyContains {
		configurationData.Flags = append(configurationData.Flags, flag)
	}
}

func (configurationData *ConfigurationData) HasFlag(flag string) bool {
	return slices.Contains(configurationData.Flags, flag)
}

func UpdateData(sourceMap map[string]string, newMap map[string]string) {
	for key, value := range newMap {
		sourceMap[key] = value
	}
}

type Configuration struct {
	Params map[string]Parameter
	Envs   []string

	Tasks             map[string]Task
	TaskGroups        map[string][]string
	ConfigurationData ConfigurationData
}

func (configuration *Configuration) HasFlag(flag string) bool {
	return configuration.ConfigurationData.HasFlag(flag)
}

func (configuration *Configuration) RunTask(taskName string) (string, error) {
	task := configuration.Tasks[taskName]
	task.SetContextData(configuration)

	taskAllowed := task.IsAllowed(configuration)

	subcommandCount := 0
	subcommandFailedCount := 0
	if taskAllowed {
		subcommandCount += 1
		successfullyRun := true
		for idx := range task.Commands {
			// check params condition met
			message, err := task.RunCommand(idx, configuration, task.ContextData)
			if err != nil && successfullyRun {
				successfullyRun = false
				fmt.Printf(color.RedString("ERROR:")+" %s\n", message)
				subcommandFailedCount += 1
			}
		}

		if successfullyRun {
			if configuration.HasFlag("verbose") {
				fmt.Println(color.CyanString("info: ") + "task \"" + task.Name + "\" completed")
			}
		}
	}

	if subcommandFailedCount > 0 {
		return color.YellowString("%v\\%v commands failed", subcommandFailedCount, subcommandCount), fmt.Errorf("task %s failed", task.Name)
	} else {
		return "", nil
	}
}

func BuildConfiguration(configFiles []ConfigFile, configurationData ConfigurationData) *Configuration {
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

	configuration := Configuration{
		Tasks:      tasks,
		Envs:       configFile.Envs,
		TaskGroups: groups,
	}

	contextData := make(map[string]string)
	for _, envFile := range configFile.EnvFiles {
		UpdateData(contextData, environment.PopulateFromEnvFile(envFile))
	}
	UpdateData(contextData, environment.PopulateVariables(configFile.Envs))

	configurationData.ContextData = contextData
	configuration.ConfigurationData = configurationData

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
