package cluster

type subscribeInfo struct {
	waiters     []chan Result
	subs        int
	result      *Result
	ct          *clusterTask
	isCacheable bool
}
