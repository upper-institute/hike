package servicediscovery

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"go.uber.org/zap"
)

type ServiceDiscoveryState struct {
	logger *zap.SugaredLogger

	serviceClusters  []*service_discovery.ServiceCluster
	serviceEndpoints []*service_discovery.ServiceEndpoints
	ingresses        []*service_discovery.Ingress
	virtualHostsMap  map[service_discovery.IngressType]virtualHosts

	Resources map[string][]types.Resource
}

func NewServiceDiscoveryState(logger *zap.SugaredLogger) *ServiceDiscoveryState {
	return &ServiceDiscoveryState{
		logger: logger,

		serviceClusters:  make([]*service_discovery.ServiceCluster, 0),
		serviceEndpoints: make([]*service_discovery.ServiceEndpoints, 0),
		ingresses:        make([]*service_discovery.Ingress, 0),
		virtualHostsMap:  make(map[service_discovery.IngressType]virtualHosts),

		Resources: map[string][]types.Resource{
			resource.EndpointType: {},
			resource.ClusterType:  {},
			resource.SecretType:   {},
			resource.RouteType:    {},
			resource.ListenerType: {},
			resource.RuntimeType:  {},
		},
	}
}

func (s *ServiceDiscoveryState) AddServiceCluster(input *service_discovery.ServiceCluster) {

	s.serviceClusters = append(s.serviceClusters, input)

}

func (s *ServiceDiscoveryState) AddServiceEndpoints(input *service_discovery.ServiceEndpoints) {

	s.serviceEndpoints = append(s.serviceEndpoints, input)

}

func (s *ServiceDiscoveryState) AddIngress(input *service_discovery.Ingress) {

	s.ingresses = append(s.ingresses, input)

}

func (s *ServiceDiscoveryState) buildVirtualHostForCluster(serviceClusterInput *service_discovery.ServiceCluster) error {

	for _, ingressInput := range s.ingresses {

		if ingressInput.IngressType != serviceClusterInput.IngressType {
			continue
		}

		if ingressInput.HealthCheck != nil {
			if ingressInput.HealthCheck.ClusterMinHealthyPercentages == nil {
				ingressInput.HealthCheck.ClusterMinHealthyPercentages = make(map[string]uint32)
			}

			ingressInput.HealthCheck.ClusterMinHealthyPercentages[serviceClusterInput.ServiceClusterName] = serviceClusterInput.MinHealthyPercentage
		}

		if serviceClusterInput.Routing != nil {

			vhs, ok := s.virtualHostsMap[ingressInput.IngressType]
			if !ok {
				vhs = make(virtualHosts)
			}

			vhs.Add(serviceClusterInput)

			s.virtualHostsMap[ingressInput.IngressType] = vhs

		}

	}

	return nil

}

func (s *ServiceDiscoveryState) buildServiceCluster() error {

	for _, serviceClusterInput := range s.serviceClusters {

		serviceCluster, err := makeServiceCluster(serviceClusterInput)
		if err != nil {
			s.logger.Warnw(err.Error(), "service_cluster", serviceClusterInput.ServiceClusterName)
			return err
		}

		s.Resources[resource.ClusterType] = append(s.Resources[resource.ClusterType], serviceCluster)

		if serviceClusterInput.Routing != nil {

			err = s.buildVirtualHostForCluster(serviceClusterInput)
			if err != nil {
				s.logger.Warnw(err.Error(), "service_cluster", serviceClusterInput.ServiceClusterName)
				return err
			}

		}

	}

	return nil

}

func (s *ServiceDiscoveryState) buildServiceEndpoints() error {

	for _, serviceEndpointInput := range s.serviceEndpoints {

		serviceEndpoints, err := makeServiceEndpoints(serviceEndpointInput)
		if err != nil {
			s.logger.Warnw(err.Error(), "service_cluster", serviceEndpointInput.ServiceClusterName)
			return err
		}

		s.Resources[resource.EndpointType] = append(s.Resources[resource.EndpointType], serviceEndpoints)

	}

	return nil

}

func (s *ServiceDiscoveryState) buildIngress() error {

	for _, ingressInput := range s.ingresses {

		httpConnManagerFilter, err := makeHttpConnectionManagerFilter(ingressInput)
		if err != nil {
			s.logger.Warnw(err.Error(), "ingress_type", ingressInput.IngressType.String())
			return err
		}

		s.Resources[resource.ListenerType] = append(
			s.Resources[resource.ListenerType],
			&listenerv3.Listener{
				Name: string(ingressInput.IngressType),
				Address: &corev3.Address{
					Address: &corev3.Address_SocketAddress{
						SocketAddress: &corev3.SocketAddress{
							Protocol: corev3.SocketAddress_TCP,
							Address:  ingressInput.ListenAddress,
							PortSpecifier: &corev3.SocketAddress_PortValue{
								PortValue: ingressInput.ListenPort,
							},
						},
					},
				},
				FilterChains: []*listenerv3.FilterChain{
					{Filters: []*listenerv3.Filter{httpConnManagerFilter}},
				},
			},
		)

	}

	return nil

}

func (s *ServiceDiscoveryState) buildRoutes() {

	for _, vhs := range s.virtualHostsMap {

		res := []types.Resource{}

		for _, vh := range vhs {
			res = append(res, vh)
		}

		s.Resources[resource.RouteType] = res

	}

}

func (s *ServiceDiscoveryState) Build() error {

	err := s.buildServiceCluster()
	if err != nil {
		return err
	}

	err = s.buildServiceEndpoints()
	if err != nil {
		return err
	}

	err = s.buildIngress()
	if err != nil {
		return err
	}

	s.buildRoutes()

	return nil

}
