package tasks

import (
	"fmt"
	"reflect"

	"github.com/fatih/color"
	"github.com/monobot/dispatch/src/models"
)

func PrintGroupTasks(groupTasks []string, configuration models.Configuration, nested bool) {
	indent := ""
	if nested {
		indent = "    "
	}

	for _, taskName := range groupTasks {
		task := configuration.Tasks[taskName]
		fmt.Printf("%s"+color.YellowString(task.Name)+":\n", indent)
		fmt.Printf("    %s%s\n", indent, task.Description)

	}
}

func Help(configuration models.Configuration) {
	// Print help message
	color.Yellow(("This is 'dispatch' help."))
	fmt.Println(("You can find more information on how to build and configure your own dispatch tasks, here:"))
	fmt.Println(("    TODO"))
	fmt.Println((""))

	indent := true
	if len(configuration.Groups) == 1 {
		indent = false
	}
	groupNames := reflect.ValueOf(configuration.Groups).MapKeys()
	for _, groupName := range groupNames {
		groupTasks := configuration.Groups[groupName.String()]
		if indent {
			color.Yellow("%s:\n", groupName)
		}
		PrintGroupTasks(groupTasks, configuration, indent)
	}
}
