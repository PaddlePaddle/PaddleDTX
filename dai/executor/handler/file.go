package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/cjqpker/slidewindow"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/aes"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecies"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/PaddlePaddle/PaddleDTX/dai/executor/storage/xuperdb"
	xdbchain "github.com/PaddlePaddle/PaddleDTX/xdb/blockchain"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/common"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
	"github.com/PaddlePaddle/PaddleDTX/xdb/pkgs/http"
)

var defaultConcurrency uint64 = 10

// Define the ExecutionType of the executor, used to download sample files during task training
const (
	ProxyExecutionMode = "Proxy"
	SelfExecutionMode  = "Self"
)

// Storage stores files locally or xuperdb
type Storage interface {
	Write(value io.Reader, key string) (string, error)
	Read(key string) (io.ReadCloser, error)
}

// FileStorage contains model storage and prediction result storage.
type FileStorage struct {
	ModelStorage      Storage
	EvaluationStorage Storage
	PredictStorage    Storage
}

// FileDownload mode for download the sample file during the task execution.
type FileDownload struct {
	Type           string           // value is 'Proxy' or 'Self'
	NodePrivateKey ecdsa.PrivateKey // the executor node's privatekey

	PrivateKey ecdsa.PrivateKey // used when the Type is 'Self'
	Host       string
}

// GetSampleFile download sample files, if f.Type is 'Self', download files from dataOwner nodes.
// If f.Type is 'Proxy', download slices from storage nodes and recover the sample file, the key
// required to decrypt the sample file and slices can be obtained through the file authorization application ID. only
// after the file owner has confirmed the executor's file authorization application, the executor node can get the sample file.
func (f *FileDownload) GetSampleFile(fileID string, chain Blockchain) (io.ReadCloser, error) {
	if f.Type == SelfExecutionMode {
		xuperdbClient := xuperdb.New(0, "", f.Host, f.PrivateKey)
		plainText, err := xuperdbClient.Read(fileID)
		if err != nil {
			return nil, errorx.Wrap(err, "failed to download the sample file from the dataOwner node, fileID: %s", fileID)
		}
		return plainText, nil
	} else {
		// 1. get the sample file info from chain
		file, err := chain.GetFileByID(fileID)
		if err != nil {
			return nil, errorx.New(errorx.ErrCodeInternal, "failed to get the sample file from contract, fileID: %s", fileID)
		}
		// 2. get the authorization ID, use the authKey to decrypt the sample file
		pubkey := ecdsa.PublicKeyFromPrivateKey(f.NodePrivateKey)
		fileAuths, err := chain.ListFileAuthApplications(&xdbchain.ListFileAuthOptions{
			Applier:    pubkey[:],
			Authorizer: file.Owner,
			Status:     xdbchain.FileAuthApproved,
			FileID:     fileID,
			TimeStart:  0,
			TimeEnd:    time.Now().UnixNano(),
			Limit:      1,
		})
		if err != nil {
			return nil, errorx.Wrap(err,
				"get the file authorization application failed, fileID: %s, Applier: %x, Authorizer: %x", fileID, pubkey[:], file.Owner)
		}
		if len(fileAuths) == 0 {
			return nil, errorx.New(errorx.ErrCodeInternal,
				"the file authorization application is empty, fileID: %s, Applier: %x, Authorizer: %x", fileID, pubkey[:], file.Owner)
		}
		// 3. obtain the derived key needed to decrypt the file through the AuthKey
		firstKey, secKey, err := f.getDecryptAuthKey(fileAuths[0].AuthKey)
		if err != nil {
			return nil, err
		}
		// 4. download slices and decrypt
		plainText, err := f.recoverFile(context.Background(), chain, file, firstKey, secKey)
		if err != nil {
			return nil, err
		}
		return plainText, nil
	}
}

// getDecryptAuthKey get the authorization key for file decryption, return firKey and secKey.
// firKey used to decrypt the file and file's Structure
// secKey used to decrypt slices, different slices of different stroage nodes use different AES Keys
func (f *FileDownload) getDecryptAuthKey(authKey []byte) (firKey aes.AESKey, secKey map[string]map[string]aes.AESKey, err error) {
	// 1 parse ecdsa.PrivateKey to EC PrivateKey key
	applierPrivateKey := ecdsa.ParsePrivateKey(f.NodePrivateKey)
	// applier's EC private key decrypt the authKey
	decryptAuthKey, err := ecies.Decrypt(&applierPrivateKey, authKey)
	if err != nil {
		return firKey, secKey, errorx.NewCode(err, errorx.ErrCodeInternal, "fail to decrypt the authKey")
	}

	// 2 unmarshal decrypt authKey
	decryptKey := make(map[string]interface{})
	if err = json.Unmarshal(decryptAuthKey, &decryptKey); err != nil {
		return firKey, secKey, errorx.NewCode(err, errorx.ErrCodeInternal, "fail to unmarshal decrypt authKey")
	}

	// 3 get the first-level derived key
	firstEncSecret, err := json.Marshal(decryptKey["firstEncSecret"])
	if err != nil {
		return firKey, secKey, errorx.Wrap(err, "failed to marshal firstEncSecret")
	}
	if err = json.Unmarshal(firstEncSecret, &firKey); err != nil {
		return firKey, secKey, errorx.NewCode(err, errorx.ErrCodeInternal, "fail to get file first encrypt key")
	}

	// 4 get the second-level derived key
	secondEncSecret, err := json.Marshal(decryptKey["secondEncSecret"])
	if err != nil {
		return firKey, secKey, errorx.Wrap(err, "failed to marshal secondEncSecret")
	}
	if err = json.Unmarshal(secondEncSecret, &secKey); err != nil {
		return firKey, secKey, errorx.NewCode(err, errorx.ErrCodeInternal, "fail to get slice second encrypt key")
	}
	return firKey, secKey, nil
}

// recoverFile recover file by pulling slices from storage nodes
// The detailed steps are as follows:
// 1. parameters check
// 2. read file info from the blockchain
// 3. decrypt the file's struct to get slice's order
// 4. download slices from the storage node, if request fails, pull slices from other storage nodes
// 5. slices decryption and combination
// 6. decrypt the combined slices to get the original file
func (f *FileDownload) recoverFile(ctx context.Context, chain Blockchain, file xdbchain.File,
	firstKey aes.AESKey, secKey map[string]map[string]aes.AESKey) (io.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)

	// get storage nodes
	allNodes, err := chain.ListNodes()
	if err != nil {
		cancel()
		return nil, errorx.Wrap(err, "failed to get nodes from blockchain")
	}
	// filter offline status storage nodes
	var nodes xdbchain.Nodes
	for _, n := range allNodes {
		if n.Online {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) == 0 {
		cancel()
		return nil, errorx.New(errorx.ErrCodeInternal, "empty online nodes")
	}
	nodesMap := common.ToNodesMap(nodes)

	// recover file structure to get slice's order
	fs, err := f.recoverChainFileStructure(firstKey, file.Structure)
	if err != nil {
		cancel()
		return nil, err
	}

	// use sliding window
	sw := slidewindow.SlideWindow{
		Total:       uint64(len(fs)),
		Concurrency: defaultConcurrency,
	}

	sw.Init = func(ctx context.Context, s *slidewindow.Session) error {
		return nil
	}
	// get the mapping of slice-storageNodes
	slicesPool := makeSlicesPool4Read(file.Slices)
	sw.Task = func(ctx context.Context, s *slidewindow.Session) error {
		slice := fs[int(s.Index())]
		// get the list of storage nodes where the slice stored
		targetPool, ok := slicesPool[slice.SliceID]
		if !ok {
			return errorx.Internal(nil, "bad file structure")
		}
		for _, target := range targetPool {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			node, exist := nodesMap[string(target.NodeID)]
			if !exist || !node.Online {
				logger.WithField("node_id", string(target.NodeID)).Warn("abnormal node")
				continue
			}
			// pull slice
			r, err := f.pull(ctx, target.ID, file.ID, node.Address)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"slice_id":    target.ID,
					"file_id":     file.ID,
					"target_node": string(node.ID),
				}).WithError(err).Warn("failed to pull slice")
				continue
			}
			defer r.Close()

			// read slice and check slice hash
			cipherText, err := ioutil.ReadAll(r)
			if err != nil {
				logger.WithError(err).Warn("failed to read slice from target node")
				continue
			}
			if len(cipherText) != int(target.Length) {
				logger.WithFields(logrus.Fields{"expected": target.Length, "got": len(cipherText)}).
					Warn("invalid slice length.")
				continue
			}
			hGot := hash.HashUsingSha256(cipherText)
			if !bytes.Equal(hGot, target.CipherHash) {
				logger.WithFields(logrus.Fields{"expected": target.CipherHash, "got": hGot}).
					Warn("invalid slice hash.")
				continue
			}

			// decrypt the encrypted slice
			plainText, err := f.recover(secKey[target.ID][string(node.ID)], cipherText)
			if err != nil {
				logger.WithError(err).Error("failed to decrypt slice")
				continue
			}

			// trim 0 at the end of file
			if s.Index() != sw.Total-1 {
				s.Set("data", plainText)
			} else {
				s.Set("data", bytes.TrimRight(plainText, string([]byte{0})))
			}

			break
		}

		if _, exist := s.Get("data"); !exist {
			return errorx.New(errorx.ErrCodeNotFound, "failed to pull slice %s", slice.SliceID)
		}

		return nil
	}

	reader, writer := io.Pipe()
	sw.Done = func(ctx context.Context, s *slidewindow.Session) error {
		data, exist := s.Get("data")
		if !exist {
			return errorx.New(errorx.ErrCodeNotFound, "failed to find data")
		}

		if _, err := writer.Write(data.([]byte)); err != nil {
			return errorx.NewCode(err, errorx.ErrCodeInternal, "failed to write")
		}

		// exit on success
		if s.Index() == uint64(len(fs)-1) {
			writer.Close()
		}
		return nil
	}

	go func() {
		defer cancel()
		if err := sw.Start(ctx); err != nil {
			writer.CloseWithError(err)
		}
	}()

	// decrypt recovered file
	fileCipherText, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read")
	}
	plainText, err := f.recover(firstKey, fileCipherText)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeCrypto, "file decryption failed")
	}
	return ioutil.NopCloser(bytes.NewReader(plainText)), nil
}

// pull used pull slices from storage nodes
func (f *FileDownload) pull(ctx context.Context, id, fileId, nodeAddress string) (io.ReadCloser, error) {
	// Add signature
	timestamp := time.Now().UnixNano()
	msg := fmt.Sprintf("%s,%s,%d", id, fileId, timestamp)
	sig, err := ecdsa.Sign(f.NodePrivateKey, hash.HashUsingSha256([]byte(msg)))
	if err != nil {
		return nil, errorx.Wrap(err, "failed to sign slice pull")
	}

	pubkey := ecdsa.PublicKeyFromPrivateKey(f.NodePrivateKey)
	url := fmt.Sprintf("http://%s/v1/slice/pull?slice_id=%s&file_id=%s&timestamp=%d&pubkey=%s&signature=%s",
		nodeAddress, id, fileId, timestamp, pubkey.String(), sig.String())

	r, err := http.Get(ctx, url)
	if err != nil {
		return nil, errorx.Wrap(err, "failed to do get slice")
	}
	return r, nil
}

// recoverChainFileStructure get file structure from blockchain and decrypt it,
// FileStructure used to get the correct file slices order
func (f *FileDownload) recoverChainFileStructure(aesKey aes.AESKey, structure []byte) (xdbchain.FileStructure, error) {
	// decrypt structure
	decStruct, err := aes.DecryptUsingAESGCM(aesKey, structure, nil)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decrypt file structure")
	}

	var fStructures xdbchain.FileStructure
	if err := fStructures.Parse(decStruct); err != nil {
		return fStructures, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to parse file structure")
	}
	return fStructures, nil
}

// recover decrypt the ciphertext using AES-GCM
func (f *FileDownload) recover(aesKey aes.AESKey, ciphertext []byte) ([]byte, error) {
	plaintext, err := aes.DecryptUsingAESGCM(aesKey, ciphertext, nil)
	if err != nil {
		return nil, errorx.NewCode(err, errorx.ErrCodeInternal, "failed to decrypt file or slices")
	}

	return plaintext, nil
}

// makeSlicesPool4Read return the mapping of storage nodes where the slice is located
// key is sliceID, value is the list of slices meta
func makeSlicesPool4Read(srs []xdbchain.PublicSliceMeta) map[string][]xdbchain.PublicSliceMeta {
	slicesPool := make(map[string][]xdbchain.PublicSliceMeta)
	for _, s := range srs {
		var ss []xdbchain.PublicSliceMeta
		if v, exist := slicesPool[s.ID]; exist {
			ss = v
		}
		ss = append(ss, s)
		slicesPool[s.ID] = ss
	}

	return slicesPool
}
