package version

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const (
	// Version is injected on release
	Version = ""
)

func NewVersionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}

	return cmd
}
