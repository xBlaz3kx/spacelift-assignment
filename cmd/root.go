package cmd

import (
	"context"
	"fmt"
	docker "github.com/docker/docker/client"
	"github.com/spacelift-io/homework-object-storage/internal/api"
	"github.com/spacelift-io/homework-object-storage/internal/discovery"
	"github.com/spacelift-io/homework-object-storage/internal/gateway"
	"github.com/spacelift-io/homework-object-storage/internal/pkg/observability"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"os/signal"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "s3-gateway",
	Short: "S3 Gateway server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, end := signal.NotifyContext(context.Background(), os.Interrupt)
		defer end()

		logger := zap.L()
		logger.Info("Starting S3 gateway server")

		// Connect to the Docker daemon
		dockerClient, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
		if err != nil {
			logger.Fatal("Failed to create Docker client", zap.Error(err))
		}

		discoveryService := discovery.NewServiceV1(dockerClient)
		gatewayService := gateway.NewServiceV1(discoveryService)

		httpServer := api.NewServer(logger, gatewayService)
		httpServer.Run(":3000")

		<-ctx.Done()
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

	observability.NewLogger("debug")
}
