package cmd_debug

import (
	"easyrice/utils"
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "debug",
	Short: "debug",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Log File : %s\n", utils.LogFile)
	},
}
