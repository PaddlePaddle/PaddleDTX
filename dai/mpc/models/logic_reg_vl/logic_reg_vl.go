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

package logic_reg_vl

import (
	"sync"
	"time"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/logic"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/psi"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
	pbLogicRegVl "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc/learners/logic_reg_vl"
)

var (
	logger = logrus.WithField("module", "mpc.models.logic_reg_vl")
)

// PSI is for vertical learning,
// initialized at the beginning of training by Learner
type PSI interface {
	// EncryptSampleIDSet to encrypt local IDs
	EncryptSampleIDSet() ([]byte, error)

	// SetReEncryptIDSet sets re-encrypted IDs from other party,
	// and tries to calculate final re-encrypted IDs
	// returns True if calculation is Done, otherwise False if still waiting for others' parts
	// returns Error if any mistake happens
	SetReEncryptIDSet(party string, reEncIDs []byte) (bool, error)

	// ReEncryptIDSet to encrypt encrypted IDs for other party
	ReEncryptIDSet(party string, encIDs []byte) ([]byte, error)

	// SetOtherFinalReEncryptIDSet sets final re-encrypted IDs of other party
	SetOtherFinalReEncryptIDSet(party string, reEncIDs []byte) error

	// IntersectParts tries to calculate intersection with all parties' samples
	// returns True with final result if calculation is Done, otherwise False if still waiting for others' samples
	// returns Error if any mistake happens
	// You'd better call it when SetReEncryptIDSet returns Done or SetOtherFinalReEncryptIDSet finishes
	IntersectParts() (bool, [][]string, []string, error)
}

// RpcHandler used to request remote mpc-node
type RpcHandler interface {
	StepPredict(req *pb.PredictRequest, peerName string) (*pb.PredictResponse, error)

	// StepPredictWithRetry sends prediction message to remote mpc-node
	// retries 2 times at most
	// inteSec indicates the interval between retry requests, in seconds
	StepPredictWithRetry(req *pb.PredictRequest, peerName string, times int, inteSec int64) (*pb.PredictResponse, error)
}

// ResultHandler handles final result which is successful or failed
// Should be called when prediction finished
type ResultHandler interface {
	SaveResult(*pbCom.PredictTaskResult)
}

type modelStatusType uint8

const (
	modelStatusStartPSI modelStatusType = iota
	modelStatusEndPSI
	modelStatusStartPredict
	modelStatusEndPredict
)

// Model was trained out by a Learner,
// and participates in the multi-parts-calculation during prediction process
// If input different parts of a sample into Models on different mpc-nodes, you'll get final predicting result after some time of multi-parts-calculation
type Model struct {
	id          string
	algo        pbCom.Algorithm
	address     string   // address indicates local mpc-node
	parties     []string // parties are other models who participates in MPC, assigned with mpc-node address usually
	params      *pbCom.TrainModels
	samplesFile []byte // sample file content for prediction
	psi         PSI
	rpc         RpcHandler    // pc is used to request remote mpc-node
	rh          ResultHandler // rh handles final result which is successful or failed
	fileRows    [][]string    // fileRows returned by psi.IntersectParts
	intersect   []string      // intersect returned by psi.IntersectParts

	predictPart          []float64 // local prediction part
	predictPartFromOther []float64 // prediction part from other party

	outcomes []float64 // final result

	procMutex sync.Mutex
	status    modelStatusType
}

// Advance does calculation with local parts of samples and communicates with other nodes in cluster to predict outcomes
// payload could be resolved by Model trained out by specific algorithm and samples
// We'd better call the method asynchronously avoid blocking the main go-routine
func (model *Model) Advance(payload []byte) (*pb.PredictResponse, error) {
	m := &pbLogicRegVl.PredictMessage{}
	err := proto.Unmarshal(payload, m)
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeParam, "failed to Unmarshal payload: %s", err.Error())
	}

	return model.advance(m)
}

// advance handles all kinds of message
func (model *Model) advance(message *pbLogicRegVl.PredictMessage) (*pb.PredictResponse, error) {
	mType := message.Type

	handleError := func(err error) {
		logger.WithField("error", err.Error()).Warning("failed to predict")
		res := &pbCom.PredictTaskResult{TaskID: model.id, ErrMsg: err.Error()}
		model.rh.SaveResult(res)
	}

	var ret *pb.PredictResponse
	switch mType {
	case pbLogicRegVl.MessageType_MsgPsiEnc: // local message
		encIDs, err := model.psi.EncryptSampleIDSet()
		if err != nil {
			go handleError(err)
			return nil, err
		}

		go func() {
			m := &pbLogicRegVl.PredictMessage{
				Type: pbLogicRegVl.MessageType_MsgPsiAskReEnc,
				VlLPsiReEncIDsReq: &pb.VLPsiReEncIDsRequest{
					EncIDs: encIDs,
				},
			}
			model.advance(m)
		}()

	case pbLogicRegVl.MessageType_MsgPsiAskReEnc: // local message
		newMess := &pbLogicRegVl.PredictMessage{
			Type:              pbLogicRegVl.MessageType_MsgPsiReEnc,
			VlLPsiReEncIDsReq: message.VlLPsiReEncIDsReq,
		}
		reM, err := model.sendMessageWithRetry(newMess, model.parties[0])
		if err != nil {
			go handleError(err)
			return nil, err
		}

		done, err := model.psi.SetReEncryptIDSet(model.parties[0], reM.VlLPsiReEncIDsResp.ReEncIDs)
		if err != nil {
			go handleError(err)
			return nil, err
		}

		if done {
			go func() {
				m := &pbLogicRegVl.PredictMessage{
					Type: pbLogicRegVl.MessageType_MsgPsiIntersect,
				}
				model.advance(m)
			}()
		}

	case pbLogicRegVl.MessageType_MsgPsiReEnc:
		reEncIDs, err := model.psi.ReEncryptIDSet(message.From, message.VlLPsiReEncIDsReq.EncIDs)
		if err != nil {
			go handleError(err)
			return nil, err
		}

		retM := &pbLogicRegVl.PredictMessage{
			Type: pbLogicRegVl.MessageType_MsgPsiReEnc,
			To:   message.From,
			From: model.address,
			VlLPsiReEncIDsResp: &pb.VLPsiReEncIDsResponse{
				TaskID:   model.id,
				ReEncIDs: reEncIDs,
			},
		}
		payload, err := proto.Marshal(retM)
		if err != nil {
			err = errorx.New(errcodes.ErrCodeInternal, "failed to Marshal payload: %s", err.Error())
			go handleError(err)
			return nil, err
		}

		ret = &pb.PredictResponse{
			TaskID:  model.id,
			Payload: payload,
		}

		err = model.psi.SetOtherFinalReEncryptIDSet(message.From, reEncIDs)
		if err != nil {
			go handleError(err)
		} else {
			go func() {
				m := &pbLogicRegVl.PredictMessage{
					Type: pbLogicRegVl.MessageType_MsgPsiIntersect,
				}
				model.advance(m)
			}()
		}

	case pbLogicRegVl.MessageType_MsgPsiIntersect: // local message
		done, newRows, intersect, err := model.psi.IntersectParts()
		if err != nil {
			go handleError(err)
			return nil, err
		}

		if done {
			model.fileRows = newRows
			model.intersect = intersect
			model.status = modelStatusEndPSI
			go func() {
				m := &pbLogicRegVl.PredictMessage{
					Type: pbLogicRegVl.MessageType_MsgPredictHup,
				}
				model.advance(m)
			}()
		}
	case pbLogicRegVl.MessageType_MsgPredictHup: // local message
		model.procMutex.Lock()
		defer model.procMutex.Unlock()
		if modelStatusEndPSI == model.status {
			model.status = modelStatusStartPredict
			predictPart, err := model.predictLocalPart()
			if err != nil {
				go handleError(err)
				return nil, err
			}

			// The party who has target tag needs the PredictPart from the party who hasn't target tag
			// So the party who hasn't target sends message , and the party who has target tag waits
			if !model.params.IsTagPart {
				newMess := &pbLogicRegVl.PredictMessage{
					Type:        pbLogicRegVl.MessageType_MsgPredictPart,
					PredictPart: predictPart,
				}
				_, err = model.sendMessageWithRetry(newMess, model.parties[0])
				if err != nil {
					go handleError(err)
					return nil, err
				}
			}

			go func() {
				m := &pbLogicRegVl.PredictMessage{
					Type: pbLogicRegVl.MessageType_MsgPredictFinal,
				}
				model.advance(m)
			}()
		}

	case pbLogicRegVl.MessageType_MsgPredictPart:
		partFromOther := message.PredictPart
		model.setPredictPartFromOther(partFromOther)
		ret = &pb.PredictResponse{
			TaskID: model.id,
		}

		go func() {
			m := &pbLogicRegVl.PredictMessage{
				Type: pbLogicRegVl.MessageType_MsgPredictFinal,
			}
			model.advance(m)
		}()

	case pbLogicRegVl.MessageType_MsgPredictFinal: // local message
		model.procMutex.Lock()
		defer model.procMutex.Unlock()

		// The party who has target tag needs the PredictPart from the party who hasn't target tag.
		// So the party who hasn't target tag stops prediction,
		// and the party who has target tag waits for the PredictPart and fullfill the prediction.

		// lock to make sure that calculate and save outcomes for just once
		if modelStatusStartPredict == model.status {
			if !model.params.IsTagPart {
				model.status = modelStatusEndPredict
				go func() {
					logger.WithField("IsTagPart", model.params.IsTagPart).Infof("model[%s] finished prediction.", model.id)
					model.rh.SaveResult(&pbCom.PredictTaskResult{
						TaskID:  model.id,
						Success: true,
					})
				}()

			} else {
				done, outcomes := model.calRealPredictValue()
				if done {
					model.status = modelStatusEndPredict
					outs, err := vl_common.PredictResultToBytes(model.params.IdName, model.intersect, outcomes)
					if err != nil {
						go handleError(err)
						return nil, err
					}
					go func() {
						logger.WithField("IsTagPart", model.params.IsTagPart).Infof("model[%s] finish prediction and outcomes are[%v].", model.id, outcomes)
						model.rh.SaveResult(&pbCom.PredictTaskResult{
							TaskID:   model.id,
							Success:  true,
							Outcomes: outs,
						})
					}()
				}
			}
		}
	}

	logger.WithFields(logrus.Fields{
		"address": model.address,
	}).Infof("model[%s] finished advance. message %s", model.id, message.Type.String())
	return ret, nil
}

func (model *Model) predictLocalPart() (predictPart []float64, err error) {
	predictPart, err = logic.PredictLocalPart(model.fileRows, model.params)
	if err != nil {
		return
	}
	model.predictPart = predictPart

	return
}

func (model *Model) setPredictPartFromOther(predictPart []float64) {
	model.predictPartFromOther = predictPart
}

func (model *Model) calRealPredictValue() (done bool, outcomes []float64) {
	if len(model.predictPart) == 0 || len(model.predictPartFromOther) == 0 {
		return
	}
	outcomes = logic.CalRealPredictValue(model.predictPart, model.predictPartFromOther)
	model.outcomes = outcomes
	done = true

	return
}

// sendMessageWithRetry sends message to remote mpc-node
// retries 2 times at most
func (model *Model) sendMessageWithRetry(message *pbLogicRegVl.PredictMessage, address string) (*pbLogicRegVl.PredictMessage, error) {
	times := 3

	var m *pbLogicRegVl.PredictMessage
	var err error
	for i := 0; i < times; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second)
		}
		m, err = model.sendMessage(message, address)
		if err == nil {
			break
		}
	}

	return m, err
}

// sendMessage sends message to remote mpc-node
func (model *Model) sendMessage(message *pbLogicRegVl.PredictMessage, address string) (*pbLogicRegVl.PredictMessage, error) {
	message.From = model.address
	message.To = address

	payload, err := proto.Marshal(message)
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeInternal, "failed to Marshal payload: %s", err.Error())
	}

	predictReq := &pb.PredictRequest{
		TaskID:  model.id,
		Algo:    model.algo,
		Payload: payload,
	}
	resp, err := model.rpc.StepPredict(predictReq, address)
	if err != nil {
		return nil, err
	}

	m := &pbLogicRegVl.PredictMessage{}
	if len(resp.Payload) != 0 {
		err := proto.Unmarshal(resp.Payload, m)
		if err != nil {
			return nil, errorx.New(errcodes.ErrCodeInternal, "failed to Unmarshal payload[%s] from[%s] and err is[%s] ", string(resp.Payload), address, err.Error())
		}
	}
	return m, nil
}

// NewModel returns a VerticalLogicRegression Model
// id is the assigned id for Model
// samplesFile is sample file content for prediction
// address indicates local mpc-node
// parties are other models who participates in MPC, assigned with mpc-node address usually
// rpc is used to request remote mpc-node
// rh handles final result which is successful or failed
// params are parameters for training model
func NewModel(id string, address string,
	params *pbCom.TrainModels, samplesFile []byte,
	parties []string, rpc RpcHandler, rh ResultHandler) (*Model, error) {

	p, err := psi.NewVLTowPartsPSI(address, samplesFile, params.GetIdName(), parties)
	if err != nil {
		return nil, err
	}

	model := &Model{
		id:          id,
		algo:        pbCom.Algorithm_LOGIC_REGRESSION_VL,
		samplesFile: samplesFile,
		address:     address,
		parties:     parties,
		params:      params,
		psi:         p,
		rpc:         rpc,
		rh:          rh,
		status:      modelStatusStartPSI,
	}

	go func() {
		m := &pbLogicRegVl.PredictMessage{
			Type: pbLogicRegVl.MessageType_MsgPsiEnc,
		}
		model.advance(m)
	}()

	return model, nil
}
