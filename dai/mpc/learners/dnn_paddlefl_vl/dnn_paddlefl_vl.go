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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	crypCom "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common/csv"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/psi"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
	pbDnnVl "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc/learners/dnn_paddlefl_vl"
	"github.com/PaddlePaddle/PaddleDTX/dai/util/docker"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

var (
	logger = logrus.WithField("module", "mpc.learners.dnn_paddlefl_vl")
)

const PADDLEFL_TASK_SAMPLE_FILE = "samples"
const PADDLEFL_TASK_LABEL_FILE = "labels"

// Distinguish files by prefix name
const PADDLEFL_TASK_CONFUSED_SAMPLE_FILE = "sample-%s-to-%s"     // source address to destination address
const PADDLEFL_TASK_CONFUSED_LABEL_FILE = "label-%s"             // destination address

const LOCAL_SAMPLE_FOLDER = "local-sample/"
const LOCAL_SAMPLE_MPC_FOLDER = "mpc-sample/"
const LOCAL_MODEL_FOLDER = "model/"

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
	StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error)
}

// ResultHandler handles final result which is successful or failed
// Should be called when learning finished
type ResultHandler interface {
	SaveResult(*pbCom.TrainTaskResult)
}

type learnerStatusType uint8

const (
	learnerStatusStartPSI learnerStatusType = iota
	learnerStatusEndPSI
	learnerStatusENVPrepare
	learnerStatusStartTrain
	learnerStatusEndTrain
)

type Learner struct {
	id          string
	algo        pbCom.Algorithm
	address     string               // address indicates local mpc-node
	parties     []string             // parties are other learners who participates in MPC, assigned with mpc-node address usually
	homoPriv    *paillier.PrivateKey // homomorphic private key
	homoPub     []byte               // homomorphic public key for transfer
	trainParams *pbCom.TrainParams
	samplesFile []byte // sample file content for training model
	psi         PSI
	procMutex   sync.Mutex
	rpc         RpcHandler    // rpc is used to request remote mpc-node
	rh          ResultHandler // rh handles final result which is successful or failed

	status learnerStatusType

	featureRows [][]string // Actual samples for training, converted by the result of psi.IntersectParts
	labelRows   [][]string
	fvSize      [3]int64 // feature vector size for every learner in mpc
	lvSize      int64    // label vector size

	role               int64    // role is the order number of the current learner in 'allParties'.
	allPaddleFLParties []string // allPaddleFLParties are all learners who participates in MPC, in increasing order of host
	containerName      string
	containerWorkspace string
	localWorkspace     string
	batchNum           int
}

func (l *Learner) Advance(payload []byte) (*pb.TrainResponse, error) {
	m := &pbDnnVl.Message{}
	err := proto.Unmarshal(payload, m)
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeParam, "failed to Unmarshal payload: %s", err.Error())
	}
	return l.advance(m)
}

// advance handles all kinds of message
func (l *Learner) advance(message *pbDnnVl.Message) (*pb.TrainResponse, error) {
	mType := message.Type

	handleError := func(err error) {
		logger.WithField("error", err.Error()).Warning("failed to train out a model")
		res := &pbCom.TrainTaskResult{TaskID: l.id, ErrMsg: err.Error()}
		l.rh.SaveResult(res)
	}

	var ret *pb.TrainResponse
	switch mType {
	case pbDnnVl.MessageType_MsgPsiEnc: // local message
		encIDs, err := l.psi.EncryptSampleIDSet()
		if err != nil {
			go handleError(err)
			return nil, err
		}

		go func() {
			m := &pbDnnVl.Message{
				Type: pbDnnVl.MessageType_MsgPsiAskReEnc,
				VlLPsiReEncIDsReq: &pb.VLPsiReEncIDsRequest{
					TaskID: l.id,
					EncIDs: encIDs,
				},
			}
			l.advance(m)
		}()
	case pbDnnVl.MessageType_MsgPsiAskReEnc:
		newMess := &pbDnnVl.Message{
			Type:              pbDnnVl.MessageType_MsgPsiReEnc,
			VlLPsiReEncIDsReq: message.VlLPsiReEncIDsReq,
		}
		done := true
		for _, party := range l.parties {
			reM, err := l.sendMessageWithRetry(newMess, party)
			if err != nil {
				go handleError(err)
				return nil, err
			}
			doneFlag, err := l.psi.SetReEncryptIDSet(party, reM.VlLPsiReEncIDsResp.ReEncIDs)
			if err != nil {
				go handleError(err)
				return nil, err
			}
			done = done && doneFlag
		}
		if done {
			go func() {
				m := &pbDnnVl.Message{
					Type: pbDnnVl.MessageType_MsgPsiIntersect,
				}
				l.advance(m)
			}()
		}

	case pbDnnVl.MessageType_MsgPsiReEnc:
		reEncIDs, err := l.psi.ReEncryptIDSet(message.From, message.VlLPsiReEncIDsReq.EncIDs)
		if err != nil {
			go handleError(err)
			return nil, err
		}

		retM := &pbDnnVl.Message{
			Type: pbDnnVl.MessageType_MsgPsiReEnc,
			To:   message.From,
			From: l.address,
			VlLPsiReEncIDsResp: &pb.VLPsiReEncIDsResponse{
				TaskID:   l.id,
				ReEncIDs: reEncIDs,
			},
		}
		payload, err := proto.Marshal(retM)
		if err != nil {
			err = errorx.New(errcodes.ErrCodeInternal, "failed to Marshal payload: %s", err.Error())
			go handleError(err)
			return nil, err
		}

		ret = &pb.TrainResponse{
			TaskID:  l.id,
			Payload: payload,
		}

		err = l.psi.SetOtherFinalReEncryptIDSet(message.From, reEncIDs)
		if err != nil {
			go handleError(err)
		} else {
			go func() {
				m := &pbDnnVl.Message{
					Type: pbDnnVl.MessageType_MsgPsiIntersect,
				}
				l.advance(m)
			}()
		}
	case pbDnnVl.MessageType_MsgPsiIntersect: // local message
		done, newRows, _, err := l.psi.IntersectParts()
		if err != nil {
			go handleError(err)
			return nil, err
		}

		if done {
			l.status = learnerStatusEndPSI
			l.setSamples(newRows)
			l.batchNum = len(newRows)

			go func() {
				m := &pbDnnVl.Message{
					Type: pbDnnVl.MessageType_MsgFLENVPrepare,
				}
				l.advance(m)
			}()
		}
	case pbDnnVl.MessageType_MsgFLENVPrepare: // local message
		l.procMutex.Lock()
		defer l.procMutex.Unlock()
		if l.status == learnerStatusEndPSI {
			l.status = learnerStatusENVPrepare

			status, err := docker.CheckRunningStatusByContainerName(l.containerName)
			if err != nil {
				go handleError(err)
				return nil, err
			}
			if status {
				go func() {
					m := &pbDnnVl.Message{
						Type: pbDnnVl.MessageType_MsgFLDataPrepare,
					}
					l.advance(m)
				}()
			} else {
				go handleError(errors.New("docker container is not running"))
				return nil, err
			}
		}
	case pbDnnVl.MessageType_MsgFLDataPrepare: // local message
		err := l.exportRawSamples()
		if err != nil {
			handleError(err)
			return nil, err
		}

		go func() {
			m := &pbDnnVl.Message{
				Type: pbDnnVl.MessageType_MsgFLDataGenerate,
			}
			l.advance(m)
		}()
	case pbDnnVl.MessageType_MsgFLDataGenerate: // local message
		// One raw file will be processed into three mpc files.

		// raw file
		sampleFolder := l.containerWorkspace + LOCAL_SAMPLE_FOLDER
		sampleFile := sampleFolder + PADDLEFL_TASK_SAMPLE_FILE

		// three mpc files
		var mpcFiles []string
		for _, address := range l.parties {
			fileName := sampleFolder + fmt.Sprintf(PADDLEFL_TASK_CONFUSED_SAMPLE_FILE, l.address, address)
			mpcFiles = append(mpcFiles, fileName)
		}
		fileName := sampleFolder + fmt.Sprintf(PADDLEFL_TASK_CONFUSED_SAMPLE_FILE, l.address, l.address)
		mpcFiles = append(mpcFiles, fileName)
		sort.Strings(mpcFiles)

		commands := []string{"python3.8", "process_data.py", "--func", "encrypt_data", "--input", sampleFile, "--out", strings.Join(mpcFiles, ",")}
		logger.Debug("PaddleFL container exec commands:", strings.Join(commands, " "))

		err := docker.RunCommand(commands, l.containerName)
		if err != nil {
			handleError(err)
			return nil, err
		}

		// must delete raw data
		defer func() {
			sampleFile = l.localWorkspace + LOCAL_SAMPLE_FOLDER + PADDLEFL_TASK_SAMPLE_FILE
			os.Remove(sampleFile)
		}()


		if l.trainParams.GetIsTagPart() {
			// raw file
			labelFile := sampleFolder + PADDLEFL_TASK_LABEL_FILE

			// three mpc files
			var outFiles []string
			for _, address := range l.parties {
				fileName := sampleFolder + fmt.Sprintf(PADDLEFL_TASK_CONFUSED_LABEL_FILE, address)
				outFiles = append(outFiles, fileName)
			}
			fileName := sampleFolder + fmt.Sprintf(PADDLEFL_TASK_CONFUSED_LABEL_FILE, l.address)
			outFiles = append(outFiles, fileName)
			sort.Strings(outFiles)

			commands = []string{"python3.8", "process_data.py", "--func", "encrypt_data", "--input", labelFile, "--out", strings.Join(outFiles, ",")}
			logger.Debug("PaddleFL container exec commands:", strings.Join(commands, " "))

			err := docker.RunCommand(commands, l.containerName)
			if err != nil {
				handleError(err)
				return nil, err
			}

			defer func() {
				labelFile = l.localWorkspace + LOCAL_SAMPLE_FOLDER + PADDLEFL_TASK_LABEL_FILE
				os.Remove(labelFile)
			}()
		}

		go func() {
			m := &pbDnnVl.Message{
				Type: pbDnnVl.MessageType_MsgFLDataSend,
			}
			l.advance(m)
		}()

	case pbDnnVl.MessageType_MsgFLDataSend:
		sampleFolder := l.localWorkspace + LOCAL_SAMPLE_FOLDER
		mpcFolder := l.localWorkspace + LOCAL_SAMPLE_MPC_FOLDER

		sampleFile := fmt.Sprintf(PADDLEFL_TASK_CONFUSED_SAMPLE_FILE, l.address, l.address)
		err := os.Rename(sampleFolder+sampleFile, mpcFolder+sampleFile)
		if err != nil {
			handleError(err)
			return nil, err
		}

		for _, party := range l.parties {
			sampleFile := fmt.Sprintf(PADDLEFL_TASK_CONFUSED_SAMPLE_FILE, l.address, party)
			_, err := os.Stat(sampleFolder + sampleFile)
			if err != nil {
				handleError(err)
				return nil, err
			}

			f, err := os.OpenFile(sampleFolder+sampleFile, os.O_RDONLY, 0600)
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

			newMess := &pbDnnVl.Message{
				Type:          pbDnnVl.MessageType_MsgFLDataExchange,
				Aby3ShareFile: []byte(sampleFile),
				Aby3ShareData: contentByte,
				VecSize:       uint64(l.fvSize[l.role]),
				Role:          uint64(l.role),
			}
			_, err = l.sendMessageWithRetry(newMess, party)
			if err != nil {
				go handleError(err)
				return nil, err
			}

			//os.Remove(sampleFolder + sampleFile)
		}

		if l.trainParams.GetIsTagPart() {
			labelFile := fmt.Sprintf(PADDLEFL_TASK_CONFUSED_LABEL_FILE, l.address)
			err := os.Rename(sampleFolder+labelFile, mpcFolder+labelFile)
			if err != nil {
				handleError(err)
				return nil, err
			}

			for _, party := range l.parties {
				labelFile = fmt.Sprintf(PADDLEFL_TASK_CONFUSED_LABEL_FILE, party)
				_, err := os.Stat(sampleFolder + labelFile)
				if err != nil {
					handleError(err)
					return nil, err
				}

				f, err := os.OpenFile(sampleFolder+labelFile, os.O_RDONLY, 0600)
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

				newMess := &pbDnnVl.Message{
					Type:          pbDnnVl.MessageType_MsgFLDataExchange,
					Aby3ShareFile: []byte(labelFile),
					Aby3ShareData: contentByte,
					VecSize:       uint64(l.lvSize),
					Role:          uint64(l.role),
				}
				_, err = l.sendMessageWithRetry(newMess, party)
				if err != nil {
					go handleError(err)
					return nil, err
				}
				//os.Remove(sampleFolder + labelFile)
			}
		}

		go func() {
			m := &pbDnnVl.Message{
				Type: pbDnnVl.MessageType_MsgFLDataStatus,
			}
			l.advance(m)
		}()
	case pbDnnVl.MessageType_MsgFLDataExchange:
		// receive confused data from other party
		fileName := string(message.Aby3ShareFile)
		if strings.HasPrefix(fileName, "label") {
			l.lvSize = int64(message.VecSize)
		} else {
			l.fvSize[int64(message.Role)] = int64(message.VecSize)
		}

		mpcFolder := l.localWorkspace + LOCAL_SAMPLE_MPC_FOLDER
		f, err := os.Create(mpcFolder + fileName)
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
		ret = &pb.TrainResponse{
			TaskID: l.id,
		}

		go func() {
			m := &pbDnnVl.Message{
				Type: pbDnnVl.MessageType_MsgFLDataStatus,
			}
			l.advance(m)
		}()
	case pbDnnVl.MessageType_MsgFLDataStatus: // local message
		mpcFolder := l.localWorkspace + LOCAL_SAMPLE_MPC_FOLDER
		fileInfoList, err := ioutil.ReadDir(mpcFolder)
		if err != nil {
			handleError(err)
			return nil, err
		}

		// Each executor provided a sample file now.
		// @todo support only two executor provided samples.
		// There must have three mpc sample file from different executor and one lable mpc file.
		if len(fileInfoList) < 4 {
			return nil, err
		}

		go func() {
			m := &pbDnnVl.Message{
				Type: pbDnnVl.MessageType_MsgTrain,
			}
			l.advance(m)
		}()

	case pbDnnVl.MessageType_MsgTrain:
		l.procMutex.Lock()
		defer l.procMutex.Unlock()
		if l.status == learnerStatusENVPrepare {
			l.status = learnerStatusStartTrain

			// sample mpc files
			mpcFolder := l.containerWorkspace + LOCAL_SAMPLE_MPC_FOLDER
			var fileNames []string
			for _, address := range l.parties {
				fileName := mpcFolder + fmt.Sprintf(PADDLEFL_TASK_CONFUSED_SAMPLE_FILE, address, l.address)
				fileNames = append(fileNames, fileName)
			}
			fileName := mpcFolder + fmt.Sprintf(PADDLEFL_TASK_CONFUSED_SAMPLE_FILE, l.address, l.address)
			fileNames = append(fileNames, fileName)
			sort.Strings(fileNames)

			// label mpc file
			labelFile := mpcFolder + fmt.Sprintf(PADDLEFL_TASK_CONFUSED_LABEL_FILE, l.address)

			var sizes string
			sizes = strconv.FormatInt(l.fvSize[0], 10) + "," + strconv.FormatInt(l.fvSize[1], 10) + "," + strconv.FormatInt(l.fvSize[2], 10)

			cmd := []string{"python3.8", "train.py",
				"--func", "train",
				"--samples", strings.Join(fileNames, ","),
				"--label", labelFile,
				"--role", strconv.Itoa(int(l.role)),
				"--parts", strings.Join(l.allPaddleFLParties, ","),
				"--parts_size", sizes,
				"--batch_num", strconv.Itoa(l.batchNum-1),
				"--output_size", strconv.Itoa(int(l.lvSize)),
				"--model_dir", l.containerWorkspace + LOCAL_MODEL_FOLDER,
			}
			logger.WithFields(logrus.Fields{
				"paddlefl role": l.role,
			}).Infof("learner[%s] execute docker cmd [%s]", l.id, strings.Join(cmd, " "))

			err := docker.RunCommand(cmd, l.containerName)

			if err != nil {
				handleError(err)
				return nil, err
			}
			l.status = learnerStatusEndTrain

			trainModels := pbCom.TrainModels{
				Path:      l.containerWorkspace + LOCAL_MODEL_FOLDER,
				IsTagPart: l.trainParams.GetIsTagPart(),
				Label:     l.trainParams.GetLabel(),
			}
			model, err := json.Marshal(trainModels)
			if err != nil {
				handleError(err)
				return nil, err
			}
			res := &pbCom.TrainTaskResult{
				TaskID:   l.id,
				Success:  true,
				Model:    model,
				TrainSet: nil,
			}
			l.rh.SaveResult(res)
		}
	}

	logger.WithFields(logrus.Fields{
		"paddlefl role": l.role,
	}).Infof("learner[%s] finished advance . message %s", l.id, mType.String())
	return ret, nil
}

// sendMessageWithRetry sends message to remote mpc-node
// retries 2 times at most
func (l *Learner) sendMessageWithRetry(message *pbDnnVl.Message, address string) (*pbDnnVl.Message, error) {
	times := 3

	var m *pbDnnVl.Message
	var err error
	for i := 0; i < times; i++ {
		m, err = l.sendMessage(message, address)
		if err == nil {
			break
		}
	}

	return m, err
}

// sendMessage sends message to remote mpc-node
func (l *Learner) sendMessage(message *pbDnnVl.Message, address string) (*pbDnnVl.Message, error) {
	message.From = l.address
	message.To = address

	payload, err := proto.Marshal(message)
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeInternal, "failed to Marshal payload: %s", err.Error())
	}

	trainReq := &pb.TrainRequest{
		TaskID:  l.id,
		Algo:    l.algo,
		Payload: payload,
	}
	resp, err := l.rpc.StepTrain(trainReq, address)
	if err != nil {
		return nil, err
	}

	m := &pbDnnVl.Message{}
	if len(resp.Payload) != 0 {
		err := proto.Unmarshal(resp.Payload, m)
		if err != nil {
			return nil, errorx.New(errcodes.ErrCodeInternal, "failed to Unmarshal payload[%s] from[%s] and err is[%s] ", string(resp.Payload), address, err.Error())
		}
	}
	return m, nil
}

// NewLearner returns a VerticalLinearDnn Learner based PaddleFL. PaddleFL's addresses will be obtained by invoking smart contract.
// id is the assigned id for Learner
// address indicates local mpc-node
// parties are other learners who participates in MPC, assigned with mpc-node address usually
// paddleFLParams are array of nodes in mpc network, and the role of the current node.
// rpc is used to request remote mpc-node
// rh handles final result which is successful or failed
// params are parameters for training model
// samplesFile contains samples for training model
func NewLearner(id string, address string, params *pbCom.TrainParams, samplesFile []byte,
	parties []string, paddleFLParams *pbCom.PaddleFLParams, rpc RpcHandler, rh ResultHandler) (*Learner, error) {

	p, err := psi.NewVLPSIByPairs(address, samplesFile, params.GetIdName(), parties)
	if err != nil {
		return nil, err
	}

	homoPriv, homoPub, err := crypCom.GenerateHomoKeyPair()
	if err != nil {
		return nil, err
	}

	// the name of paddlefl container is the domain name of the node's address
	role := paddleFLParams.Role
	u := strings.Split(paddleFLParams.Nodes[role], ":")
	containerName := u[0]

	l := &Learner{
		id:                 id,
		algo:               pbCom.Algorithm_DNN_PADDLEFL_VL,
		address:            address,
		parties:            parties,
		homoPriv:           homoPriv,
		homoPub:            homoPub,
		psi:                p,
		trainParams:        params,
		samplesFile:        samplesFile,
		rpc:                rpc,
		rh:                 rh,
		status:             learnerStatusStartPSI,
		containerName:      containerName,
		containerWorkspace: fmt.Sprintf(docker.PADDLEFL_CONTAINER_WORKSPACE, id),
		localWorkspace:     fmt.Sprintf(docker.PADDLEFL_LOCAL_WORKSPACE, id),
		allPaddleFLParties: paddleFLParams.Nodes,
		role:               int64(role),
		fvSize:             [3]int64{},
	}

	// le @todo

	// start training
	go func() {
		// Interim solutions to consistency issues
		time.Sleep(50 * time.Millisecond)
		m := &pbDnnVl.Message{
			Type: pbDnnVl.MessageType_MsgPsiEnc,
		}
		l.advance(m)
	}()
	return l, nil
}

// setSamples, produce the result of psi.IntersectParts
func (l *Learner) setSamples(fileRows [][]string) {
	if l.trainParams.GetIsTagPart() {
		var labelRows [][]string
		var featureRows [][]string
		lable_position := -1
		for k, v := range fileRows[0] {
			if v == l.trainParams.Label {
				lable_position = k
				break
			}
		}
		if lable_position == -1 {
			return
		}

		dataLen := len(fileRows[0])
		for _, row := range fileRows {
			labelRows = append(labelRows, row[lable_position:lable_position+1])
			featureRows = append(featureRows, append(row[0:lable_position], row[lable_position+1:dataLen]...))
		}
		l.featureRows = featureRows
		l.labelRows = labelRows
		l.fvSize[l.role] = int64(len(featureRows[0]))
		l.lvSize = int64(len(labelRows[0]))
	} else {
		l.featureRows = fileRows
		l.fvSize[l.role] = int64(len(fileRows[0]))
	}
}

func (l *Learner) exportRawSamples() error {
	sampleFolder := l.localWorkspace + LOCAL_SAMPLE_FOLDER
	err := os.MkdirAll(sampleFolder, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(l.localWorkspace+LOCAL_SAMPLE_MPC_FOLDER, os.ModePerm)
	if err != nil {
		return err
	}

	featureFile := sampleFolder + PADDLEFL_TASK_SAMPLE_FILE
	err = csv.WriteRowsToFile(l.featureRows, featureFile)
	if err != nil {
		return err
	}

	if l.trainParams.GetIsTagPart() {
		labelDataFile := sampleFolder + PADDLEFL_TASK_LABEL_FILE
		err := csv.WriteRowsToFile(l.labelRows, labelDataFile)
		if err != nil {
			return err
		}
	}
	return nil
}
