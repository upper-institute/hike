package awsdriver

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/upper-institute/ops-control/pkg/parameter"
	"go.uber.org/zap"
)

const (
	SSMParameterPathSeparator   = "/"
	SSMParameterPathPrefixQuery = "ssm_parameter_path_prefix"
)

type ssmParameterStore struct {
	ssmClient *ssm.Client

	logger *zap.SugaredLogger
}

func NewSSMParameterStore(
	ssmClient *ssm.Client,
	logger *zap.SugaredLogger,
) parameter.Store {
	return &ssmParameterStore{
		ssmClient: ssmClient,
		logger:    logger.With("driver", "aws_ssm_parameter_store"),
	}
}

func (s *ssmParameterStore) Pull(ctx context.Context, options *parameter.PullRequest) error {

	getParametersByPathReq := ssm.NewGetParametersByPathPaginator(
		s.ssmClient,
		&ssm.GetParametersByPathInput{
			Path:           aws.String(options.Url.Path),
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

			name := aws.ToString(param.Name)
			value := aws.ToString(param.Value)

			sep := strings.LastIndex(name, SSMParameterPathSeparator)

			pathPrefix := name[:sep]
			key := name[sep+1:]

			s.logger.Infow("Pull operation", "key", key, "path_prefix", pathPrefix)

			param, err := options.NewFromURLString(key, value)
			if err != nil {
				return err
			}

			param.Metadata.Set(SSMParameterPathPrefixQuery, pathPrefix)

			options.Result <- param

		}

	}

	return nil

}

func (s *ssmParameterStore) Put(ctx context.Context, param *parameter.Parameter) error {

	pathPrefix := param.Metadata.Get(SSMParameterPathPrefixQuery)

	s.logger.Infow("Put operation", "path_prefix", pathPrefix)

	_, err := s.ssmClient.PutParameter(ctx, &ssm.PutParameterInput{
		Name:  aws.String(fmt.Sprintf("%s%s", pathPrefix, param.GetKey())),
		Value: aws.String(param.GetURLString()),
	})

	return err

}
