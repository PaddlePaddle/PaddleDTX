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

package errcodes

// error code list
const (
	// 00xx common error
	ErrCodeInternal = "PX0001" // internal error
	ErrCodeParam    = "PX0002" // parameters error
	ErrCodeConfig   = "PX0003" // configuration error
	ErrCodeNotFound = "PX0004" // target not found
	ErrCodeEncoding = "PX0005" // encoding error
	ErrCodeUnknown  = "PX0006" // unknown error

	// mpc errors
	ErrCodeTooMuchTasks          = "PX0011" // the number of task reach upper-limit
	ErrCodeTaskExists            = "PX0012" // task already exists
	ErrCodePSISamplesFile        = "PX0013" // mistake happened when PSI read IDs from file
	ErrCodePSIEncryptSampleIDSet = "PX0014" // mistake happened when PSI encrypts SampleIDSet
	ErrCodePSIReEncryptIDSet     = "PX0015" // mistake happened when PSI encrypts EncryptedSampleIDSet for other party
	ErrCodePSIIntersectParts     = "PX0016" // mistake happened when PSI intersects all parts
	ErrCodePSIRearrangeFile      = "PX0016" // mistake happened when PSI rearranges file with intersected IDs
	ErrCodeRPCFindNoPeer         = "PX0017" // find no peer when do rpc request
	ErrCodeRPCConnect            = "PX0018" // failed to get connection
	ErrCodeTaskDeleted           = "PX0019" // failed to delete task
	ErrCodeDataSetSplit          = "PX0020" // failed to split data set
	ErrCodeGetTrainSet           = "PX0021" // failed to get training set when evaluate model
	ErrCodeGetPredictSet         = "PX0022" // failed to split predicting set when evaluate model
	ErrCodeStartTask             = "PX0023" // failed to start task
	ErrCodeTriggerTooMuch        = "PX0024" // LiveEvaluator be triggered more than once for same pause round
)
