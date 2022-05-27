package psi

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestVlThreePartsPsi(t *testing.T) {
	path, _ := os.Getwd()

	vp1Address := "address1"
	vp2Address := "address2"
	vp3Address := "address3"

	vp1SamplesFile := readTestData(path + "/testdata/dataA.csv")
	vp1SamplesParties := []string{vp2Address, vp3Address}
	vp1, err := NewVLPSIByPairs(vp1Address, vp1SamplesFile, "id", vp1SamplesParties)
	checkErr(err)

	vp2SamplesFile := readTestData(path + "/testdata/dataB.csv")
	vp2SamplesParties := []string{vp1Address, vp3Address}
	vp2, err := NewVLPSIByPairs(vp2Address, vp2SamplesFile, "id", vp2SamplesParties)
	checkErr(err)

	vp3SamplesFile := readTestData(path + "/testdata/dataC.csv")
	vp3SamplesParties := []string{vp2Address, vp1Address}
	vp3, err := NewVLPSIByPairs(vp3Address, vp3SamplesFile, "id", vp3SamplesParties)
	checkErr(err)

	vp1EnId, err := vp1.EncryptSampleIDSet()
	checkErr(err)
	vp2EnId, err := vp2.EncryptSampleIDSet()
	checkErr(err)
	vp3EnId, err := vp3.EncryptSampleIDSet()
	checkErr(err)

	vp2ReEnId, err := vp2.ReEncryptIDSet(vp1Address, vp1EnId)
	checkErr(err)
	vp3ReEnId, err := vp3.ReEncryptIDSet(vp1Address, vp1EnId)
	checkErr(err)

	vp12ReEnId, err := vp1.ReEncryptIDSet(vp2Address, vp2EnId)
	checkErr(err)
	vp13ReEnId, err := vp1.ReEncryptIDSet(vp3Address, vp3EnId)
	checkErr(err)
	err = vp1.SetOtherFinalReEncryptIDSet(vp2Address, vp12ReEnId)
	checkErr(err)
	err = vp1.SetOtherFinalReEncryptIDSet(vp3Address, vp13ReEnId)
	checkErr(err)

	_, err = vp1.SetReEncryptIDSet(vp2Address, vp2ReEnId)
	checkErr(err)
	_, err = vp1.SetReEncryptIDSet(vp3Address, vp3ReEnId)
	checkErr(err)

	vp1b, vp1Row, vp1intersect, err := vp1.IntersectParts()
	checkErr(err)

	fmt.Println("vp1 intersect status:", vp1b)
	fmt.Println("vp1 intersect row:", len(vp1Row))
	fmt.Println("vp1 intersect intersect:", len(vp1intersect), vp1intersect)
}

func TestVLTwoPartsPsi(t *testing.T) {
	path, _ := os.Getwd()

	vp1Address := "address1"
	vp2Address := "address2"

	vp1SamplesFile := readTestData(path + "/testdata/dataA.csv")
	vp1SamplesParties := []string{vp2Address}
	vp1, err := NewVLPSIByPairs(vp1Address, vp1SamplesFile, "id", vp1SamplesParties)
	checkErr(err)

	vp2SamplesFile := readTestData(path + "/testdata/dataB.csv")
	vp2SamplesParties := []string{vp1Address}
	vp2, err := NewVLPSIByPairs(vp2Address, vp2SamplesFile, "id", vp2SamplesParties)
	checkErr(err)

	vp1EnId, err := vp1.EncryptSampleIDSet()
	checkErr(err)
	vp2EnId, err := vp2.EncryptSampleIDSet()
	checkErr(err)

	vp12ReEnId, err := vp1.ReEncryptIDSet(vp2Address, vp2EnId)
	checkErr(err)

	vp21ReEnId, err := vp2.ReEncryptIDSet(vp1Address, vp1EnId)
	checkErr(err)

	_, err = vp1.SetReEncryptIDSet(vp2Address, vp21ReEnId)
	checkErr(err)

	err = vp1.SetOtherFinalReEncryptIDSet(vp2Address, vp12ReEnId)
	checkErr(err)

	vp1b, vp1Row, vp1intersect, err := vp1.IntersectParts()
	checkErr(err)

	fmt.Println("vp1 intersect status:", vp1b)
	fmt.Println("vp1 intersect row:", len(vp1Row))
	fmt.Println("vp1 intersect intersect:", len(vp1intersect), vp1intersect)

}

func readTestData(filename string) []byte {
	file, err := os.Open(filename)
	checkErr(err)

	text, err := ioutil.ReadAll(file)
	checkErr(err)
	return text
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
