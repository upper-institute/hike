package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/upper-institute/ops-control/internal/parameter"
	"go.uber.org/zap"
)

const SsmParameterPathSeparator = "/"

type ssmParameterStore struct {
	ssmClient *ssm.Client

	logger *zap.SugaredLogger
}

func NewSSMParameterStore(
	ssmClient *ssm.Client,
	logger *zap.SugaredLogger,
) parameter.ParameterStore {
	return &ssmParameterStore{
		ssmClient: ssmClient,
		logger:    logger,
	}
}

func (s *ssmParameterStore) Load(ctx context.Context, pathProvider parameter.ParameterPathProvider, paramSet *parameter.ParameterSet) error {

	paramPath := pathProvider.GetParameterPath()

	getParametersByPathReq := ssm.NewGetParametersByPathPaginator(
		s.ssmClient,
		&ssm.GetParametersByPathInput{
			Path:           aws.String(paramPath),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
		},
	)

	for getParametersByPathReq.HasMorePages() {

		getParametersByPathPage, err := getParametersByPathReq.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, param := range getParametersByPathPage.Parameters {

			key := aws.ToString(param.Name)
			value := aws.ToString(param.Value)

			sep := strings.LastIndex(key, SsmParameterPathSeparator)

			if sep > -1 {
				key = key[sep+1:]
			}

			err := paramSet.Add(key, value)
			if err != nil {
				return err
			}

		}

	}

	return nil

}
