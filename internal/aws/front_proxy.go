package aws

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	httpv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/upper-institute/ops-control/internal/envoy"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type FrontProxy struct {
	*envoy.GenericConfiguration

	Config aws.Config

	NamespacesNames          []string
	MatchNamespaceNamePrefix bool
	XdsClusterName           string
}

func (f *FrontProxy) getListNamespacesInputFilters() []types.NamespaceFilter {

	condition := types.FilterConditionEq

	if f.MatchNamespaceNamePrefix {
		condition = types.FilterConditionBeginsWith
	}

	namesFilter := types.NamespaceFilter{
		Name:      types.NamespaceFilterNameName,
		Values:    []string{},
		Condition: condition,
	}

	namesFilter.Values = append(namesFilter.Values, f.NamespacesNames...)

	return []types.NamespaceFilter{namesFilter}

}

func (f *FrontProxy) getListServicesInputFilters(ctx context.Context, client *servicediscovery.Client) ([]types.ServiceFilter, error) {

	listNamespacesReq := servicediscovery.NewListNamespacesPaginator(
		client,
		&servicediscovery.ListNamespacesInput{
			Filters: f.getListNamespacesInputFilters(),
		},
	)

	namespaceIdsFilter := types.ServiceFilter{
		Name:      types.ServiceFilterNameNamespaceId,
		Values:    []string{},
		Condition: types.FilterConditionEq,
	}

	for listNamespacesReq.HasMorePages() {

		listNamespacesPage, err := listNamespacesReq.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, namespace := range listNamespacesPage.Namespaces {
			namespaceIdsFilter.Values = append(namespaceIdsFilter.Values, aws.ToString(namespace.Id))
		}

	}

	return []types.ServiceFilter{namespaceIdsFilter}, nil

}

func (f *FrontProxy) addGrpcServiceCluster(service types.ServiceSummary) {

	f.Resources[resource.ClusterType] = append(
		f.Resources[resource.ClusterType],
		&clusterv3.Cluster{
			Name:                 aws.ToString(service.Id),
			ConnectTimeout:       durationpb.New(15 * time.Second),
			ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_EDS},
			LbPolicy:             clusterv3.Cluster_ROUND_ROBIN,
			DnsLookupFamily:      clusterv3.Cluster_V4_ONLY,
			Http2ProtocolOptions: &corev3.Http2ProtocolOptions{
				MaxConcurrentStreams: wrapperspb.UInt32(2147483647),
			},
			EdsClusterConfig: &clusterv3.Cluster_EdsClusterConfig{
				ServiceName: aws.ToString(service.Name),
				EdsConfig: &corev3.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
						ApiConfigSource: &corev3.ApiConfigSource{
							TransportApiVersion:       resource.DefaultAPIVersion,
							ApiType:                   corev3.ApiConfigSource_GRPC,
							SetNodeOnFirstMessageOnly: true,
							GrpcServices: []*corev3.GrpcService{{
								TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{ClusterName: f.XdsClusterName},
								},
							}},
						},
					},
				},
			},
		},
	)

}

func (f *FrontProxy) addHttp1ServiceCluster(service types.ServiceSummary) {

	protocolOptions, err := anypb.New(&httpv3.HttpProtocolOptions{
		UpstreamProtocolOptions: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_HttpProtocolOptions{
					HttpProtocolOptions: &corev3.Http1ProtocolOptions{},
				},
			},
		},
	})

	if err != nil {
		panic(err)
	}

	f.Resources[resource.ClusterType] = append(
		f.Resources[resource.ClusterType],
		&clusterv3.Cluster{
			Name:                 aws.ToString(service.Id),
			ConnectTimeout:       durationpb.New(15 * time.Second),
			ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_EDS},
			LbPolicy:             clusterv3.Cluster_ROUND_ROBIN,
			DnsLookupFamily:      clusterv3.Cluster_V4_ONLY,
			TypedExtensionProtocolOptions: map[string]*anypb.Any{
				"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": protocolOptions,
			},
			EdsClusterConfig: &clusterv3.Cluster_EdsClusterConfig{
				ServiceName: aws.ToString(service.Name),
				EdsConfig: &corev3.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
						ApiConfigSource: &corev3.ApiConfigSource{
							TransportApiVersion:       resource.DefaultAPIVersion,
							ApiType:                   corev3.ApiConfigSource_GRPC,
							SetNodeOnFirstMessageOnly: true,
							GrpcServices: []*corev3.GrpcService{{
								TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{ClusterName: f.XdsClusterName},
								},
							}},
						},
					},
				},
			},
		},
	)

}

func (f *FrontProxy) discoverService(ctx context.Context, client *servicediscovery.Client, service types.ServiceSummary) error {

	listServiceTagsRes, err := client.ListTagsForResource(ctx, &servicediscovery.ListTagsForResourceInput{ResourceARN: service.Arn})
	if err != nil {
		return err
	}

	serviceTags := NewTagsFromTagList(listServiceTagsRes.Tags)

	parameter := &Parameter{
		Config: f.Config,
		Path:   serviceTags.ConfigurationPath,
	}

	err = parameter.LoadParameters(ctx)
	if err != nil {
		return err
	}

	servicePort := uint32(0)
	servicePortParam := ""

	switch {

	case serviceTags.IsApplication(ApplicationTag_GrpcService):

		f.addGrpcServiceCluster(service)

		servicePortParam = parameter.GetStringValue(GrpcServicePortParam)

	case serviceTags.IsApplication(ApplicationTag_Http1Server):

		f.addHttp1ServiceCluster(service)

		servicePortParam = parameter.GetStringValue(Http1ServerPortParam)

	}

	if len(servicePortParam) > 0 {

		servicePortInt, err := strconv.Atoi(servicePortParam)

		if err != nil {
			return err
		}

		servicePort = uint32(servicePortInt)
	}

	listInstancesReq := servicediscovery.NewListInstancesPaginator(
		client,
		&servicediscovery.ListInstancesInput{
			ServiceId: service.Id,
		},
	)

	clusterName := aws.ToString(service.Id)

	var lbEndpoints []*endpointv3.LbEndpoint

	for listInstancesReq.HasMorePages() {

		listInstancesPage, err := listInstancesReq.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, instance := range listInstancesPage.Instances {

			lbEndpoint := &endpointv3.LbEndpoint{}

			switch {

			case serviceTags.IsApplication(ApplicationTag_GrpcService), serviceTags.IsApplication(ApplicationTag_Http1Server):

				address, ok := instance.Attributes["AWS_INSTANCE_IPV4"]
				if !ok {
					break
				}

				lbEndpoint.HostIdentifier = &endpointv3.LbEndpoint_Endpoint{
					Endpoint: &endpointv3.Endpoint{
						Address: &corev3.Address{
							Address: &corev3.Address_SocketAddress{
								SocketAddress: &corev3.SocketAddress{
									Protocol: corev3.SocketAddress_TCP,
									Address:  address,
									PortSpecifier: &corev3.SocketAddress_PortValue{
										PortValue: servicePort,
									},
								},
							},
						},
					},
				}

			}

			lbEndpoints = append(lbEndpoints, lbEndpoint)

		}

	}

	if len(lbEndpoints) > 0 {
		f.Resources[resource.EndpointType] = append(
			f.Resources[resource.EndpointType],
			&endpointv3.ClusterLoadAssignment{
				ClusterName: clusterName,
				Endpoints: []*endpointv3.LocalityLbEndpoints{{
					LbEndpoints: lbEndpoints,
				}},
			},
		)
	}

	return nil

}

func (f *FrontProxy) LoadConfigurationFromCloudMap(ctx context.Context) error {

	client := servicediscovery.NewFromConfig(f.Config)

	listServicesFilters, err := f.getListServicesInputFilters(ctx, client)

	if err != nil {
		return err
	}

	listServicesReq := servicediscovery.NewListServicesPaginator(
		client,
		&servicediscovery.ListServicesInput{
			Filters: listServicesFilters,
		},
	)

	for listServicesReq.HasMorePages() {

		listServicesPage, err := listServicesReq.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, service := range listServicesPage.Services {

			err = f.discoverService(ctx, client, service)
			if err != nil {
				return err
			}

		}

	}

	return nil

}
