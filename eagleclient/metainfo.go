package eagleclient

import (
	"fmt"
	"github.com/duyanghao/eagle/eagleclient/balancer"
	"github.com/duyanghao/eagle/eagleclient/balancer/picker"
	"github.com/duyanghao/eagle/eagleclient/balancer/resolver/endpoint"
	pb "github.com/duyanghao/eagle/proto/metainfo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func (e *BtEngine) newMetaInfoClient() (pb.MetaInfoClient, error) {
	rsv, err := endpoint.NewResolverGroup("eagleclient")
	if err != nil {
		return nil, err
	}
	rsv.SetEndpoints(e.seeders)

	name := fmt.Sprintf("eagleclient-%s", picker.RoundrobinBalanced.String())
	balancer.RegisterBuilder(balancer.Config{
		Policy: picker.RoundrobinBalanced,
		Name:   name,
		Logger: zap.NewExample(),
	})
	conn, err := grpc.Dial(fmt.Sprintf("endpoint://eagleclient/"), grpc.WithInsecure(), grpc.WithBalancerName(name))
	if err != nil {
		return nil, fmt.Errorf("failed to dial seeder: %s", err)
	}
	return pb.NewMetaInfoClient(conn), nil
}
