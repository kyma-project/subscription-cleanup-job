package model

import "fmt"

type HyperscalerType string

const (
	GCP      HyperscalerType = "gcp"
	Azure    HyperscalerType = "azure"
	AWS      HyperscalerType = "aws"
	Alicloud HyperscalerType = "alicloud"
)

func NewHyperscalerType(provider string) (HyperscalerType, error) {

	hyperscalerType := HyperscalerType(provider)

	switch hyperscalerType {
	case GCP, Azure, AWS, Alicloud:
		return hyperscalerType, nil
	case "gcp_cf-sa30":
		return GCP, nil
	}
	return "", fmt.Errorf("unknown Hyperscaler provider type: %s", provider)
}
