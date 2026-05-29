package cloudprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/kyma-project/subscription-cleanup-job/cmd/subscriptioncleanup/model"

	"github.com/sirupsen/logrus"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type awsResourceCleaner struct {
	credentials awsCredentialsConfig
	market      model.Market
}

type awsCredentialsConfig struct {
	accessKeyID     string
	secretAccessKey string
}

func NewAwsResourcesCleaner(secretData map[string][]byte, market model.Market) (ResourceCleaner, error) {
	awsConfig, err := toAwsConfig(secretData)
	if err != nil {
		return nil, err
	}

	return awsResourceCleaner{
		credentials: awsConfig,
		market:      market,
	}, nil
}

func (ac awsResourceCleaner) Do() error {
	allRegions, err := ac.getAllRegions()
	if err != nil {
		return err
	}

	for _, region := range allRegions {
		logrus.Printf("Switching to region %v", region)
		ec2Client, err := ac.newAwsEC2Client(ac.credentials, region)
		if err != nil {
			return err
		}

		err = ac.deleteVolumes(ec2Client)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ac awsResourceCleaner) deleteVolumes(ec2Client *ec2.Client) error {
	volumes, err := ec2Client.DescribeVolumes(context.TODO(), &ec2.DescribeVolumesInput{})
	if err != nil {
		return err
	}

	for _, volume := range volumes.Volumes {
		if volume.State == types.VolumeStateInUse {
			return fmt.Errorf("There is an EC2 instance which uses this volume with id: %v", *volume.VolumeId)
		}
	}

	var errs []error
	for _, volume := range volumes.Volumes {
		logrus.Printf("Deleting volume with id %v", *volume.VolumeId)
		_, err := ec2Client.DeleteVolume(context.TODO(), &ec2.DeleteVolumeInput{
			VolumeId: volume.VolumeId,
		})

		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete volume %v: %w", *volume.VolumeId, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (ac awsResourceCleaner) getAllRegions() ([]string, error) {
	// This is a temporary solution ; list of available regions must be passed to the SCJ job
	switch ac.market {
	case model.GlobalMarket:
		return []string{
			"eu-central-1",
			"eu-south-1",
			"eu-west-2",
			"ca-central-1",
			"sa-east-1",
			"us-east-1",
			"us-west-2",
			"ap-northeast-1",
			"ap-northeast-2",
			"ap-south-1",
			"ap-southeast-1",
			"ap-southeast-2",
			"eu-west-1", // used for trials
		}, nil
	case model.ChineseMarket:
		return []string{"cn-north-1", "cn-northwest-1"}, nil
	case model.USGovMarket:
		return []string{"us-gov-east-1", "us-gov-west-1"}, nil
	}

	return nil, fmt.Errorf("unsupported AWS market: %v", ac.market)
}
}

func toAwsConfig(secretData map[string][]byte) (awsCredentialsConfig, error) {
	accessKeyID, exists := secretData["accessKeyID"]
	if !exists {
		return awsCredentialsConfig{}, fmt.Errorf("AccessKeyID was not provided in secret!")
	}

	secretAccessKey, exists := secretData["secretAccessKey"]
	if !exists {
		return awsCredentialsConfig{}, fmt.Errorf("SecretAccessKey was not provided in secret!")
	}

	return awsCredentialsConfig{
		accessKeyID:     string(accessKeyID),
		secretAccessKey: string(secretAccessKey),
	}, nil
}

func (ac awsResourceCleaner) newAwsEC2Client(awsCredentialConfig awsCredentialsConfig, region string) (*ec2.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(awsCredentialConfig.accessKeyID, awsCredentialConfig.secretAccessKey, "")

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, err
	}

	return ec2.NewFromConfig(cfg), nil
}
