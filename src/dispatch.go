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
			if len(taskNameSplit) == 1 {
				parsedParams[taskNameSplit[0]] = ""
			} else {
				if len(taskNameSplit) > 2 {
					panic("Invalid param")
				}
				parsedParams[taskNameSplit[0]] = taskNameSplit[1]
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
	configuredParamValues := map[string]models.ConfiguredParamValue{}

	for _, taskName := range tasksRequested {
		taskToRun, ok := configuration.Tasks[taskName]
		if !ok {
			fmt.Printf("Unkwown task %s!\n", taskName)
			return
		}

		for _, param := range taskToRun.Params {
			value, ok := parsedParams[param.Name]
			paramType := param.Type
			if !ok {
				value = param.Default
			}
			configuredParamValues[param.Name] = models.ConfiguredParamValue{Value: value, Type: paramType}
		}
	}

	// RUN TASKS
	for _, taskName := range tasksRequested {
		taskToRun := configuration.Tasks[taskName]
		_, helpBeingRequested := parsedParams["help"]
		if taskName == "help" {
			models.Help(configuration)
		} else {
			if helpBeingRequested {
				taskToRun.Help(0, true)
			} else {
				taskToRun.Run(configuration)
			}
		}
	}
}
