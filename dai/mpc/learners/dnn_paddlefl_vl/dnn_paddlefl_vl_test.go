package dnn_paddlefl_vl

import (
	"errors"
	"io/ioutil"
	"log"
	"testing"
	"time"

	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

type rpc struct {
	reqC  map[string]chan *pb.TrainRequest
	respC map[string]chan *pb.TrainResponse
}

func NewRpc(parties []string, req1 chan *pb.TrainRequest, rep1 chan *pb.TrainResponse, req2 chan *pb.TrainRequest, rep2 chan *pb.TrainResponse) *rpc {
	ret := &rpc{}
	ret.reqC = make(map[string]chan *pb.TrainRequest)
	ret.respC = make(map[string]chan *pb.TrainResponse)
	ret.reqC[parties[0]] = req1
	ret.respC[parties[0]] = rep1
	ret.reqC[parties[1]] = req2
	ret.respC[parties[1]] = rep2
	return ret
}

func (r *rpc) StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error) {
	r.reqC[peerName] <- req
	resp := <-r.respC[peerName]
	if resp != nil {
		return resp, nil
	}
	return nil, errors.New("test response error")
}

type resHandler struct {
	modelC chan *[]byte
}

func NewResHandle() *resHandler {
	var modelC1 = make(chan *[]byte)
	return &resHandler{
		modelC: modelC1,
	}
}

func (rd *resHandler) SaveResult(res *pbCom.TrainTaskResult) {
	if res.Success {
		rd.modelC <- &res.Model
	} else {
		log.Printf("training failed, and reason is %s.", res.ErrMsg)
		rd.modelC <- nil
	}
}

func TestAdvance(t *testing.T) {
	ids := []string{"127.0.0.1:8180", "127.0.0.1:8280", "127.0.0.1:8380"}
	addresses := ids
	parties := [][]string{
		{addresses[1], addresses[2]},
		{addresses[0], addresses[2]},
		{addresses[0], addresses[1]},
	}

	trainFile := []string{
		"../../testdata/vl/dnn_paddlefl/train_dataA.csv",
		"../../testdata/vl/dnn_paddlefl/train_dataB.csv",
		"../../testdata/vl/dnn_paddlefl/train_dataC.csv",
	}
	samplesFile1, err := ioutil.ReadFile(trainFile[0])
	checkErr(err, t)
	samplesFile2, err := ioutil.ReadFile(trainFile[1])
	checkErr(err, t)
	samplesFile3, err := ioutil.ReadFile(trainFile[2])
	checkErr(err, t)

	params := []*pbCom.TrainParams{
		{
			Label:     "MEDV",
			RegMode:   0,
			RegParam:  0.1,
			Alpha:     0.1,
			Amplitude: 0.0001,
			Accuracy:  10,
			IsTagPart: false,
			IdName:    "id",
			BatchSize: 4,
		},
		{
			Label:     "MEDV",
			RegMode:   0,
			RegParam:  0.1,
			Alpha:     0.1,
			Amplitude: 0.0001,
			Accuracy:  10,
			IsTagPart: false,
			IdName:    "id",
			BatchSize: 4,
		},
		{
			Label:     "MEDV",
			RegMode:   0,
			RegParam:  0.1,
			Alpha:     0.1,
			Amplitude: 0.0001,
			Accuracy:  10,
			IsTagPart: true,
			IdName:    "id",
			BatchSize: 4,
		},
	}

	req01 := make(chan *pb.TrainRequest)
	resp01 := make(chan *pb.TrainResponse)
	req02 := make(chan *pb.TrainRequest)
	resp02 := make(chan *pb.TrainResponse)

	req10 := make(chan *pb.TrainRequest)
	resp10 := make(chan *pb.TrainResponse)
	req12 := make(chan *pb.TrainRequest)
	resp12 := make(chan *pb.TrainResponse)

	req20 := make(chan *pb.TrainRequest)
	resp20 := make(chan *pb.TrainResponse)
	req21 := make(chan *pb.TrainRequest)
	resp21 := make(chan *pb.TrainResponse)

	rpcs := [3]*rpc{}
	rpcs[0] = NewRpc(parties[0], req01, resp01, req02, resp02)
	rpcs[1] = NewRpc(parties[1], req10, resp10, req12, resp12)
	rpcs[2] = NewRpc(parties[2], req20, resp20, req21, resp21)

	var learner1, learner2, learner0 *Learner
	go func() {
		learner0, err = NewLearner(ids[0], addresses[0], params[0], samplesFile1, parties[0], &pbCom.PaddleFLParams{
			Role:  0,
			Nodes: []string{"paddlefl-env1:38302", "paddlefl-env2:38303", "paddlefl-env3:38304"},
		}, rpcs[0], NewResHandle())
		checkErr(err, t)
	}()
	go func() {
		learner1, err = NewLearner(ids[1], addresses[1], params[1], samplesFile2, parties[1], &pbCom.PaddleFLParams{
			Role:  1,
			Nodes: []string{"paddlefl-env1:38302", "paddlefl-env2:38303", "paddlefl-env3:38304"},
		}, rpcs[1], NewResHandle())
		checkErr(err, t)
	}()
	go func() {
		learner2, err = NewLearner(ids[2], addresses[2], params[2], samplesFile3, parties[2], &pbCom.PaddleFLParams{
			Role:  2,
			Nodes: []string{"paddlefl-env1:38302", "paddlefl-env2:38303", "paddlefl-env3:38304"},
		}, rpcs[2], NewResHandle())
		checkErr(err, t)
	}()

	for {
		select {
		case resv := <-req01:
			if learner1 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := learner1.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "learner1", err.Error())
			}
			resp01 <- resp

		case resv := <-req02:
			if learner2 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := learner2.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "learner2", err.Error())
			}
			resp02 <- resp

		case resv := <-req10:
			if learner0 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := learner0.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "learner0", err.Error())
			}
			resp10 <- resp

		case resv := <-req12:
			if learner2 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := learner2.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "learner2", err.Error())
			}
			resp12 <- resp

		case resv := <-req20:
			if learner0 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := learner0.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "learner0", err.Error())
			}
			resp20 <- resp

		case resv := <-req21:
			if learner1 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := learner1.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "learner1", err.Error())
			}
			resp21 <- resp
		}

	}

}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
