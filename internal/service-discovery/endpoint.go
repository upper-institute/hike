package servicediscovery

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
)

func makeServiceEndpoints(input *service_discovery.ServiceEndpoints) (*endpointv3.ClusterLoadAssignment, error) {

	lbEndpoints := []*endpointv3.LbEndpoint{}

	for _, endpoint := range input.Endpoints {

		lbEndpoints = append(
			lbEndpoints,
			&endpointv3.LbEndpoint{
				HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
					Endpoint: &endpointv3.Endpoint{
						Address: &corev3.Address{
							Address: &corev3.Address_SocketAddress{
								SocketAddress: &corev3.SocketAddress{
									Protocol: endpoint.Protocol,
									Address:  endpoint.Address,
									PortSpecifier: &corev3.SocketAddress_PortValue{
										PortValue: endpoint.PortValue,
									},
								},
							},
						},
					},
				},
			},
		)

	}

	return &endpointv3.ClusterLoadAssignment{
		ClusterName: input.ServiceClusterName,
		Endpoints: []*endpointv3.LocalityLbEndpoints{{
			LbEndpoints: lbEndpoints,
		}},
	}, nil

}
