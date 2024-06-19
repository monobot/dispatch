package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/monobot/dispatch/src/discovery"
	"github.com/monobot/dispatch/src/models"
)

func parseCommandLineArgs() ([]string, map[string]string) {
	tasksRequested := []string{}
	parsedParams := map[string]string{}
	for _, param := range os.Args[1:] {
		if !strings.HasPrefix(param, "-") {
			tasksRequested = append(tasksRequested, param)
		} else {
			// configuration params
			param = strings.TrimPrefix(param, "-")
			equal := regexp.MustCompile(`=`)
			taskNameSplit := equal.Split(param, -1)

			paramName := taskNameSplit[0]
			if paramName == "h" {
				paramName = "help"
			}

			if paramName == "v" {
				paramName = "verbose"
			}

			if len(taskNameSplit) == 1 {
				parsedParams[paramName] = ""
			} else {
				if len(taskNameSplit) > 2 {
					panic("Invalid param")
				}
				parsedParams[paramName] = taskNameSplit[1]
			}
		}
	}

	if len(tasksRequested) == 0 {
		tasksRequested = []string{"help"}
	}
	return tasksRequested, parsedParams
}

func main() {
	tasksRequested, parsedParams := parseCommandLineArgs()
	configuration := models.BuildConfiguration(discovery.TaskDiscovery(), parsedParams)

	// COLLECT VALUES FOR ALL THE PARAMS
	configuredParamValues := map[string]string{}

	for _, taskName := range tasksRequested {
		taskToRun, ok := configuration.Tasks[taskName]
		if !ok {
			fmt.Printf("Unkwown task %s!\n", taskName)
			return
		}

		for _, param := range taskToRun.Params {
			value, ok := parsedParams[param.Name]
			if !ok {
				value = param.Default
			}
			configuredParamValues[param.Name] = value
		}
	}

	configuration.UpdateContextData(configuredParamValues)

	// RUN TASKS
	for _, taskName := range tasksRequested {
		taskToRun := configuration.Tasks[taskName]

		if taskName == "help" {
			models.Help(configuration)
		} else {
			if configuration.HasFlag("help") {
				taskToRun.Help(0, true)
			} else {
				taskToRun.Run(configuration)
			}
		}
	}
}
