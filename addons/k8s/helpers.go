package k8s

import (
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func EnvApply(env map[string]string) []*applyCoreV1.EnvVarApplyConfiguration {
	ret := make([]*applyCoreV1.EnvVarApplyConfiguration, 0, len(env))
	for k, v := range env {
		ret = append(ret, applyCoreV1.EnvVar().WithName(k).WithValue(v))
	}
	return ret
}
