package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	sdtypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	cachetypes "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	parameterapi "github.com/upper-institute/ops-control/gen/api/parameter"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	parameter "github.com/upper-institute/ops-control/internal/parameter"
	sdinternal "github.com/upper-institute/ops-control/internal/service-discovery"
	"go.uber.org/zap"
)

type cloudMapServiceTag_ParameterPathProvider struct {
	v string
}

func (c *cloudMapServiceTag_ParameterPathProvider) FromTags(tagName string, tags []sdtypes.Tag) {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == tagName {
			c.v = aws.ToString(tag.Value)
			break
		}
	}
}

func (c *cloudMapServiceTag_ParameterPathProvider) GetParameterPath() string {
	return c.v
}

type CloudMapServiceDiscovery struct {
	namespacesNames  []string
	parameterPathTag string
	xdsClusterName   string

	cloudMapClient *servicediscovery.Client

	logger *zap.SugaredLogger

	parameterStore          parameter.ParameterStore
	parameterFileDownloader parameter.ParameterFileDownloader
}

func NewCloudMapServiceDiscovery(
	namespacesNames []string,
	parameterPathTag string,
	xdsClusterName string,
	cloudMapClient *servicediscovery.Client,
	logger *zap.SugaredLogger,
	parameterStore parameter.ParameterStore,
	parameterFileDownloader parameter.ParameterFileDownloader,
) sdinternal.ServiceDiscoveryService {

	return &CloudMapServiceDiscovery{
		namespacesNames:         namespacesNames,
		parameterPathTag:        parameterPathTag,
		xdsClusterName:          xdsClusterName,
		cloudMapClient:          cloudMapClient,
		logger:                  logger.With("xds_cluster", xdsClusterName, "namespaces_names", namespacesNames),
		parameterStore:          parameterStore,
		parameterFileDownloader: parameterFileDownloader,
	}

}

func (c *CloudMapServiceDiscovery) getListNamespacesInputFilters() []sdtypes.NamespaceFilter {

	c.logger.Debugw("Building NamespaceFilter to get namespaces IDs from namespaces names")

	namesFilter := sdtypes.NamespaceFilter{
		Name:      sdtypes.NamespaceFilterNameName,
		Values:    c.namespacesNames,
		Condition: sdtypes.FilterConditionEq,
	}

	return []sdtypes.NamespaceFilter{namesFilter}

}

func (c *CloudMapServiceDiscovery) getListServicesInputFilters(ctx context.Context) ([]sdtypes.ServiceFilter, error) {

	listNamespacesReq := servicediscovery.NewListNamespacesPaginator(
		c.cloudMapClient,
		&servicediscovery.ListNamespacesInput{
			Filters: c.getListNamespacesInputFilters(),
		},
	)

	namespaceIdsFilter := sdtypes.ServiceFilter{
		Name:      sdtypes.ServiceFilterNameNamespaceId,
		Values:    []string{},
		Condition: sdtypes.FilterConditionEq,
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

	return []sdtypes.ServiceFilter{namespaceIdsFilter}, nil

}

func (c *CloudMapServiceDiscovery) discoverService(ctx context.Context, sdState *sdinternal.ServiceDiscoveryState, service sdtypes.ServiceSummary) error {

	paramSet := parameter.NewParameterSet(c.parameterFileDownloader, c.logger)

	serviceName := aws.ToString(service.Name)

	c.logger.Debugw("Starting service discovery process (AWS Cloud Map)", "service_name", serviceName)

	listServiceTagsRes, err := c.cloudMapClient.ListTagsForResource(ctx, &servicediscovery.ListTagsForResourceInput{ResourceARN: service.Arn})
	if err != nil {
		return err
	}

	paramPathProvider := &cloudMapServiceTag_ParameterPathProvider{}
	paramPathProvider.FromTags(c.parameterPathTag, listServiceTagsRes.Tags)

	c.logger.Debugw("Load parameter path from service tag", "service_name", serviceName, "tag_key", c.parameterPathTag)

	err = c.parameterStore.Load(ctx, paramPathProvider, paramSet)
	if err != nil {
		return err
	}

	wnIngress := parameterapi.WellKnown_WELL_KNOWN_INGRESS.String()

	if paramSet.HasFile(wnIngress) {

		c.logger.Infow("Ingress configuration file found", "service_name", serviceName)

		addIngressInput := &service_discovery.Ingress{}

		err = paramSet.ParseProtoJson(ctx, wnIngress, addIngressInput)
		if err != nil {
			return err
		}

		addIngressInput.XdsClusterName = c.xdsClusterName

		sdState.AddIngress(addIngressInput)

		return nil

	}

	c.logger.Infow("Service cluster configuration file found", "service_name", serviceName)

	wnServiceCluster := parameterapi.WellKnown_WELL_KNOWN_SERVICE_CLUSTER.String()

	if !paramSet.HasFile(wnServiceCluster) {
		return nil
	}

	addServiceCluster := &service_discovery.ServiceCluster{}

	err = paramSet.ParseProtoJson(ctx, wnServiceCluster, addServiceCluster)
	if err != nil {
		return err
	}

	addServiceCluster.ServiceClusterName = serviceName
	addServiceCluster.XdsClusterName = c.xdsClusterName

	sdState.AddServiceCluster(addServiceCluster)

	listInstancesReq := servicediscovery.NewListInstancesPaginator(
		c.cloudMapClient,
		&servicediscovery.ListInstancesInput{
			ServiceId: service.Id,
		},
	)

	addServiceEndpointsInput := &service_discovery.ServiceEndpoints{
		ServiceClusterName: serviceName,
		Endpoints:          []*service_discovery.Endpoint{},
	}

	for listInstancesReq.HasMorePages() {

		listInstancesPage, err := listInstancesReq.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, instance := range listInstancesPage.Instances {

			address, ok := instance.Attributes["AWS_INSTANCE_IPV4"]
			if !ok {
				continue
			}

			c.logger.Infow("Add endpoint from AWS Cloud Map service instance", "service_name", serviceName, "instance_ipv4", address)

			addServiceEndpointsInput.Endpoints = append(
				addServiceEndpointsInput.Endpoints,
				&service_discovery.Endpoint{
					Protocol:  corev3.SocketAddress_TCP,
					Address:   address,
					PortValue: addServiceCluster.UpstreamPort,
				},
			)

		}

	}

	sdState.AddServiceEndpoints(addServiceEndpointsInput)

	return nil

}

func (c *CloudMapServiceDiscovery) Discover(ctx context.Context) (map[string][]cachetypes.Resource, error) {

	listServicesFilters, err := c.getListServicesInputFilters(ctx)
	if err != nil {
		return nil, err
	}

	listServicesReq := servicediscovery.NewListServicesPaginator(
		c.cloudMapClient,
		&servicediscovery.ListServicesInput{
			Filters: listServicesFilters,
		},
	)

	sdState := sdinternal.NewServiceDiscoveryState(c.logger)

	for listServicesReq.HasMorePages() {

		listServicesPage, err := listServicesReq.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, service := range listServicesPage.Services {

			err = c.discoverService(ctx, sdState, service)
			if err != nil {
				return nil, err
			}

		}

	}

	c.logger.Infow("Build envoy service discovery resources map")

	err = sdState.Build()
	if err != nil {
		return nil, err
	}

	return sdState.Resources, nil

}
