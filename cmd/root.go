// Copyright © 2022

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"log"

	"github.com/hashicorp-services/tfe-migrate/version"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	// o       *output.Output

	// Required to leverage viper defaults for optional Flags
	bindPFlags = func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.Fatal(aurora.Red(err))
		}
	}
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tfe-migrate",
	Short: "A CLI to assist with TFE Migrations.",
	Long: `Migration of Terraform Enterprise.
	More words here.
	And maybe here.`,
	SilenceUsage:     true,
	SilenceErrors:    true,
	Version:          version.String(),
	PersistentPreRun: bindPFlags, // Bind here to avoid having to call this in every subcommand
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(aurora.Red(err))
	}

	// // Close output stream always before exiting
	// if err := rootCmd.Execute(); err != nil {
	// 	o.Close()
	// 	log.Fatal(aurora.Red(err))
	// } else {
	// 	o.Close()
	// }
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file, can be used to store common flags, (default is ./.tfe-migrate.hcl).")
	// rootCmd.PersistentFlags().String("source-hostname", "", "The source hostname. Can also be set with the environment variable SOURCE_HOSTNAME.")
	// rootCmd.PersistentFlags().String("source-organization", "", "The source Organization. Can also be set with the environment variable SOURCE_ORGANIZATION.")
	// rootCmd.PersistentFlags().String("source-token", "", "The source API token used to authenticate. Can also be set with the environment variable SOURCE_TOKEN.")

	// // required
	// rootCmd.MarkPersistentFlagRequired("source-hostname")
	// rootCmd.MarkPersistentFlagRequired("source-organization")
	// rootCmd.MarkPersistentFlagRequired("source-token")

	// // ENV aliases
	// viper.BindEnv("source-hostname", "SOURCE_HOSTNAME")
	// viper.BindEnv("source-organization", "SOURCE_ORGANIZATION")
	// viper.BindEnv("source-token", "SOURCE_TOKEN")

	// Turn off completion option
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in current & home directory with name ".tfe-migrate" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".tfe-migrate")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	isConfigFile := false
	if err := viper.ReadInConfig(); err == nil {
		isConfigFile = true // Capture information here to bring after all flags are loaded (namely which output type)
	}

	// Some hacking here to let viper use the cobra required flags, simplifies this checking
	// in one place rather than each command
	// More info: https://github.com/spf13/viper/issues/397
	postInitCommands(rootCmd.Commands())

	// // Initialize output
	// o = output.New(*viperBool("json"))

	// Print if config file was found
	if isConfigFile {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// copy.pasta function
func postInitCommands(commands []*cobra.Command) {
	for _, cmd := range commands {
		presetRequiredFlags(cmd)
		if cmd.HasSubCommands() {
			postInitCommands(cmd.Commands())
		}
	}
}

// copy.pasta function
func presetRequiredFlags(cmd *cobra.Command) {
	viper.BindPFlags(cmd.Flags())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			cmd.Flags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}
