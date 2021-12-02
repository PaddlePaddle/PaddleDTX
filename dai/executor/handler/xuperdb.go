package handler

import (
	"context"
	"io"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/ecdsa"
	httpclient "github.com/PaddlePaddle/PaddleDTX/xdb/client/http"
)

type XuperDB struct {
	PrivateKey ecdsa.PrivateKey
	Address    string
	Ns         string
	ExpireTime int64
}

// Write stores samples in xuperDB
func (f *XuperDB) Write(ctx context.Context, r io.Reader, name string) (string, error) {
	client, err := httpclient.New(f.Address)
	if err != nil {
		return "", err
	}
	opt := httpclient.WriteOptions{
		PrivateKey: f.PrivateKey.String(),

		Namespace:   f.Ns,
		FileName:    name,
		ExpireTime:  f.ExpireTime,
		Description: "store samples",
	}

	resp, err := client.Write(context.Background(), r, opt)
	if err != nil {
		return "", err
	}
	return resp.FileID, nil
}

// Read gets samples from xuperDB
func (f *XuperDB) Read(ctx context.Context, fileID string) (io.ReadCloser, error) {
	client, err := httpclient.New(f.Address)
	if err != nil {
		return nil, err
	}

	opt := httpclient.ReadOptions{
		PrivateKey: f.PrivateKey.String(),
		FileID:     fileID,
	}

	reader, err := client.Read(context.Background(), opt)
	if err != nil {
		return nil, err
	}

	return reader, nil
}
