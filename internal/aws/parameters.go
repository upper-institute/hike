package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

var (
	GrpcServicePortParam = "GRPC_SERVICE_PORT"
	Http1ServerPortParam = "HTTP1_SERVER_PORT"
)

type Parameter struct {
	Config     aws.Config
	Path       string
	Parameters map[string]string
}

func (p *Parameter) GetStringValue(name string) string {

	key := strings.Join([]string{p.Path, name}, "/")

	value, ok := p.Parameters[key]

	if !ok {
		return ""
	}

	return value

}

func (s *Parameter) LoadParameters(ctx context.Context) error {

	s.Parameters = make(map[string]string)

	client := ssm.NewFromConfig(s.Config)

	getParametersByPathReq := ssm.NewGetParametersByPathPaginator(
		client,
		&ssm.GetParametersByPathInput{
			Path:           aws.String(s.Path),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
		},
	)

	for getParametersByPathReq.HasMorePages() {

		getParametersByPathPage, err := getParametersByPathReq.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, parameter := range getParametersByPathPage.Parameters {

			s.Parameters[aws.ToString(parameter.Name)] = aws.ToString(parameter.Value)

		}

	}

	return nil

}
