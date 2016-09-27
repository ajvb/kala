package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// This represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "cobrinha",
	Short: "A brief description of your command",
	Long:  `A loooooooonger description of your command.`,
	Run: func(cmd *cobra.Command, args []string) {

		log.Println("sdfsf")
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(ViperInit)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/config.yaml)")
}

func ViperInit() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(".")      // path to look for the config file in
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {
		log.Println("Failed to read config", err)
	}
}
