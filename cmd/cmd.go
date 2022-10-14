package cmd

import (
	"account-management/router"
	"account-management/service"

	"github.com/spf13/cobra"
)

var messageChannels = []string{"request"}

// serveCmd represents the serve command
var RootCmd = &cobra.Command{
	Use:   "",
	Short: "Api server",
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Api server",
	Run: func(cmd *cobra.Command, args []string) {
		api := router.InitAPIServer(messageChannels)
		api.Start()
	},
}

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Task queue",
	Run: func(cmd *cobra.Command, args []string) {
		useWorker, _ := cmd.Flags().GetBool("useWorker")
		numWorkers, _ := cmd.Flags().GetInt("numWorker")

		taskQueue := service.NewTaskQueue(useWorker, numWorkers, messageChannels)
		taskQueue.Start()
	},
}

func init() {

	queueCmd.Flags().Bool("useWorker", false, "use workers for concurrent processing")
	queueCmd.Flags().Int("numWorker", 1, "number of workers for concurrent processing")
	RootCmd.AddCommand(apiCmd)
	RootCmd.AddCommand(queueCmd)
}
