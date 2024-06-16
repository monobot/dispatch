package main

import (
	"fmt"
	"strings"
	"regexp"
	"os"

	"github.com/monobot/dispatch/src/discovery"
	"github.com/monobot/dispatch/src/models"
	// "golang.org/x/exp/slices"
	"github.com/monobot/dispatch/src/tasks"
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
			parsedParams[taskNameSplit[0]] = taskNameSplit[1]
		}
	}

	if len(tasksRequested) == 0 {
		tasksRequested = []string{"help"}
	}
	return tasksRequested, parsedParams
}
func main() {
	configuration := models.BuildConfiguration(discovery.TaskDiscovery())

	tasksRequested, parsedParams := parseCommandLineArgs()

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
			configuredParamValues[param.Name] = models.ConfiguredParamValue{Value:value, Type:paramType}
		}
	}

	// RUN TASKS
	for _, taskName := range tasksRequested {
		taskToRun := configuration.Tasks[taskName]
		if taskName == "help" {
			tasks.Help(configuration)
		} else {
			if true {
				taskToRun.Help()
			} else {
				for _, command := range taskToRun.Commands {
					fmt.Printf("%s\n",command.Command)
					fmt.Printf("%v\n",taskToRun.Params[0].Value)
				}
			}
		}
	}

}
