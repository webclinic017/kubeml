package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	controllerClient "github.com/diegostock12/thesis/ml/pkg/controller/client"
)

var (
	 taskId string
	 outputFile string


	 historyCmd = &cobra.Command{
	 	Use: "history",
	 	Short: "Check training history for task",
	 	RunE: getHistory,
	 }
)

// getHistory gets a training history based on the taskId and pretty
// prints it for easy reference
func getHistory(_ *cobra.Command, _ []string) error {
	controller := controllerClient.MakeClient()

	history, err := controller.GetHistory(taskId)
	if err != nil {
		return err
	}
	fmt.Println(history)
	return nil
}

func init()  {
	rootCmd.AddCommand(historyCmd)

	historyCmd.Flags().StringVar(&taskId, "network", "", "Id of the train task (required)" )
	historyCmd.Flags().StringVarP(&outputFile, "outputFile", "o", "", "Output file to save the results")

	historyCmd.MarkFlagRequired("network")
}

