package servicediscovery

import (
	"strings"
	"time"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"google.golang.org/protobuf/types/known/durationpb"
)

type virtualHosts map[string]*routev3.VirtualHost

func (v virtualHosts) Add(serviceClusterInput *service_discovery.ServiceCluster) {

	routing := serviceClusterInput.Routing

	routeToClusterAction := &routev3.Route_Route{
		Route: &routev3.RouteAction{
			ClusterSpecifier: &routev3.RouteAction_Cluster{
				Cluster: serviceClusterInput.ServiceClusterName,
			},
			Timeout: durationpb.New(60 * time.Second),
			MaxStreamDuration: &routev3.RouteAction_MaxStreamDuration{
				GrpcTimeoutHeaderMax: durationpb.New(60 * time.Second),
			},
		},
	}

	if len(routing.VirtualHostName) == 0 {
		routing.VirtualHostName = "default_vh"
	}

	routes := []*routev3.Route{}

	for _, matchPathPrefix := range routing.MatchPathPrefixes {

		route := &routev3.Route{
			Match: &routev3.RouteMatch{
				PathSpecifier: &routev3.RouteMatch_Prefix{
					Prefix: matchPathPrefix,
				},
			},
			Action: routeToClusterAction,
		}

		routes = append(routes, route)

	}

	virtualHost, ok := v[routing.VirtualHostName]
	if !ok {
		virtualHost = &routev3.VirtualHost{
			Name:    routing.VirtualHostName,
			Domains: routing.MatchDomains,
			Routes:  []*routev3.Route{},
			Cors: &routev3.CorsPolicy{
				AllowMethods: "GET, PUT, DELETE, POST, OPTIONS",
				AllowHeaders: "*",
				MaxAge:       "1728000",
				AllowOriginStringMatch: []*matcherv3.StringMatcher{{
					MatchPattern: &matcherv3.StringMatcher_Prefix{
						Prefix: "*",
					},
				}},
			},
		}
	}

	corsPolicy := serviceClusterInput.CorsPolicy

	if corsPolicy != nil {

		if len(corsPolicy.ExposeHeaders) > 0 {
			virtualHost.Cors.ExposeHeaders = strings.Join(corsPolicy.ExposeHeaders, ",")
		}

	}

	virtualHost.Routes = append(virtualHost.Routes, routes...)

	v[routing.VirtualHostName] = virtualHost

}
