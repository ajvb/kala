package cmd

import (
	"fmt"
	"log"

	"github.com/ajvb/kala/job"
	"github.com/spf13/cobra"
)

var runCommandCmd = &cobra.Command{
	Use:   "run",
	Short: "run a job",
	Long:  `runs the specific job immediately and returns the result`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("Must include a command")
		} else if len(args) > 1 {
			log.Fatal("Must only include a command")
		}

		j := &job.Job{
			Command: args[0],
		}

		out, err := j.RunCmd()
		if err != nil {
			log.Fatalf("Command Failed with err: %s", err)
		} else {
			fmt.Println("Command Succeeded!")
			fmt.Println("Output: ", out)
		}
	},
}

func init() {
	RootCmd.AddCommand(runCommandCmd)
}
