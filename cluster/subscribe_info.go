package cluster

type SubscribeInfo struct {
	waiters []chan Result
	subs    int
	result  *Result
	err     error
	ct      *clusterTask
}
