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

package simple

import (
	"context"
	"io"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/PaddlePaddle/PaddleDTX/xdb/config"
	"github.com/PaddlePaddle/PaddleDTX/xdb/engine/slicer"
	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"
)

const (
	defaultBlockSize = 4 * 1024 * 1024 // 4MB
	defaultQueueSize = 4
)

var (
	logger = logrus.WithField("module", "simple-slicer")
)

// SimpleSlicer cuts a file to several slices by blockSize
type SimpleSlicer struct {
	blockSize uint32
	queueSize uint32
}

// New create a SimpleSlicer instance by configuration
func New(conf *config.SimpleSlicerConf) (*SimpleSlicer, error) {
	blockSize := uint32(conf.BlockSize)
	if blockSize == 0 {
		blockSize = defaultBlockSize
	}
	logger.WithField("blockSize", blockSize).Info("slicer initialization")

	queueSize := uint32(conf.QueueSize)
	if queueSize == 0 {
		queueSize = defaultQueueSize
	}
	logger.WithField("queueSize", queueSize).Info("slicer initialization")

	s := &SimpleSlicer{
		blockSize: blockSize,
		queueSize: queueSize,
	}

	return s, nil
}

// Slice reads from IO and cut the data into slices by blockSize, and digests every slice by sha256
func (ss *SimpleSlicer) Slice(ctx context.Context, r io.Reader, opt *slicer.SliceOptions,
	onErr func(err error)) chan slicer.Slice {
	resCh := make(chan slicer.Slice, ss.queueSize)

	go func() {
	SLICEACTION:
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			buf := make([]byte, ss.blockSize)

			// TODO: cancelable read
			_, err := io.ReadFull(r, buf)
			switch err {
			case nil:
				resCh <- makeSlice(buf)
				continue SLICEACTION
			case io.ErrUnexpectedEOF:
				resCh <- makeSlice(buf)
			case io.EOF:
				// end of file
			default:
				onErr(errorx.NewCode(err, errorx.ErrCodeInternal, "failed to read file during Slice"))
			}

			close(resCh)
			return
		}
	}()

	return resCh
}

// GetBlockSize returns size of slice
func (ss *SimpleSlicer) GetBlockSize() int {
	return int(ss.blockSize)
}

// makeSlice digests a slice by sha256 to make a SliceMeta
func makeSlice(bs []byte) slicer.Slice {
	h := hash.HashUsingSha256(bs)
	id, _ := uuid.NewRandom()
	return slicer.Slice{
		SliceMeta: slicer.SliceMeta{
			ID:     id.String(),
			Hash:   h,
			Length: uint64(len(bs)),
		},
		Data: bs,
	}
}
