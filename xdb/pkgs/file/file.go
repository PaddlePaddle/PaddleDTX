package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const PrivateKeyFileName = "private.key"
const PublicKeyFileName = "public.key"

const UserKeyFilePath = "./ukeys"
const AuthKeyFilePath = "./authkeys"
const KeyFilePath = "./keys"

// ReadFile read the file contents
func ReadFile(path, filename string) ([]byte, error) {
	if strings.LastIndex(path, "/") != len(path)-1 {
		path = path + "/"
	}
	filename = path + filename

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ReadFile [%v] failed, err is [%v]", filename, err)
	}
	return content, nil
}

// WriteFile write the file
func WriteFile(path, filename string, content []byte) error {
	if strings.LastIndex(path, "/") != len(path)-1 {
		path = path + "/"
	}
	// if the path does not exist, it will be created first
	if err := os.MkdirAll(path, os.ModePerm); nil != err {
		return fmt.Errorf("failed to create output dir before create account:%s", err)
	}
	filename = path + filename

	if _, err := os.Stat(filename); err == nil {
		// file existed
		return fmt.Errorf("WriteFile failed, [%v] is existed, err is [%v]", filename, err)
	}
	err := ioutil.WriteFile(filename, content, 0666)
	return err
}

// IsFileExisted judge if the file exists
func IsFileExisted(path, filename string) (bool, error) {
	if strings.LastIndex(path, "/") != len(path)-1 {
		path = path + "/"
	}
	filename = path + filename

	_, err := os.Stat(filename)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
