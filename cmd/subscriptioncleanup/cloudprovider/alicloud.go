package cloudprovider

type alicloudResourceCleaner struct {
}

func NewAliCloudResourcesCleaner(secretData map[string][]byte) ResourceCleaner {
	return &alicloudResourceCleaner{}
}

func (rc alicloudResourceCleaner) Do() error {
	return nil
}
