package cluster

type RetryMode int

const (
	Requeue RetryMode = iota
	Immediate
)
