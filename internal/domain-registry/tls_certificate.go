package domainregistry

import (
	"context"
	"crypto"
	"crypto/x509"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"github.com/upper-institute/ops-control/internal/parameter"
)

type tlsCertificate struct {
	ingressTls     *service_discovery.IngressTLS
	client         *lego.Client
	parameter      *parameter.Parameter
	http01Provider *HTTP01Provider

	privateKey crypto.PrivateKey
}

func (t *tlsCertificate) Restore(ctx context.Context, key string) {

}

func (t *tlsCertificate) Obtain(ctx context.Context) error {

	request := certificate.ObtainRequest{
		Domains:                        t.ingressTls.Domains,
		Bundle:                         true,
		MustStaple:                     true,
		AlwaysDeactivateAuthorizations: false,
		PrivateKey:                     t.privateKey,
	}

	resource, err := t.client.Certificate.Obtain(request)
	if err != nil {
		return err
	}

	cert, err := x509.ParseCertificate(resource.Certificate)

	cert.NotAfter

	return nil

}

func (t *tlsCertificate) Renew() {

}

func (t *tlsCertificate) Save() {

}
