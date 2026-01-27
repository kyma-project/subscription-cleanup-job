package cloudprovider

import (
	"context"
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

	for _, region := range allRegions.Regions {
		logrus.Printf("Switching to region %v", *region.RegionName)
		ec2Client, err := ac.newAwsEC2Client(ac.credentials, *region.RegionName)
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

	for _, volume := range volumes.Volumes {
		logrus.Printf("Deleting volume with id %v", *volume.VolumeId)
		ec2Client.DeleteVolume(context.TODO(), &ec2.DeleteVolumeInput{
			VolumeId: volume.VolumeId,
		})
	}

	return nil
}

func (ac awsResourceCleaner) getAllRegions() (ec2.DescribeRegionsOutput, error) {
	var allRegions bool
	var region string

	switch ac.market {
	case model.GlobalMarket:
		allRegions = false
		region = "us-east-1"
	case model.ChineseMarket:
		allRegions = true
		region = "cn-north-1"
	case model.USGovMarket:
		allRegions = true
		region = "us-gov-west-1"
	default:
		return ec2.DescribeRegionsOutput{}, fmt.Errorf("unsupported AWS market: %v", market)
	}

	ec2Client, err := ac.newAwsEC2Client(ac.credentials, region)
	if err != nil {
		return ec2.DescribeRegionsOutput{}, err
	}

	regionOutput, err := ec2Client.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{AllRegions: &allRegions})
	if err != nil {
		return ec2.DescribeRegionsOutput{}, err
	}

	return *regionOutput, nil
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
