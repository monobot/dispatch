package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/monobot/dispatch/src/discovery"
	"github.com/monobot/dispatch/src/models"
	"github.com/monobot/dispatch/src/tasks"
	"golang.org/x/exp/slices"
)

func main() {
	configuration := models.BuildConfiguration(discovery.TaskDiscovery())
	tasksRequested := os.Args[1:]
	if len(tasksRequested) == 0 {
		tasksRequested = []string{"help"}
	}

	for _, taskName := range tasksRequested {
		fmt.Printf("%s\n", taskName)
	}
	if slices.Contains(tasksRequested, "help") {
		tasks.Help(configuration)
	}

	flag.Bool("h", false, "Show task help")
	for _, param := range configuration.Params {
		switch param.Type {
		case "string":
			{
				flag.String(param.Name, param.Default, param.Description)
			}
		default:
			{
				flag.String(param.Name, param.Default, param.Description)
			}

		}
	}

	fmt.Printf("%+v\n", configuration.Envs)
	fmt.Printf("%+v\n", configuration.Params)

	flag.Parse()
	fmt.Println("tail:", flag.Args())
}
