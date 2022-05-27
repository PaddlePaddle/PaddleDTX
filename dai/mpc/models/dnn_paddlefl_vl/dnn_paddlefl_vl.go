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

package dnn_paddlefl_vl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/linear"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/psi"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common/csv"
	pbDnnVl "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc/learners/dnn_paddlefl_vl"
	"github.com/PaddlePaddle/PaddleDTX/dai/util/docker"
)

var (
	logger = logrus.WithField("module", "mpc.models.dnn_paddlefl_vl")
)

const PADDLEFL_TASK_TRAIN_SAMPLE_FILE = "samples-predict"
const PADDLEFL_TASK_TRAIN_SAMPLE_CONFUSED_FILE = "from-%s-to-%s-sample-predict"
const LOCAL_OUTPUT_FOLDER = "output/"

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
	modelStatusENVPrepare
	modelStatusStartPredict
	modelStatusEndPredict
	modelStatusStartRecovery
	modelStatusEndRecovery
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
	rpc         RpcHandler    // rpc is used to request remote mpc-node
	rh          ResultHandler // rh handles final result which is successful or failed
	fileRows    [][]string    // fileRows returned by psi.IntersectParts
	intersect   []string      // intersect returned by psi.IntersectParts

	predictPart          []float64 // local prediction part
	predictPartFromOther []float64 // prediction part from other party
	outcomes             []float64 // final result

	procMutex sync.Mutex
	status    modelStatusType

	fvSize [3]int64 // feature vector size for every learner in mpc
	lvSize int64    // label vector size

	role               int64    // role is the order number of the current learner in 'allParties'.
	allPaddleFLParties []string // allPaddleFLParties are all learners who participates in MPC, in increasing order of host
	containerName      string
	containerWorkspace string
	localWorkspace     string
	modelPath          string
	batchNum           int
}

// Advance does calculation with local parts of samples and communicates with other nodes in cluster to predict outcomes
// payload could be resolved by Model trained out by specific algorithm and samples
// We'd better call the method asynchronously avoid blocking the main go-routine
func (model *Model) Advance(payload []byte) (*pb.PredictResponse, error) {
	m := &pbDnnVl.PredictMessage{}
	err := proto.Unmarshal(payload, m)
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeParam, "failed to Unmarshal payload: %s", err.Error())
	}

	return model.advance(m)
}

// advance handles all kinds of message
func (model *Model) advance(message *pbDnnVl.PredictMessage) (*pb.PredictResponse, error) {
	mType := message.Type

	handleError := func(err error) {
		logger.WithField("error", err.Error()).Warning("failed to predict")
		res := &pbCom.PredictTaskResult{TaskID: model.id, ErrMsg: err.Error()}
		model.rh.SaveResult(res)
	}

	var ret *pb.PredictResponse
	switch mType {
	case pbDnnVl.MessageType_MsgPsiEnc: // local message
		encIDs, err := model.psi.EncryptSampleIDSet()
		if err != nil {
			go handleError(err)
			return nil, err
		}

		go func() {
			m := &pbDnnVl.PredictMessage{
				Type: pbDnnVl.MessageType_MsgPsiAskReEnc,
				VlLPsiReEncIDsReq: &pb.VLPsiReEncIDsRequest{
					EncIDs: encIDs,
				},
			}
			model.advance(m)
		}()
	case pbDnnVl.MessageType_MsgPsiAskReEnc:
		newMess := &pbDnnVl.PredictMessage{
			Type:              pbDnnVl.MessageType_MsgPsiReEnc,
			VlLPsiReEncIDsReq: message.VlLPsiReEncIDsReq,
		}

		done := true
		for _, party := range model.parties {
			reM, err := model.sendMessageWithRetry(newMess, party)
			if err != nil {
				go handleError(err)
				return nil, err
			}
			doneFlag, err := model.psi.SetReEncryptIDSet(party, reM.VlLPsiReEncIDsResp.ReEncIDs)
			if err != nil {
				go handleError(err)
				return nil, err
			}
			done = done && doneFlag
		}

		if done {
			go func() {
				m := &pbDnnVl.PredictMessage{
					Type: pbDnnVl.MessageType_MsgPsiIntersect,
				}
				model.advance(m)
			}()
		}
	case pbDnnVl.MessageType_MsgPsiReEnc:
		reEncIDs, err := model.psi.ReEncryptIDSet(message.From, message.VlLPsiReEncIDsReq.EncIDs)
		if err != nil {
			go handleError(err)
			return nil, err
		}

		retM := &pbDnnVl.PredictMessage{
			Type: pbDnnVl.MessageType_MsgPsiReEnc,
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
			return nil, err
		} else {
			go func() {
				m := &pbDnnVl.PredictMessage{
					Type: pbDnnVl.MessageType_MsgPsiIntersect,
				}
				model.advance(m)
			}()
		}

	case pbDnnVl.MessageType_MsgPsiIntersect: // local message
		done, newRows, intersect, err := model.psi.IntersectParts()
		if err != nil {
			go handleError(err)
			return nil, err
		}

		if done {
			model.status = modelStatusEndPSI
			model.fileRows = newRows
			model.intersect = intersect
			model.fvSize[model.role] = int64(len(newRows[0]))
			model.batchNum = len(newRows)

			go func() {
				m := &pbDnnVl.PredictMessage{
					Type: pbDnnVl.MessageType_MsgFLENVPrepare,
				}
				model.advance(m)
			}()
		}
	case pbDnnVl.MessageType_MsgFLENVPrepare: // local message
		model.procMutex.Lock()
		defer model.procMutex.Unlock()
		if model.status == modelStatusEndPSI {
			model.status = modelStatusENVPrepare

			status, err := docker.CheckRunningStatusByContainerName(model.containerName)
			if err != nil {
				go handleError(err)
				return nil, err
			}
			if status {
				m := &pbDnnVl.PredictMessage{
					Type: pbDnnVl.MessageType_MsgFLDataPrepare,
				}
				model.advance(m)
			} else {
				go handleError(errors.New("docker container is not running"))
				return nil, err
			}
		}
	case pbDnnVl.MessageType_MsgFLDataPrepare:
		err := model.exportRawSamples()
		if err != nil {
			handleError(err)
			return nil, err
		}

		go func() {
			m := &pbDnnVl.PredictMessage{
				Type: pbDnnVl.MessageType_MsgFLDataGenerate,
			}
			model.advance(m)
		}()

	case pbDnnVl.MessageType_MsgFLDataGenerate:
		// three files will be generated by mpc process
		var outFiles []string
		for _, address := range model.parties {
			fileName := model.containerWorkspace + fmt.Sprintf(PADDLEFL_TASK_TRAIN_SAMPLE_CONFUSED_FILE, model.address, address)
			outFiles = append(outFiles, fileName)
		}
		fileName := model.containerWorkspace + fmt.Sprintf(PADDLEFL_TASK_TRAIN_SAMPLE_CONFUSED_FILE, model.address, model.address)
		outFiles = append(outFiles, fileName)
		sort.Strings(outFiles)

		// raw samples
		sampleFile := model.containerWorkspace + PADDLEFL_TASK_TRAIN_SAMPLE_FILE
		cmd := []string{"python3.8", "process_data.py", "--func", "encrypt_data", "--input", sampleFile, "--out", strings.Join(outFiles, ",")}
		logger.Info(cmd)
		err := docker.RunCommand(cmd, model.containerName)
		if err != nil {
			handleError(err)
			return nil, err
		}

		// must delete raw data
		defer func() {
			sampleFile = model.localWorkspace + PADDLEFL_TASK_TRAIN_SAMPLE_FILE
			err = os.Remove(sampleFile)
		}()

		go func() {
			m := &pbDnnVl.PredictMessage{
				Type: pbDnnVl.MessageType_MsgFLDataSend,
			}
			model.advance(m)
		}()
	case pbDnnVl.MessageType_MsgFLDataSend:
		for _, party := range model.parties {
			shareFile := fmt.Sprintf(PADDLEFL_TASK_TRAIN_SAMPLE_CONFUSED_FILE, model.address, party)
			_, err := os.Stat(model.localWorkspace + shareFile)
			if err != nil {
				handleError(err)
				return nil, err
			}

			f, err := os.OpenFile(model.localWorkspace+shareFile, os.O_RDONLY, 0600)
			defer f.Close()
			if err != nil {
				handleError(err)
				return nil, err
			}
			contentByte, err := ioutil.ReadAll(f)
			if err != nil {
				handleError(err)
				return nil, err
			}

			newMess := &pbDnnVl.PredictMessage{
				Type:          pbDnnVl.MessageType_MsgFLDataExchange,
				Aby3ShareFile: []byte(shareFile),
				Aby3ShareData: contentByte,
				VecSize:       uint64(model.fvSize[model.role]),
				Role:          uint64(model.role),
			}
			_, err = model.sendMessageWithRetry(newMess, party)
			if err != nil {
				go handleError(err)
				return nil, err
			}
			//os.Remove(model.localWorkspace + shareFile)
		}

		go func() {
			m := &pbDnnVl.PredictMessage{
				Type: pbDnnVl.MessageType_MsgFLDataStatus,
			}
			model.advance(m)
		}()

	case pbDnnVl.MessageType_MsgFLDataExchange:
		// receive confused data from other party
		f, err := os.Create(model.localWorkspace + string(message.Aby3ShareFile))
		defer f.Close()
		if err != nil {
			handleError(err)
			return nil, err
		}
		model.fvSize[int64(message.Role)] = int64(message.VecSize)

		_, err = f.Write(message.Aby3ShareData)
		if err != nil {
			handleError(err)
			return nil, err
		}
		ret = &pb.PredictResponse{
			TaskID: model.id,
		}

		go func() {
			m := &pbDnnVl.PredictMessage{
				Type: pbDnnVl.MessageType_MsgFLDataStatus,
			}
			model.advance(m)
		}()
	case pbDnnVl.MessageType_MsgFLDataStatus: // local message
		fileInfoList, err := ioutil.ReadDir(model.localWorkspace)
		if err != nil {
			handleError(err)
			return nil, err
		}
		num := 0
		for i := range fileInfoList {
			infos := strings.Split(fileInfoList[i].Name(), "-")
			if len(infos) == 6 && infos[len(infos)-3] == model.address && infos[len(infos)-1] == "predict" {
				num++
			}
		}
		if num < 3 {
			return nil, err
		}

		go func() {
			m := &pbDnnVl.PredictMessage{
				Type: pbDnnVl.MessageType_MsgPredictHup,
			}
			model.advance(m)
		}()
	case pbDnnVl.MessageType_MsgPredictHup:

		model.procMutex.Lock()
		defer model.procMutex.Unlock()
		if model.status == modelStatusENVPrepare {
			model.status = modelStatusStartPredict

			var fileNames []string
			for _, address := range model.parties {
				fileName := model.containerWorkspace + fmt.Sprintf(PADDLEFL_TASK_TRAIN_SAMPLE_CONFUSED_FILE, address, model.address)
				fileNames = append(fileNames, fileName)
			}
			fileName := model.containerWorkspace + fmt.Sprintf(PADDLEFL_TASK_TRAIN_SAMPLE_CONFUSED_FILE, model.address, model.address)
			fileNames = append(fileNames, fileName)
			sort.Strings(fileNames)

			var sizes string
			logger.Info(model.fvSize[0], model.fvSize[1], model.fvSize[2])
			sizes = strconv.FormatInt(model.fvSize[0], 10) + "," + strconv.FormatInt(model.fvSize[1], 10) + "," + strconv.FormatInt(model.fvSize[2], 10)

			os.MkdirAll(model.localWorkspace+LOCAL_OUTPUT_FOLDER, os.ModePerm)

			outputFile := model.containerWorkspace + LOCAL_OUTPUT_FOLDER + "out.part" + strconv.FormatInt(model.role, 10)
			cmd := []string{"python3.8", "train.py",
				"--func", "infer",
				"--samples", strings.Join(fileNames, ","),
				"--role", strconv.Itoa(int(model.role)),
				"--parts", strings.Join(model.allPaddleFLParties, ","),
				"--parts_size", sizes,
				"--batch_num", strconv.Itoa(model.batchNum - 1),
				"--output_file", outputFile,
				"--model_dir", model.modelPath,
			}
			logger.WithFields(logrus.Fields{
				"paddlefl role": model.role,
			}).Infof("model[%s] execute docker cmd [%s]", model.id, strings.Join(cmd, " "))

			err := docker.RunCommand(cmd, model.containerName)

			if err != nil {
				handleError(err)
				return nil, err
			}
			model.status = modelStatusEndPredict
		}

		go func() {
			m := &pbDnnVl.PredictMessage{
				Type: pbDnnVl.MessageType_MsgPredictResultSend,
			}
			model.advance(m)
		}()

	case pbDnnVl.MessageType_MsgPredictResultSend:

		if model.params.GetIsTagPart() {
			// current node is the receiver
			go func() {
				m := &pbDnnVl.PredictMessage{
					Type: pbDnnVl.MessageType_MsgPredictResultStatus,
				}
				model.advance(m)
			}()
		} else {
			fileName := "out.part" + strconv.FormatInt(model.role, 10)
			outputFile := model.localWorkspace + LOCAL_OUTPUT_FOLDER + "out.part" + strconv.FormatInt(model.role, 10)
			f, err := os.OpenFile(outputFile, os.O_RDONLY, 0600)
			defer f.Close()
			if err != nil {
				handleError(err)
				return nil, err
			}

			contentByte, err := ioutil.ReadAll(f)
			if err != nil {
				handleError(err)
				return nil, err
			}

			for _, party := range model.parties {
				newMess := &pbDnnVl.PredictMessage{
					Type:          pbDnnVl.MessageType_MsgPredictResultExchange,
					Aby3ShareFile: []byte(fileName),
					Aby3ShareData: contentByte,
				}
				_, err = model.sendMessageWithRetry(newMess, party)
				if err != nil {
					go handleError(err)
					return nil, err
				}
			}
		}

	case pbDnnVl.MessageType_MsgPredictResultExchange:
		outputFolder := model.localWorkspace + LOCAL_OUTPUT_FOLDER

		f, err := os.Create(outputFolder + string(message.Aby3ShareFile))
		defer f.Close()
		if err != nil {
			handleError(err)
			return nil, err
		}
		_, err = f.Write(message.Aby3ShareData)
		if err != nil {
			handleError(err)
			return nil, err
		}
		ret = &pb.PredictResponse{
			TaskID: model.id,
		}
		go func() {
			m := &pbDnnVl.PredictMessage{
				Type: pbDnnVl.MessageType_MsgPredictResultStatus,
			}
			model.advance(m)
		}()

	case pbDnnVl.MessageType_MsgPredictResultStatus:
		if model.params.GetIsTagPart() {
			outputFolder := model.localWorkspace + LOCAL_OUTPUT_FOLDER
			fileInfoList, err := ioutil.ReadDir(outputFolder)
			if err != nil {
				handleError(err)
				return nil, err
			}

			if len(fileInfoList) < 3 {
				return nil, err
			}

			model.procMutex.Lock()
			defer model.procMutex.Unlock()
			if model.status == modelStatusEndPredict {
				model.status = modelStatusStartRecovery
			} else {
				return nil, err
			}

			go func() {
				m := &pbDnnVl.PredictMessage{
					Type: pbDnnVl.MessageType_MsgPredictResultRecovery,
				}
				model.advance(m)
			}()
		} else {
			return nil, nil
		}
	case pbDnnVl.MessageType_MsgPredictResultRecovery:
		outputFile := model.containerWorkspace + LOCAL_OUTPUT_FOLDER + "out"
		realOut := model.containerWorkspace + LOCAL_OUTPUT_FOLDER + "real"
		cmd := []string{"python3.8", "process_data.py", "--func", "decrypt_data", "--path", outputFile, "--out", realOut}
		logger.Info(cmd)
		err := docker.RunCommand(cmd, model.containerName)
		if err != nil {
			handleError(err)
			return nil, err
		}

		newMess := &pbDnnVl.PredictMessage{
			Type: pbDnnVl.MessageType_MsgPredictStop,
		}
		for _, party := range model.parties {
			_, err := model.sendMessageWithRetry(newMess, party)
			if err != nil {
				go handleError(err)
				return nil, err
			}
		}

		c := []float64{}
		f, err := os.OpenFile(model.localWorkspace+LOCAL_OUTPUT_FOLDER+"real", os.O_RDONLY, 0600)
		defer f.Close()
		if err != nil {
			handleError(err)
			return nil, err
		}
		rd := bufio.NewReader(f)
		for {
			line, err := rd.ReadString('\n')
			line = strings.TrimSpace(line)
			if err != nil || io.EOF == err {
				break
			}
			one, err := strconv.ParseFloat(line, 64)
			logger.Info(line,one, err)
			c = append(c, one)
		}
		outs, err := vl_common.PredictResultToBytes(model.params.IdName, model.intersect, c)
		model.status = modelStatusEndRecovery

		go func() {
			model.rh.SaveResult(&pbCom.PredictTaskResult{
				TaskID:   model.id,
				Success:  true,
				Outcomes: outs,
			})
		}()
	case pbDnnVl.MessageType_MsgPredictStop:
		logger.WithField("IsTagPart", model.params.IsTagPart).Infof("model[%s] finished prediction.", model.id)
		model.rh.SaveResult(&pbCom.PredictTaskResult{
			TaskID:  model.id,
			Success: true,
		})
		ret = &pb.PredictResponse{
			TaskID: model.id,
		}
	}

	logger.Infof("model[%s] finished advance. message %s", model.id, message.Type.String())
	return ret, nil
}

func (model *Model) predictLocalPart() (predictPart []float64, err error) {
	predictPart, err = linear.PredictLocalPart(model.fileRows, model.params)
	if err != nil {
		return
	}
	model.predictPart = predictPart

	return
}

func (model *Model) setPredictPartFromOther(predictPart []float64) {
	model.predictPartFromOther = predictPart
}

func (model *Model) deStandardizeOutput() (done bool, outcomes []float64) {
	if len(model.predictPart) == 0 || len(model.predictPartFromOther) == 0 {
		return
	}
	outcomes = linear.DeStandardizeOutput(model.params, model.predictPart, model.predictPartFromOther)
	model.outcomes = outcomes
	done = true

	return
}

// sendMessageWithRetry sends message to remote mpc-node
// retries 2 times at most
func (model *Model) sendMessageWithRetry(message *pbDnnVl.PredictMessage, address string) (*pbDnnVl.PredictMessage, error) {
	times := 3

	var m *pbDnnVl.PredictMessage
	var err error
	for i := 0; i < times; i++ {
		m, err = model.sendMessage(message, address)
		if err == nil {
			break
		}
	}

	return m, err
}

// sendMessage sends message to remote mpc-node
func (model *Model) sendMessage(message *pbDnnVl.PredictMessage, address string) (*pbDnnVl.PredictMessage, error) {
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

	m := &pbDnnVl.PredictMessage{}
	if len(resp.Payload) != 0 {
		err := proto.Unmarshal(resp.Payload, m)
		if err != nil {
			return nil, errorx.New(errcodes.ErrCodeInternal, "failed to Unmarshal payload[%s] from[%s] and err is[%s] ", string(resp.Payload), address, err.Error())
		}
	}
	return m, nil
}

// NewModel returns a VerticalLinearRegression Model
// id is the assigned id for Model
// samplesFile is sample file content for prediction
// address indicates local mpc-node
// parties are other models who participates in MPC, assigned with mpc-node address usually
// paddleFLParams are array of nodes in mpc network, and the role of the current node.
// rpc is used to request remote mpc-node
// rh handles final result which is successful or failed
// params are parameters for training model
func NewModel(id string, address string,
	params *pbCom.TrainModels, samplesFile []byte,
	parties []string, paddleFLParams *pbCom.PaddleFLParams, rpc RpcHandler, rh ResultHandler) (*Model, error) {

	p, err := psi.NewVLPSIByPairs(address, samplesFile, params.GetIdName(), parties)
	if err != nil {
		return nil, err
	}

	// the name of paddlefl container is the domain name of the node's address
	role := paddleFLParams.Role
	u := strings.Split(paddleFLParams.Nodes[role], ":")
	containerName := u[0]

	model := &Model{
		id:                 id,
		algo:               pbCom.Algorithm_LINEAR_REGRESSION_VL,
		samplesFile:        samplesFile,
		address:            address,
		parties:            parties,
		params:             params,
		psi:                p,
		rpc:                rpc,
		rh:                 rh,
		status:             modelStatusStartPSI,
		role:               int64(role),
		containerName:      containerName,
		allPaddleFLParties: paddleFLParams.Nodes,
		fvSize:             [3]int64{},
		containerWorkspace: fmt.Sprintf(docker.PADDLEFL_CONTAINER_WORKSPACE, id),
		localWorkspace:     fmt.Sprintf(docker.PADDLEFL_LOCAL_WORKSPACE, id),
		modelPath:          params.Path,
	}

	go func() {
		// Interim solutions to consistency issues
		time.Sleep(1 * time.Second)
		m := &pbDnnVl.PredictMessage{
			Type: pbDnnVl.MessageType_MsgPsiEnc,
		}
		model.advance(m)
	}()

	return model, nil
}

func (m *Model) exportRawSamples() error {
	err := os.MkdirAll(m.localWorkspace, os.ModePerm)
	if err != nil {
		return err
	}

	featureFile := m.localWorkspace + PADDLEFL_TASK_TRAIN_SAMPLE_FILE
	err = csv.WriteRowsToFile(m.fileRows, featureFile)
	return err
}
