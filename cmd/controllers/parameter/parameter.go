package parameter

import (
	"github.com/spf13/cobra"
)

var (
	ParameterCmd = &cobra.Command{
		Use:   "parameter",
		Short: "Parameter management related commands",
	}
)

func init() {

	ParameterCmd.AddCommand(templateFileCmd)

}
