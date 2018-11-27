package util

import (
	"fmt"
	"github.com/samsung-cnct/cma-ssh/pkg/apis/cluster/common"
)

func GetStatus(statuses []common.StatusPhase) (common.StatusPhase, error) {
	if len(statuses) == 0 {
		return common.EmptyClusterPhase, nil
	}

	statusCheck := common.ReadyResourcePhase
	for _, status := range statuses {
		if status != statusCheck && status != common.ReadyResourcePhase {
			return status, fmt.Errorf("non-uniform machine statuses")
		}

		statusCheck = status
	}

	return statusCheck, nil
}
