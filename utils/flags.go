package utils

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func CreateStringFlag(cmd *cobra.Command, description, name, short, thisdefault string) {
	envname := strings.ToUpper(name)
	viper.BindEnv(envname)
	cmd.Flags().StringP(name, short, thisdefault, description)
	viper.BindPFlag(envname, cmd.Flags().Lookup(name))
}

func CreateIntFlag(cmd *cobra.Command, description, name, short string, thisdefault int) {
	envname := strings.ToUpper(name)
	viper.BindEnv(envname)
	cmd.Flags().IntP(name, short, thisdefault, description)
	viper.BindPFlag(envname, cmd.Flags().Lookup(name))
}

func CreateShortBoolFlag(cmd *cobra.Command, description, name, short string, thisdefault bool) {

	envname := strings.ToUpper(name)
	viper.BindEnv(envname)
	cmd.Flags().BoolP(name, short, thisdefault, description)
	viper.BindPFlag(envname, cmd.Flags().Lookup(name))
}
