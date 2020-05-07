package picker

import (
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
)

// Picker defines balancer Picker methods.
type Picker interface {
	balancer.Picker
	String() string
}

// Config defines picker configuration.
type Config struct {
	// Policy specifies etcd clientv3's built in balancer policy.
	Policy Policy

	// Logger defines picker logging object.
	Logger *zap.Logger

	// SubConnToResolverAddress maps each gRPC sub-connection to an address.
	// Basically, it is a list of addresses that the Picker can pick from.
	SubConnToResolverAddress map[balancer.SubConn]resolver.Address
}

// Policy defines balancer picker policy.
type Policy uint8

const (
	// Error is error picker policy.
	Error Policy = iota

	// RoundrobinBalanced balances loads over multiple endpoints
	// and implements failover in roundrobin fashion.
	RoundrobinBalanced

	// Custom defines custom balancer picker.
	// TODO: custom picker is not supported yet.
	Custom
)

func (p Policy) String() string {
	switch p {
	case Error:
		return "picker-error"

	case RoundrobinBalanced:
		return "picker-roundrobin-balanced"

	case Custom:
		panic("'custom' picker policy is not supported yet")

	default:
		panic(fmt.Errorf("invalid balancer picker policy (%d)", p))
	}
}

// New creates a new Picker.
func New(cfg Config) Picker {
	switch cfg.Policy {
	case Error:
		panic("'error' picker policy is not supported here; use 'picker.NewErr'")

	case RoundrobinBalanced:
		return newRoundrobinBalanced(cfg)

	case Custom:
		panic("'custom' picker policy is not supported yet")

	default:
		panic(fmt.Errorf("invalid balancer picker policy (%d)", cfg.Policy))
	}
}
