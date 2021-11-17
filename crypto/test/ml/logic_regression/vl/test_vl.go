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
	"math"
	"strings"

	"github.com/PaddlePaddle/PaddleDTX/crypto/client/service/xchain"
	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	ml_common "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

var xcc = new(xchain.XchainCryptoClient)

func main() {
	// step 1: 导入测试数据集
	label := "Label"
	labelName := "Iris-setosa"
	featuresA, err := readFeaturesFromCSVFile("./testdata/trainA.csv", label, labelName)
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}
	featuresB, err := readFeaturesFromCSVFile("./testdata/trainB.csv", label, labelName)
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}
	dataSetA := &ml_common.DataSet{
		Features: featuresA,
	}
	dataSetB := &ml_common.DataSet{
		Features: featuresB,
	}

	// step 2: 测试数据集标准化
	standardizedDataSetA := xcc.LogRegVLStandardizeDataSet(dataSetA, "")
	jsonStandardizedDataSetA, _ := json.Marshal(standardizedDataSetA)
	log.Printf("standardizedDataSetA is %s", jsonStandardizedDataSetA)

	standardizedDataSetB := xcc.LogRegVLStandardizeDataSet(dataSetB, "Label")
	jsonStandardizedDataSetB, _ := json.Marshal(standardizedDataSetB)
	log.Printf("standardizedDataSetB is %s", jsonStandardizedDataSetB)

	// step 3: 测试数据集预处理
	trainDataSetA := xcc.LogRegVLPreProcessDataSet(standardizedDataSetA)
	jsonTrainSetA, _ := json.Marshal(trainDataSetA)
	log.Printf("trainSetA is %s", jsonTrainSetA)

	trainDataSetB := xcc.LogRegVLPreProcessDataSetTagPart(standardizedDataSetB, "Label")
	jsonTrainSetB, _ := json.Marshal(trainDataSetB)
	log.Printf("trainSetB is %s", jsonTrainSetB)

	trainSetA := trainDataSetA.TrainSet
	trainSetB := trainDataSetB.TrainSet

	// step 4: 生成同态加密密钥
	paillierPrivateKeyA, err := xcc.GeneratePaillierPrivateKey(paillier.DefaultPrimeLength)
	if err != nil {
		log.Printf("GeneratePaillierPrivateKey err is %v", err)
		return
	}
	paillierPrivateKeyB, err := xcc.GeneratePaillierPrivateKey(paillier.DefaultPrimeLength)
	if err != nil {
		log.Printf("GeneratePaillierPrivateKey err is %v", err)
		return
	}

	// step 5: 使用同态加密技术来解决梯度计算过程和损失函数计算过程中的参数交换
	regMode := ml_common.RegNone
	alpha := 0.01
	accuracy := 10
	lambda := 0.1
	amplitude := 0.001

	// 创建模型参数
	thetasA := make([]float64, len(trainSetA[0])-1)
	thetasB := make([]float64, len(trainSetB[0])-2)

	// 存储模型参数的临时数据
	tempsA := make([]float64, len(trainSetA[0])-1)
	tempsB := make([]float64, len(trainSetB[0])-2)

	lastCostA := 0.0
	lastCostB := 0.0
	round := 0
	for {
		for i := 0; i < len(thetasA); i++ {
			gradA, err := calGradForA(thetasA, thetasB, trainSetA, trainSetB, i, accuracy, regMode, lambda, paillierPrivateKeyA, paillierPrivateKeyB)
			if err != nil {
				log.Printf("calGrad for A err is %v", err)
				return
			}
			tempsA[i] = thetasA[i] - alpha*gradA
		}

		for i := 0; i < len(thetasB); i++ {
			gradB, err := calGradForB(thetasA, thetasB, trainSetA, trainSetB, i, accuracy, regMode, lambda, paillierPrivateKeyA, paillierPrivateKeyB)
			if err != nil {
				log.Printf("calGrad for B err is %v", err)
				return
			}
			tempsB[i] = thetasB[i] - alpha*gradB
		}

		// 更新每个特征维度的系数theta
		for i := 0; i < len(thetasA); i++ {
			thetasA[i] = tempsA[i]
		}
		for i := 0; i < len(thetasB); i++ {
			thetasB[i] = tempsB[i]
		}

		currentCostA, err := calCostForA(thetasA, thetasB, trainSetA, trainSetB, accuracy, regMode, lambda, paillierPrivateKeyA, paillierPrivateKeyB)
		if err != nil {
			log.Printf("calCostForA err is %v", err)
			return
		}
		deltaA := math.Abs(currentCostA - lastCostA)

		currentCostB, err := calCostForB(thetasA, thetasB, trainSetA, trainSetB, accuracy, regMode, lambda, paillierPrivateKeyA, paillierPrivateKeyB)
		if err != nil {
			log.Printf("calCostForA err is %v", err)
			return
		}
		deltaB := math.Abs(currentCostB - lastCostB)

		log.Printf("round[%v] costForA is %v, costForB is %v,deltaA is %v, deltaB is %v", round, currentCostA, currentCostB, deltaA, deltaB)
		round++

		// 根据损失函数来判断模型是否收敛
		if deltaA < amplitude && deltaB < amplitude {
			break
		}

		lastCostA = currentCostA
		lastCostB = currentCostB
	}

	log.Printf("thetasA before DeStandardize is %v", thetasA)
	log.Printf("thetasB before DeStandardize is %v", thetasB)

	// -- 联邦预测 start

	paramsA := make(map[string]float64)
	paramsB := make(map[string]float64)

	for i := 0; i < len(trainDataSetA.FeatureNames); i++ {
		paramsA[trainDataSetA.FeatureNames[i]] = thetasA[i]
	}

	paramsB["Intercept"] = thetasB[0]
	for i := 0; i < len(trainDataSetB.FeatureNames)-1; i++ {
		paramsB[trainDataSetB.FeatureNames[i]] = thetasB[i+1]
	}

	log.Printf("thetasA before DeStandardize is %v", paramsA)
	log.Printf("thetasB before DeStandardize is %v", paramsB)

	featuresForPredictA, err :=
		readFeaturesFromCSVFile("./testdata/verifyA.csv", label, labelName)
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}
	featuresForPredictB, err := readFeaturesFromCSVFile("./testdata/verifyB.csv", label, labelName)
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}

	for i := 0; i < len(featuresForPredictA[0].Sets); i++ {
		inputA := make(map[string]float64)
		inputA["Sepal Length"] = featuresForPredictA[0].Sets[i]
		inputA["Sepal Width"] = featuresForPredictA[1].Sets[i]
		standardizeInputA := xcc.LogRegVLStandardizeLocalInput(trainDataSetA.XbarParams, trainDataSetA.SigmaParams, inputA)
		predictA := xcc.LogRegVLPredictLocalPart(paramsA, standardizeInputA)

		inputB := make(map[string]float64)
		inputB["Petal Length"] = featuresForPredictB[0].Sets[i]
		inputB["Petal Width"] = featuresForPredictB[1].Sets[i]
		standardizeInputB := xcc.LogRegVLStandardizeLocalInput(trainDataSetB.XbarParams, trainDataSetB.SigmaParams, inputB)
		predictB := xcc.LogRegVLPredictLocalTagPart(paramsB, standardizeInputB)

		realValue := featuresForPredictB[2].Sets[i]

		predictSum := predictA + predictB
		// TODO:后面用同态结合泰勒展开来做
		// 计算 1/(1+e^-(w'x))
		predictReal := 1 / (1 + math.Exp(-1*predictSum))
		log.Printf("joint learning predict result is:%v, real result is:%v", predictReal, realValue)
	}

	//  -- 联邦预测 end
}

// calGradForB 标签方计算梯度
func calGradForB(thetasA, thetasB []float64, trainSetA, trainSetB [][]float64, featureIndex, accuracy, regMode int, regParam float64, paillierPrivateKeyA, paillierPrivateKeyB *paillier.PrivateKey) (float64, error) {
	paillierPublicKeyA := &paillierPrivateKeyA.PublicKey
	paillierPublicKeyB := &paillierPrivateKeyB.PublicKey

	// A 计算加密中间参数，传给 B
	// 参与方A计算predictValue(j-A)，predictValue(j-A)^2，并对它们分别使用公钥pubKey-A进行同态加密
	localGradAndCostPartA, err := xcc.LogRegVLCalLocalGradAndCost(thetasA, trainSetA, accuracy, regMode, regParam, paillierPublicKeyA)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	// B 计算加密中间参数
	// 参与方B计算predictValue(j-B) - realValue(j)，(predictValue(j-B) - realValue(j))^2，并使用公钥pubKey-B进行同态加密
	localGradAndCostPartB, err := xcc.LogRegVLCalLocalGradAndCostTagPart(thetasB, trainSetB, accuracy, regMode, regParam, paillierPublicKeyB)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	encLocalGradientPartA := localGradAndCostPartA.EncPart
	rawLocalGradientPartB := localGradAndCostPartB.RawPart

	// B 计算最终加密梯度，传给 A
	encGraForB, err := xcc.LogRegVLCalEncGradientTagPart(rawLocalGradientPartB, encLocalGradientPartA, trainSetB, featureIndex, accuracy, paillierPublicKeyA)
	if err != nil {
		log.Printf("CalEncLocalGradientTagPart for B err is %v", err)
		return 0, err
	}

	// A 解密梯度，传给 B
	decGraForB := xcc.LogRegVLDecryptGradient(encGraForB.EncGrad, paillierPrivateKeyA)

	// B 移除随机数得到最终梯度
	realGraForB := xcc.LogRegVLRetrieveRealGradient(decGraForB, accuracy, encGraForB.RandomNoise)
	graForB := xcc.LogRegVLCalGradient(realGraForB)

	return graForB, err
}

// calGradForA 非标签方计算梯度
func calGradForA(thetasA, thetasB []float64, trainSetA, trainSetB [][]float64, featureIndex, accuracy, regMode int, regParam float64, paillierPrivateKeyA, paillierPrivateKeyB *paillier.PrivateKey) (float64, error) {
	paillierPublicKeyA := &paillierPrivateKeyA.PublicKey
	paillierPublicKeyB := &paillierPrivateKeyB.PublicKey

	// B 计算加密中间参数，传给 A
	localGradAndCostPartB, err := xcc.LogRegVLCalLocalGradAndCostTagPart(thetasB, trainSetB, accuracy, regMode, regParam, paillierPublicKeyB)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	// A 计算加密中间参数
	localGradAndCostPartA, err := xcc.LogRegVLCalLocalGradAndCost(thetasA, trainSetA, accuracy, regMode, regParam, paillierPublicKeyA)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	rawLocalGradientPartA := localGradAndCostPartA.RawPart
	encLocalGradientPartB := localGradAndCostPartB.EncPart

	// A 计算最终加密梯度，传给 B
	encGraForA, err := xcc.LogRegVLCalEncGradient(rawLocalGradientPartA, encLocalGradientPartB, trainSetA, featureIndex, accuracy, paillierPublicKeyB)
	if err != nil {
		log.Printf("CalEncLocalGradient for A err is %v", err)
		return 0, err
	}

	// B 解密梯度，传给 A
	decGraForA := xcc.LogRegVLDecryptGradient(encGraForA.EncGrad, paillierPrivateKeyB)

	// A 移除随机数得到最终梯度
	realGraForA := xcc.LogRegVLRetrieveRealGradient(decGraForA, accuracy, encGraForA.RandomNoise)
	graForA := xcc.LogRegVLCalGradient(realGraForA)

	return graForA, err
}

// calCostForB 标签方计算损失
func calCostForB(thetasA, thetasB []float64, trainSetA, trainSetB [][]float64, accuracy, regMode int, regParam float64, paillierPrivateKeyA, paillierPrivateKeyB *paillier.PrivateKey) (float64, error) {
	paillierPublicKeyA := &paillierPrivateKeyA.PublicKey
	paillierPublicKeyB := &paillierPrivateKeyB.PublicKey

	// A 加密中间参数，传给 B
	localGradAndCostPartA, err := xcc.LogRegVLCalLocalGradAndCost(thetasA, trainSetA, accuracy, regMode, regParam, paillierPublicKeyA)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	// B 计算加密中间参数
	localGradAndCostPartB, err := xcc.LogRegVLCalLocalGradAndCostTagPart(thetasB, trainSetB, accuracy, regMode, regParam, paillierPublicKeyB)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	encLocalGradientPartA := localGradAndCostPartA.EncPart
	rawLocalGradientPartB := localGradAndCostPartB.RawPart

	// B 计算最终加密损失，传给 A
	encCostForB, err := xcc.LogRegVLEvaluateEncCostTagPart(rawLocalGradientPartB, encLocalGradientPartA, trainSetB, accuracy, paillierPublicKeyA)
	if err != nil {
		log.Printf("EvaluateEncLocalCostTag for B err is %v", err)
		return 0, err
	}

	// A 解密损失，传给 B
	decCostForB := xcc.LogRegVLDecryptCost(encCostForB.EncCost, paillierPrivateKeyA)

	// B 移除随机数得到最终损失
	realCostForB := xcc.LogRegVLRetrieveRealCost(decCostForB, accuracy, encCostForB.RandomNoise)
	costForB := xcc.LogRegVLCalCost(realCostForB)

	return costForB, err
}

// calCostForA 非标签方计算损失
func calCostForA(thetasA, thetasB []float64, trainSetA, trainSetB [][]float64, accuracy, regMode int, regParam float64, paillierPrivateKeyA, paillierPrivateKeyB *paillier.PrivateKey) (float64, error) {
	paillierPublicKeyA := &paillierPrivateKeyA.PublicKey
	paillierPublicKeyB := &paillierPrivateKeyB.PublicKey

	// B 加密中间参数，传给 A
	localGradAndCostPartB, err := xcc.LogRegVLCalLocalGradAndCostTagPart(thetasB, trainSetB, accuracy, regMode, regParam, paillierPublicKeyB)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	// A 计算加密中间参数
	localGradAndCostPartA, err := xcc.LogRegVLCalLocalGradAndCost(thetasA, trainSetA, accuracy, regMode, regParam, paillierPublicKeyA)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	rawLocalGradientPartA := localGradAndCostPartA.RawPart
	encLocalGradientPartB := localGradAndCostPartB.EncPart

	// A 计算最终加密损失，传给 B
	encCostForA, err := xcc.LogRegVLEvaluateEncCost(rawLocalGradientPartA, encLocalGradientPartB, trainSetA, accuracy, paillierPublicKeyB)
	if err != nil {
		log.Printf("evaluateEncLocalCost for A err is %v", err)
		return 0, err
	}

	// B 解密损失，传给 A
	decCostForA := xcc.LogRegVLDecryptCost(encCostForA.EncCost, paillierPrivateKeyB)

	// A 移除随机数得到最终损失
	realCostForA := xcc.LogRegVLRetrieveRealCost(decCostForA, accuracy, encCostForA.RandomNoise)
	costForA := xcc.LogRegVLCalCost(realCostForA)

	return costForA, err
}

// readFeaturesFromCSVFile 从 csv 文件中读取样本特征
func readFeaturesFromCSVFile(path string, label, labelName string) ([]*ml_common.DataFeature, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	plainFile := string(content)
	r := csv.NewReader(strings.NewReader(plainFile))
	ss, _ := r.ReadAll()

	features, err := ml_common.ImportFeaturesForLogReg(ss, label, labelName)
	if err != nil {
		return nil, err
	}
	return features, nil
}
