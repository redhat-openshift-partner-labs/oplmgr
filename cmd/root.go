/*
Copyright © 2021 Melvin Hillsman <mrhillsman@redhat.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string
var ClusterId string
var Namespace string
var Company string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "oplmgr SUB-COMMAND",
	Short: "Used primarily by pg_timetable.",
	Long: `Program to manipulate provisioning, sleep, wake, and deletion of clusters
created using OpenShift Partner Labs. You will need the cluster_id and timezone
to successfully use this application:

oplmgr provision --cluster-id 177933cc-e47a-42fc-9ad9-f6efbb27f44a
oplmgr delete --cluster-id 177933cc-e47a-42fc-9ad9-f6efbb27f44a
oplmgr sleep --cluster-id 177933cc-e47a-42fc-9ad9-f6efbb27f44a
oplmgr wake --cluster-id 177933cc-e47a-42fc-9ad9-f6efbb27f44a

Provisioning happens at 7am of the timezone provided. Here are the timezones we set:

UTC-5    'America/Panama'    americas
UTC+1    'Africa/Algiers'    emea
UTC+7    'Asia/Jakarta'      apac`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		if err != nil {
			log.Printf("Unable to provide usage details: %v\n", err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Unable to run initial execute: %v\n", err)
		os.Exit(100)
	}
}

func init() {
	//cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.oplmgr.yaml)")
	rootCmd.PersistentFlags().StringVar(&ClusterId, "cluster-id", "", "id of cluster to interact with")
	rootCmd.PersistentFlags().StringVar(&Namespace, "namespace", "hive", "namespace to interact with")
	rootCmd.PersistentFlags().StringVar(&Company, "company", "opl", "company name provided by request form")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.MarkPersistentFlagRequired("cluster-id")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Unable to get user's home directory: %v\n", err)
		}

		// Search config in home directory with name ".oplmgr" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".oplmgr")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_, err = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
