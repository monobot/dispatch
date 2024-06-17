package models

import (
	"fmt"

	"reflect"
	"os/exec"
	"strings"

	"github.com/monobot/dispatch/src/environment"

	"github.com/fatih/color"
	"golang.org/x/exp/slices"
)

type ConfigCondition struct {
	Variable  string `json:"variable"`
	Value     string `json:"value"`
	Allowance bool   `json:"allowance"`
}

func (configCondition ConfigCondition) Help(indentCount int) {

	indentString := getIndentString(indentCount)

	allowance := "Deny"
	if configCondition.Allowance {
		allowance = "Allow"
	}

	fmt.Printf( "%s    When " +color.RedString(configCondition.Variable) + " equals " +color.RedString(configCondition.Value) +" then "+color.RedString(allowance) +"\n", indentString )
}

type ConfigCommand struct {
	Command    string            `json:"command"`
	Conditions []ConfigCondition `json:"conditions,omitempty"`
	Allowed    bool
}

func (configCommand ConfigCommand) Help(indentCount int) {
	indentString := getIndentString(indentCount)

	fmt.Printf("%s    - \"%s\"\n", indentString, configCommand.Command)
	if len(configCommand.Conditions) > 0 {
		fmt.Printf("%s    Conditions:\n", indentString)
		for _, condition := range configCommand.Conditions {
			condition.Help(indentCount+1)
		}
	}
}

func (configCommand ConfigCommand) CalculateCommands() []string {
	// check that the condition is met
	return strings.Fields(configCommand.Command)
}

type ConfiguredParamValue struct {
	Type  string // choices: string, int, bool
	Value string
}

type ConfigParam struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // choices: string, int, bool
	Default     string `json:"default"`
	Mandatory   bool   `json:"mandatory"`
	Value       string
}

func (configParam ConfigParam) Help (indentCount int) {
	indentString := getIndentString(indentCount)

	fmt.Printf("%s    %s\n", indentString, configParam.Name)
	if configParam.Description != "" {
		fmt.Printf("%s        %s\n", indentString, configParam.Description)
	}
	if configParam.Type != "" {
		fmt.Printf("%s        type: %s\n", indentString, configParam.Type)
	}
	if configParam.Default != "" {
		fmt.Printf("%s        default: %s\n", indentString, configParam.Default)
	}
	if configParam.Mandatory {
		fmt.Printf("%s        mandatory: %v\n", indentString, configParam.Mandatory)
	}
}

type ConfigTask struct {
	Name        string          `json:"name"`
	Group       string          `group:"name"`
	Description string          `json:"description,omitempty"`
	Commands    []ConfigCommand `json:"commands"`
	Envs        []string        `json:"envs,omitempty"`
	Params      []ConfigParam   `json:"params,omitempty"`
}

func (task ConfigTask) Help(indentCount int, detailed bool) {
	indentString := getIndentString(indentCount)

	fmt.Printf("%s"+color.BlueString(task.Name)+":\n", indentString)
	if task.Description != "" {
		fmt.Printf("%s    %s\n", indentString, task.Description)
	}
	if detailed {
		if len(task.Commands) > 0 {
			color.Cyan("%s    Commands:\n", indentString)
			for _, command := range task.Commands {
				command.Help(indentCount+1)
			}
		}
		if len(task.Envs) > 0 {
			color.Cyan("%s    Environments:\n", indentString)
			environments := strings.Join(task.Envs, ", ")
			fmt.Printf("%s        %s\n", indentString, environments)
		}
		if len(task.Params) > 0 {
			color.Cyan("%s    Params:\n", indentString)
			for _, param := range task.Params {
				param.Help(indentCount+1)
			}
		}
	}
	fmt.Println("")
}

func (task ConfigTask) Run() {
	for _, calculatedCommand := range task.CalculateCommands() {
		if len(calculatedCommand) > 0 {
			fmt.Println(strings.Join(calculatedCommand, " "))
			baseCmd := calculatedCommand[0]
			cmdArgs := calculatedCommand[1:]

			out, err := exec.Command(baseCmd, cmdArgs...).Output()
			if err != nil {
				fmt.Printf("%s",err)
			}
			if out != nil {
				fmt.Printf("%s",out)
			}
		}
	}
}

func (task ConfigTask) CalculateCommandsAllowance() {
	for _, command := range task.Commands {
		allowance := true
		for _, condition := range command.Conditions {
			allowed := true
			if condition.Allowance {
				allowed = condition.Variable == condition.Value
			} else {
				allowed = condition.Variable != condition.Value
			}
			if allowance && !allowed {
				allowance = allowed
			}
		}
		command.Allowed = allowance
	}
}

func (task ConfigTask) CalculateCommands() [][]string {
	calculatedCommands := [][]string{}
	for _, command := range task.Commands {
		calculatedCommands = append(calculatedCommands, command.CalculateCommands())
	}

	return calculatedCommands
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
	// configure default tasks
	configFile := ConfigFile{
		Envs: []string{},
		Tasks: []ConfigTask{
			{
				Name:        "help",
				Description: "Show this help",
				Commands:    []ConfigCommand{},
			},
		},
	}
	for _, innerConfig := range configFiles {
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
		Envs:          environment.PopulateVariables(configFile.Envs),
		Params:        make(map[string]ConfigParam),
		Tasks:         tasks,
		Groups:        groups,
	}
}

func getIndentString(nestCount int) string {
	indent := ""
	for i := 0; i < nestCount; i++ {
		indent += "    "
	}
	return indent
}

func PrintHelpGroupTasks(groupTasks []string, configuration Configuration, indentCount int, detailed bool) {
	for _, taskName := range groupTasks {
		task := configuration.Tasks[taskName]
		task.Help(indentCount, detailed)
	}
}

func Help(configuration Configuration) {
	// Print help message
	fmt.Println("")
	title := color.New(color.FgRed).Add(color.Bold)
	title.Println("THIS IS 'dispatch' HELP.")
	fmt.Println(("You can find more information on how to build and configure your own dispatch tasks, here:"))
	fmt.Println(("    TODO"))
	fmt.Println((""))

	// enviornments
	color.Yellow("Environments:\n")
	environments := strings.Join(configuration.ConfigFile.Envs, ", ")
	fmt.Printf("    %s\n\n", environments)

	// tasks
	indentCount := 0
	if len(configuration.Groups) > 1 {
		indentCount += 1
	}
	groupNames := reflect.ValueOf(configuration.Groups).MapKeys()
	for _, groupName := range groupNames {
		groupTasks := configuration.Groups[groupName.String()]
		if len(configuration.Groups) > 1 {
			color.Yellow("%s:\n", groupName)
		}
		PrintHelpGroupTasks(groupTasks, configuration, indentCount, false)
	}
}
