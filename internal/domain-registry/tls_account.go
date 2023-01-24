package domainregistry

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"github.com/upper-institute/ops-control/internal/parameter"
)

type tlsAccountPayload struct {
	Registration *registration.Resource `json:"registration,omitempty"`
	PrivateKey   []byte                 `json:"private_key,omitempty"`
}

type tlsAccount struct {
	*tlsAccountPayload

	ingressTls *service_discovery.IngressTLS

	privateKey crypto.PrivateKey
	parameter  *parameter.Parameter
}

func (t *tlsAccount) tryRecoverRegistration(ctx context.Context) error {
	// couldn't load account but got a key. Try to look the account up.
	config := lego.NewConfig(t)
	config.CADirURL = t.ingressTls.CaDirUrl
	config.UserAgent = t.ingressTls.UserAgent

	client, err := lego.NewClient(config)
	if err != nil {
		return err
	}

	reg, err := client.Registration.ResolveAccountByKey()

	if err != nil {

		resource, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})

		if err != nil {
			return err
		}

		reg = resource

	}

	t.Registration = reg

	return nil
}

func (t *tlsAccount) load(ctx context.Context) error {

	err := t.parameter.Load(ctx)

	switch {

	case err == parameter.FileNotFoundErr:

		privateKey, err := generatePrivateKey(certcrypto.KeyType(t.ingressTls.KeyType))
		if err != nil {
			return err
		}

		t.privateKey = privateKey

		err = t.save(ctx)
		if err != nil {
			return err
		}

	case err != nil:
		return err

	}

	err = json.Unmarshal(t.parameter.GetFile().Bytes(), t.tlsAccountPayload)
	if err != nil {
		return err
	}

	if t.privateKey == nil {

		privateKey, err := loadPrivateKey(t.PrivateKey)
		if err != nil {
			return err
		}

		t.privateKey = privateKey

	}

	if t.Registration == nil || t.Registration.Body.Status == "" {

		err = t.tryRecoverRegistration(ctx)
		if err != nil {
			return err
		}

		err = t.save(ctx)
		if err != nil {
			return err
		}

	}

	return nil

}

func (t *tlsAccount) save(ctx context.Context) error {

	pkBuf := bytes.NewBuffer(nil)

	err := encodePrivateKey(t.privateKey, pkBuf)
	if err != nil {
		return err
	}

	t.PrivateKey = pkBuf.Bytes()

	data, err := json.Marshal(t.tlsAccountPayload)
	if err != nil {
		return err
	}

	file := t.parameter.GetFile()

	file.Reset()
	file.Write(data)

	return t.parameter.Push(ctx)

}

func (t *tlsAccount) GetEmail() string {
	return t.ingressTls.AccountEmail
}

func (t *tlsAccount) GetRegistration() *registration.Resource {
	return t.Registration
}

func (t *tlsAccount) GetPrivateKey() crypto.PrivateKey {
	return t.privateKey
}
