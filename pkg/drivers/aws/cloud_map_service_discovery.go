package awsdriver

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	servicediscoverytypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	parameterapi "github.com/upper-institute/ops-control/gen/api/parameter"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"github.com/upper-institute/ops-control/pkg/parameter"
	"github.com/upper-institute/ops-control/pkg/servicemesh"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

type cloudMapServiceDiscovery_operation struct {
	ctx       context.Context
	resources *servicemesh.Resources
	service   servicediscoverytypes.ServiceSummary

	logger *zap.SugaredLogger
}

type cloudMapServiceDiscovery struct {
	namespacesNames []string
	parameterUriTag string

	parameterCacheOptions *parameter.CacheOptions
	cloudMapClient        *servicediscovery.Client

	logger *zap.SugaredLogger
}

func NewCloudMapServiceDiscovery(
	namespacesNames []string,
	parameterUriTag string,
	parameterCacheOptions *parameter.CacheOptions,
	cloudMapClient *servicediscovery.Client,
	logger *zap.SugaredLogger,
) servicemesh.EnvoyDiscoveryService {
	return &cloudMapServiceDiscovery{
		namespacesNames,
		parameterUriTag,
		parameterCacheOptions,
		cloudMapClient,
		logger,
	}
}

func (c *cloudMapServiceDiscovery) getListNamespacesInputFilters() []servicediscoverytypes.NamespaceFilter {

	c.logger.Debugw("Building NamespaceFilter to get namespaces IDs from namespaces names")

	namesFilter := servicediscoverytypes.NamespaceFilter{
		Name:      servicediscoverytypes.NamespaceFilterNameName,
		Values:    c.namespacesNames,
		Condition: servicediscoverytypes.FilterConditionEq,
	}

	return []servicediscoverytypes.NamespaceFilter{namesFilter}

}

func (c *cloudMapServiceDiscovery) getListServicesInputFilters(ctx context.Context) ([]servicediscoverytypes.ServiceFilter, error) {

	listNamespacesReq := servicediscovery.NewListNamespacesPaginator(
		c.cloudMapClient,
		&servicediscovery.ListNamespacesInput{
			Filters: c.getListNamespacesInputFilters(),
		},
	)

	namespaceIdsFilter := servicediscoverytypes.ServiceFilter{
		Name:      servicediscoverytypes.ServiceFilterNameNamespaceId,
		Values:    []string{},
		Condition: servicediscoverytypes.FilterConditionEq,
	}

	for listNamespacesReq.HasMorePages() {

		listNamespacesPage, err := listNamespacesReq.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, namespace := range listNamespacesPage.Namespaces {

			namespaceId := aws.ToString(namespace.Id)

			c.logger.Infow("Found namespace from AWS Cloud Map", "namespace_id", namespaceId)

			namespaceIdsFilter.Values = append(namespaceIdsFilter.Values, namespaceId)

		}

	}

	return []servicediscoverytypes.ServiceFilter{namespaceIdsFilter}, nil

}

func (c *cloudMapServiceDiscovery) getServiceParameters(op *cloudMapServiceDiscovery_operation) (*parameter.Cache, error) {

	listServiceTagsRes, err := c.cloudMapClient.ListTagsForResource(op.ctx, &servicediscovery.ListTagsForResourceInput{ResourceARN: op.service.Arn})

	if err != nil {
		return nil, err
	}

	uriStr := ""

	for _, tag := range listServiceTagsRes.Tags {
		if aws.ToString(tag.Key) == c.parameterUriTag {
			uriStr = aws.ToString(tag.Value)
			break
		}
	}

	op.logger.Debugw("Load parameter path from service tag", "tag_key", c.parameterUriTag, "parameter_path_value", uriStr)

	if len(uriStr) == 0 {
		op.logger.Infow("Ignoring service discovery 'case parameter_path is empty")
		return nil, nil
	}

	parameterCache, err := c.parameterCacheOptions.NewFromURLString(uriStr)
	if err != nil {
		return nil, err
	}

	err = parameterCache.Restore(op.ctx)
	if err != nil {
		return nil, err
	}

	return parameterCache, nil

}

func (c *cloudMapServiceDiscovery) discoverService(op *cloudMapServiceDiscovery_operation) (*service_discovery.Service, error) {

	op.logger.Debugw("Starting service discovery process (AWS Cloud Map)")

	parameterCache, err := c.getServiceParameters(op)
	if parameterCache == nil || err != nil {
		return nil, err
	}

	if !parameterCache.HasWellKnown(parameterapi.WellKnown_WN_SERVICE_MESH_SERVICE) {
		op.logger.Warnw("No service mesh service parameter found")
		return nil, nil
	}

	op.logger.Infow("Service mesh service parameter found")

	param := parameterCache.GetWellKnown(parameterapi.WellKnown_WN_SERVICE_MESH_SERVICE)

	if param.GetType() != parameterapi.ParameterType_PARAMETER_TYPE_FILE {
		op.logger.Errorw("Service mesh service parameter must be a file type")
		return nil, nil
	}

	op.logger.Debugw("Loading service mesh service parameter file")

	err = param.Load(op.ctx)
	if err != nil {
		return nil, err
	}

	svc := &service_discovery.Service{}

	op.logger.Debugw("Parsing service mesh service parameter file")

	err = protojson.Unmarshal(param.GetFile().Bytes(), svc)
	if err != nil {
		return nil, err
	}

	return svc, nil

}

func (c *cloudMapServiceDiscovery) Discover(ctx context.Context, svcCh chan *service_discovery.Service) {

	listServicesFilters, err := c.getListServicesInputFilters(ctx)
	if err != nil {
		c.logger.Error(err)
		return
	}

	listServicesReq := servicediscovery.NewListServicesPaginator(
		c.cloudMapClient,
		&servicediscovery.ListServicesInput{
			Filters: listServicesFilters,
		},
	)

	for listServicesReq.HasMorePages() {

		listServicesPage, err := listServicesReq.NextPage(ctx)
		if err != nil {
			c.logger.Error(err)
			return
		}

		op := &cloudMapServiceDiscovery_operation{
			ctx:       ctx,
			resources: servicemesh.NewResources(),
		}

		for _, service := range listServicesPage.Services {

			op.service = service
			op.logger = c.logger.With("service_name", aws.ToString(service.Name))

			svc, err := c.discoverService(op)
			if err != nil {
				c.logger.Error(err)
				continue
			}

			if svc != nil {
				op.logger.Info("Sending service through discovery channel")
				svcCh <- svc
			}

		}

	}

}
