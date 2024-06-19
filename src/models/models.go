package models

import (
	"fmt"

	"bytes"
	"os/exec"
	"reflect"
	"strings"
	"text/template"

	"github.com/monobot/dispatch/src/environment"

	"github.com/fatih/color"
	"golang.org/x/exp/slices"
)

type Condition struct {
	Variable  string `json:"variable" yaml:"variable"`
	Value     string `json:"value" yaml:"value"`
	Allowance bool   `json:"allowance" yaml:"allowance"`
}

func (condition Condition) HelpString() string {
	allowance := "Deny"
	if condition.Allowance {
		allowance = "Allow"
	}
	return color.RedString(condition.Variable)+" equals "+color.RedString(condition.Value)+" then "+color.RedString(allowance)
}

func (condition Condition) Help(indentCount int) {
	indentString := getIndentString(indentCount)

	conditionString := condition.HelpString()
	fmt.Printf("%s    When "+conditionString+"\n", indentString)
}

type Command struct {
	Command    string      `json:"command" yaml:"command"`
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Allowed    bool
}

func (command Command) Help(indentCount int) {
	indentString := getIndentString(indentCount)

	fmt.Printf("%s    - \"%s\"\n", indentString, command.Command)
	if len(command.Conditions) > 0 {
		fmt.Printf("%s    Conditions:\n", indentString)
		for _, condition := range command.Conditions {
			condition.Help(indentCount + 1)
		}
	}
}

func (command Command) Run(configuration *Configuration) {
	allowance := true
	failConditionString := ""
	for _, condition := range command.Conditions {
		parsedValue := configuration.ContextData[condition.Variable]
		if condition != (Condition{}) {
			allowed := false
			if condition.Allowance {
				allowed = parsedValue == condition.Value
			} else {
				allowed = parsedValue != condition.Value
			}

			if allowance && !allowed {
				allowance = allowed
				failConditionString = condition.HelpString()
			}
		}
	}

	template, err := template.New("commandTemplate").Parse(command.Command)
	if err != nil {
		// this should not pass silently
		fmt.Println(err)
	}
	var outputBytes bytes.Buffer
	if err := template.Execute(&outputBytes, configuration.ContextData); err != nil {
		// this should not pass silently
		fmt.Println(err)
	}

	runCommand := outputBytes.String()
	if allowance {
		if configuration.HasFlag("verbose") {
			fmt.Printf(color.YellowString("running ")+"\"%s\":\n", runCommand)
		}

		splittedCommand := strings.Fields(runCommand)

		if len(splittedCommand) > 0 {
			baseCmd := splittedCommand[0]
			cmdArgs := splittedCommand[1:]

			output, _ := exec.Command(baseCmd, cmdArgs...).CombinedOutput()
			stringOutput := string(output)
			if stringOutput != "" {
				fmt.Println(stringOutput)
			}
		}
	} else {
		if configuration.HasFlag("verbose") {
			fmt.Printf(color.YellowString("Command") + " \"%s\" " + color.YellowString("not run.\n"), runCommand)
			fmt.Println("    condition: " +failConditionString+" not met\n")
		}
	}
}

type Parameter struct {
	Name      string `json:"name" yaml:"name"`
	Default   string `json:"default" yaml:"default"`
	Mandatory bool   `json:"mandatory" yaml:"mandatory"`
}

func (param Parameter) Help(indentCount int) {
	indentString := getIndentString(indentCount)

	fmt.Printf("%s    %s\n", indentString, param.Name)
	if param.Default != "" {
		fmt.Printf("%s        default: %s\n", indentString, param.Default)
	}
	if param.Mandatory {
		fmt.Printf("%s        mandatory: %v\n", indentString, param.Mandatory)
	}
}

type Task struct {
	Name        string      `json:"name" yaml:"name"`
	Group       string      `json:"group" yaml:"group"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Commands    []Command   `json:"commands" yaml:"commands"`
	Envs        []string    `json:"envs,omitempty" yaml:"envs,omitempty"`
	Params      []Parameter `json:"params,omitempty" yaml:"params,omitempty"`
}

func (task Task) Help(indentCount int, detailed bool) {
	indentString := getIndentString(indentCount)

	fmt.Printf("%s"+color.BlueString(task.Name)+":\n", indentString)
	if task.Description != "" {
		fmt.Printf("%s    %s\n", indentString, task.Description)
	}
	if detailed {
		if len(task.Commands) > 0 {
			color.Cyan("%s    Commands:\n", indentString)
			for _, command := range task.Commands {
				command.Help(indentCount + 1)
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
				param.Help(indentCount + 1)
			}
		}
	}
	fmt.Println("")
}

func (task Task) Run(configuration *Configuration) {
	allowance := true
	parameterString := ""
	for _, param := range task.Params {
		allowed := true

		if param.Mandatory {
			allowed = configuration.HasFlag(param.Name)
		}

		if allowance && !allowed {
			parameterString = "parameter \""+color.YellowString(param.Name)+"\" is mandatory"
			allowance = allowed
		}
	}

	if allowance {
		for _, command := range task.Commands {
			// check params condition met

			command.Run(configuration)
		}
	} else {
		if configuration.HasFlag("verbose") {
			fmt.Printf("task \"%s\" not run, "+parameterString+"\n", task.Name)
		}
	}
}

type ConfigFile struct {
	Envs  []string `json:"envs" yaml:"envs"`
	Tasks []Task   `json:"tasks" yaml:"tasks"`
}

func (configFile *ConfigFile) TaskNames() []string {
	taskNames := []string{}
	for _, task := range configFile.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	return taskNames
}

func (configFile *ConfigFile) Combine(newConfigFile ConfigFile) ConfigFile {
	combinedEnvironments := configFile.Envs
	for _, env := range newConfigFile.Envs {
		if !slices.Contains(combinedEnvironments, env) {
			combinedEnvironments = append(combinedEnvironments, env)
		}
	}

	newTasks := configFile.Tasks
	currentTaskNames := configFile.TaskNames()
	for _, task := range newConfigFile.Tasks {
		if slices.Contains(currentTaskNames, task.Name) {
			panic(fmt.Sprintf("Task names should not be duplicated!\n    Task %s is duplicated", task.Name))
		}
		newTasks = append(newTasks, task)
	}

	return ConfigFile{
		Envs:  combinedEnvironments,
		Tasks: newTasks,
	}
}

type Configuration struct {
	ConfigFile   ConfigFile
	Params       map[string]Parameter
	Tasks        map[string]Task
	Groups       map[string][]string
	ContextData  map[string]string
}

func (configuration *Configuration) HasFlag(flag string) bool {
	_, ok := configuration.ContextData[flag]

	return ok
}

func (configuration *Configuration) UpdateContextData(contextData map[string]string) {
	for param, value := range contextData {
		currentValue, hasKey := configuration.ContextData[param]
		if !hasKey || currentValue == "" {
			configuration.ContextData[param] = value
		}
	}
}

func BuildConfiguration(configFiles []ConfigFile, contextData map[string]string) *Configuration {
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
		ConfigFile:   configFile,
		Tasks:        tasks,
		Groups:       groups,
	}
	configuration.ContextData = contextData
	configuration.UpdateContextData(environment.PopulateVariables(configFile.Envs))

	return &configuration
}

func getIndentString(nestCount int) string {
	indent := ""
	for i := 0; i < nestCount; i++ {
		indent += "    "
	}
	return indent
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
		PrintHelpGroupTasks(groupTasks, configuration, indentCount)
	}
}
