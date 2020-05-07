package picker

import (
	"context"

	"google.golang.org/grpc/balancer"
)

// NewErr returns a picker that always returns err on "Pick".
func NewErr(err error) Picker {
	return &errPicker{p: Error, err: err}
}

type errPicker struct {
	p   Policy
	err error
}

func (ep *errPicker) String() string {
	return ep.p.String()
}

func (ep *errPicker) Pick(context.Context, balancer.PickOptions) (balancer.SubConn, func(balancer.DoneInfo), error) {
	return nil, nil, ep.err
}
