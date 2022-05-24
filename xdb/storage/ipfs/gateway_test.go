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

package ipfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	cidLib "github.com/ipfs/go-cid"
	shell "github.com/ipfs/go-ipfs-api"
	files "github.com/ipfs/go-ipfs-files"
	u "github.com/ipfs/go-ipfs-util"
)

func TestNewGateway(t *testing.T) {
	gw, err := NewGateway([]string{"localhost:5001", "localhost:5002", "localhost:5003", "127.0.0.1:5004", "127.0.0.1:5005"}, time.Millisecond*300)
	//gw, err := NewGateway([]string{"localhost:5001"})
	checkErr(err, t)

	tGw := gw.(*gateway)
	t.Logf("Gateway.backups is : %v", tGw.backups)
	for i, cli := range tGw.clients {
		t.Logf("Gateway.clients[%v] is : ShellID[%v], Shell[%v]", i, cli.shID, cli.sh)

	}
	for k, shs := range tGw.successors {
		t.Logf("Gateway.successors[%v] is : %v", []byte(k), shs)

		for i, sh := range shs {
			t.Logf("-- [%d] Gateway.successor is : %v", i, sh)
		}
	}
}

func TestAdd(t *testing.T) {
	gw, err := NewGateway([]string{"localhost:5001", "127.0.0.1:5002"}, time.Millisecond*300)
	checkErr(err, t)

	key := "18f168b6-2ef2-491e-8b26-4aa6df18378a"
	file := strings.NewReader("Hello world!")
	cid, err := gw.Add(key, file)
	checkErr(err, t)

	t.Logf("Added [%s]", cid)
}

func TestCat(t *testing.T) {
	gw, err := NewGateway([]string{"localhost:5001", "127.0.0.1:5002"}, time.Millisecond*300)
	checkErr(err, t)

	key := "18f168b6-2ef2-491e-8b26-4aa6df18378a"
	cid := "QmQzCQn4puG4qu8PVysxZmscmQ5vT1ZXpqo7f58Uh9QfyY"

	file, err := gw.Cat(key, cid)
	checkErr(err, t)

	cont, err := ioutil.ReadAll(file)
	checkErr(err, t)

	t.Logf("File [%s] content is [%s]", cid, string(cont))
	file.Close()
}

func TestPinLs(t *testing.T) {
	gw, err := NewGateway([]string{"localhost:5001", "127.0.0.1:5002"}, time.Millisecond*300)
	checkErr(err, t)

	key := "18f168b6-2ef2-491e-8b26-4aa6df18378a"
	cid := "QmQzCQn4puG4qu8PVysxZmscmQ5vT1ZXpqo7f58Uh9QfyY"

	pins, err := gw.PinLs(key, cid)
	checkErr(err, t)
	t.Logf("File [%s] PinInfo is %v", cid, pins)

	cid = "QmbjQDmjdu13KJmqgDWhJfo32oFbydMUwuakpj8HUYmLbL"
	pins, err = gw.PinLs(key, cid)
	checkErr(err, t)
	t.Logf("File [%s] PinInfo is %v", cid, pins)
}

func TestUnpin(t *testing.T) {
	gw, err := NewGateway([]string{"localhost:5001", "127.0.0.1:5002"}, time.Millisecond*300)
	checkErr(err, t)

	key := "18f168b6-2ef2-491e-8b26-4aa6df18378a"
	cid := "QmQzCQn4puG4qu8PVysxZmscmQ5vT1ZXpqo7f58Uh9QfyY"

	err = gw.Unpin(key, cid)
	checkErr(err, t)

	t.Logf("File [%s] is unpined", cid)
}

func TestLBStratege(t *testing.T) {
	var gw Gateway
	var err error
	addPeers := func() {
		gw, err = NewGateway([]string{"127.0.0.1:5001", "localhost:5001", "127.0.0.1:5002", "localhost:5002"}, time.Millisecond*300)
		checkErr(err, t)
	}

	minusPeers := func() {
		gw, err = NewGateway([]string{"127.0.0.1:5001", "localhost:5001"}, time.Millisecond*300)
		checkErr(err, t)
	}

	for i := 0; i < 20; i++ {
		minusPeers()

		// add
		key := fmt.Sprintf("18f168b6-2ef2-491e-8b26-4aa6df18378a-%d", i)
		file := strings.NewReader(fmt.Sprintf("Hello world! V[%d]", i))

		cid, err := gw.Add(key, file)
		checkErr(err, t)

		t.Logf("Added [%s]", cid)

		// capacity expansion
		addPeers()

		// cat
		fRet, err := gw.Cat(key, cid)
		checkErr(err, t)

		cont, err := ioutil.ReadAll(fRet)
		checkErr(err, t)

		t.Logf("File [%s] content is [%s]", cid, string(cont))
		fRet.Close()

		// unpin
		err = gw.Unpin(key, cid)
		checkErr(err, t)

		t.Logf("File [%s] is unpined", cid)
	}
}

func TestRawFileMethods(t *testing.T) {
	sh := shell.NewShell("localhost:5001")

	for i := 0; i < 20; i++ {
		// add
		file := strings.NewReader(fmt.Sprintf("Hello world! R[%d]", i))

		cid, err := sh.Add(file)
		checkErr(err, t)

		t.Logf("Added [%s]", cid)

		// cat
		fRet, err := sh.Cat(cid)
		checkErr(err, t)

		cont, err := ioutil.ReadAll(fRet)
		checkErr(err, t)

		t.Logf("File [%s] content is [%s]", cid, string(cont))
		fRet.Close()

		// pin ls
		var raw struct{ Keys map[string]shell.PinInfo }
		err = sh.Request("pin/ls", cid).Option("type", "recursive").Exec(context.Background(), &raw)
		checkErr(err, t)
		t.Logf("Pined Files are [%v]", raw)

		// unpin
		err = sh.Unpin(cid)
		checkErr(err, t)
		t.Logf("File [%s] is unpined", cid)
		err = sh.Request("pin/ls", cid).Option("type", "recursive").Exec(context.Background(), &raw)
		if err != nil {
			if strings.HasSuffix(err.Error(), "is not pinned") {
				t.Logf("File [%s] is unpined really", cid)
			} else {
				checkErr(err, t)
			}
		}

		// unpin again
		err = sh.Unpin(cid)
		if err != nil {
			if strings.HasSuffix(err.Error(), "not pinned or pinned indirectly") {
				t.Logf("File [%s] is not pinned or pinned indirectly", cid)
			} else {
				checkErr(err, t)
			}
		}
	}
}

func TestRawBlockMethod(t *testing.T) {
	sh := shell.NewShell("localhost:5001")

	block := []byte("I love China!")
	c := cidLib.NewCidV0(u.Hash(block))
	t.Logf("Created CID: %s ", c)

	blocktem := []byte("I love China very much!")
	ctem := cidLib.NewCidV0(u.Hash(blocktem))
	t.Logf("Created CID2: %s ", ctem)

	// put a block and pin it
	var out struct {
		Key string
	}
	fr := files.NewBytesFile(block)
	slf := files.NewSliceDirectory([]files.DirEntry{files.FileEntry("", fr)})
	fileReader := files.NewMultiFileReader(slf, true)

	err := sh.Request("block/put").
		Option("pin", true).
		Body(fileReader).
		Exec(context.Background(), &out)

	checkErr(err, t)
	t.Logf("Put a block and its Key is: %v ", out.Key)

	// get a block
	blockGot, err := sh.BlockGet(c.String())
	checkErr(err, t)
	t.Logf("Got a block and its content is: %v ", string(blockGot))

	// unpin a block
	err = sh.Request("pin/rm", c.String()).Option("recursive", true).Exec(context.Background(), nil)
	checkErr(err, t)

	// remove a block
	err = sh.Request("block/rm", c.String()).Exec(context.Background(), nil)
	checkErr(err, t)
	t.Logf("Remove a block: %v ", c.String())
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
