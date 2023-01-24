package domainregistry

import (
	"context"

	"github.com/go-acme/lego/v4/lego"
	paramapi "github.com/upper-institute/ops-control/gen/api/parameter"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"github.com/upper-institute/ops-control/internal/parameter"
)

type TLSOptions struct {
	HTTP01Provider *HTTP01Provider
}

func (options *TLSOptions) Build(ctx context.Context, parameterCache *parameter.Cache, ingressTls *service_discovery.IngressTLS) (*tlsCertificate, error) {

	if len(ingressTls.UserAgent) == 0 {
		ingressTls.UserAgent = DefaultUserAgent
	}

	if len(ingressTls.CaDirUrl) == 0 {
		ingressTls.CaDirUrl = lego.LEDirectoryStaging
	}

	if len(ingressTls.KeyType) == 0 {
		ingressTls.KeyType = DefaultKeyType
	}

	if !parameterCache.HasWellKnown(paramapi.WellKnown_WELL_KNOWN_TLS_ACCOUNT) {
		return nil, MissingWellKnownTlsAccountErr
	}

	tlsAccParam := parameterCache.GetWellKnown(paramapi.WellKnown_WELL_KNOWN_TLS_ACCOUNT)

	tlsAcc := &tlsAccount{
		ingressTls: ingressTls,
		parameter:  tlsAccParam,
	}

	err := tlsAcc.load(ctx)
	if err != nil {
		return nil, err
	}

	keyType, err := toCryptoKeyType(ingressTls.KeyType)
	if err != nil {
		return nil, err
	}

	config := lego.NewConfig(tlsAcc)
	config.CADirURL = ingressTls.CaDirUrl
	config.UserAgent = ingressTls.UserAgent
	config.Certificate = lego.CertificateConfig{
		KeyType: keyType,
		Timeout: DefaultCertificateTimeout,
	}
	config.HTTPClient.Timeout = DefaultCertificateTimeout

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	if !parameterCache.HasWellKnown(paramapi.WellKnown_WELL_KNOWN_TLS_CERTIFICATE) {
		return nil, MissingWellKnownTlsAccountErr
	}

	tlsCertParam := parameterCache.GetWellKnown(paramapi.WellKnown_WELL_KNOWN_TLS_CERTIFICATE)

	err = client.Challenge.SetHTTP01Provider(options.HTTP01Provider)
	if err != nil {
		return nil, err
	}

	tlsCert := &tlsCertificate{
		ingressTls:     ingressTls,
		client:         client,
		parameter:      tlsCertParam,
		http01Provider: options.HTTP01Provider,
	}

	return tlsCert, nil

}
