package main

import (
	"fmt"
	"os"
	"os/exec"
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
				taskToRun.Help()
			} else {
				for _, calculatedCommand := range taskToRun.CalculateCommands() {
					if len(calculatedCommand) > 0 {
						fmt.Printf("%v %v\n", calculatedCommand, len(calculatedCommand))
						fmt.Println(strings.Join(calculatedCommand, " "))
						baseCmd := calculatedCommand[0]
						cmdArgs := calculatedCommand[1:]

						cmd := exec.Command(baseCmd, cmdArgs...)
						_, err := cmd.Output()
						if err != nil {
							panic(err)
						}
					}
				}
			}
		}
	}

}
