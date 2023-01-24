package awsdriver

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	servicediscoverytypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/upper-institute/hike/pkg/parameter"
	"github.com/upper-institute/hike/pkg/servicemesh"
	paramapi "github.com/upper-institute/hike/proto/api/parameter"
	sdapi "github.com/upper-institute/hike/proto/api/service-discovery"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

type cloudMapServiceDiscovery_operation struct {
	ctx context.Context

	serviceSummary servicediscoverytypes.ServiceSummary

	service *sdapi.Service

	logger *zap.SugaredLogger
}

type cloudMapServiceDiscovery struct {
	namespacesNames []string
	parameterUriTag string

	parameterSourceOptions *parameter.SourceOptions
	cloudMapClient         *servicediscovery.Client

	logger *zap.SugaredLogger
}

func NewCloudMapServiceDiscovery(
	namespacesNames []string,
	parameterUriTag string,
	parameterSourceOptions *parameter.SourceOptions,
	cloudMapClient *servicediscovery.Client,
	logger *zap.SugaredLogger,
) servicemesh.EnvoyDiscoveryService {
	return &cloudMapServiceDiscovery{
		namespacesNames,
		parameterUriTag,
		parameterSourceOptions,
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

func (c *cloudMapServiceDiscovery) getServiceParameters(op *cloudMapServiceDiscovery_operation) (*parameter.Source, error) {

	listServiceTagsRes, err := c.cloudMapClient.ListTagsForResource(
		op.ctx,
		&servicediscovery.ListTagsForResourceInput{ResourceARN: op.serviceSummary.Arn},
	)

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

	parameterCache, err := c.parameterSourceOptions.NewFromURLString(uriStr)
	if err != nil {
		return nil, err
	}

	err = parameterCache.Restore(op.ctx)
	if err != nil {
		return nil, err
	}

	return parameterCache, nil

}

func (c *cloudMapServiceDiscovery) discoverEndpoints(op *cloudMapServiceDiscovery_operation) error {

	listInstancesReq := servicediscovery.NewListInstancesPaginator(
		c.cloudMapClient,
		&servicediscovery.ListInstancesInput{
			ServiceId: op.serviceSummary.Id,
		},
	)

	lbEndpoints := []*endpointv3.LbEndpoint{}

	for listInstancesReq.HasMorePages() {

		listInstancesPage, err := listInstancesReq.NextPage(op.ctx)
		if err != nil {
			return err
		}

		for _, instance := range listInstancesPage.Instances {

			address, ok := instance.Attributes["AWS_INSTANCE_IPV4"]
			if !ok {
				op.logger.Infow("Instance without AWS_INSTANCE_IPV4 key")
				continue
			}

			lbEndpoints = append(
				lbEndpoints,
				&endpointv3.LbEndpoint{
					HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
						Endpoint: &endpointv3.Endpoint{
							Address: &corev3.Address{
								Address: &corev3.Address_SocketAddress{
									SocketAddress: &corev3.SocketAddress{
										Protocol: corev3.SocketAddress_TCP,
										Address:  address,
										PortSpecifier: &corev3.SocketAddress_PortValue{
											PortValue: op.service.ListenPort,
										},
									},
								},
							},
						},
					},
				},
			)

		}

	}

	for _, loadAssignment := range op.service.EnvoyEndpoints {
		loadAssignment.ClusterName = op.service.ServiceName
		loadAssignment.Endpoints = []*endpointv3.LocalityLbEndpoints{{
			LbEndpoints: lbEndpoints,
		}}
	}

	return nil

}

func (c *cloudMapServiceDiscovery) discoverService(op *cloudMapServiceDiscovery_operation) (*sdapi.Service, error) {

	op.logger.Debugw("Starting service discovery process (AWS Cloud Map)")

	parameterCache, err := c.getServiceParameters(op)
	if parameterCache == nil || err != nil {
		return nil, err
	}

	if !parameterCache.HasWellKnown(paramapi.WellKnown_WN_SERVICE_MESH_SERVICE) {
		op.logger.Warnw("No service mesh service parameter found")
		return nil, nil
	}

	op.logger.Infow("Service mesh service parameter found")

	param := parameterCache.GetWellKnown(paramapi.WellKnown_WN_SERVICE_MESH_SERVICE)

	if param.GetType() != paramapi.ParameterType_PT_FILE {
		op.logger.Errorw("Service mesh service parameter must be a file type")
		return nil, nil
	}

	op.logger.Debugw("Loading service mesh service parameter file")

	err = param.Load(op.ctx)
	if err != nil {
		return nil, err
	}

	op.service = &sdapi.Service{}

	op.logger.Debugw("Parsing service mesh service parameter file")

	err = protojson.Unmarshal(param.GetFile().Bytes(), op.service)
	if err != nil {
		return nil, err
	}

	op.service.ServiceId = aws.ToString(op.serviceSummary.Id)

	if len(op.service.ServiceName) == 0 {
		op.service.ServiceName = aws.ToString(op.serviceSummary.Name)
	}

	if op.service.EnvoyCluster != nil {

		op.logger.Debugw("Loading endpoints from Cloud Map")

		err = c.discoverEndpoints(op)
		if err != nil {
			return nil, err
		}

	}

	return op.service, nil

}

func (c *cloudMapServiceDiscovery) Discover(ctx context.Context, svcCh chan *sdapi.Service) {

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
			ctx: ctx,
		}

		for _, serviceSummary := range listServicesPage.Services {

			op.serviceSummary = serviceSummary
			op.logger = c.logger.With("service_name", aws.ToString(serviceSummary.Name))

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
