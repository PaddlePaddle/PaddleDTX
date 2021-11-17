// Copyright (c) 2021 PaddlePaddle Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

// Mpc is used to handle requests for training and prediction
type Mpc interface {
	Predict(*pb.PredictRequest) (*pb.PredictResponse, error)
	Train(*pb.TrainRequest) (*pb.TrainResponse, error)
}

// Service is implementation for mpc.Cluster
type Service struct {
	mpc Mpc
}

// Step @implementation mpc.Cluster.Step
func (s *Service) Step(ctx context.Context, in *pb.StepRequest) (resp *pb.StepResponse, err error) {
	if trainReq := in.GetTrainRequest(); trainReq != nil {
		var trainResp *pb.TrainResponse
		trainResp, err = s.mpc.Train(trainReq)
		if err == nil {
			resp = &pb.StepResponse{
				Payload: &pb.StepResponse_TrainResponse{
					TrainResponse: trainResp,
				},
			}
		}
	} else {
		var predictResp *pb.PredictResponse
		predictReq := in.GetPredictRequest()
		predictResp, err = s.mpc.Predict(predictReq)
		if err == nil {
			resp = &pb.StepResponse{
				Payload: &pb.StepResponse_PredictResponse{
					PredictResponse: predictResp,
				},
			}
		}
	}
	return
}

// NewService to create a Service instance
func NewService(m Mpc) *Service {
	s := &Service{
		mpc: m,
	}

	return s
}

// RegisterClusterServer to register mpc.cluster.Service to grpcServer
func (s *Service) RegisterClusterServer(grpcServer *grpc.Server) {
	pb.RegisterClusterServer(grpcServer, s)
}
