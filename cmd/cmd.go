package cmd

import (
	"account-management/router"
	"account-management/service"

	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var RootCmd = &cobra.Command{
	Use:   "",
	Short: "Api server",
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Api server",
	Run: func(cmd *cobra.Command, args []string) {
		router.Start()
	},
}

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Task queue",
	Run: func(cmd *cobra.Command, args []string) {
		service.StartTaskQueue()
	},
}

func init() {

	RootCmd.AddCommand(apiCmd)
	RootCmd.AddCommand(queueCmd)
}