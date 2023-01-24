// package main

// import (
// 	"time"

// 	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
// 	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
// 	jwt_authnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
// 	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
// 	"google.golang.org/protobuf/encoding/protojson"
// 	"google.golang.org/protobuf/types/known/durationpb"
// )

// func main() {

// 	frontProxyCfg := &service_discovery.Ingress{
// 		XdsClusterName: "xds-cluster",
// 		ListenPort:     8081,
// 		ListenAddress:  "0.0.0.0",
// 		EnableCors:     true,
// 		HealthCheck: &service_discovery.HealthCheck{
// 			Path: "/healthz",
// 		},
// 		IngressType: service_discovery.IngressType_INGRESS_TYPE_WEB_SECURE_TRAFFIC,
// 		Domains: []*service_discovery.IngressDomain{{
// 			Zone:       "dev.pomwm.com",
// 			RecordName: "us-east-1",
// 			Ttl:        durationpb.New(30 * time.Second),
// 			CnameValue: "aaaaxx",
// 		}},
// 		GrpcWeb: &service_discovery.GrpcWeb{
// 			Enabled: true,
// 			JwtAuthentication: &jwt_authnv3.JwtAuthentication{
// 				Rules: []*jwt_authnv3.RequirementRule{
// 					{
// 						Match: &routev3.RouteMatch{
// 							PathSpecifier: &routev3.RouteMatch_Prefix{
// 								Prefix: "/api/",
// 							},
// 						},
// 						RequirementType: &jwt_authnv3.RequirementRule_Requires{
// 							Requires: &jwt_authnv3.JwtRequirement{
// 								RequiresType: &jwt_authnv3.JwtRequirement_ProviderName{
// 									ProviderName: "google",
// 								},
// 							},
// 						},
// 					},
// 				},
// 				Providers: map[string]*jwt_authnv3.JwtProvider{
// 					"google": &jwt_authnv3.JwtProvider{
// 						Issuer:    "https://securetoken.google.com/dev-pom",
// 						Audiences: []string{"dev-pom"},
// 						FromHeaders: []*jwt_authnv3.JwtHeader{
// 							{
// 								Name:        "authorization",
// 								ValuePrefix: "Bearer ",
// 							},
// 						},
// 						JwksSourceSpecifier: &jwt_authnv3.JwtProvider_RemoteJwks{
// 							RemoteJwks: &jwt_authnv3.RemoteJwks{
// 								HttpUri: &corev3.HttpUri{
// 									Uri: "https://www.googleservice_discoverys.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com",
// 									HttpUpstreamType: &corev3.HttpUri_Cluster{
// 										Cluster: "googleservice_discoverys",
// 									},
// 									Timeout: durationpb.New(20 * time.Second),
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	// http1ServerConfig := &service_discovery.ServiceDiscovery_AddServiceClusterInput{
// 	// 	ServiceClusterName:   "http1-server",
// 	// 	XdsClusterName:       "xds-cluster",
// 	// 	ServiceType:          service_discovery.ServiceDiscovery_SERVICE_TYPE_HTTP1_SERVER,
// 	// 	MinHealthyPercentage: 70,
// 	// 	IngressType:          service_discovery.ServiceDiscovery_INGRESS_TYPE_WEB_SECURE_TRAFFIC,
// 	// 	Routing: &service_discovery.ServiceDiscovery_Routing{
// 	// 		MatchDomains:      []string{"backstage.dev.*"},
// 	// 		MatchPathPrefixes: []string{"/"},
// 	// 	},
// 	// 	UpstreamPort: 7007,
// 	// }

// 	// grcpServiceConfig := &service_discovery.ServiceDiscovery_AddServiceClusterInput{
// 	// 	ServiceClusterName:   "grpc-server",
// 	// 	XdsClusterName:       "xds-cluster",
// 	// 	ServiceType:          service_discovery.ServiceDiscovery_SERVICE_TYPE_GRPC_SERVICE,
// 	// 	MinHealthyPercentage: 70,
// 	// 	IngressType:          service_discovery.ServiceDiscovery_INGRESS_TYPE_WEB_SECURE_TRAFFIC,
// 	// 	Routing: &service_discovery.ServiceDiscovery_Routing{
// 	// 		MatchDomains:      []string{"*"},
// 	// 		MatchPathPrefixes: []string{"/grpc.reflection", "/wealth.proto"},
// 	// 	},
// 	// 	UpstreamPort: 9090,
// 	// }

// 	val, _ := protojson.Marshal(frontProxyCfg)
// 	println(string(val))
// 	// val, _ = protojson.Marshal(http1ServerConfig)
// 	// println(string(val))
// 	// val, _ = protojson.Marshal(grcpServiceConfig)
// 	// println(string(val))

// }
