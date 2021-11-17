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
	featuresA, err := readFeaturesFromCSVFile("./testdata/train_dataA.csv")
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}
	featuresB, err := readFeaturesFromCSVFile("./testdata/train_dataB.csv")
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
	standardizedDataSetA := xcc.LinRegVLStandardizeDataSet(dataSetA)
	jsonStandardizedDataSetA, _ := json.Marshal(standardizedDataSetA)
	log.Printf("standardizedDataSetA is %s", jsonStandardizedDataSetA)

	standardizedDataSetB := xcc.LinRegVLStandardizeDataSet(dataSetB)
	jsonStandardizedDataSetB, _ := json.Marshal(standardizedDataSetB)
	log.Printf("standardizedDataSetB is %s", jsonStandardizedDataSetB)

	// step 3: 测试数据集预处理
	trainDataSetA := xcc.LinRegVLPreProcessDataSet(standardizedDataSetA)
	jsonTrainSetA, _ := json.Marshal(trainDataSetA)
	log.Printf("trainSetA is %s", jsonTrainSetA)

	trainDataSetB := xcc.LinRegVLPreProcessDataSetTagPart(standardizedDataSetB, "MEDV")
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
	var alpha = 0.01
	var amplitude = 0.0001
	var lambda = 0.1
	accuracy := 10

	// 存储模型参数
	thetasA := make([]float64, len(trainSetA[0])-1)
	thetasB := make([]float64, len(trainSetB[0])-2)

	// 存储临时数据
	tempsA := make([]float64, len(trainSetA[0])-1)
	tempsB := make([]float64, len(trainSetB[0])-2)

	lastCostA := 0.0
	lastCostB := 0.0
	round := 0
	for {
		// 使用随机梯度下降 stochastic gradient descent
		trainSetAThisRound := [][]float64{trainSetA[round%len(trainSetA)]}
		trainSetBThisRound := [][]float64{trainSetB[round%len(trainSetB)]}

		for i := 0; i < len(thetasA); i++ {
			gradA, err := calGradForA(thetasA, thetasB, trainSetAThisRound, trainSetBThisRound, i, regMode, accuracy, lambda, paillierPrivateKeyA, paillierPrivateKeyB)
			if err != nil {
				log.Printf("calGrad for A err is %v", err)
				return
			}
			tempsA[i] = thetasA[i] - alpha*gradA
		}

		for i := 0; i < len(thetasB); i++ {
			gradB, err := calGradForB(thetasA, thetasB, trainSetAThisRound, trainSetBThisRound, i, regMode, accuracy, lambda, paillierPrivateKeyA, paillierPrivateKeyB)
			if err != nil {
				log.Printf("calGrad for B err is %v", err)
				return
			}
			tempsB[i] = thetasB[i] - alpha*gradB
		}

		// 更新 thetas
		for i := 0; i < len(thetasA); i++ {
			thetasA[i] = tempsA[i]
		}
		for i := 0; i < len(thetasB); i++ {
			thetasB[i] = tempsB[i]
		}

		currentCostA, err := calCostForA(thetasA, thetasB, trainSetAThisRound, trainSetBThisRound, regMode, accuracy, lambda, paillierPrivateKeyA, paillierPrivateKeyB)
		if err != nil {
			log.Printf("calCostForB err is %v", err)
			return
		}
		deltaA := math.Abs(currentCostA - lastCostA)

		currentCostB, err := calCostForB(thetasA, thetasB, trainSetAThisRound, trainSetBThisRound, regMode, accuracy, lambda, paillierPrivateKeyA, paillierPrivateKeyB)
		if err != nil {
			log.Printf("calCostForB err is %v", err)
			return
		}
		deltaB := math.Abs(currentCostB - lastCostB)

		log.Printf("round[%v] costForA is %v, costForB is %v,deltaA is %v, deltaB is %v", round, currentCostA, currentCostB, deltaA, deltaB)
		round++

		if deltaA < amplitude && deltaB < amplitude {
			break
		}

		lastCostA = currentCostA
		lastCostB = currentCostB
	}

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

	// 读取预测数据集
	predictDataA, err := readFeaturesFromCSVFile("./testdata/predict_dataA.csv")
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}
	predictDataB, err := readFeaturesFromCSVFile("./testdata/predict_dataB.csv")
	if err != nil {
		log.Printf("readFeaturesFromCSVFile failed: %v", err)
		return
	}

	inputA := make(map[string]float64)
	inputB := make(map[string]float64)
	for i := 0; i < len(predictDataA[0].Sets); i++ {
		for j := 0; j < len(predictDataA); j++ {
			inputA[predictDataA[j].FeatureName] = predictDataA[j].Sets[i]
		}
		for j := 0; j < len(predictDataB); j++ {
			inputB[predictDataB[j].FeatureName] = predictDataB[j].Sets[i]
		}

		// 标准化样本预测数据并预测
		standardizeInputA := xcc.LinRegVLStandardizeLocalInput(trainDataSetA.XbarParams, trainDataSetA.SigmaParams, inputA)
		predictA := xcc.LinRegVLPredictLocalPart(paramsA, standardizeInputA)

		standardizeInputB := xcc.LinRegVLStandardizeLocalInput(trainDataSetB.XbarParams, trainDataSetB.SigmaParams, inputB)
		predictB := xcc.LinRegVLPredictLocalTagPart(paramsB, standardizeInputB)

		// 逆标准化并得到最终结果
		predictSum := predictA + predictB
		predictReal := xcc.LinRegVLDeStandardizeOutput(trainDataSetB.XbarParams["MEDV"], trainDataSetB.SigmaParams["MEDV"], predictSum)
		log.Printf("predictReal after joint learning DeStandardizeOutput is %v", predictReal)
	}

	//  -- 联邦预测 end
}

// calGradForB 标签方计算梯度
func calGradForB(thetasA, thetasB []float64, trainSetA, trainSetB [][]float64, featureIndex, regMode, accuracy int, regParam float64, paillierPrivateKeyA, paillierPrivateKeyB *paillier.PrivateKey) (float64, error) {
	paillierPublicKeyA := &paillierPrivateKeyA.PublicKey
	paillierPublicKeyB := &paillierPrivateKeyB.PublicKey

	// A 计算加密中间参数，传给 B
	// 参与方A计算predictValue(j-A)，predictValue(j-A)^2，并对它们分别使用公钥pubKey-A进行同态加密
	localGradientPartA, err := xcc.LinRegVLCalLocalGradAndCost(thetasA, trainSetA, accuracy, regMode, regParam, paillierPublicKeyA)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	// B 计算加密中间参数
	// 参与方B计算predictValue(j-B) - realValue(j)，(predictValue(j-B) - realValue(j))^2，并使用公钥pubKey-B进行同态加密
	localGradientPartB, err := xcc.LinRegVLCalLocalGradAndCostTagPart(thetasB, trainSetB, accuracy, regMode, regParam, paillierPublicKeyB)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	encLocalGradientPartA := localGradientPartA.EncPart
	rawLocalGradientPartB := localGradientPartB.RawPart

	// B 计算最终加密梯度，传给 A
	encGraForB, err := xcc.LinRegVLCalEncGradientTagPart(rawLocalGradientPartB, encLocalGradientPartA, trainSetB, featureIndex, accuracy, paillierPublicKeyA)
	if err != nil {
		log.Printf("CalEncLocalGradientTagPart for B err is %v", err)
		return 0, err
	}

	// A 解密梯度，传给 B
	decGraForB := xcc.LinRegVLDecryptGradient(encGraForB.EncGrad, paillierPrivateKeyA)

	// B 移除随机数得到最终梯度
	realGraForB := xcc.LinRegVLRetrieveRealGradient(decGraForB, accuracy, encGraForB.RandomNoise)
	graForB := xcc.LinRegVLCalGradient(realGraForB)

	return graForB, err
}

// calGradForA 非标签方计算梯度
func calGradForA(thetasA, thetasB []float64, trainSetA, trainSetB [][]float64, featureIndex, regMode, accuracy int, regParam float64, paillierPrivateKeyA, paillierPrivateKeyB *paillier.PrivateKey) (float64, error) {
	paillierPublicKeyA := &paillierPrivateKeyA.PublicKey
	paillierPublicKeyB := &paillierPrivateKeyB.PublicKey

	// B 计算加密中间参数, 传给 A
	// 参与方B计算predictValue(j-B) - realValue(j)，(predictValue(j-B) - realValue(j))^2，并使用公钥pubKey-B进行同态加密
	localGradientPartB, err := xcc.LinRegVLCalLocalGradAndCostTagPart(thetasB, trainSetB, accuracy, regMode, regParam, paillierPublicKeyB)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	// A 计算加密中间参数
	// predictValue(j-A)，predictValue(j-A)^2，并对它们分别使用公钥pubKey-A进行同态加密
	localGradientPartA, err := xcc.LinRegVLCalLocalGradAndCost(thetasA, trainSetA, accuracy, regMode, regParam, paillierPublicKeyA)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	rawLocalGradientPartA := localGradientPartA.RawPart
	encLocalGradientPartB := localGradientPartB.EncPart

	// A 计算最终加密梯度，传给 B
	encGraForA, err := xcc.LinRegVLCalEncGradient(rawLocalGradientPartA, encLocalGradientPartB, trainSetA, featureIndex, accuracy, paillierPublicKeyB)
	if err != nil {
		log.Printf("CalEncLocalGradient for A err is %v", err)
		return 0, err
	}

	// B 解密梯度，传给 A
	// 参与方B使用私钥解密 encGraForA，得到被添加了随机数RanNumA的梯度graForA，然后将其发送给参与方A
	decGraForA := xcc.LinRegVLDecryptGradient(encGraForA.EncGrad, paillierPrivateKeyB)

	// A 移除随机数得到最终梯度
	realGraForA := xcc.LinRegVLRetrieveRealGradient(decGraForA, accuracy, encGraForA.RandomNoise)
	graForA := xcc.LinRegVLCalGradient(realGraForA)

	return graForA, err
}

// calCostForB 标签方计算损失
func calCostForB(thetasA, thetasB []float64, trainSetA, trainSetB [][]float64, regMode, accuracy int, regParam float64, paillierPrivateKeyA, paillierPrivateKeyB *paillier.PrivateKey) (float64, error) {
	paillierPublicKeyA := &paillierPrivateKeyA.PublicKey
	paillierPublicKeyB := &paillierPrivateKeyB.PublicKey

	// A本地计算加密中间参数，并传给 B
	// 参与方A计算predictValue(j-A)，predictValue(j-A)^2，并对它们分别使用公钥pubKey-A进行同态加密
	localGradientPartA, err := xcc.LinRegVLCalLocalGradAndCost(thetasA, trainSetA, accuracy, regMode, regParam, paillierPublicKeyA)
	if err != nil {
		return 0, err
	}

	// B计算加密中间参数
	// 参与方B计算 predictValue(j-B) - realValue(j)，(predictValue(j-B) - realValue(j))^2，并使用公钥pubKey-B进行同态加密
	localGradientPartB, err := xcc.LinRegVLCalLocalGradAndCostTagPart(thetasB, trainSetB, accuracy, regMode, regParam, paillierPublicKeyB)
	if err != nil {
		return 0, err
	}

	// B 计算最终加密的损失，并传给 A
	encLocalGradientPartA := localGradientPartA.EncPart
	rawLocalGradientPartB := localGradientPartB.RawPart
	encCostForB, err := xcc.LinRegVLEvaluateEncCostTagPart(rawLocalGradientPartB, encLocalGradientPartA, trainSetB, paillierPublicKeyA)
	if err != nil {
		return 0, err
	}

	// A 解密，并传给 B
	decCostForB := xcc.LinRegVLDecryptCost(encCostForB.EncCost, paillierPrivateKeyA)

	// B 移除随机数，得到最终损失
	realCostForB := xcc.LinRegVLRetrieveRealCost(decCostForB, accuracy, encCostForB.RandomNoise)
	costForB := xcc.LinRegVLCalCost(realCostForB)

	return costForB, err
}

// calCostForA 非标签方计算损失
func calCostForA(thetasA, thetasB []float64, trainSetA, trainSetB [][]float64, regMode, accuracy int, regParam float64, paillierPrivateKeyA, paillierPrivateKeyB *paillier.PrivateKey) (float64, error) {
	paillierPublicKeyA := &paillierPrivateKeyA.PublicKey
	paillierPublicKeyB := &paillierPrivateKeyB.PublicKey

	// 先计算A的两个特征维度中第1个特征：常数项的梯度参数
	//	featureIndex := 0

	// 先由A进行计算，生成加密中间参数
	// 参与方A计算predictValue(j-A)，predictValue(j-A)^2，并对它们分别使用公钥pubKey-A进行同态加密
	localGradientPartA, err := xcc.LinRegVLCalLocalGradAndCost(thetasA, trainSetA, accuracy, regMode, regParam, paillierPublicKeyA)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	// 再由B进行计算，生成加密中间参数
	// 参与方B计算predictValue(j-B) - realValue(j)，(predictValue(j-B) - realValue(j))^2，并使用公钥pubKey-B进行同态加密
	localGradientPartB, err := xcc.LinRegVLCalLocalGradAndCostTagPart(thetasB, trainSetB, accuracy, regMode, regParam, paillierPublicKeyB)
	if err != nil {
		log.Printf("calLocalGradientPart for A err is %v", err)
		return 0, err
	}

	// 参与方A计算加密梯度信息
	// 参与方A执行同态运算encGraForA = encByB(predictValue(j-A)*xAj(i))
	//									 + encByB(predictValue(j-B) - realValue(j)) * xAj(i)
	//									 + encByB(RanNumA)
	// 其中，encByB(predictValue(j-A)*xAj(i))是A用B的公钥加密得到的，A自己产生随机数RanNumA再用B的公钥加密，得到encByB(RanNumA)
	rawLocalGradientPartA := localGradientPartA.RawPart
	encLocalGradientPartB := localGradientPartB.EncPart

	// 通过损失函数评估损失

	encCostForA, err := xcc.LinRegVLEvaluateEncCost(rawLocalGradientPartA, encLocalGradientPartB, trainSetA, paillierPublicKeyB)
	if err != nil {
		log.Printf("evaluateEncLocalCost for A err is %v", err)
		return 0, err
	}

	decCostForA := xcc.LinRegVLDecryptCost(encCostForA.EncCost, paillierPrivateKeyB)

	// 参与方A从decCostForA中移除随机数RanNumA，得到最终用来更新损失函数的计算结果
	realCostForA := xcc.LinRegVLRetrieveRealCost(decCostForA, accuracy, encCostForA.RandomNoise)
	costForA := xcc.LinRegVLCalCost(realCostForA)

	return costForA, err
}

// readFeaturesFromCSVFile 从 csv 文件中读取样本特征
func readFeaturesFromCSVFile(path string) ([]*ml_common.DataFeature, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	plainFile := string(content)
	r := csv.NewReader(strings.NewReader(plainFile))
	ss, _ := r.ReadAll()

	features, err := xcc.LinRegImportFeatures(ss)
	if err != nil {
		return nil, err
	}
	return features, nil
}
