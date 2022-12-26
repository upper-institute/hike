package parameter

import (
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

type TemplateInput struct {
	Env map[string]string
}

func (t *TemplateInput) LoadEnvs() {

	envs := os.Environ()
	t.Env = make(map[string]string)

	for _, env := range envs {

		sep := strings.IndexRune(env, '=')

		t.Env[env[:sep]] = env[sep+1:]

	}

}

var (
	templateFileCmd = &cobra.Command{
		Use:   "template-file",
		Short: "Produce output from template file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {

			source, err := template.ParseFiles(args[0])
			if err != nil {
				return err
			}

			rendered, err := os.Create(args[1])
			if err != nil {
				return err
			}

			defer rendered.Close()

			os.Environ()

			input := &TemplateInput{}
			input.LoadEnvs()

			return source.Execute(rendered, input)
		},
	}
)
