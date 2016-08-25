package cmd

import (
	"fmt"
	"log"
	"os"

	"flag"

	"github.com/ajvb/kala/job"
	"github.com/spf13/cobra"
)

var RunCommandCmd = &cobra.Command{
	Use:   "run_command",
	Short: "A brief description of your command",
	Long:  `A loooooooonger description of your command.`,
	Run: func(cmd *cobra.Command, args []string) {

		flag.Parse()
		log.Println(os.Args[2])

		if len(os.Args) < 2 {
			log.Fatal("Must include a command")
		} else if len(os.Args) > 3 {
			log.Fatal("Must only include a command")
		}

		j := &job.Job{
			Command: os.Args[2],
		}

		err := j.RunCmd()
		if err != nil {
			log.Fatalf("Command Failed with err: %s", err)
		} else {
			fmt.Println("Command Succeeded!")
		}
	},
}

func init() {

	RootCmd.AddCommand(RunCommandCmd)

}
