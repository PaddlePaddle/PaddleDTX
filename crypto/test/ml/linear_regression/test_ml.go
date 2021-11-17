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

package main

import (
	"encoding/json"
	"log"

	"github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	ml_common "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

var xcc = new(xchain.XchainCryptoClient)

func main() {
	// --- 多元线性回归（中心化） start ---

	// step 1: 测试数据集标准化

	// 构造测试数据集
	dataSize := 30

	// ---feature1
	feature1Sets := make(map[int]float64)
	for i := 0; i < dataSize; i++ {
		feature1Sets[i] = 100 + float64(i)*10
	}
	feature1 := &ml_common.DataFeature{
		FeatureName: "size",
		Sets:        feature1Sets,
	}

	// ---feature2
	feature2Sets := make(map[int]float64)
	for i := 0; i < dataSize; i++ {
		feature2Sets[i] = float64(i) + 1
	}
	feature2 := &ml_common.DataFeature{
		FeatureName: "floor",
		Sets:        feature2Sets,
	}

	// ---feature4
	feature4Sets := make(map[int]float64)
	for i := 0; i < dataSize; i++ {
		feature4Sets[i] = float64(i) + 2
	}
	feature4 := &ml_common.DataFeature{
		FeatureName: "window",
		Sets:        feature4Sets,
	}

	// ---feature5
	feature5Sets := make(map[int]float64)
	for i := 0; i < dataSize; i++ {
		feature5Sets[i] = float64(i) + 3
	}
	feature5 := &ml_common.DataFeature{
		FeatureName: "room",
		Sets:        feature5Sets,
	}

	// ---feature3
	feature3Sets := make(map[int]float64)
	// 50000*size + 30000*floor + 20000*window + 10000*room + 3000
	for i := 0; i < dataSize; i++ {
		feature3Sets[i] = 50000*feature1Sets[i] + 30000*feature2Sets[i] + 20000*feature4Sets[i] + 10000*feature5Sets[i] + 3000
	}
	feature3 := &ml_common.DataFeature{
		FeatureName: "price",
		Sets:        feature3Sets,
	}

	var features []*ml_common.DataFeature
	features = append(features, feature1)
	features = append(features, feature2)
	features = append(features, feature3)
	features = append(features, feature4)
	features = append(features, feature5)

	dataSet := &ml_common.DataSet{
		Features: features,
	}

	standardizedDataSet := xcc.LinRegStandardizeDataSet(dataSet)
	jsonStandardizedDataSet, _ := json.Marshal(standardizedDataSet)
	log.Printf("standardizedDataSet is %s", jsonStandardizedDataSet)

	// step 2: 测试数据集预处理
	trainSet := xcc.LinRegPreProcessDataSet(standardizedDataSet, "price")
	jsonTrainSet, _ := json.Marshal(trainSet)
	log.Printf("trainSet is %s", jsonTrainSet)

	// step 3: 模型训练
	var alpha = 0.001
	var amplitude = 0.000001
	var lambda = 0.1
	model := xcc.LinRegTrainModel(trainSet, alpha, amplitude, ml_common.RegNone, lambda)
	jsonModel, _ := json.Marshal(model)
	log.Printf("model using RegNone is %s", jsonModel)

	model = xcc.LinRegTrainModel(trainSet, alpha, amplitude, ml_common.RegLasso, lambda)
	jsonModel, _ = json.Marshal(model)
	log.Printf("model using RegLasso is %s", jsonModel)

	model = xcc.LinRegTrainModel(trainSet, alpha, amplitude, ml_common.RegRidge, lambda)
	jsonModel, _ = json.Marshal(model)
	log.Printf("model using RegRidge is %s", jsonModel)

	costForRegLasso := xcc.LinRegEvaluateModelSuperParamByCV(dataSet, "price", alpha, amplitude, ml_common.RegLasso, lambda, ml_common.CvLoo, 0)
	log.Printf("costForRegLasso using RegLasso is %v, lambda is %v", costForRegLasso, lambda)

	costForRegRidge := xcc.LinRegEvaluateModelSuperParamByCV(dataSet, "price", alpha, amplitude, ml_common.RegRidge, lambda, ml_common.CvLoo, 0)
	log.Printf("costForRegLasso using RegRidge is %v, lambda is %v", costForRegRidge, lambda)

	lambda = 0.5

	costForRegLasso = xcc.LinRegEvaluateModelSuperParamByCV(dataSet, "price", alpha, amplitude, ml_common.RegLasso, lambda, ml_common.CvLoo, 0)
	log.Printf("costForRegLasso using RegLasso is %v, lambda is %v", costForRegLasso, lambda)

	costForRegRidge = xcc.LinRegEvaluateModelSuperParamByCV(dataSet, "price", alpha, amplitude, ml_common.RegRidge, lambda, ml_common.CvLoo, 0)
	log.Printf("costForRegRidge using RegRidge is %v, lambda is %v", costForRegRidge, lambda)

	lambda = 0.01

	costForRegLasso = xcc.LinRegEvaluateModelSuperParamByCV(dataSet, "price", alpha, amplitude, ml_common.RegLasso, lambda, ml_common.CvLoo, 0)
	log.Printf("costForRegLasso using RegLasso is %v, lambda is %v", costForRegLasso, lambda)

	costForRegRidge = xcc.LinRegEvaluateModelSuperParamByCV(dataSet, "price", alpha, amplitude, ml_common.RegRidge, lambda, ml_common.CvLoo, 0)
	log.Printf("costForRegRidge using RegRidge is %v, lambda is %v", costForRegRidge, lambda)

	lambda = 0.001

	costForRegLasso = xcc.LinRegEvaluateModelSuperParamByCV(dataSet, "price", alpha, amplitude, ml_common.RegLasso, lambda, ml_common.CvLoo, 0)
	log.Printf("costForRegLasso using RegLasso is %v, lambda is %v", costForRegLasso, lambda)

	costForRegRidge = xcc.LinRegEvaluateModelSuperParamByCV(dataSet, "price", alpha, amplitude, ml_common.RegRidge, lambda, ml_common.CvLoo, 0)
	log.Printf("costForRegRidge using RegRidge is %v, lambda is %v", costForRegRidge, lambda)

	// --- 线性回归（中心化） end ---
}
