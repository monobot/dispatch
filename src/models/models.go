package models

import (
	"fmt"
	"os"

	"bytes"
	"os/exec"
	"reflect"
	"strings"
	"text/template"

	"github.com/monobot/dispatch/src/environment"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"

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
	return "variable \"" + color.RedString(condition.Variable) + "\" equals \"" + color.RedString(condition.Value) + "\" then " + color.RedString(allowance)
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
		fmt.Printf("%s        Conditions:\n", indentString)
		for _, condition := range command.Conditions {
			condition.Help(indentCount + 2)
		}
	}
}

func (command Command) Run(configuration *Configuration) {
	logFields := log.Fields{"command": command.Command}

	allowance := true
	failConditionString := ""
	for _, condition := range command.Conditions {
		contextValue := configuration.ContextData.Data[condition.Variable]
		if condition != (Condition{}) {
			allowed := false
			if condition.Allowance {
				allowed = contextValue == condition.Value
			} else {
				allowed = contextValue != condition.Value
			}

			if allowance && !allowed {
				allowance = allowed
				failConditionString = condition.HelpString()
			}
			log.WithFields(logFields).Debugf("condition \"%v\" validated to: %v", condition.HelpString(), allowed)
		}
	}

	template, err := template.New("commandTemplate").Parse(command.Command)
	if err != nil {
		// this should not pass silently
		fmt.Println(err)
	}
	var outputBytes bytes.Buffer
	if err := template.Execute(&outputBytes, configuration.ContextData.Data); err != nil {
		// this should not pass silently
		fmt.Println(err)
	}

	runCommand := outputBytes.String()
	if allowance {
		log.WithFields(logFields).Debug("command is running")
		fmt.Printf(color.YellowString("running ")+"\"%s\":\n", runCommand)

		splitCommand := strings.Fields(runCommand)

		if len(splitCommand) > 0 {
			baseCmd := splitCommand[0]
			cmdArgs := splitCommand[1:]

			command := exec.Command(baseCmd, cmdArgs...)

			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			if err := command.Run(); err != nil {
				fmt.Println("could not run command: ", err)
			}
		}
	} else {
		log.WithFields(logFields).Debug("command not running")
		if configuration.HasFlag("verbose") {
			fmt.Printf(color.YellowString("Command")+" \"%s\" "+color.YellowString("not run.\n"), runCommand)
			fmt.Println("    condition: " + failConditionString + " not met\n")
		}
	}
}

type Parameter struct {
	Name      string `json:"name" yaml:"name"`
	Default   string `json:"default" yaml:"default"`
	Mandatory bool   `json:"mandatory" yaml:"mandatory"`
}

func (param Parameter) HelpString() string {
	mandatoryString := "is not"
	if param.Mandatory {
		mandatoryString = "is"
	}
	return mandatoryString + color.YellowString(" mandatory")
}

func (param Parameter) Help(indentCount int) {
	indentString := getIndentString(indentCount)

	defaultString := ""
	if param.Default != "" {
		defaultString = " default: " + color.YellowString(param.Default)
	}
	mandatoryString := ""
	if param.Mandatory {
		mandatoryString = " " + param.HelpString()
	}
	fmt.Printf("%s    "+color.YellowString(param.Name)+"%s%s\n", indentString, mandatoryString, defaultString)
}

type Task struct {
	Name        string      `json:"name" yaml:"name"`
	Group       string      `json:"group" yaml:"group"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Commands    []Command   `json:"commands" yaml:"commands"`
	Envs        []string    `json:"envs,omitempty" yaml:"envs,omitempty"`
	Params      []Parameter `json:"params,omitempty" yaml:"params,omitempty"`
	EnvsValues  map[string]string
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
	logFields := log.Fields{"task": task.Name}
	allowance := true
	parameterString := ""
	for _, param := range task.Params {
		allowed := true

		if param.Mandatory {
			allowed = configuration.HasFlag(param.Name)
		}

		if allowance && !allowed {
			parameterString = "parameter \"" + color.YellowString(param.Name) + "\" is mandatory"
			allowance = allowed
		}
		log.WithFields(logFields).Debugf("\"%v\" validated to: %v", param.HelpString(), allowed)
	}

	if allowance {
		log.WithFields(logFields).Debug("task running")
		for _, command := range task.Commands {
			// check params condition met
			command.Run(configuration)
		}
	} else {
		log.WithFields(logFields).Debug("task not running")
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
	Params      map[string]Parameter
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
		TaskGroups: groups,
	}
	contextData.UpdateData(environment.PopulateVariables(configFile.Envs))
	contextData.UpdateData(envsValues)

	configuration.ContextData = contextData
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
	fmt.Println("You can find more information on how to build and configure your own dispatch tasks, here:")
	fmt.Println("    TODO")
	fmt.Println("")

	// environments
	// TODO Rebuild environments
	// color.Yellow("Environments:\n")
	// environments := strings.Join(configuration.ConfigFile.Envs, ", ")
	// fmt.Printf("    %s\n\n", environments)

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
