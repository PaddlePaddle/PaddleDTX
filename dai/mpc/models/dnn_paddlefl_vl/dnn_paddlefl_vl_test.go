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
	reqC  map[string]chan *pb.PredictRequest
	respC map[string]chan *pb.PredictResponse
}

func NewRpc(parties []string, req1 chan *pb.PredictRequest, rep1 chan *pb.PredictResponse, req2 chan *pb.PredictRequest, rep2 chan *pb.PredictResponse) *rpc {
	ret := &rpc{}
	ret.reqC = make(map[string]chan *pb.PredictRequest)
	ret.respC = make(map[string]chan *pb.PredictResponse)
	ret.reqC[parties[0]] = req1
	ret.respC[parties[0]] = rep1
	ret.reqC[parties[1]] = req2
	ret.respC[parties[1]] = rep2
	return ret
}

func (r *rpc) StepPredict(req *pb.PredictRequest, peerName string) (*pb.PredictResponse, error) {
	r.reqC[peerName] <- req
	resp := <-r.respC[peerName]
	if resp != nil {
		return resp, nil
	}
	return nil, errors.New("test response error")
}

type resHandler struct {
	outcomes chan *[]byte
}

func NewResHandle() *resHandler {
	var out = make(chan *[]byte)
	return &resHandler{
		outcomes: out,
	}
}

func (rd *resHandler) SaveResult(res *pbCom.PredictTaskResult) {
	if res.Success {
		rd.outcomes <- &res.Outcomes
	} else {
		log.Printf("training failed, and reason is %s.", res.ErrMsg)
		rd.outcomes <- nil
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
		"../../testdata/vl/dnn_paddlefl/predict_dataA.csv",
		"../../testdata/vl/dnn_paddlefl/predict_dataB.csv",
		"../../testdata/vl/dnn_paddlefl/predict_dataC.csv",
	}
	samplesFile1, err := ioutil.ReadFile(trainFile[0])
	checkErr(err, t)
	samplesFile2, err := ioutil.ReadFile(trainFile[1])
	checkErr(err, t)
	samplesFile3, err := ioutil.ReadFile(trainFile[2])
	checkErr(err, t)

	params := []*pbCom.TrainModels{
		{
			Label:     "MEDV",
			IsTagPart: false,
			IdName:    "id",
		},
		{
			Label:     "MEDV",
			IsTagPart: false,
			IdName:    "id",
		},
		{
			Label:     "MEDV",
			IsTagPart: true,
			IdName:    "id",
		},
	}

	req01 := make(chan *pb.PredictRequest)
	resp01 := make(chan *pb.PredictResponse)
	req02 := make(chan *pb.PredictRequest)
	resp02 := make(chan *pb.PredictResponse)

	req10 := make(chan *pb.PredictRequest)
	resp10 := make(chan *pb.PredictResponse)
	req12 := make(chan *pb.PredictRequest)
	resp12 := make(chan *pb.PredictResponse)

	req20 := make(chan *pb.PredictRequest)
	resp20 := make(chan *pb.PredictResponse)
	req21 := make(chan *pb.PredictRequest)
	resp21 := make(chan *pb.PredictResponse)

	rpcs := [3]*rpc{}
	rpcs[0] = NewRpc(parties[0], req01, resp01, req02, resp02)
	rpcs[1] = NewRpc(parties[1], req10, resp10, req12, resp12)
	rpcs[2] = NewRpc(parties[2], req20, resp20, req21, resp21)
	var modele0, modele1, modele2 *Model

	go func() {
		modele0, err = NewModel(ids[0], addresses[0], params[0], samplesFile1, parties[0], &pbCom.PaddleFLParams{
			Role:  0,
			Nodes: []string{"paddlefl-env1:38302", "paddlefl-env2:38303", "paddlefl-env3:38304"},
		}, rpcs[0], NewResHandle())
		checkErr(err, t)
	}()
	go func() {
		modele1, err = NewModel(ids[1], addresses[1], params[1], samplesFile2, parties[1], &pbCom.PaddleFLParams{
			Role:  1,
			Nodes: []string{"paddlefl-env1:38302", "paddlefl-env2:38303", "paddlefl-env3:38304"},
		}, rpcs[1], NewResHandle())
		checkErr(err, t)
	}()
	go func() {
		modele2, err = NewModel(ids[2], addresses[2], params[2], samplesFile3, parties[2], &pbCom.PaddleFLParams{
			Role:  2,
			Nodes: []string{"paddlefl-env1:38302", "paddlefl-env2:38303", "paddlefl-env3:38304"},
		}, rpcs[2], NewResHandle())
		checkErr(err, t)
	}()

	for {
		select {
		case resv := <-req01:
			if modele1 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := modele1.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "modele1", err.Error())
			}
			resp01 <- resp

		case resv := <-req02:
			if modele2 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := modele2.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "modele2", err.Error())
			}
			resp02 <- resp

		case resv := <-req10:
			if modele0 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := modele0.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "modele0", err.Error())
			}
			resp10 <- resp

		case resv := <-req12:
			if modele2 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := modele2.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "modele2", err.Error())
			}
			resp12 <- resp

		case resv := <-req20:
			if modele0 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := modele0.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "modele0", err.Error())
			}
			resp20 <- resp

		case resv := <-req21:
			if modele1 == nil {
				time.Sleep(time.Duration(10) * time.Millisecond) // spare time for initiate learner
			}
			message := resv.GetPayload()
			resp, err := modele1.Advance(message)
			if err != nil {
				log.Printf("[%s].Advance err: %s", "modele1", err.Error())
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
