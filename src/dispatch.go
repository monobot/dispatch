package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/monobot/dispatch/src/discovery"
	"github.com/monobot/dispatch/src/models"
)

func parseCommandLineArgs() ([]string, models.ConfigurationData, error) {
	configurationData := models.ConfigurationData{}
	tasksRequested := []string{}
	parsedParams := map[string]string{}
	flags := []string{}
	err := error(nil)
	for _, param := range os.Args[1:] {
		if !strings.HasPrefix(param, "-") {
			tasksRequested = append(tasksRequested, param)
		} else {
			// configuration params
			param = strings.TrimPrefix(param, "-")
			equal := regexp.MustCompile(`=`)
			taskNameSplit := equal.Split(param, -1)

			paramMap := map[string]string{
				"-h":       "help",
				"-help":    "help",
				"-v":       "verbose",
				"-verbose": "verbose",
				"-dry-run": "dry-run",
				"-dr":      "dry-run",
				"-dryrun":  "dry-run",
				"-dry":     "dry-run",
			}

			paramName, ok := paramMap[taskNameSplit[0]]
			if ok {
				flags = append(flags, paramName)
			} else {
				if string(param[0]) == "-" {
					return nil, configurationData, errors.New("Invalid param -" + param)
				}

				if len(taskNameSplit) > 1 {
					if len(taskNameSplit) > 2 {
						return nil, configurationData, errors.New(strings.Join(taskNameSplit, "="))
					}
					parsedParams[taskNameSplit[0]] = taskNameSplit[1]
				}
			}
		}
	}

	if len(tasksRequested) == 0 {
		tasksRequested = []string{"help"}
	}

	configurationData.ContextData = parsedParams
	configurationData.Flags = flags
	return tasksRequested, configurationData, err
}

func main() {
	tasksRequested, contextData, err := parseCommandLineArgs()
	if err != nil {
		fmt.Printf(color.RedString("Error!")+" parsing command line arguments \"%s\"", err)
		return
	}
	configuration := models.BuildConfiguration(discovery.TaskDiscovery(), contextData)

	totalCount := 0
	failedCount := 0
	// RUN TASKS
	for _, taskName := range tasksRequested {
		_, ok := configuration.Tasks[taskName]
		if !ok {
			fmt.Printf("unknown task %s!\n", taskName)
			return
		}
		totalCount += 1
		if taskName == "help" {
			models.Help(configuration)
		} else {
			if configuration.HasFlag("help") {
				configuration.Tasks[taskName].Help(0, true)
			} else {
				message, err := configuration.RunTask(taskName)
				if err != nil {
					failedCount += 1
					fmt.Printf(color.RedString(taskName) + " " + message + "\n")
				}
			}
		}
	}
	if failedCount > 0 {
		fmt.Printf("\n%v\\%v tasks "+color.RedString("failed\n"), failedCount, totalCount)
	}
}
