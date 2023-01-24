package domainregistry

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"io"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
)

const (
	DefaultUserAgent          = "ops-control/latest"
	DefaultKeyType            = "RSA4096"
	DefaultCertificateTimeout = time.Second * 30
)

func generatePrivateKey(keyType certcrypto.KeyType) (crypto.PrivateKey, error) {
	privateKey, err := certcrypto.GeneratePrivateKey(keyType)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func encodePrivateKey(privateKey crypto.PrivateKey, writer io.Writer) error {
	pemKey := certcrypto.PEMBlock(privateKey)
	return pem.Encode(writer, pemKey)
}

func loadPrivateKey(keyBytes []byte) (crypto.PrivateKey, error) {
	keyBlock, _ := pem.Decode(keyBytes)

	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(keyBlock.Bytes)
	}

	return nil, UnknownPrivateKeyTypeErr
}

func toCryptoKeyType(keyTypeStr string) (certcrypto.KeyType, error) {

	switch strings.ToUpper(keyTypeStr) {
	case "EC256":
		return certcrypto.EC256, nil
	case "EC384":
		return certcrypto.EC384, nil
	case "RSA2048":
		return certcrypto.RSA2048, nil
	case "RSA4096":
		return certcrypto.RSA4096, nil
	case "RSA8192":
		return certcrypto.RSA8192, nil
	}

	return certcrypto.KeyType(""), UnknownPrivateKeyTypeErr

}
