package cmds

import (
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify configuration",
	Run: func(cmd *cobra.Command, args []string) {
		//configPath, _ := cmd.Flags().GetString("config")
		//section := parser.ParseFile(configPath)
	},
}

func init() {
	verifyCmd.PersistentFlags().String("config", "/etc/declarch/config.conf", "Configuration file (default is /etc/declarch/config.conf)")

	rootCmd.AddCommand(verifyCmd)
}
