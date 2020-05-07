// Copyright 2020 duyanghao
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
