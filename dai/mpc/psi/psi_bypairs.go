package psi

import (
	"crypto/ecdsa"
	"sync"

	"github.com/PaddlePaddle/PaddleDTX/xdb/errorx"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	csv "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common/csv"
	"github.com/PaddlePaddle/PaddleDTX/dai/errcodes"
)

// vlPsiByPairs implements VLPSI
type vlPsiByPairs struct {
	name          string
	privkey       *ecdsa.PrivateKey // local ecc private key for ID encryption
	samplesFile   []byte            // csv file content subjected to specified form
	samplesIdName string            // feature name for samples ID, used to extract IDs
	parties       map[string]bool   // names of other parties who participate MPC

	// intermediate results
	// see vl_common.psi for more
	ids    []string
	rows   [][]string
	encIDs []byte

	reEncryptIDSetsFromOthers    sync.Map // stores re-encrypted ID-Sets returned by other parties
	finalReEncryptIDSetsOfOthers sync.Map // stores final re-encrypted ID-Sets for other parties
	middleIntersect              sync.Map // stores the intersections of the current and the one other for other parties

	// final results
	done      bool
	newRows   [][]string
	intersect []string
}

// EncryptSampleIDSet encrypt sample ID list using own public key
func (vp *vlPsiByPairs) EncryptSampleIDSet() ([]byte, error) {
	err := vp.readSamples()
	if err != nil {
		return []byte{}, errorx.New(errcodes.ErrCodePSISamplesFile, "mistake[%s] happened when PSI read IDs from file", err.Error())
	}

	encIDs, err := vl_common.EncryptSampleIDSet(vp.ids, &vp.privkey.PublicKey)
	if err != nil {
		return []byte{}, errorx.New(errcodes.ErrCodePSIEncryptSampleIDSet, "mistake[%s] happened when PSI encrypt SampleIDSet", err.Error())
	}

	vp.encIDs = encIDs
	return vp.encIDs, nil
}

func (vp *vlPsiByPairs) SetReEncryptIDSet(party string, reEncIDs []byte) (bool, error) {
	if _, ok := vp.parties[party]; !ok {
		// if from unknown party, ignore
		return false, nil
	}

	vp.reEncryptIDSetsFromOthers.LoadOrStore(party, reEncIDs)

	return true, nil
}

// ReEncryptIDSet re-encrypt ID list for other party using own private key
func (vp *vlPsiByPairs) ReEncryptIDSet(party string, encIDs []byte) ([]byte, error) {

	reEncIDs, errRe := vl_common.ReEncryptIDSet(encIDs, vp.privkey)

	// if from unknown party, don't care about any Error
	if _, ok := vp.parties[party]; !ok {
		if errRe != nil {
			reEncIDs = []byte{}
		}
		return reEncIDs, nil
	}

	if errRe != nil {
		return []byte{}, errorx.New(errcodes.ErrCodePSIReEncryptIDSet, "mistake[%s] happened when PSI encrypt EncryptedSampleIDSet for other party[%s]", errRe.Error(), party)
	}

	return reEncIDs, nil
}

func (vp *vlPsiByPairs) SetOtherFinalReEncryptIDSet(party string, reEncIDs []byte) error {
	// if from unknown party, don't store result
	if _, ok := vp.parties[party]; !ok {
		return nil
	}
	vp.finalReEncryptIDSetsOfOthers.LoadOrStore(party, reEncIDs)
	return nil
}

// IntersectParts calculate intersections of all parties samples, and re-arrange sample files
func (vp *vlPsiByPairs) IntersectParts() (bool, [][]string, []string, error) {
	if vp.done {
		return vp.done, vp.newRows, vp.intersect, nil
	}

	var newRows [][]string
	for party := range vp.parties {
		reEncIDsBySelf, ok := vp.reEncryptIDSetsFromOthers.Load(party)
		if !ok {
			return false, newRows, nil, nil
		}

		reEncIDsByOther, ok := vp.finalReEncryptIDSetsOfOthers.Load(party)
		if !ok {
			return false, newRows, nil, nil
		}
		intersect, err := vl_common.IntersectTwoParts(vp.ids, reEncIDsBySelf.([]byte), reEncIDsByOther.([]byte))
		if err != nil {
			return false, newRows, intersect, errorx.New(errcodes.ErrCodePSIIntersectParts, "mistake[%s] happened when PSI intersect all parts", err.Error())
		}
		vp.middleIntersect.LoadOrStore(party, intersect)
	}

	var intersect []string
	firstParty := true
	for party := range vp.parties {
		if firstParty {
			firstParty = false
			if intersectInf, ok := vp.middleIntersect.Load(party); ok {
				intersect = intersectInf.([]string)
			} else {
				return false, newRows, intersect, nil
			}
			continue
		}
		var compare []string
		if compareInf, ok := vp.middleIntersect.Load(party); ok {
			compare = compareInf.([]string)
		} else {
			return false, newRows, intersect, nil
		}
		intersect = vp.getIntersection(intersect, compare)
	}

	vp.intersect = intersect
	newRows, err := vl_common.RearrangeFileWithIntersectIDs(vp.rows, vp.samplesIdName, intersect)
	if err != nil {
		return false, newRows, intersect, errorx.New(errcodes.ErrCodePSIRearrangeFile, "mistake[%s] happened when PSI rearrange file with intersected IDs", err.Error())
	}
	vp.newRows = newRows
	vp.done = true

	return vp.done, vp.newRows, vp.intersect, nil
}

func (vp *vlPsiByPairs) getIntersection(a, b []string) []string {
	compareMap := make(map[string]bool)
	for _, v := range b {
		compareMap[v] = true
	}
	var ret []string
	for _, v := range a {
		if compareMap[v] == true {
			ret = append(ret, v)
		}
	}
	return ret
}

// readSamples retrieve ID list from sample file rows
func (vp *vlPsiByPairs) readSamples() error {
	rows, IDs, err := csv.ReadIDsFromFileRows(vp.samplesFile, vp.samplesIdName)

	if err != nil {
		return err
	}

	vp.ids = IDs
	vp.rows = rows

	return nil
}

// NewVLPSIByPairs create a VLPSI instance and initiate it, calculating intersections of multiple nodes by pairs.
// name is to name the PSI instance
// parties are names of other parties who participate MPC
// sampleFile is csv file content subjected to specified form
// sampleIdName is used to extract IDs
func NewVLPSIByPairs(name string, samplesFile []byte, samplesIdName string, parties []string) (VLPSI, error) {
	if len(parties) <= 0 {
		return nil, errorx.New(errcodes.ErrCodeParam, "no parties in PSI")
	}

	p := &vlPsiByPairs{
		name:          name,
		samplesFile:   samplesFile,
		samplesIdName: samplesIdName,
	}

	p.parties = make(map[string]bool)
	for _, party := range parties {
		p.parties[party] = true
	}

	// create local ecc private key and public key pari for ID encryption
	privkey, err := vl_common.GeneratePSIKeyPair()
	if err != nil {
		return nil, errorx.New(errcodes.ErrCodeInternal, "mistake[%s] happened when PSI GeneratePSIKeyPair", err.Error())
	}

	p.privkey = privkey

	return p, nil
}
