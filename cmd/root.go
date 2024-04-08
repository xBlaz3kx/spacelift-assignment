package cmd

import (
	"fmt"
	"github.com/spacelift-io/homework-object-storage/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "s3-gateway",
	Short: "S3 Gateway server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		logger := zap.L()
		logger.Info("Starting S3 gateway server")

		api.NewServer(logger, nil, ":3000")
	},
	Version: "0.0.1",
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.homework-object-storage.yaml)")

	rootCmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		zap.L().Fatal("Failed to execute command", zap.Error(err))
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".homework-object-storage" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".homework-object-storage")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
