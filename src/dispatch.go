package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/monobot/dispatch/src/discovery"
	"github.com/monobot/dispatch/src/models"
)

func parseCommandLineArgs() ([]string, models.ContextData) {
	tasksRequested := []string{}
	parsedParams := map[string]string{}
	flags := []string{}
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
				flags = append(flags, "help")
			}

			if paramName == "v" {
				flags = append(flags, "verbose")
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
	contextData := models.ContextData{Data: parsedParams, Flags: flags}
	return tasksRequested, contextData
}

func main() {
	tasksRequested, contextData := parseCommandLineArgs()
	configuration := models.BuildConfiguration(discovery.TaskDiscovery(), contextData)

	// RUN TASKS
	for _, taskName := range tasksRequested {
		_, ok := configuration.Tasks[taskName]
		if !ok {
			fmt.Printf("Unkwown task %s!\n", taskName)
			return
		}
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
