package awsdriver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	sdapi "github.com/upper-institute/hike/proto/api/service-discovery"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
)

const domainSeparator = "."

type route53DomainRegistry_Registration struct {
	ctx    context.Context
	record *sdapi.DnsRecord

	logger *zap.SugaredLogger

	hostedZoneId string
	fqdn         string
	recordType   types.RRType
}

type Route53DomainRegistry struct {
	route53Client *route53.Client

	logger *zap.SugaredLogger
}

func NewRoute53DomainRegistry(
	route53Client *route53.Client,
	logger *zap.SugaredLogger,
) *Route53DomainRegistry {
	return &Route53DomainRegistry{
		route53Client: route53Client,
		logger:        logger,
	}
}

func (id *Route53DomainRegistry) listRecords(registration *route53DomainRegistry_Registration) ([]*types.ResourceRecordSet, error) {

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

		recordFqdn := strings.TrimRight(aws.ToString(recordSet.Name), domainSeparator)

		registration.logger.Debugw("Found record", "record_fqdn", recordFqdn, "match_fqdn", registration.fqdn)

		if recordFqdn == registration.fqdn {
			registration.logger.Debugw("Match existing record", "record_fqdn", recordFqdn)
			records = append(records, &recordSet)
		}
	}

	return records, nil

}

func (id *Route53DomainRegistry) changeRecord(registration *route53DomainRegistry_Registration, changes []types.Change) (*route53.ChangeResourceRecordSetsOutput, error) {

	changeRecordSetInput := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(registration.hostedZoneId),
		ChangeBatch: &types.ChangeBatch{
			Comment: aws.String("Managed by Ops Control"),
			Changes: changes,
		},
	}

	return id.route53Client.ChangeResourceRecordSets(registration.ctx, changeRecordSetInput)

}

func (id *Route53DomainRegistry) registerCname(registration *route53DomainRegistry_Registration) error {

	registration.recordType = types.RRTypeCname

	recordSet, err := id.listRecords(registration)
	if err != nil {
		return err
	}

	resourceRecords := []types.ResourceRecord{{
		Value: aws.String(registration.record.CnameValue),
	}}

	if len(recordSet) == 0 {

		registration.logger.Infow("Inserting record (action create)", "record_fqdn", registration.fqdn)

		_, err = id.changeRecord(registration, []types.Change{{
			Action: types.ChangeActionCreate,
			ResourceRecordSet: &types.ResourceRecordSet{
				Name:            aws.String(registration.fqdn),
				Type:            types.RRTypeCname,
				TTL:             aws.Int64(int64(registration.record.Ttl.AsDuration().Seconds())),
				ResourceRecords: resourceRecords,
			},
		}})

		if err != nil {
			return err
		}
	}

	for _, record := range recordSet {

		for _, rr := range record.ResourceRecords {

			if aws.ToString(rr.Value) == registration.record.CnameValue {

				registration.logger.Debugw("No need to update cname value, skipping action", "record_fqdn", aws.ToString(record.Name))

				return nil
			}
		}

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

func (id *Route53DomainRegistry) registerDnsRecord(ctx context.Context, record *sdapi.DnsRecord) error {

	logger := id.logger.With("zone", record.Zone, "record_name", record.RecordName)

	listHostedZonesOutput, err := id.route53Client.ListHostedZonesByName(ctx, &route53.ListHostedZonesByNameInput{
		DNSName:  aws.String(record.Zone),
		MaxItems: aws.Int32(1),
	})
	if err != nil {
		return err
	}

	if len(listHostedZonesOutput.HostedZones) == 0 {
		return fmt.Errorf("Found 0 hosted zones (ingress zone: %s)", record.Zone)
	}

	if !record.Ttl.IsValid() {
		record.Ttl = durationpb.New(30 * time.Second)
	}

	registration := &route53DomainRegistry_Registration{
		ctx:          ctx,
		hostedZoneId: aws.ToString(listHostedZonesOutput.HostedZones[0].Id),
		record:       record,
		logger:       logger,
		fqdn: strings.Join(
			[]string{
				strings.TrimRight(record.RecordName, domainSeparator),
				strings.TrimLeft(record.Zone, domainSeparator),
			},
			domainSeparator,
		),
	}

	if len(record.CnameValue) > 0 {

		logger.Infow("Ingress domain is a CNAME value")

		if err := id.registerCname(registration); err != nil {
			return err
		}
	}

	return nil
}

func (r *Route53DomainRegistry) Discover(ctx context.Context, svcCh chan *sdapi.Service) {

	defer close(svcCh)

}
