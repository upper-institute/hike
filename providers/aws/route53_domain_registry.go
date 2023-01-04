package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	domainregistry "github.com/upper-institute/ops-control/internal/domain-registry"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
)

const domainSeparator = "."

type route53DomainRegistry_Registration struct {
	ctx           context.Context
	ingressDomain *service_discovery.IngressDomain

	logger *zap.SugaredLogger

	hostedZoneId string
	fqdn         string
	recordType   types.RRType
}

type route53DomainRegistry struct {
	route53Client *route53.Client

	logger *zap.SugaredLogger
}

func NewRoute53DomainRegistry(
	route53Client *route53.Client,
	logger *zap.SugaredLogger,
) domainregistry.DomainRegistryService {
	return &route53DomainRegistry{
		route53Client: route53Client,
		logger:        logger,
	}
}

func (id *route53DomainRegistry) listRecords(registration *route53DomainRegistry_Registration) ([]*types.ResourceRecordSet, error) {

	listResourceRecordSetsOutput, err := id.route53Client.ListResourceRecordSets(
		registration.ctx,
		&route53.ListResourceRecordSetsInput{
			HostedZoneId:    aws.String(registration.hostedZoneId),
			StartRecordName: aws.String(registration.fqdn),
			StartRecordType: registration.recordType,
		},
	)
	if err != nil {
		return nil, err
	}

	var records []*types.ResourceRecordSet

	for _, recordSet := range listResourceRecordSetsOutput.ResourceRecordSets {

		recordFqdn := aws.ToString(recordSet.Name)

		registration.logger.Debugw("Found record", "record_fqdn", recordFqdn)

		if recordFqdn == registration.fqdn {
			registration.logger.Debugw("Match", "record_fqdn", recordFqdn)
			records = append(records, &recordSet)
		}
	}

	return records, nil

}

func (id *route53DomainRegistry) changeRecord(registration *route53DomainRegistry_Registration, changes []types.Change) (*route53.ChangeResourceRecordSetsOutput, error) {

	changeRecordSetInput := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(registration.hostedZoneId),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String("Managed by Ops Control"),
			Changes: []types.Change{{
				Action: types.ChangeActionUpsert,
			}},
		},
	}

	return id.route53Client.ChangeResourceRecordSets(registration.ctx, changeRecordSetInput)

}

func (id *route53DomainRegistry) registerCname(registration *route53DomainRegistry_Registration) error {

	registration.recordType = types.RRTypeCname

	recordSet, err := id.listRecords(registration)
	if err != nil {
		return err
	}

	resourceRecords := []types.ResourceRecord{{
		Value: aws.String(registration.ingressDomain.CnameValue),
	}}

	if len(recordSet) == 0 {

		registration.logger.Infow("Changing record (action create)", "record_fqdn", registration.fqdn)

		_, err = id.changeRecord(registration, []types.Change{{
			Action: types.ChangeActionCreate,
			ResourceRecordSet: &types.ResourceRecordSet{
				Name:            aws.String(registration.fqdn),
				Type:            types.RRTypeCname,
				TTL:             aws.Int64(int64(registration.ingressDomain.Ttl.AsDuration().Seconds())),
				ResourceRecords: resourceRecords,
			},
		}})

		if err != nil {
			return err
		}
	}

	for _, record := range recordSet {

		registration.logger.Infow("Changing record (action upsert)", "record_fqdn", aws.ToString(record.Name))

		record.ResourceRecords = resourceRecords

		_, err = id.changeRecord(registration, []types.Change{{
			Action:            types.ChangeActionUpsert,
			ResourceRecordSet: record,
		}})

		if err != nil {
			break
		}

	}

	return err

}

func (id *route53DomainRegistry) RegisterIngressDomain(ctx context.Context, ingressDomain *service_discovery.IngressDomain) error {

	logger := id.logger.With("zone", ingressDomain.Zone, "record_name", ingressDomain.RecordName)

	listHostedZonesOutput, err := id.route53Client.ListHostedZonesByName(ctx, &route53.ListHostedZonesByNameInput{
		DNSName:  aws.String(ingressDomain.Zone),
		MaxItems: aws.Int32(1),
	})
	if err != nil {
		return err
	}

	if len(listHostedZonesOutput.HostedZones) == 0 {
		return fmt.Errorf("Found 0 hosted zones (ingress zone: %s)", ingressDomain.Zone)
	}

	if !ingressDomain.Ttl.IsValid() {
		ingressDomain.Ttl = durationpb.New(30 * time.Second)
	}

	registration := &route53DomainRegistry_Registration{
		ctx:           ctx,
		hostedZoneId:  aws.ToString(listHostedZonesOutput.HostedZones[0].Id),
		ingressDomain: ingressDomain,
		logger:        logger,
		fqdn: strings.Join(
			[]string{
				strings.TrimRight(ingressDomain.RecordName, domainSeparator),
				strings.TrimLeft(ingressDomain.Zone, domainSeparator),
			},
			domainSeparator,
		),
	}

	if len(ingressDomain.CnameValue) > 0 {

		logger.Infow("Ingress domain is a CNAME value")

		if err := id.registerCname(registration); err != nil {
			return err
		}
	}

	return nil
}
