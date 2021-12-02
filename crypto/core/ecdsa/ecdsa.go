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

package ecdsa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
)

/*
	Generates the commonly used ecdsa account, with which to identify users and nodes.
*/

const (
	PublicKeyLength  = 64
	PrivateKeyLength = 32
	SignatureLength  = 64
)

var (
	defaultCurve elliptic.Curve
)

type PublicKey [PublicKeyLength]byte

type PrivateKey [PrivateKeyLength]byte

type Signature [SignatureLength]byte

func init() {
	defaultCurve = elliptic.P256()
}

func (pk PublicKey) String() string {
	return hex.EncodeToString(pk[:])
}

func (sk PrivateKey) String() string {
	return hex.EncodeToString(sk[:])
}

func (s Signature) String() string {
	return hex.EncodeToString(s[:])
}

// GenerateKeyPair generate a key pair
func GenerateKeyPair() (privkey PrivateKey, pubkey PublicKey, err error) {
	privateKey, err := ecdsa.GenerateKey(defaultCurve, rand.Reader)
	if err != nil {
		// will never happen
		return
	}

	privkey = MarshalPrivateKey(privateKey)
	pubkey = MarshalPublicKey(&privateKey.PublicKey)
	return
}

// ParsePrivateKey parse from local type to EC private key
func ParsePrivateKey(privkey PrivateKey) ecdsa.PrivateKey {
	D := new(big.Int).SetBytes(privkey[:])

	x, y := defaultCurve.ScalarBaseMult(D.Bytes())

	return ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: defaultCurve,
			X:     x,
			Y:     y,
		},
		D: D,
	}
}

// ParsePublicKey parse from local type to EC public key
func ParsePublicKey(pubkey PublicKey) (ecdsa.PublicKey, error) {
	x := new(big.Int).SetBytes(pubkey[:32])
	y := new(big.Int).SetBytes(pubkey[32:])

	if !defaultCurve.IsOnCurve(x, y) {
		return ecdsa.PublicKey{}, errors.New("public key not on curve")
	}

	return ecdsa.PublicKey{
		Curve: defaultCurve,
		X:     x,
		Y:     y,
	}, nil
}

// MarshalPrivateKey marshal private key to local types
func MarshalPrivateKey(privkey *ecdsa.PrivateKey) PrivateKey {
	var res PrivateKey
	copy(res[:], padStart(privkey.D.Bytes(), 32))
	return res
}

// MarshalPublicKey marshal public key to local types
func MarshalPublicKey(pubkey *ecdsa.PublicKey) PublicKey {
	x := padStart(pubkey.X.Bytes(), 32)
	y := padStart(pubkey.Y.Bytes(), 32)
	var res PublicKey
	copy(res[:], x)
	copy(res[32:], y)

	return res
}

// PublicKeyFromPrivateKey
func PublicKeyFromPrivateKey(privkey PrivateKey) PublicKey {
	ecPrivkey := ParsePrivateKey(privkey)
	return MarshalPublicKey(&ecPrivkey.PublicKey)
}

// Sign sign a digest
func Sign(privkey PrivateKey, digest []byte) (Signature, error) {
	privateKey := ParsePrivateKey(privkey)

	r, s, err := ecdsa.Sign(rand.Reader, &privateKey, digest)
	if err != nil {
		return Signature{}, fmt.Errorf("failed to sign digest %w", err)
	}

	rr, ss := padStart(r.Bytes(), 32), padStart(s.Bytes(), 32)
	var sig Signature
	copy(sig[:], rr)
	copy(sig[32:], ss)
	return sig, nil
}

// Verify verify a signature
func Verify(pubkey PublicKey, digest []byte, signature Signature) error {
	publicKey, err := ParsePublicKey(pubkey)
	if err != nil {
		return err
	}

	rr, ss := signature[:32], signature[32:]
	r, s := new(big.Int).SetBytes(rr), new(big.Int).SetBytes(ss)

	if !ecdsa.Verify(&publicKey, digest, r, s) {
		return errors.New("failed to verify")
	}

	return nil
}

// DecodePrivateKeyFromString decode ec private key from string
func DecodePrivateKeyFromString(s string) (PrivateKey, error) {
	var privateKey PrivateKey

	bs, err := hex.DecodeString(s)
	if err != nil {
		return PrivateKey{}, fmt.Errorf("invalid private key format")
	}
	if len(bs) != PrivateKeyLength {
		return PrivateKey{}, fmt.Errorf("invalid private key length")
	}
	copy(privateKey[:], bs)
	return privateKey, nil
}

// DecodePublicKeyFromString decode ec public key from string
func DecodePublicKeyFromString(s string) (PublicKey, error) {
	var publicKey PublicKey

	bs, err := hex.DecodeString(s)
	if err != nil {
		return PublicKey{}, fmt.Errorf("invalid public key format")
	}
	if len(bs) != PublicKeyLength {
		return PublicKey{}, fmt.Errorf("invalid public key length")
	}
	copy(publicKey[:], bs)
	if _, err := ParsePublicKey(publicKey); err != nil {
		return PublicKey{}, fmt.Errorf("invalid ecdsa public key")
	}

	return publicKey, nil
}

// DecodeSignatureFromString decode ec signature from string
func DecodeSignatureFromString(s string) (Signature, error) {
	var sig Signature

	bs, err := hex.DecodeString(s)
	if err != nil {
		return Signature{}, err
	}
	if len(bs) != PublicKeyLength {
		return Signature{}, fmt.Errorf("invalid signature length")
	}
	copy(sig[:], bs)
	return sig, nil
}
