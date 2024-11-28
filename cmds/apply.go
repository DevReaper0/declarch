package cmds

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/DevReaper0/declarch/parser"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		section := parser.ParseFile(configPath)
		parserTester(section)
	},
}

func init() {
	applyCmd.PersistentFlags().StringP("config", "c", "/etc/declarch/config.conf", "Configuration file (default is /etc/declarch/config.conf)")

	rootCmd.AddCommand(applyCmd)
}

func parserTester(section *parser.Section) {
	fmt.Println(section.GetFirst("bakery/secrets/password", "!!!!"))
	fmt.Println(section.GetAll("bakery/secrets/password"))
	fmt.Println()
	fmt.Println(section.GetFirst("bakery/employees", "!!!!"))
	fmt.Println(section.GetAll("bakery/employees"))
	fmt.Println()
	fmt.Println(section.GetFirst("cakes/number", "!!!!"))
	fmt.Println(section.GetAll("cakes/number"))
	fmt.Println()
	fmt.Println(section.GetFirst("cakes/colors", "!!!!"))
	fmt.Println(section.GetAll("cakes/colors"))
	fmt.Println()
	fmt.Println(section.GetFirst("bakery/cakes/colors", "!!!!"))
	fmt.Println(section.GetAll("bakery/cakes/colors"))
	fmt.Println()
	fmt.Println(section.GetFirst("add_baker", "!!!!"))
	fmt.Println(section.GetAll("add_baker"))
}
