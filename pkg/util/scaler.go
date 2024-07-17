package util

import (
	gst "github.com/elankath/gardener-scaling-types"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
)

// ParseClusterAutoscalerConfig parses the cluster autoscaler configuration from the given file path.
func ParseClusterAutoscalerConfig(cfgPath string) (gst.AutoScalerConfig, error) {
	autoScalerCfg := gst.AutoScalerConfig{}
	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		return autoScalerCfg, err
	}
	err = json.Unmarshal(cfgBytes, autoScalerCfg)
	return autoScalerCfg, err
}

func MapScalingRecommenderConfig(caCfg gst.AutoScalerConfig) {

}
