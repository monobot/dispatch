package models

import (
	"fmt"
	"os"
	"os/exec"

	"bytes"
	"strings"
	"text/template"

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
	return "variable \"" + color.RedString(condition.Variable) + "\" equals \"" + color.RedString(condition.Value) + "\" then " + color.RedString(allowance)
}

func (condition Condition) Help(indentCount int) {
	indentString := getIndentString(indentCount)

	conditionString := condition.HelpString()
	fmt.Printf("%s    When "+conditionString+"\n", indentString)
}

type Executable interface {
	IsAllowed(configuration *Configuration) bool
	Run(configuration *Configuration) (string, error)
	Help(configuration *Configuration)
}

type Command struct {
	Command string `json:"command" yaml:"command"`

	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
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

func (command Command) IsAllowed(configuration *Configuration, contextData map[string]string) bool {
	allowance := true
	failConditionString := ""
	for _, condition := range command.Conditions {
		contextValue := configuration.ConfigurationData.ContextData[condition.Variable]
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

			if !allowance && configuration.HasFlag("verbose") {
				if condition.Allowance {
					fmt.Println("    condition: " + failConditionString + " not met\n")
				} else {
					fmt.Println("    condition: " + failConditionString + " applied\n")
				}
			}
		}
	}

	return allowance
}

type Parameter struct {
	Name    string `json:"name" yaml:"name"`
	Default string `json:"default" yaml:"default"`
}

func (param Parameter) HelpString() string {
	mandatoryString := "is not"
	return mandatoryString + color.YellowString(" mandatory")
}

func (param Parameter) Help(indentCount int) {
	indentString := getIndentString(indentCount)

	defaultString := ""
	if param.Default != "" {
		defaultString = " default: " + color.YellowString(param.Default)
	}

	fmt.Printf("%s    "+color.YellowString(param.Name)+"%s%s\n", indentString, defaultString)
}

type Task struct {
	Name        string `json:"name" yaml:"name"`
	Group       string `json:"group" yaml:"group"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Hidden      bool   `json:"hidden" yaml:"hidden"`

	Commands []Command `json:"commands" yaml:"commands"`
	Async    bool      `json:"async" yaml:"async"`

	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Params     []Parameter `json:"params,omitempty" yaml:"params,omitempty"`

	ContextData map[string]string
}

func (task *Task) SetContextData(configuration *Configuration) {
	contextData := configuration.ConfigurationData.ContextData
	if contextData == nil {
		contextData = make(map[string]string)
	}

	for _, param := range task.Params {
		currentValue, ok := contextData[param.Name]
		if !ok || currentValue == "" {
			contextData[param.Name] = param.Default
		}
	}

	task.ContextData = contextData
}

func (task Task) Help(indentCount int, detailed bool) {
	indentString := getIndentString(indentCount)

	if !task.Hidden {
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

			if len(task.Params) > 0 {
				color.Cyan("%s    Params:\n", indentString)
				for _, param := range task.Params {
					param.Help(indentCount + 1)
				}
			}
		}
		fmt.Println("")
	}
}

func (task *Task) IsAllowed(configuration *Configuration) bool {
	allowance := true
	failConditionString := ""
	for _, condition := range task.Conditions {
		contextValue := task.ContextData[condition.Variable]
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

			if !allowance && configuration.HasFlag("verbose") {
				if condition.Allowance {
					fmt.Println("    condition: " + failConditionString + " not met\n")
				} else {
					fmt.Println("    condition: " + failConditionString + " applied\n")
				}
			}
		}
	}

	return allowance
}

func (task Task) RunCommand(index int, configuration *Configuration, contextData map[string]string) (string, error) {
	command := task.Commands[index]

	allowance := command.IsAllowed(configuration, task.ContextData)

	taskPrefix := "TASK:"
	if strings.HasPrefix(command.Command, taskPrefix) {
		trimmedCommand := strings.TrimPrefix(command.Command, taskPrefix)
		_, ok := configuration.Tasks[trimmedCommand]
		if ok {
			return configuration.RunTask(trimmedCommand)
		}
	}

	commandTemplate, err := template.New("commandTemplate").Option("missingkey=error").Parse(command.Command)
	if err != nil {
		message := " \"" + command.Command + "\", can not be parsed"
		return message, err
	}

	var outputBytes bytes.Buffer
	if err := commandTemplate.Execute(&outputBytes, contextData); err != nil {
		message := " \"" + command.Command + "\", not all arguments could be inferred"
		return message, err
	}

	runCommand := outputBytes.String()

	if allowance {
		isDryRun := configuration.HasFlag("dry-run")
		prefix := color.YellowString("running ")
		fmt.Printf(prefix + "\"" + runCommand + "\"\n")

		if !isDryRun {
			splitCommand := strings.Fields(runCommand)

			if len(splitCommand) > 0 {
				baseCmd := splitCommand[0]
				cmdArgs := splitCommand[1:]

				command := exec.Command(baseCmd, cmdArgs...)

				command.Stdout = os.Stdout
				command.Stderr = os.Stderr

				err := command.Run()

				if err != nil {
					return color.YellowString("could not run command ") + " %v", err
				}
			}
		}
	} else {
		prefix := color.YellowString("skipping ")
		fmt.Printf(prefix + "\"" + runCommand + "\"\n")
	}

	return "", nil
}

type ConfigFile struct {
	Envs     []string `json:"envs" yaml:"envs"`
	EnvFiles []string `json:"env_files" yaml:"env_files"`
	Tasks    []Task   `json:"tasks" yaml:"tasks"`
}

func (configFile *ConfigFile) TaskNames() []string {
	taskNames := []string{}
	for _, task := range configFile.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	return taskNames
}

func (configFile *ConfigFile) Combine(newConfigFile ConfigFile) ConfigFile {
	combinedEnvFiles := append(configFile.EnvFiles, newConfigFile.EnvFiles...)

	combinedEnvs := configFile.Envs
	for _, env := range newConfigFile.Envs {
		if !slices.Contains(combinedEnvs, env) {
			combinedEnvs = append(combinedEnvs, env)
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
		Envs:     combinedEnvs,
		EnvFiles: combinedEnvFiles,
		Tasks:    newTasks,
	}
}

func getIndentString(nestCount int) string {
	indent := ""
	for i := 0; i < nestCount; i++ {
		indent += "    "
	}
	return indent
}
