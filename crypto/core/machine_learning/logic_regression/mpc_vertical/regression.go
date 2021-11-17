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

package mpc_vertical

import (
	"log"
	"math"
	"math/big"
	"strconv"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/rand"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

// 纵向联合学习，基于半同态加密方案的多元逻辑回归算法
// PHE Multiple Variable Logic Regression Model based on Gradient Descent method

// LocalGradAndCostPart 迭代的中间参数，包含未加密的和同态加密参数，用于计算梯度和损失
type LocalGradAndCostPart struct {
	EncPart *EncLocalGradAndCostPart // 加密参数
	RawPart *RawLocalGradAndCostPart // 原始参数
}

// EncLocalGradAndCostPart 迭代的中间同态加密参数，用于计算梯度和损失
type EncLocalGradAndCostPart struct {
	EncPart1   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算y - 0.5，无标签方: 计算preValA; 并使用公钥pubKey-B进行同态加密;
	EncPart2   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算(y - 0.5)*preValB，无标签方:计算preValA^2/8; 并使用公钥pubKey-B进行同态加密
	EncPart3   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算preValB^2/8，并使用公钥pubKey-B进行同态加密
	EncPart4   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算preValB/4，并使用公钥pubKey-B进行同态加密
	EncPart5   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算0.5 + preValB/4 - y，并使用公钥pubKey-B进行同态加密
	EncRegCost *big.Int         // 正则化损失，被同态加密
}

// RawLocalGradAndCostPart 迭代中间同态加密参数，用于计算梯度和损失
type RawLocalGradAndCostPart struct {
	RawPart1   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算y - 0.5, 无标签方: 计算preValA
	RawPart2   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算(y - 0.5)*preValB， 无标签方: 计算preValA^2/8
	RawPart3   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算preValB^2/8
	RawPart4   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算preValB/4
	RawPart5   map[int]*big.Int // 对每一条数据（ID编号），有标签方：计算0.5 + preValB/4 - y
	RawRegCost *big.Int         // 正则化损失
}

// ---------------------- 纵向联合学习的模型训练原理相关 start ----------------------
//
// 我们回忆一下，logic regression的损失函数和梯度
//
// ---------- 损失的计算 ----------
//
// 在多元逻辑回归中，我们改用交叉熵(Cross Entropy)，来作为损失函数。原因在于，我们在模型中引入了Sigmoid函数。
// Sigmoid函数：1/(1+e^-x)，也就是说，最终模型为：1/(1+e^-(w*x)) = 1/(1+e^-(θ(0) + θ(1)*x(1) + θ(2)*x(2) + ... + θ(n)*x(n)))
// Sigmoid函数的特性是函数输出为0和1之间。 我们利用该特性来解决二分类问题。
// 但是，引入该函数后，如果继续使用MSE来作为损失函数，该函数不是一个凹函数或者凸函数。
// 也就是说，存在大量的局部最优点，导致难以使用梯度下降方法来寻找到全局最优点，进而确定模型的最佳参数。
//
// 交叉熵损失函数：
// 总特征数量n, 总样本数量m
// J(θ)=-1/m*CostSum(hθ(x(j)),y(j))
//
// for j:=0; j++; j<m
// {
//		CostSum += y(j)*log(hθ(x(j)) + (1-y(j))*log(1-hθ(x(j)))
// }
// 其中，y的值总是为0或者为1
//
// hθ(x(j)) = 1/(1+e^-(w'x)) = 1/(1+e^-(θ(0) + θ(1)*x(1) + θ(2)*x(2) + ... + θ(n)*x(n)))
//
// 参数集合w = [θ(1),θ(2),...,θ(n)]
// 整个模型的参数集合w' = (w, θ(0)) = [θ(0),θ(1),θ(2),...,θ(n)]
//
// 如果要完成去中心化计算损失函数，核心是通过同态和随机mask，完成log(hθ(x(j))和log(1-hθ(x(j))的计算，而y(j)和(1-y(j))由标签方掌握，可以基于同态做明文*密文的运算。
// 问题是，我们可以通过同态和随机mask完成线性方程式的计算，log函数并不支持我们修改过的同态函数，hθ函数中的e^x也不支持同态
// 因此，需要把核心函数log(hθ(x(j))和log(1-hθ(x(j))进行泰勒展开。预估泰勒展开到二次项可以满足精度要求。
//
// 把hθ(x(j)的多元变量考虑成1个x（这不会影响泰勒展开求导），用Sigmoid函数：1/(1+e^-x)来表示。然后对log(hθ(x(j))和log(1-hθ(x(j))来做泰勒展开到二次项。
//
// log(hθ) = ln(1/(1+e^-x))，求导：
// 1阶导数为：f'(x) = (ln(1+e^-x)^-1)' = (-1*ln(1+e^-x))' = -(ln(1+e^-x))' = -(-1 * e^-x)/(1+e^-x) = e^-x/(1+e^-x)
// 2阶导数为：f''(x) = (e^-x * (1+e^-x)^-1)' = (e^-x)' * (1+e^-x)^-1 + (e^-x) * ((1+e^-x)^-1)'
//					= -e^-x / (1+e^-x) + e^-x * e^-x / (1+e^-x)^2
// f(0)=ln(0.5), f'(0)=0.5, f''(0)=-1/4
// 所以，log(hθ) = ln(1/(1+e^-x))进行泰勒展开到二次项，结果为：f(x)=f(0)+f'(0)(x)+f''(0)(x^2)/2! = ln(0.5) + x/2 - x^2/8
//
// log(1-hθ) = ln(1 - 1/(1+e^-x))，求导：
// 1阶导数为：f'(x) = (ln(1 - 1/(1+e^-x)))' = (ln(e^-x / (1+e^-x)))' = (1+e^-x)/e^-x * (e^-x / (1+e^-x))' = -1/(1+e^-x)
// 2阶导数为：f''(x) = (-1/(1+e^-x))' = -((1+e^-x)^-1)' = -1 * -1 * (1+e^-x)^-2 * -1 * e^-x = -e^-x / (1+e^-x)^2
// f(0)=ln(0.5), f'(0)=-0.5, f''(0)=-1/4
// 所以，log(1-hθ) = ln(1 - 1/(1+e^-x))进行泰勒展开到二次项，结果为：f(x)=f(0)+f'(0)(x)+f''(0)(x^2)/2! = ln(0.5) - x/2 - x^2/8
//
// 此外，在计算x^2时， (predictValue(j-A) + predictValue(j-B))^2
//		= (A(j) + B(j))^2
//		= A(j)^2 + B(j)^2 + 2*A(j)*B(j)
//
// 综上，考虑两方场景，也就是无标签方A（部分特征）和有标签方B(部分特征+标签)
//
// 损失函数：
// J(θ)=-1/m*CostSum(hθ(x(j)),y(j))
//
// for j:=0; j++; j<m
// {
//		CostSum += y(j)*log(hθ(x(j)) + (1-y(j))*log(1-hθ(x(j))
// }
// 其中，y的值总是为0或者为1
//
// 那么，Cost = y*log(hθ(x) + (1-y)*log(1-hθ(x)) = y*(ln(0.5) + x/2 - x^2/8) + (1-y)*(ln(0.5) - x/2 - x^2/8)
//			= ln(0.5) + (y - 0.5)*x - x^2/8 = ln(0.5) + (y - 0.5)*(preValA + preValB) - (preValA + preValB)^2/8
//			= ln(0.5) + (y - 0.5)*preValA + (y - 0.5)*preValB - preValA^2/8 - preValB^2/8 - preValA*preValB/4
//
// 无标签方A（部分特征）通过损失函数评估损失时：
// 		step 1. B用自己的同态公钥加密数据，获得encByB(y - 0.5)、encByB((y - 0.5)*preValB)、encByB(preValB^2/8)、encByB(preValB/4)，将加密数据发送给A
//		step 2. A用B的公钥执行同态运算：ln(0.5) + encByB(y - 0.5)*preValA + encByB((y - 0.5)*preValB) - preValA^2/8
//						- encByB(preValB^2/8) - preValA*encByB(preValB/4) + ranNumA
//		step 3. A将同态运算结果发给B
//		step 4. B使用自己的私钥解密同态运算结果，将解密结果发给A
//		step 5. A从解密结果中移除ranNumA，获得真正结果，用来更新xAj的梯度graAj
//
// 有标签方B(部分特征+标签)通过损失函数评估损失时：
// 		step 1. A用自己的同态公钥加密数据，获得encByA(preValA)、encByA(preValA^2/8)，将加密数据发送给B
//		step 2. B用A的公钥执行同态运算：ln(0.5) + (y - 0.5)*encByA(preValA) + (y - 0.5)*preValB - encByA(preValA^2/8)
//						- preValB^2/8 - encByA(preValA)*preValB/4 + ranNumB
//		step 3. B将同态运算结果发给A
//		step 4. A使用自己的私钥解密同态运算结果，将解密结果发给B
//		step 5. B从解密结果中移除ranNumB，获得真正结果，用来更新xBj的梯度graBj
//
// 根据最近两次损失函数的差值评估整荡幅度是否符合目标要求，是否可以结束梯度下降的收敛计算
//
// 特别提醒：
// 在上面计算时，需要考虑精度，同态加密（仅支持整数），所以加密前放大指定精度，乘法运算的常数项又放大了指定精度。所以，加法算式中的每一项需要放大（精度*精度），最后在计算结束后，进行缩小。
//
// ---------- 梯度的计算 ----------
//
// 根据上文计算损失函数时的介绍
// 计算出w' = (w, θ(0)) = [θ(0),θ(1),θ(2),...,θ(n)]中的每个θ(i)，来得到模型w'
// for i:=0; i++; i<n
// {
// 		// 计算第i个特征的参数θ(i):
//		θ(i) = θ(i) − α*Grad(i)
// }
//
// 第i个特征的Grad(i):
// for j:=0; j++; i<m
// {
//	Grad(i) += (predictValue(j) - realValue(j))*trainSet[j][i]
// }
//
// predictValue(j) = hθ(x(j)) = 1/(1+e^-(w'x)) = 1/(1+e^-(θ(0) + θ(1)*x(1) + θ(2)*x(2) + ... + θ(n)*x(n)))
//
// Grad(i) = Grad(i)/m
//
// 同样的，predictValue(j)的计算，需要多方合作完成。
//
// 如果要完成去中心化计算梯度，核心是通过同态和随机mask，完成hθ(x(j))的计算，而y(j)由标签方掌握，可以基于同态做明文*密文的运算。
// 问题是，我们可以通过同态和随机mask完成线性方程式的计算，以e为底数的指数函数e^x并不支持同态
// 因此，需要把核心函数hθ(x(j))进行泰勒展开。预估泰勒展开到二次项可以满足精度要求。
//
// hθ(x) = 1/(1+e^-x))，求导：
// 1阶导数为：f'(x) = ((1+e^-x)^-1)' = -1 * (1+e^-x)^-2 * e^-x * -1 = e^-x / (1+e^-x)^2
// 2阶导数为：f''(x) = (e^-x * (1+e^-x)^-2)' = (e^-x)' * (1+e^-x)^-2 + (e^-x) * ((1+e^-x)^-2)'
//					= -e^-x / (1+e^-x)^2 + 2 * e^-x * e^-x / (1+e^-x)^3
// f(0)=0.5, f'(0)=1/4, f''(0)=0
// 所以，hθ(x) = 1/(1+e^-x)进行泰勒展开到二次项，结果为：f(x)=f(0)+f'(0)(x)+f''(0)(x^2)/2! = 0.5 + x/4
//
// 那么，梯度的计算：Grad(i) = (predictValue(j) - realValue(j))*trainSet[j][i]
//						= (0.5 + x/4 - y)*x(i) = (0.5 + (preValA + preValB)/4 - y)*x(i)
//						= (0.5 + preValA/4 + (preValB/4 - y))*x(i)
//						= x(i)*preValA/4 + x(i)*(0.5 + preValB/4 - y)
//
// 无标签方A（部分特征）计算本地特征的梯度时：
// 		step 1. B用自己的同态公钥加密数据，获得encByB(0.5 + preValB/4 - y)，将加密数据发送给A
//		step 2. A用B的公钥执行同态运算：x(i)*preValA/4 + x(i)*encByB(0.5 + preValB/4 - y) + ranNumA
//		step 3. A将同态运算结果发给B
//		step 4. B使用自己的私钥解密同态运算结果，将解密结果发给A
//		step 5. A从解密结果中移除ranNumA，获得真正结果，用来更新xAj的梯度graAj
//
// 有标签方B(部分特征+标签)计算本地特征的梯度时：
// 		step 1. A用自己的同态公钥加密数据，获得encByA(preValA/4)，将加密数据发送给B
//		step 2. B用A的公钥执行同态运算：x(i)*encByA(preValA/4) + x(i)*(0.5 + preValB/4 - y) + ranNumB
//		step 3. B将同态运算结果发给A
//		step 4. A使用自己的私钥解密同态运算结果，将解密结果发给B
//		step 5. B从解密结果中移除ranNumB，获得真正结果，用来更新xBj的梯度graBj
//
// ---------------------- 纵向联合学习的模型训练原理相关 end ----------------------

// StandardizeDataSet 将样本数据集合进行标准化处理，将均值变为0，标准差变为1
// Z-score Normalization
// x' = (x - avgerage(x)) / standard_deviation(x)
// 特别注意：对于逻辑回归，标签值不做标准化处理
func StandardizeDataSet(sourceDataSet *common.DataSet, label string) *common.StandardizedDataSet {
	// Calculate the means and standard deviations for each feature

	// 特征的均值
	xbarParams := make(map[string]float64)
	// 特征的标准差
	sigmaParams := make(map[string]float64)

	for _, feature := range sourceDataSet.Features {
		var sum float64 = 0      // 样本总值
		var sigmaSum float64 = 0 // 每个样本值与全体样本值的平均数之差的平方值之和

		// 计算总值
		for _, value := range feature.Sets {
			sum += value
		}

		// 计算均值
		xbars := sum / float64(len(feature.Sets))

		// 计算每个样本值与全体样本值的平均数之差的平方值之和
		for _, value := range feature.Sets {
			sigmaSum += math.Pow(value-xbars, 2)
		}

		// 计算方差: 每个样本值与全体样本值的平均数之差的平方值的平均数
		variance := sigmaSum / float64(len(feature.Sets))

		// 计算标准差: 方差的平方根
		sigmas := math.Sqrt(variance)

		xbarParams[feature.FeatureName] = xbars
		sigmaParams[feature.FeatureName] = sigmas
	}

	// 遍历各个特征维度的特征值集合，根据均值和标准差进行标准化处理
	var newFeatures []*common.DataFeature

	for _, feature := range sourceDataSet.Features {
		// 先获得该特征维度的名称
		var newDataFeature common.DataFeature
		newDataFeature.FeatureName = feature.FeatureName

		// 声明该特征维度的经过标准化处理后的特征值集合
		newSets := make(map[int]float64)

		// 如果是目标维度
		if feature.FeatureName == label {
			// 直接赋值，不执行标准化操作
			for key, value := range feature.Sets {
				newSets[key] = value
			}
		} else {
			xbars := xbarParams[feature.FeatureName]
			sigmas := sigmaParams[feature.FeatureName]

			// 对特征值进行标准化处理，对数据集的每一条数据的每个特征的减去该特征均值后除以特征标准差。
			for key, value := range feature.Sets {
				newValue := (value - xbars) / sigmas
				newSets[key] = newValue
			}
		}

		newDataFeature.Sets = newSets

		newFeatures = append(newFeatures, &newDataFeature)
	}

	var standardizedDataSet common.StandardizedDataSet

	standardizedDataSet.Features = newFeatures
	standardizedDataSet.XbarParams = xbarParams
	standardizedDataSet.SigmaParams = sigmaParams
	standardizedDataSet.OriginalFeatures = sourceDataSet.Features

	return &standardizedDataSet
}

// PreProcessDataSet 预处理标签方的标准化数据集
// 将样本数据集转化为一个m*(n+2)的矩阵, 其中每行对应一个样本，每行的第一列为编号ID，第二列为1（截距intercept），其它列分别对应一种特征的值
func PreProcessDataSet(sourceDataSet *common.StandardizedDataSet, label string) *common.TrainDataSet {
	var featureNames []string

	// 特征数量 - 有多少个特征维度
	featureNum := len(sourceDataSet.Features)

	// 样本数量 - 取某一个特征的的样本数量
	sampleNum := len(sourceDataSet.Features[0].Sets)

	// 将样本数据集转化为一个m*(n+2)的矩阵, 其中每行对应一个样本，每行的第一列为编号ID，第二列为1（截距intercept），其它列分别对应一种特征的值
	trainSet := make([][]float64, sampleNum)
	originalTrainSet := make([][]float64, sampleNum)
	for i := 0; i < sampleNum; i++ {
		trainSet[i] = make([]float64, featureNum+2)
		originalTrainSet[i] = make([]float64, featureNum+2)
	}

	i := 0 // i表示遍历到第i个样本
	for key, _ := range sourceDataSet.Features[0].Sets {
		trainSet[key][0] = float64(key)
		originalTrainSet[key][0] = float64(key)

		// 每个样本的第2列为1
		trainSet[key][1] = 1
		originalTrainSet[key][1] = 1
		i++
	}

	// 遍历所有的特征维度
	i = 1 // i表示遍历到第i个特征维度
	for _, feature := range sourceDataSet.Features {
		// 遍历每个特征维度的每个样本
		// 如果是目标维度
		if feature.FeatureName == label {
			// 遍历某个特征维度的所有样本
			for key, value := range feature.Sets {
				// 把值放在训练样本的最后一列
				trainSet[key][featureNum+1] = value
			}
		} else { // 如果不是目标维度
			for key, value := range feature.Sets {
				trainSet[key][i+1] = value
			}
			i++

			featureNames = append(featureNames, feature.FeatureName)
		}
	}

	// 遍历所有的特征维度
	i = 1 // i表示遍历到第i个特征维度
	for _, feature := range sourceDataSet.OriginalFeatures {
		// 遍历每个特征维度的每个样本
		// 如果是目标维度
		if feature.FeatureName == label {
			// 遍历某个特征维度的所有样本
			for key, value := range feature.Sets {
				// 把值放在训练样本的最后一列
				originalTrainSet[key][featureNum+1] = value
			}
		} else { // 如果不是目标维度
			for key, value := range feature.Sets {
				originalTrainSet[key][i+1] = value
			}
			i++
		}
	}

	if len(label) != 0 {
		featureNames = append(featureNames, label)
	}

	dataSet := &common.TrainDataSet{
		FeatureNames:     featureNames,              // 特征名称的集合
		TrainSet:         trainSet,                  // 特征集合
		XbarParams:       sourceDataSet.XbarParams,  // 特征的均值
		SigmaParams:      sourceDataSet.SigmaParams, // 特征的标准差
		OriginalTrainSet: originalTrainSet,          // 原始特征集合
	}

	return dataSet
}

// PreProcessDataSetNoTag 预处理非标签方的标准化训练数据集
// 将样本数据集转化为一个m*(n+1)的矩阵, 其中每行对应一个样本，每行的第一列为编号ID，其它列分别对应一种特征的值
func PreProcessDataSetNoTag(sourceDataSet *common.StandardizedDataSet) *common.TrainDataSet {
	var featureNames []string

	// 特征数量 - 有多少个特征维度
	featureNum := len(sourceDataSet.Features)

	// 样本数量 - 取某一个特征的的样本数量
	sampleNum := len(sourceDataSet.Features[0].Sets)

	// 将样本数据集转化为一个m*(n+2)的矩阵, 其中每行对应一个样本，每行的第一列为编号ID，第二列为1（截距intercept），其它列分别对应一种特征的值
	trainSet := make([][]float64, sampleNum)
	originalTrainSet := make([][]float64, sampleNum)
	for i := 0; i < sampleNum; i++ {
		trainSet[i] = make([]float64, featureNum+1)
		originalTrainSet[i] = make([]float64, featureNum+1)
	}

	i := 0 // i表示遍历到第i个样本
	for key, _ := range sourceDataSet.Features[0].Sets {
		// 每个样本的第1列为编号ID
		trainSet[key][0] = float64(key)
		originalTrainSet[key][0] = float64(key)
		i++
	}

	// 遍历所有的特征维度
	i = 1 // i表示遍历到第i个特征维度
	for _, feature := range sourceDataSet.Features {
		// 遍历每个特征维度的每个样本
		for key, value := range feature.Sets {
			trainSet[key][i] = value
		}
		i++

		featureNames = append(featureNames, feature.FeatureName)

	}

	// 遍历所有的特征维度
	i = 1 // i表示遍历到第i个特征维度
	for _, feature := range sourceDataSet.OriginalFeatures {
		// 遍历每个特征维度的每个样本
		for key, value := range feature.Sets {
			originalTrainSet[key][i] = value
		}
		i++
	}

	dataSet := &common.TrainDataSet{
		FeatureNames:     featureNames,              // 特征名称的集合
		TrainSet:         trainSet,                  // 特征集合
		XbarParams:       sourceDataSet.XbarParams,  // 特征的均值
		SigmaParams:      sourceDataSet.SigmaParams, // 特征的标准差
		OriginalTrainSet: originalTrainSet,          // 原始特征集合
	}

	return dataSet
}

// CalLocalGradAndCostTagPart 标签方为计算本地模型中，每个特征的参数做准备，计算本地同态加密结果
// 对每一条数据（ID编号j），计算y - 0.5，(y - 0.5)*preValB，preValB^2/8，preValB/4，并对它们分别使用公钥pubKey-B进行同态加密
// 注意：高性能的同态运算仅能处理整数，对于浮点数有精度损失，所以必须在参数中指定精度来进行处理
//
// - thetas 上一轮训练得到的模型参数
// - trainSet 预处理过的训练数据
// - accuracy 同态加解密精确到小数点后的位数
// - regMode 正则模式
// - regParam 正则参数
// - publicKey 标签方同态公钥
func CalLocalGradAndCostTagPart(thetas []float64, trainSet [][]float64, accuracy int, regMode int, regParam float64, publicKey *paillier.PublicKey) (*LocalGradAndCostPart, error) {
	// 对每一条数据（ID编号），计算y - 0.5
	rawPart1 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算(y - 0.5)*preValB
	rawPart2 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算preValB^2/8
	rawPart3 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算preValB/4
	rawPart4 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算0.5 + preValB/4 - y
	rawPart5 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算y - 0.5，并使用公钥pubKey-B进行同态加密
	encPart1 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算(y - 0.5)*preValB，并使用公钥pubKey-B进行同态加密
	encPart2 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算preValB^2/8，并使用公钥pubKey-B进行同态加密
	encPart3 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算preValB/4，并使用公钥pubKey-B进行同态加密
	encPart4 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算0.5 + preValB/4 - y，并使用公钥pubKey-B进行同态加密
	encPart5 := make(map[int]*big.Int)

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		id, predictValue := predict(thetas, trainSet[i])

		// 计算y - 0.5
		rawPart1Value := trainSet[i][len(trainSet[i])-1] - 0.5
		// 精度处理转int后，才可以使用同态加密和同态运算
		rawPart1ValueInt := big.NewInt(int64(math.Round(rawPart1Value * math.Pow(10, float64(accuracy)))))
		rawPart1[id] = rawPart1ValueInt
		// encByB(y - 0.5)
		// 使用同态公钥加密数据
		encPart1Value, err := publicKey.EncryptSupNegNum(rawPart1ValueInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}
		encPart1[id] = encPart1Value

		// 计算(y - 0.5)*preValB
		rawPart2Value := rawPart1Value * predictValue
		// 精度处理转int后，才可以使用同态加密和同态运算
		rawPart2ValueInt := big.NewInt(int64(math.Round(rawPart2Value * math.Pow(10, float64(accuracy)))))
		rawPart2[id] = rawPart2ValueInt
		// encByB((y - 0.5)*preValB)
		// 使用同态公钥加密数据
		encPart2Value, err := publicKey.EncryptSupNegNum(rawPart2ValueInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}
		encPart2[id] = encPart2Value

		// 计算preValB^2/8
		rawPart3Value := math.Pow(predictValue, 2) / 8
		// 精度处理转int后，才可以使用同态加密和同态运算
		rawPart3ValueInt := big.NewInt(int64(math.Round(rawPart3Value * math.Pow(10, float64(accuracy)))))
		rawPart3[id] = rawPart3ValueInt
		// encByB(preValB^2/8)
		// 使用同态公钥加密数据
		encPart3Value, err := publicKey.EncryptSupNegNum(rawPart3ValueInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}
		encPart3[id] = encPart3Value

		// 计算preValB/4
		rawPart4Value := predictValue / 4
		// 精度处理转int后，才可以使用同态加密和同态运算
		rawPart4ValueInt := big.NewInt(int64(math.Round(rawPart4Value * math.Pow(10, float64(accuracy)))))
		rawPart4[id] = rawPart4ValueInt
		// encByB(preValB/4)
		// 使用同态公钥加密数据
		encPart4Value, err := publicKey.EncryptSupNegNum(rawPart4ValueInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}
		encPart4[id] = encPart4Value

		// 计算0.5 + preValB/4 - y
		rawPart5Value := 0.5 + predictValue*0.25 - trainSet[i][len(trainSet[i])-1]
		// 精度处理转int后，才可以使用同态加密和同态运算
		rawPart5ValueInt := big.NewInt(int64(math.Round(rawPart5Value * math.Pow(10, float64(accuracy)))))
		rawPart5[id] = rawPart5ValueInt
		// encByB(0.5 + preValB/4 - y)
		// 使用同态公钥加密数据
		encPart5Value, err := publicKey.EncryptSupNegNum(rawPart5ValueInt)
		encPart5[id] = encPart5Value
	}

	regCost := 0.0
	switch regMode {
	case common.RegLasso:
		regCost = CalLassoRegCost(thetas, len(trainSet), regParam)
	case common.RegRidge:
		regCost = CalRidgeRegCost(thetas, len(trainSet), regParam)
	default:
	}

	// 精度处理转int后，才可以使用同态加密和同态运算
	rawRegCost := big.NewInt(int64(math.Round(regCost * math.Pow(10, float64(accuracy)))))
	// 使用同态公钥加密数据
	encRegCost, err := publicKey.EncryptSupNegNum(rawRegCost)
	if err != nil {
		log.Printf("Paillier Encrypt err is %v", err)
		return nil, err
	}

	// 生成标签方的中间原始参数，用于计算梯度和损失
	rawPart := &RawLocalGradAndCostPart{
		RawPart1:   rawPart1,
		RawPart2:   rawPart2,
		RawPart3:   rawPart3,
		RawPart4:   rawPart4,
		RawPart5:   rawPart5,
		RawRegCost: rawRegCost,
	}

	// 生成标签方的中间同态加密参数，用于计算梯度和损失
	encPart := &EncLocalGradAndCostPart{
		EncPart1:   encPart1,
		EncPart2:   encPart2,
		EncPart3:   encPart3,
		EncPart4:   encPart4,
		EncPart5:   encPart5,
		EncRegCost: encRegCost,
	}

	// 生成标签方的中间同态加密参数，用于计算梯度和损失
	localGradAndCostTagPart := &LocalGradAndCostPart{
		RawPart: rawPart,
		EncPart: encPart,
	}

	return localGradAndCostTagPart, nil
}

// CalLocalGradAndCostPart 非标签方为计算本地模型中，每个特征的参数做准备，计算本地同态加密结果
// 对于梯度：A计算preValA/4，再用自己的同态公钥加密数据，获得encByA(preValA/4)，将加密数据发送给B
// 对于损失：A计算preValA、preValA^2/8，再用自己的同态公钥加密数据，获得encByA(preValA)、encByA(preValA^2/8)，将加密数据发送给B
//
// - thetas 上一轮训练得到的模型参数
// - trainSet 预处理过的训练数据
// - accuracy 同态加解密精确到小数点后的位数
// - regMode 正则模式
// - regParam 正则参数
// - publicKey 非标签方同态公钥
func CalLocalGradAndCostPart(thetas []float64, trainSet [][]float64, accuracy int, regMode int, regParam float64, publicKey *paillier.PublicKey) (*LocalGradAndCostPart, error) {
	// 对每一条数据（ID编号），计preValA
	rawPart1 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算preValA^2/8
	rawPart2 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算preValA/4

	// 对每一条数据（ID编号），计preValA，并使用公钥pubKey-A进行同态加密
	encPart1 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算preValA^2/8，并使用公钥pubKey-A进行同态加密
	encPart2 := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算preValA/4，并使用公钥pubKey-A进行同态加密

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		id, predictValue := predictNoTag(thetas, trainSet[i])

		// 精度处理转int后，才可以使用同态加密和同态运算，放大1个精度
		predictValueInt := big.NewInt(int64(math.Round(predictValue * math.Pow(10, float64(accuracy)))))
		// 使用同态公钥加密数据
		encPredictValue, err := publicKey.EncryptSupNegNum(predictValueInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 计算preValA^2/8，放大1个精度
		predictValue2 := math.Pow(predictValue, 2) / 8
		predictValue2Int := big.NewInt(int64(math.Round(predictValue2 * math.Pow(10, float64(accuracy)))))
		// 使用同态公钥加密数据
		encPredictValue2, err := publicKey.EncryptSupNegNum(predictValue2Int)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		rawPart1[id] = predictValueInt
		rawPart2[id] = predictValue2Int

		encPart1[id] = encPredictValue
		encPart2[id] = encPredictValue2
	}

	regCost := 0.0
	switch regMode {
	case common.RegLasso:
		regCost = CalLassoRegCost(thetas, len(trainSet), regParam)
	case common.RegRidge:
		regCost = CalRidgeRegCost(thetas, len(trainSet), regParam)
	default:
	}

	// 精度处理转int后，才可以使用同态加密和同态运算
	rawRegCost := big.NewInt(int64(math.Round(regCost * math.Pow(10, float64(accuracy)))))
	// 使用同态公钥加密数据
	encRegCost, err := publicKey.EncryptSupNegNum(rawRegCost)
	if err != nil {
		log.Printf("Paillier Encrypt err is %v", err)
		return nil, err
	}

	// 生成非标签方的中间原始参数，用于计算梯度和损失
	rawPart := &RawLocalGradAndCostPart{
		RawPart1:   rawPart1,
		RawPart2:   rawPart2,
		RawRegCost: rawRegCost,
	}

	// 生成非标签方的中间同态加密参数，用于计算梯度和损失
	encPart := &EncLocalGradAndCostPart{
		EncPart1:   encPart1,
		EncPart2:   encPart2,
		EncRegCost: encRegCost,
	}

	// 生成费标签方的中间同态加密参数，用于计算梯度和损失
	localGradAndCostPart := &LocalGradAndCostPart{
		RawPart: rawPart,
		EncPart: encPart,
	}

	return localGradAndCostPart, nil
}

// CalLassoRegCost 计算使用L1 Lasso进行正则化后的损失函数，来评估当前模型的损失
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// L1 = λ/m * (|θ(0)| + |θ(1)| + ... + |θ(n)|)，其中，|θ|表示θ的绝对值
//
// - thetas 当前的模型参数
// - trainSetSize 训练样本个数
// - regParam 正则参数
func CalLassoRegCost(thetas []float64, trainSetSize int, regParam float64) float64 {
	lassoRegCost := 0.0
	// 遍历特征的每一行
	for i := 0; i < len(thetas); i++ {
		thetasCost := math.Abs(thetas[i])

		lassoRegCost += thetasCost
	}

	lassoRegCost = regParam * lassoRegCost / float64(trainSetSize)
	return lassoRegCost
}

// CalRidgeRegCost 计算使用L2 Ridge进行正则化后的损失函数，来评估当前模型的损失
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// L2 = λ/2m * (θ(0)^2 + θ(1)^2 + ... + θ(n)^2)
//
// - thetas 当前的模型参数
// - trainSetSize 训练样本个数
// - regParam 正则参数
func CalRidgeRegCost(thetas []float64, trainSetSize int, regParam float64) float64 {
	ridgeRegCost := 0.0
	// 遍历特征的每一行
	for i := 0; i < len(thetas); i++ {
		thetasCost := math.Pow(thetas[i], 2)

		ridgeRegCost += thetasCost
	}

	ridgeRegCost = regParam * ridgeRegCost / (2 * float64(trainSetSize))
	return ridgeRegCost
}

// CalEncLocalGradient 非标签方聚合双方的中间加密参数，为本地特征计算模型参数
// 计算本地加密梯度，并提交随机数干扰
// 参与方A执行同态运算
// 梯度的计算：Grad(i) = (predictValue(j) - realValue(j))*trainSet[j][i]
//						= (0.5 + x/4 - y)*x(i) = (0.5 + (preValA + preValB)/4 - y)*x(i)
//						= (0.5 + preValA/4 + (preValB/4 - y))*x(i)
//						= x(i)*preValA/4 + x(i)*(0.5 + preValB/4 - y)
//
// - localPart 非标签方本地的明文梯度数据
// - tagPart 标签方的加密梯度数据
// - trainSet 非标签方训练样本集合
// - featureIndex 指定特征的索引
// - accuracy 同态加解密精度
// - publicKey 标签方同态公钥
func CalEncLocalGradient(localPart *RawLocalGradAndCostPart, tagPart *EncLocalGradAndCostPart, trainSet [][]float64, featureIndex, accuracy int, publicKey *paillier.PublicKey) (*common.EncLocalGradient, error) {
	encGradMap := make(map[int]*big.Int)

	// 生成 RanNumA，用于梯度值的混淆
	randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, rand.KeyLengthInt64)
	if err != nil {
		return nil, err
	}
	ranNum := big.NewInt(0).SetBytes(randomBytes)

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		// 获取该行数据的id
		// TODO: 后续优化下数据结构，来提升性能
		id := int(math.Floor(trainSet[i][0] + 0.5))

		// 计算x(i)/4*scale精度
		scaleFactor := big.NewInt(int64(math.Round(trainSet[i][featureIndex+1] * 0.25 * math.Pow(10, float64(accuracy)))))
		// 计算x(i)*preValA/4，1个精度的明文*scale精度
		rawValue1 := new(big.Int).Mul(localPart.RawPart1[id], scaleFactor)

		// 计算x(i)*scale精度
		scaleFactor = big.NewInt(int64(math.Round(trainSet[i][featureIndex+1] * math.Pow(10, float64(accuracy)))))
		// 计算x(i)*encByB(0.5 + preValB/4 - y)，1个精度的密文*scale精度
		encValue2 := publicKey.CypherPlainMultiply(tagPart.EncPart5[id], scaleFactor)

		// 计算 x(i)*preValA/4 + x(i)*encByB(0.5 + preValB/4 - y) + ranNumA
		// 密文与原文的同态加法
		addResult := publicKey.CypherPlainsAdd(encValue2, rawValue1, ranNum)

		encGradMap[id] = addResult
	}

	encLocalGradient := &common.EncLocalGradient{
		EncGrad:     encGradMap,
		RandomNoise: ranNum,
	}

	return encLocalGradient, nil
}

// CalEncLocalGradientTagPart 标签方聚合双方的中间加密参数，为本地特征计算模型参数
// 计算本地加密梯度，并提交随机数干扰
// 参与方B计算加密梯度信息
// 梯度的计算：Grad(i) = (predictValue(j) - realValue(j))*trainSet[j][i]
//						= (0.5 + x/4 - y)*x(i) = (0.5 + (preValA + preValB)/4 - y)*x(i)
//						= (0.5 + preValA/4 + (preValB/4 - y))*x(i)
//						= x(i)*preValA/4 + x(i)*(0.5 + preValB/4 - y)
//
// - localPart 标签方本地的明文梯度数据
// - otherPart 非标签方的加密梯度数据
// - trainSet 标签方训练样本集合
// - featureIndex 指定特征的索引
// - accuracy 同态加解密精度
// - publicKey 非标签方同态公钥
func CalEncLocalGradientTagPart(tagPart *RawLocalGradAndCostPart, otherPart *EncLocalGradAndCostPart, trainSet [][]float64, featureIndex, accuracy int, publicKey *paillier.PublicKey) (*common.EncLocalGradient, error) {
	encGradMap := make(map[int]*big.Int)

	// 生成 RanNumB，用于梯度值的混淆
	randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, rand.KeyLengthInt64)
	if err != nil {
		return nil, err
	}
	ranNum := big.NewInt(0).SetBytes(randomBytes)

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		// 获取该行数据的id
		// TODO: 后续优化下数据结构，来提升性能
		id := int(math.Floor(trainSet[i][0] + 0.5))

		// 计算x(i)/4*scale精度
		scaleFactor := big.NewInt(int64(math.Round(trainSet[i][featureIndex+1] * 0.25 * math.Pow(10, float64(accuracy)))))
		// 计算x(i)*encByA(preValA)/4，1个精度的密文*scale精度
		encValue1 := publicKey.CypherPlainMultiply(otherPart.EncPart1[id], scaleFactor)

		// 计算x(i)*scale精度
		scaleFactor = big.NewInt(int64(math.Round(trainSet[i][featureIndex+1] * math.Pow(10, float64(accuracy)))))
		// 计算x(i)*(0.5 + preValB/4 - y)，1个精度的密文*scale精度
		rawValue2 := new(big.Int).Mul(tagPart.RawPart5[id], scaleFactor)

		// 计算 x(i)*encByA(preValA)/4 + x(i)*(0.5 + preValB/4 - y) + ranNumB
		// 密文与原文的同态加法
		addResult := publicKey.CypherPlainsAdd(encValue1, rawValue2, ranNum)

		encGradMap[id] = addResult
	}

	encLocalGradient := &common.EncLocalGradient{
		EncGrad:     encGradMap,
		RandomNoise: ranNum,
	}

	return encLocalGradient, nil
}

// DecryptGradient 为另一参与方解密加密的梯度
// - encGradMap 加密的梯度信息
// - privateKey 己方同态私钥
func DecryptGradient(encGradMap map[int]*big.Int, privateKey *paillier.PrivateKey) map[int]*big.Int {
	// 解密后的梯度信息
	gradMap := make(map[int]*big.Int)

	for id, encGrad := range encGradMap {
		rawGrad := privateKey.DecryptSupNegNum(encGrad)

		gradMap[id] = rawGrad
	}

	return gradMap
}

// RetrieveRealGradient 从解密后的梯度信息中，移除随机数噪音，还原己方真实的梯度数据
// - decGradMap 解密的梯度信息
// - accuracy 同态加解密精度
// - randomInt 己方梯度的噪音值
func RetrieveRealGradient(decGradMap map[int]*big.Int, accuracy int, randomInt *big.Int) map[int]float64 {
	// 解密后的梯度信息（含随机数噪声）
	gradMap := make(map[int]float64)

	for id, decGrad := range decGradMap {
		minusRandomInt := new(big.Int).Mul(big.NewInt(-1), randomInt)
		rawGrad := new(big.Int).Add(decGrad, minusRandomInt)

		rawGradStr := rawGrad.String()
		rawGradFloat64, _ := strconv.ParseFloat(rawGradStr, 64)
		rawGradFloat64 = rawGradFloat64 / math.Pow(10, float64(accuracy)*2)

		gradMap[id] = rawGradFloat64
	}

	return gradMap
}

// CalGradient 根据还原的明文梯度数据计算梯度值
func CalGradient(gradMap map[int]float64) float64 {
	var decGradSum float64 = 0

	for _, decGrad := range gradMap {
		decGradSum += decGrad
	}

	gradient := decGradSum / float64(len(gradMap))

	return gradient
}

// CalGradientWithLassoReg 使用L1正则(Lasso)计算梯度
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// Grad_new(i) = (Grad_new(i) + λ*sgn(θ(i)))/m
// 其中，trainSet[j][i] 表示第j个样本的的第i个特征的值
func CalGradientWithLassoReg(thetas []float64, gradMap map[int]float64, featureIndex int, regParam float64) float64 {
	gradient := CalGradient(gradMap)

	// get sgn(θ(i)
	sgnTheta := 0.0
	switch {
	case thetas[featureIndex] > 0:
		sgnTheta = 1
	case thetas[featureIndex] < 0:
		sgnTheta = -1
	default: // θ(i)=0
		sgnTheta = 0
	}

	// λ*sgn(θ(i)))/m
	lassoReg := regParam * sgnTheta / float64(len(gradMap))

	gradientWithLassoReg := gradient + lassoReg

	return gradientWithLassoReg
}

// CalGradientWithRidgeReg 使用L2正则(Ridge)计算梯度
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// Grad_new(i) = (Grad_new(i) + λ*θ(i))/m
// 其中，trainSet[j][i] 表示第j个样本的的第i个特征的值
func CalGradientWithRidgeReg(thetas []float64, gradMap map[int]float64, featureIndex int, regParam float64) float64 {
	gradient := CalGradient(gradMap)

	// λ*θ(i))/m
	ridgeReg := regParam * thetas[featureIndex] / float64(len(gradMap))

	gradientWithRidgeReg := gradient + ridgeReg

	return gradientWithRidgeReg
}

// EvaluateEncLocalCost 非标签方根据损失函数来评估当前模型的损失，衡量模型是否已经收敛
// TODO 增加泛化支持：
// L1 = λ/m * (|θ(0)| + |θ(1)| + ... + |θ(n)|)，其中，|θ|表示θ的绝对值
// L2 = λ/2m * (θ(0)^2 + θ(1)^2 + ... + θ(n)^2)
//
// 参与方A执行同态运算ln(0.5) + encByB(y - 0.5)*preValA + encByB((y - 0.5)*preValB) - preValA^2/8
//				- encByB(preValB^2/8) - preValA*encByB(preValB/4) + ranNumA
//
// - localPart 本地的明文损失数据
// - tagPart 标签方的加密损失数据
// - trainSet 非标签方训练样本集合
// - accuracy 同态加解密精度
// - publicKey 标签方同态公钥
func EvaluateEncLocalCost(localPart *RawLocalGradAndCostPart, tagPart *EncLocalGradAndCostPart, trainSet [][]float64, accuracy int, publicKey *paillier.PublicKey) (*common.EncLocalCost, error) {
	costSum := make(map[int]*big.Int)

	// 生成 RanNumA，用于损失值的混淆
	randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, rand.KeyLengthInt64)
	if err != nil {
		return nil, err
	}
	ranNum := big.NewInt(0).SetBytes(randomBytes)

	// 计算ln(0.5)
	lnHalf := math.Log(0.5)
	// 2个精度的明文
	lnHalfValueInt := big.NewInt(int64(math.Round(lnHalf * math.Pow(10, 2*float64(accuracy)))))

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		// 获取该行数据的id
		// TODO: 后续优化下数据结构，来提升性能
		id := int(math.Floor(trainSet[i][0] + 0.5))

		// 计算encByB(y - 0.5)*preValA，2个精度的密文
		encValue1 := publicKey.CypherPlainMultiply(tagPart.EncPart1[id], localPart.RawPart1[id])

		// 计算encByB((y - 0.5)*preValB)，1个精度的密文*scale精度
		scaleFactor := big.NewInt(int64(math.Round(math.Pow(10, float64(accuracy)))))
		encValue2 := publicKey.CypherPlainMultiply(tagPart.EncPart2[id], scaleFactor)

		// 计算preValA^2/8，1个精度的原文*scale精度
		rawValue3 := new(big.Int).Mul(localPart.RawPart2[id], scaleFactor)
		// 计算-preValA^2/8
		rawValue3 = new(big.Int).Mul(rawValue3, big.NewInt(-1))

		// 计算encByB(preValB^2/8)，1个精度的密文*scale精度
		encValue4 := publicKey.CypherPlainMultiply(tagPart.EncPart3[id], scaleFactor)
		// 计算-encByB(preValB^2/8)
		encValue4 = publicKey.CypherPlainMultiply(encValue4, big.NewInt(-1))

		// 计算preValA*encByB(preValB/4)，2个精度的密文
		encValue5 := publicKey.CypherPlainMultiply(tagPart.EncPart4[id], localPart.RawPart1[id])
		// 计算-preValA*encByB(preValB/4)
		encValue5 = publicKey.CypherPlainMultiply(encValue5, big.NewInt(-1))

		// 计算 ln(0.5) + encByB(y - 0.5)*preValA + encByB((y - 0.5)*preValB) - preValA^2/8
		//		- encByB(preValB^2/8) - preValA*encByB(preValB/4) + ranNumA
		// 密文同态加法
		addResult := publicKey.CyphersAdd(encValue1, encValue2, encValue4, encValue5)
		// 密文与原文的同态加法
		addResult = publicKey.CypherPlainsAdd(addResult, lnHalfValueInt, rawValue3, ranNum)

		costSum[id] = addResult
	}

	encLocalCost := &common.EncLocalCost{
		EncCost:     costSum,
		RandomNoise: ranNum,
	}

	return encLocalCost, nil
}

// EvaluateEncLocalCostTag 标签方使用同态运算，根据损失函数来计算当前模型的加密损失
// TODO 增加泛化支持：
// 参与方B执行同态运算encCostForB = ln(0.5) + (y - 0.5)*encByA(preValA) + (y - 0.5)*preValB - encByA(preValA^2/8)
//						- preValB^2/8 - encByA(preValA)*preValB/4 + ranNumB
//
// - localPart 本地的明文损失数据
// - otherPart 非标签方的加密损失数据
// - trainSet 标签方训练样本集合
// - accuracy 同态加解密精度
// - publicKey 非标签方同态公钥
func EvaluateEncLocalCostTag(localPart *RawLocalGradAndCostPart, otherPart *EncLocalGradAndCostPart, trainSet [][]float64, accuracy int, publicKey *paillier.PublicKey) (*common.EncLocalCost, error) {
	costSum := make(map[int]*big.Int)

	// 生成 RanNumB，用于损失值的混淆
	randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, rand.KeyLengthInt64)
	if err != nil {
		return nil, err
	}
	ranNum := big.NewInt(0).SetBytes(randomBytes)

	// 计算ln(0.5)
	lnHalf := math.Log(0.5)
	// 2个精度的明文
	lnHalfValueInt := big.NewInt(int64(math.Round(lnHalf * math.Pow(10, 2*float64(accuracy)))))

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		// 获取该行数据的id
		// TODO: 后续优化下数据结构，来提升性能
		id := int(math.Floor(trainSet[i][0] + 0.5))

		// 计算(y - 0.5)*encByA(preValA)，2个精度的密文
		encValue1 := publicKey.CypherPlainMultiply(otherPart.EncPart1[id], localPart.RawPart1[id])

		// 计算(y - 0.5)*preValB，1个精度的原文*scale精度
		scaleFactor := big.NewInt(int64(math.Round(math.Pow(10, float64(accuracy)))))
		rawValue2 := new(big.Int).Mul(localPart.RawPart2[id], scaleFactor)

		// 计算encByA(preValA^2/8)，1个精度的密文*scale精度
		encValue3 := publicKey.CypherPlainMultiply(otherPart.EncPart2[id], scaleFactor)
		// 计算-encByA(preValA^2/8)
		encValue3 = publicKey.CypherPlainMultiply(encValue3, big.NewInt(-1))

		// 计算preValB^2/8，1个精度的密文*scale精度
		rawValue4 := new(big.Int).Mul(localPart.RawPart3[id], scaleFactor)
		// 计算 -preValB^2/8
		rawValue4 = new(big.Int).Mul(rawValue4, big.NewInt(-1))

		// 计算encByA(preValA)*preValB/4，2个精度的密文
		encValue5 := publicKey.CypherPlainMultiply(otherPart.EncPart1[id], localPart.RawPart4[id])
		// 计算-encByA(preValA)*preValB/4
		encValue5 = publicKey.CypherPlainMultiply(encValue5, big.NewInt(-1))

		// 计算 ln(0.5) + (y - 0.5)*encByA(preValA) + (y - 0.5)*preValB - encByA(preValA^2/8)
		//		- preValB^2/8 - encByA(preValA)*preValB/4 + ranNumB
		// 密文同态加法
		addResult := publicKey.CyphersAdd(encValue1, encValue3, encValue5)
		// 密文与原文的同态加法
		addResult = publicKey.CypherPlainsAdd(addResult, lnHalfValueInt, rawValue2, rawValue4, ranNum)

		costSum[id] = addResult
	}

	encLocalCost := &common.EncLocalCost{
		EncCost:     costSum,
		RandomNoise: ranNum,
	}

	return encLocalCost, nil
}

// DecryptCost 为其他方解密带噪音的损失
// - encCostMap 加密的损失信息
// - privateKey 己方同态私钥
func DecryptCost(encCostMap map[int]*big.Int, privateKey *paillier.PrivateKey) map[int]*big.Int {
	// 解密后的梯度信息
	costMap := make(map[int]*big.Int)

	for id, encCost := range encCostMap {
		rawCost := privateKey.DecryptSupNegNum(encCost)

		costMap[id] = rawCost
	}

	return costMap
}

// RetrieveRealCost 从解密后的梯度信息中，移除随机数噪音，恢复真实损失
// - decCostMap 解密的损失信息
// - accuracy 同态加解密精度
// - randomInt 损失的噪音值
func RetrieveRealCost(decCostMap map[int]*big.Int, accuracy int, randomInt *big.Int) map[int]float64 {
	// 解密后的梯度信息（含随机数噪声）
	costMap := make(map[int]float64)

	for id, decCost := range decCostMap {
		minusRandomInt := new(big.Int).Mul(big.NewInt(-1), randomInt)
		rawCost := new(big.Int).Add(decCost, minusRandomInt)

		rawCostStr := rawCost.String()
		rawCostFloat64, _ := strconv.ParseFloat(rawCostStr, 64)
		rawCostFloat64 = rawCostFloat64 / math.Pow(10, 2*float64(accuracy))

		costMap[id] = rawCostFloat64
	}

	return costMap
}

// CalCost 根据还原的损失信息计算损失值
func CalCost(costMap map[int]float64) float64 {
	var decCostSum float64 = 0

	for _, decCost := range costMap {
		decCostSum += decCost
	}

	cost := decCostSum / (-1 * float64(len(costMap)))
	return cost
}

// predict 标签方用训练得到的模型对样本做预测计算
// thetas example: const, feature1_theta, feature2_theta
// sample example: id, 1, feature1, feature2...
func predict(thetas []float64, sample []float64) (int, float64) {
	var predictValue = thetas[0]

	for i := 1; i < len(thetas); i++ {
		predictValue += thetas[i] * sample[i+1]
	}

	// TODO: 后续优化下数据结构，来提升性能
	id := int(math.Floor(sample[0] + 0.5))

	return id, predictValue
}

// predictNoTag 非标签方用训练得到的模型对样本做预测计算
// thetas example: feature1_theta, feature2_theta
// sample example: id, feature1, feature2
func predictNoTag(thetas []float64, sample []float64) (int, float64) {
	var predictValue = 0.0

	for i := 0; i < len(thetas); i++ {
		predictValue += thetas[i] * sample[i+1]
	}

	// TODO: 后续优化下数据结构，来提升性能
	id := int(math.Floor(sample[0] + 0.5))

	return id, predictValue
}

// PredictLocalPartNoTag 非标签方使用经过标准化处理的本地数据进行预测
// - thetas feature_name->feature_theta
// - standardizedInput feature_name->value
func PredictLocalPartNoTag(thetas, standardizedInput map[string]float64) float64 {
	var predictValue = 0.0

	for key, _ := range thetas {
		predictValue += thetas[key] * standardizedInput[key]
	}

	return predictValue
}

// PredictLocalPartTag 标签方使用经过标准化处理的本地数据进行预测
// - thetas feature_name->feature_theta，包含"Intercept"
// - standardizedInput feature_name->value
func PredictLocalPartTag(thetas, standardizedInput map[string]float64) float64 {
	var predictValue = thetas["Intercept"]

	for key, _ := range standardizedInput {
		predictValue += thetas[key] * standardizedInput[key]
	}

	return predictValue
}

// StandardizeLocalInput 各参与方使用之前训练集合的标准化参数对本地的要进行预测的数据进行标准化处理
// - xbars feature_name->样本均值
// - sigmas feature_name->样本标准差
// - input feature_name->样本值
func StandardizeLocalInput(xbars, sigmas, input map[string]float64) map[string]float64 {
	standardizedInput := make(map[string]float64)

	for key, _ := range input {
		standardizedInput[key] = (input[key] - xbars[key]) / sigmas[key]
	}

	return standardizedInput
}
