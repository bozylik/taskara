package cluster

type SubscribeInfo struct {
	waiters     []chan Result
	subs        int
	result      *Result
	ct          *clusterTask
	isCacheable bool
}
