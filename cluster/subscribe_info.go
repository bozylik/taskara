package cluster

import "time"

type subscribeInfo struct {
	waiters  []chan Result
	subs     int
	result   *Result
	ct       *clusterTask
	cacheTTL time.Duration
}
