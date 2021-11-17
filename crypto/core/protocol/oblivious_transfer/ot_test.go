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

package oblivious_transfer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestOT(t *testing.T) {
	msg0 := "msg 0 for ot protocol"
	msg1 := "msg 1 for ot protocol"

	var msgs []string
	msgs = append(msgs, msg0)
	msgs = append(msgs, msg1)

	senderPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	receiverPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	ReceiverPublicKeyForSender, err := ReceiverChoose(receiverPrivateKey, &senderPrivateKey.PublicKey, IndexOne)
	if err != nil {
		t.Errorf("ReceiverPublicKeyForSender err is %v", err)
		return
	}
	cts, err := SenderEncryptMsg(senderPrivateKey, ReceiverPublicKeyForSender, msgs)
	if err != nil {
		t.Errorf("SenderEncryptMsg err is %v", err)
		return
	}
	msgChosen, err := ReceiverRetrieveMsg(receiverPrivateKey, &senderPrivateKey.PublicKey, cts, IndexOne)
	if err != nil {
		t.Errorf("ReceiverRetrieveMsg err is %v", err)
		return
	}
	t.Logf("msgChosen is: %s", msgChosen)
}
