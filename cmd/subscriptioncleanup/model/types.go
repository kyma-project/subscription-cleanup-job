package model

import "fmt"

type HyperscalerType string

const (
	GCP   HyperscalerType = "gcp"
	Azure HyperscalerType = "azure"
	AWS   HyperscalerType = "aws"
)

func NewHyperscalerType(provider string) (HyperscalerType, error) {

	hyperscalerType := HyperscalerType(provider)

	switch hyperscalerType {
	case GCP, Azure, AWS:
		return hyperscalerType, nil
	case "gcp_cf-sa30":
		return GCP, nil
	}
	return "", fmt.Errorf("unknown Hyperscaler provider type: %s", provider)
}
