package p2pclient

import (
	"fmt"
	"github.com/duyanghao/eagle/p2pclient/balancer"
	"github.com/duyanghao/eagle/p2pclient/balancer/picker"
	"github.com/duyanghao/eagle/p2pclient/balancer/resolver/endpoint"
	pb "github.com/duyanghao/eagle/proto/metainfo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func (e *BtEngine) newMetaInfoClient() (pb.MetaInfoClient, error) {
	rsv, err := endpoint.NewResolverGroup("p2pclient")
	if err != nil {
		return nil, err
	}
	rsv.SetEndpoints(e.seeders)

	name := fmt.Sprintf("p2pclient-%s", picker.RoundrobinBalanced.String())
	balancer.RegisterBuilder(balancer.Config{
		Policy: picker.RoundrobinBalanced,
		Name:   name,
		Logger: zap.NewExample(),
	})
	conn, err := grpc.Dial(fmt.Sprintf("endpoint://p2pclient/"), grpc.WithInsecure(), grpc.WithBalancerName(name))
	if err != nil {
		return nil, fmt.Errorf("failed to dial seeder: %s", err)
	}
	return pb.NewMetaInfoClient(conn), nil
}
