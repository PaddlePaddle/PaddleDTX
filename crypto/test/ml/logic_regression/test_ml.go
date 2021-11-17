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
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	ml_common "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

var xcc = new(xchain.XchainCryptoClient)

func main() {
	// --- 多元逻辑回归（中心化） start ---

	// step 1: 测试数据集标准化

	// 引入Iris鸢尾花数据集，其数据集包含了150个样本，都属于鸢尾属下的三个亚属，分别是山鸢尾Iris-setosa、变色鸢尾Iris-versicolor和维吉尼亚鸢尾Iris-virginica
	// 数据集中，各个亚属占比相同，均为50个，即1/3。
	// 数据特征维度分别是花萼长度Sepal Length、花萼宽度Sepal Width、花瓣长度Petal Length、花瓣宽度Petal Width。

	label := "Label"
	labelName := "Iris-setosa"
	features, err := readFeaturesFromCSVFile("./testdata/train.csv", label, labelName)
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}

	dataSet := &ml_common.DataSet{
		Features: features,
	}

	standardizedDataSet := xcc.LogRegStandardizeDataSet(dataSet, "Label")
	jsonStandardizedDataSet, _ := json.Marshal(standardizedDataSet)
	log.Printf("standardizedDataSet is %s", jsonStandardizedDataSet)

	// step 2: 测试数据集预处理
	trainSet := xcc.LogRegPreProcessDataSet(standardizedDataSet, "Label")
	jsonTrainSet, _ := json.Marshal(trainSet)
	log.Printf("trainSet is %s", jsonTrainSet)

	// step 3: 模型训练
	var alpha = 0.0001
	var amplitude = 0.0000001
	var lambda = 0.1

	model := xcc.LogRegTrainModel(trainSet, alpha, amplitude, ml_common.RegNone, lambda)
	jsonModel, _ := json.Marshal(model)
	log.Printf("model using RegNone is %s", jsonModel)

	modelLasso := xcc.LogRegTrainModel(trainSet, alpha, amplitude, ml_common.RegLasso, lambda)
	jsonModel, _ = json.Marshal(modelLasso)
	log.Printf("model using RegLasso is %s", jsonModel)

	modelRidge := xcc.LogRegTrainModel(trainSet, alpha, amplitude, ml_common.RegRidge, lambda)
	jsonModel, _ = json.Marshal(modelRidge)
	log.Printf("model using RegRidge is %s", jsonModel)

	// -- 使用验证集合进行预测，测试模型准确度 --

	verifyFeatures, err := readFeaturesFromCSVFile("./testdata/verify.csv", label, labelName)
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}

	for i := 0; i < len(verifyFeatures[0].Sets); i++ {
		input := make(map[string]float64)

		for _, feature := range verifyFeatures {
			input[feature.FeatureName] = feature.Sets[i]
		}

		realResult := verifyFeatures[len(verifyFeatures)-1].Sets[i]

		standardizeInput := xcc.LogRegStandardizeLocalInput(trainSet.XbarParams, trainSet.SigmaParams, input)

		predictResult := xcc.LogRegPredictByLocalInput(model.Params, standardizeInput)
		log.Printf("PredictByLocalInput result[RegNone] is %v, realResult is: %v", predictResult, realResult)
		predictResult = xcc.LogRegPredictByLocalInput(modelLasso.Params, standardizeInput)
		log.Printf("PredictByLocalInput result[RegLasso] is %v, realResult is: %v", predictResult, realResult)
		predictResult = xcc.LogRegPredictByLocalInput(modelRidge.Params, standardizeInput)
		log.Printf("PredictByLocalInput result[RegRidge] is %v, realResult is: %v", predictResult, realResult)
	}

	// --- 逻辑回归（中心化） end ---
}

// readFeaturesFromCSVFile 从 csv 文件中读取样本特征
func readFeaturesFromCSVFile(path, label, labelName string) ([]*ml_common.DataFeature, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	plainFile := string(content)
	r := csv.NewReader(strings.NewReader(plainFile))
	ss, _ := r.ReadAll()

	features, err := xcc.LogRegImportFeatures(ss, label, labelName)
	if err != nil {
		return nil, err
	}
	return features, nil
}
