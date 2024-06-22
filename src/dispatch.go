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

func parseCommandLineArgs() ([]string, models.ContextData, error) {
	contextData := models.ContextData{}
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
				"-v":       "verbose",
				"-dry-run": "dry-run",
				"-dryrun":  "dry-run",
				"-dry":     "dry-run",
			}

			paramName, ok := paramMap[taskNameSplit[0]]
			if ok {
				flags = append(flags, paramName)
			} else {
				if string(param[0]) == "-" {
					return nil, contextData, errors.New("Invalid param -" + param)
				}

				if len(taskNameSplit) > 1 {
					if len(taskNameSplit) > 2 {
						return nil, contextData, errors.New(strings.Join(taskNameSplit, "="))
					}
					parsedParams[taskNameSplit[0]] = taskNameSplit[1]
				}
			}
		}
	}

	if len(tasksRequested) == 0 {
		tasksRequested = []string{"help"}
	}

	contextData.Data = parsedParams
	contextData.Flags = flags
	return tasksRequested, contextData, err
}

func main() {
	tasksRequested, contextData, err := parseCommandLineArgs()
	if err != nil {
		fmt.Printf(color.RedString("Error!")+" parsing command line arguments \"%s\"", err)
		return
	}
	fmt.Printf("contextData: %v\n", contextData)
	configuration := models.BuildConfiguration(discovery.TaskDiscovery(), contextData)

	// RUN TASKS
	for _, taskName := range tasksRequested {
		_, ok := configuration.Tasks[taskName]
		if !ok {
			fmt.Printf("unknown task %s!\n", taskName)
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
