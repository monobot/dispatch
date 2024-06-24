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

func (command Command) IsAllowed(configuration *Configuration) bool {
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
		}
	}

	if !allowance && configuration.HasFlag("verbose") {
		fmt.Println("    condition: " + failConditionString + " not met\n")
	}

	return allowance
}
func (command Command) Run(configuration *Configuration) (string, error) {
	allowance := command.IsAllowed(configuration)

	task, ok := configuration.Tasks[command.Command]
	if ok {
		return task.Run(configuration)
	}

	commandTemplate, err := template.New("commandTemplate").Option("missingkey=error").Parse(command.Command)
	if err != nil {
		message := " \"" + command.Command + "\", can not be parsed"
		return message, err
	}

	var outputBytes bytes.Buffer
	if err := commandTemplate.Execute(&outputBytes, configuration.ContextData.Data); err != nil {
		message := " \"" + command.Command + "\", not all arguments could be inferred"
		return message, err
	}

	runCommand := outputBytes.String()

	if allowance {
		isDryRun := configuration.HasFlag("dry-run")
		prefix := color.YellowString("running ")
		if isDryRun {
			prefix = color.CyanString("DRY-RUN ")
		}
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
	}

	return "", nil
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

	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Params     []Parameter `json:"params,omitempty" yaml:"params,omitempty"`
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

func (task Task) IsAllowed(configuration *Configuration) bool {
	allowance := true
	failConditionString := ""
	for _, condition := range task.Conditions {
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
		}
	}

	if !allowance && configuration.HasFlag("verbose") {
		fmt.Println("    condition: " + failConditionString + " not met\n")
	}

	return allowance
}
func (task Task) Run(configuration *Configuration) (string, error) {
	taskAllowed := task.IsAllowed(configuration)

	subcommandCount := 0
	subcommandFailedCount := 0
	if taskAllowed {
		subcommandCount += 1
		successfullyRun := true
		for _, command := range task.Commands {
			// check params condition met
			message, err := command.Run(configuration)
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

func getIndentString(nestCount int) string {
	indent := ""
	for i := 0; i < nestCount; i++ {
		indent += "    "
	}
	return indent
}
