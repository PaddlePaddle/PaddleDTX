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
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/rand"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

// 纵向联合学习，基于半同态加密方案的多元线性回归算法
// PHE Multiple Variable Linear Regression Model based on Gradient Descent method

// LocalGradientPart 迭代的中间参数，包含未加密的和同态加密参数，用于计算梯度和损失
type LocalGradientPart struct {
	EncPart *EncLocalGradientPart `json:"enc_part"` // 加密参数
	RawPart *RawLocalGradientPart `json:"raw_part"` // 原始参数
}

// EncLocalGradientPart 迭代的中间同态加密参数，用于计算梯度和损失
type EncLocalGradientPart struct {
	EncGradPart       map[int]*big.Int `json:"enc_grad_part"`        // 对每一条数据（ID编号），计算predictValue(j-A)，并使用公钥pubKey-A进行同态加密
	EncGradPartSquare map[int]*big.Int `json:"enc_grad_part_square"` // 对每一条数据（ID编号），计算predictValue(j-A)^2，并使用公钥pubKey-A进行同态加密
	EncRegCost        *big.Int         `json:"enc_reg_cost"`         // 正则化损失，被同态加密
}

// RawLocalGradientPart 迭代中间同态加密参数，用于计算梯度和损失
type RawLocalGradientPart struct {
	RawGradPart       map[int]*big.Int `json:"raw_grad_part"`        // 对每一条数据（ID编号），计算predictValue(j-A)
	RawGradPartSquare map[int]*big.Int `json:"raw_grad_part_square"` // 对每一条数据（ID编号），计算predictValue(j-A)^2
	RawRegCost        *big.Int         `json:"raw_reg_cost"`         // 正则化损失
}

// ---------------------- 纵向联合学习的模型训练原理相关 start ----------------------
//
// 先分析无需进行泛化（如正则化）的场景 --- 批量梯度下降方法
//
// step 1: 假设两方场景
// 参与方A拥有的特征数量为nA，参与方B拥有的特征数量为nB，参与方B拥有标签。
// 参与方A和参与方B完成PSI后，确定的用于训练的总样本数量为m
//
// Cost损失函数为J(θ(0),θ(1)...,θ(n))，计算过程如下：
// for j:=0; j++; j<m
// {
//		CostSum += (predictValue(j) - realValue(j))^2
// }
//
// 其中，(predictValue(j) - realValue(j))^2
//		= (predictValue(j-A) + predictValue(j-B) - realValue(j))^2
//		= (uA(j) + uB(j) - y(j))^2
//		= uA(j)^2 + (uB(j) - y(j))^2 + 2*uA(j)*(uB(j) - y(j))
//
// 其中，predictValue(j-A) = θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(nA)*xj(nA), A样本集合中的顺序
// predictValue(j-B) = θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(nB)*xj(nB), B样本集合中的顺序
// predictValue(j) = predictValue(j-A) + predictValue(j-B), realValue(j) = yj.
//
// predictValue(j) - realValue(j) = θ(0) + w*x(j) - y(j)
// 参数集合w = [θ(1),θ(2),...,θ(n)]
// 整个模型的参数集合w' = (w, θ(0)) = [θ(0),θ(1),θ(2),...,θ(n)]
//
// 最终：J(θ(0),θ(1)...,θ(n)) = CostSum/2m
// 备注：为什么除以m，因为需要评估在整个样本集（大小为m）的平均损失。为什么除以2，是因为后面求解的时候会求偏导，会产生一个2，正好消掉
//
// ------------
//
// step 2: 设定目标并求解
// 优化目标为寻找J(θ(0),θ(1)...,θ(n))到达最小值时，w的最优解，也就是所有θ的最优解
//
// 求解原理（批量梯度下降Batch Gradient Descent）：
// 1. 将所有的θ初始化为0
// 2. 不断修改所有的θ,以使得J(θ(0),θ(1)...,θ(n))越来越小，直到无限逼近最小值点。
//		2.1 怎么定义无限逼近？连续两次逼近的差值的绝对值 |J[k] - J[k-1]| < 指定波幅amplitude
//		2.2 算法如下所示：
// 			do
// 			{
//				// 其中i为特征维度的index，也就是0,1,2,...,n
//				for i:=0; i<n; i++
//				{
//					θ(i)_old = θ(i)_new
//					θ(i)_new = θ(i)_old - α*ΔJ(θ(0),θ(1)...,θ(n))/Δθ(i)
//				}
// 			} while (|J[new] - J[old]| > amplitude)
//
// 方法：分别对不同的θ求偏导，来寻找最优解
// 令g(i) = ΔJ(θ(0),θ(1)...,θ(n))/Δθ(i)，其中i为特征维度的index，也就是0,1,2,...,n
// j为样本编号值，从0自增到总样本数量m，i为特征维度的index
// g(i) = (Δ/Δθ(i))*(1/2)*(θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(i)*xj(i) + ... + θ(n)*xj(n) - yj)^2
// 那么，g(i) = (1/2m) * SumFromZeroToM((θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(i)*xj(i) + ... + θ(n)*xj(n) - yj) * 2xj(i))
//			= (1/2m) * SumFromZeroToM((predictValue(j) - realValue(j)) * 2xj(i))
//			= (1/m) * SumFromZeroToM((predictValue(j) - realValue(j)) * xj(i))
//			= (1/m) * SumFromZeroToM((predictValue(j-A) + predictValue(j-B) - realValue(j)) * xj(i))
//			= (1/m) * (SumFromZeroToM(predictValue(j-A)*xj(i)) + SumFromZeroToM((predictValue(j-B) - realValue(j)) * xj(i))
//			= (1/m) * SumFromZeroToM(predictValue(j-A)*xj(i)) + (1/m) * SumFromZeroToM((predictValue(j-B) - realValue(j)) * xj(i))
//			= Grad(i)
//
// Grad(i)的代码定义在下面会介绍
//
// ------------
//
// step 3: 结论
// 单个参与方场景下：
// 计算出w' = (w, θ(0)) = [θ(0),θ(1),θ(2),...,θ(n)]中的每个θ(i)，来得到模型w'
// for i:=0; i++; i<n
// {
// 		// 计算第i个特征的参数θ(i):
//		θ(i) = θ(i) − α*Grad(i)
// }
//
// 第i个特征的Grad(i):
// for j:=0; j++; j<m
// {
//	Grad(i) += (predictValue(j) - realValue(j))*trainSet[j][i]
// }
//
// Grad(i) = Grad(i)/m
//
// trainSet[j][i] 表示第j个样本的的第i个特征的值
//
// 多个参与方场景下：
// w'分散在多方，所以每方需要掌握自己的模型参数w'-A，w'-B...
// 对于任何一个参与方来说，如果要获得自己模型中的θ(i)，就需要获得相应的Grad(i)
// 但是Grad(i)的计算过程，依赖于多方同时参与。
// 注意：Grad(i) = predictValue(j-A)*xj(i) + (predictValue(j-B) - realValue(j)) * xj(i)，
// 		其中，j为样本编号值，从0自增到总样本数量m，i为特征维度的index
//
// 此外，要衡量模型是否已经收敛，以便顺势结束训练过程，还需要计算损失函数。
// Cost损失函数为J(θ(0),θ(1)...,θ(n))，计算过程如下：
//		Cost(j) = (predictValue(j) - realValue(j))^2
//			 = predictValue(j-A)^2 + (predictValue(j-B) - realValue(j))^2
//				 + 2*predictValue(j-A)*(predictValue(j-B) - realValue(j))
//
// 在2方的场景下，如上例，梯度和损失函数的计算需求如下：
//		3.1 当参与方A需要计算自己的模型中的θ(i)和损失函数时，自己掌握predictValue(j-A), xj(i),
//			还需要参与方B贡献predictValue(j-B) - realValue(j)
//		3.2 当参与方B需要计算自己的模型中的θ(i)和损失函数时，自己掌握predictValue(j-B) - realValue(j), xj(i),
//			还需要参与方A贡献predictValue(j-A)
//
// ------------
//
// 由此，引入半同态加密技术PHE，来解决梯度计算过程和损失函数计算过程中的参数交换问题
// 假设已经完成训练样本的标准化和预处理
//
// step 1:
//		1.1 参与方A生成公钥pubKey-A，将公钥pubKey-A传输给参与方B。参与方B生成公钥pubKey-B，将公钥pubKey-B传输给参与方A
// 		1.2 针对每一轮的梯度下降，重复执行以下步骤，直至完成收敛
// step 2:
//		2.1 参与方A计算predictValue(j-A)，predictValue(j-A)^2，并对它们分别使用公钥pubKey-A进行同态加密
//		2.2 参与方A将同态加密后的结果encByA(predictValue(j-A))和encByA（predictValue(j-A)^2）发给参与方B
// step 3:
//		3.1 参与方B计算predictValue(j-B) - realValue(j)，(predictValue(j-B) - realValue(j))^2，并使用公钥pubKey-B进行同态加密
//		3.2 参与方B将同态加密后的结果encByB(predictValue(j-B) - realValue(j))和encByB((predictValue(j-B) - realValue(j))^2)发给参与方A
// step 4:
//		4.1 参与方A执行同态运算encGraForA = encByB(predictValue(j-A)*xAj(i))
//									 + encByB(predictValue(j-B) - realValue(j)) * xAj(i)
//									 + encByB(RanNumA)
//			其中，encByB(predictValue(j-A)*xAj(i))是A用B的公钥加密得到的，A自己产生随机数RanNumA再用B的公钥加密，得到encByB(RanNumA)
//		4.2 参与方A将同态运算后的结果encGraForA发给参与方B
// step 5:
//		5.1 参与方B执行同态运算encGraForB = encByA(predictValue(j-A)*xBj(i))
//									 + encByA(predictValue(j-B) - realValue(j)) * xBj(i)
//									 + encByA(RanNumB)
//			其中，encByA(predictValue(j-B) - realValue(j))是B用A的公钥加密得到的，B自己产生随机数RanNumB再用A的公钥加密，得到encByA(RanNumB)
//		5.2 参与方B将同态运算后的结果encGraForB发给参与方A
// step 6:
//		6.1 参与方A使用私钥解密encGraForB，得到被添加了随机数RanNumB的梯度graForB，然后将其发送给参与方B
//		6.2 参与方B使用私钥解密encGraForA，得到被添加了随机数RanNumA的梯度graForA，然后将其发送给参与方A
// step 7:
//		7.1 参与方A从graForA中移除随机数RanNumA，得到最终用来更新xAj的梯度graAj
//		7.2 参与方B从graForB中移除随机数RanNumB，得到最终用来更新xBj的梯度graBj
//
// ---------------------- 纵向联合学习的模型训练原理相关 end ----------------------

// StandardizeDataSet 将样本数据集合进行标准化处理，将均值变为0，标准差变为1
// Z-score Normalization
// x' = (x - avgerage(x)) / standard_deviation(x)
func StandardizeDataSet(sourceDataSet *common.DataSet) *common.StandardizedDataSet {
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

		xbars := xbarParams[feature.FeatureName]
		sigmas := sigmaParams[feature.FeatureName]

		// 声明该特征维度的经过标准化处理后的特征值集合
		newSets := make(map[int]float64)
		// 对特征值进行标准化处理，对数据集的每一条数据的每个特征的减去该特征均值后除以特征标准差
		for key, value := range feature.Sets {
			newValue := (value - xbars) / sigmas
			newSets[key] = newValue
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

// DeStandardizeBothThetas 利用两方训练的模型结果逆标准化模型参数
// trainDataSetA 非标签方训练数据集
// trainDataSetB 标签方训练数据集
// originalThetasA 非标签方模型
// originalThetasB 标签方模型
func DeStandardizeBothThetas(trainDataSetA, trainDataSetB *common.TrainDataSet, originalThetasA, originalThetasB []float64) []float64 {
	thetas := make([]float64, len(originalThetasA)+len(originalThetasB))
	// 拷贝常数项
	copy(thetas, originalThetasB[0:1])
	// 拷贝A的特征系数
	copy(thetas[1:], originalThetasA)
	// 拷贝B的非常数特征系数和标签
	copy(thetas[len(originalThetasA)+1:], originalThetasB[1:])

	xbars := make([]float64, len(thetas))
	sigmas := make([]float64, len(thetas))

	for i := 0; i < len(originalThetasA); i++ {
		xbars[i] = trainDataSetA.XbarParams[trainDataSetA.FeatureNames[i]]
		sigmas[i] = trainDataSetA.SigmaParams[trainDataSetA.FeatureNames[i]]
	}

	for i := 0; i < len(originalThetasB); i++ {
		xbars[i+len(originalThetasA)] = trainDataSetB.XbarParams[trainDataSetB.FeatureNames[i]]
		sigmas[i+len(originalThetasA)] = trainDataSetB.SigmaParams[trainDataSetB.FeatureNames[i]]
	}

	// 可以同时处理有标签和无标签的情况
	for i := 1; i < len(thetas); i++ {
		thetas[0] -= thetas[i] * (xbars[i-1] / sigmas[i-1])
		thetas[i] = (thetas[i] * sigmas[len(sigmas)-1]) / sigmas[i-1]
	}

	thetas[0] *= sigmas[len(sigmas)-1]
	thetas[0] += xbars[len(xbars)-1]

	return thetas
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

// PreProcessDataSet 预处理标签方的标准化数据集
// 将样本数据集转化为一个m*(n+2)的矩阵, 其中每行对应一个样本，每行的第一列为编号ID，第二列为1（截距intercept），其它列分别对应一种特征的值
func PreProcessDataSet(sourceDataSet *common.StandardizedDataSet, targetFeatureName string) *common.TrainDataSet {
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
		if feature.FeatureName == targetFeatureName {
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
		if feature.FeatureName == targetFeatureName {
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

	if len(targetFeatureName) != 0 {
		featureNames = append(featureNames, targetFeatureName)
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

// CalLocalGradientTagPart 标签方为计算本地模型中，每个特征的参数做准备，计算本地同态加密结果
// 对每一条数据（ID编号j），计算predictValue(j-B) - realValue(j)，(predictValue(j-B) - realValue(j))^2，并对它们分别使用公钥pubKey-A进行同态加密
// 注意：高性能的同态运算仅能处理整数，对于浮点数有精度损失，所以必须在参数中指定精度来进行处理
//
// - thetas 上一轮训练得到的模型参数
// - trainSet 预处理过的训练数据
// - accuracy 同态加解密精确到小数点后的位数
// - regMode 正则模式
// - regParam 正则参数
// - publicKey 标签方同态公钥
func CalLocalGradientTagPart(thetas []float64, trainSet [][]float64, accuracy int, regMode int, regParam float64, publicKey *paillier.PublicKey) (*LocalGradientPart, error) {
	// 对每一条数据（ID编号），计算predictValue(j-B) - realValue(j)
	rawGradPart := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算(predictValue(j-B) - realValue(j))^2
	rawGradPartSquare := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算predictValue(j-B) - realValue(j)，并使用公钥pubKey-B进行同态加密
	encGradPart := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算(predictValue(j-B) - realValue(j))^2，并使用公钥pubKey-B进行同态加密
	encGradPartSquare := make(map[int]*big.Int)

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		id, predictValue := predict(thetas, trainSet[i])

		// 用预测值减去实际值以获取误差，每行的最后一列是实际值
		deviation := predictValue - trainSet[i][len(trainSet[i])-1]

		// 精度处理转int后，才可以使用同态加密和同态运算
		deviationInt := big.NewInt(int64(math.Round(deviation * math.Pow(10, float64(accuracy)))))
		// 使用同态公钥加密数据
		encDeviation, err := publicKey.EncryptSupNegNum(deviationInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 精度处理转int后，才可以使用同态加密和同态运算
		deviationSquareInt := new(big.Int).Mul(deviationInt, deviationInt)
		// 使用同态公钥加密数据
		encDeviationSquare, err := publicKey.EncryptSupNegNum(deviationSquareInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		rawGradPart[id] = deviationInt
		rawGradPartSquare[id] = deviationSquareInt

		encGradPart[id] = encDeviation
		encGradPartSquare[id] = encDeviationSquare
	}

	// 根据正则模型计算损失
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
	rawPart := &RawLocalGradientPart{
		RawGradPart:       rawGradPart,
		RawGradPartSquare: rawGradPartSquare,
		RawRegCost:        rawRegCost,
	}

	// 生成标签方的中间同态加密参数，用于计算梯度和损失
	encPart := &EncLocalGradientPart{
		EncGradPart:       encGradPart,
		EncGradPartSquare: encGradPartSquare,
		EncRegCost:        encRegCost,
	}

	// 生成标签方的中间同态加密参数，用于计算梯度和损失
	localGradientPart := &LocalGradientPart{
		RawPart: rawPart,
		EncPart: encPart,
	}

	return localGradientPart, nil
}

// CalLocalGradientPart 非标签方为计算本地模型中，每个特征的参数做准备，计算本地同态加密结果
// 对每一条数据（ID编号j），计算predictValue(j-A)，predictValue(j-A)^2，并对它们分别使用公钥pubKey-A进行同态加密
//
// - thetas 上一轮训练得到的模型参数
// - trainSet 预处理过的训练数据
// - accuracy 同态加解密精确到小数点后的位数
// - regMode 正则模式
// - regParam 正则参数
// - publicKey 非标签方同态公钥
func CalLocalGradientPart(thetas []float64, trainSet [][]float64, accuracy int, regMode int, regParam float64, publicKey *paillier.PublicKey) (*LocalGradientPart, error) {
	// 对每一条数据（ID编号），计算predictValue(j-A)
	rawGradPart := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算predictValue(j-A)^2
	rawGradPartSquare := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算predictValue(j-A)，并使用公钥pubKey-A进行同态加密
	encGradPart := make(map[int]*big.Int)

	// 对每一条数据（ID编号），计算predictValue(j-A)^2，并使用公钥pubKey-A进行同态加密
	encGradPartSquare := make(map[int]*big.Int)

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		id, predictValue := predictNoTag(thetas, trainSet[i])

		// 精度处理转int后，才可以使用同态加密和同态运算
		predictValueInt := big.NewInt(int64(math.Round(predictValue * math.Pow(10, float64(accuracy)))))
		// 使用同态公钥加密数据
		encPredictValue, err := publicKey.EncryptSupNegNum(predictValueInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 精度处理转int后，才可以使用同态加密和同态运算
		predictValueSquareInt := new(big.Int).Mul(predictValueInt, predictValueInt)
		// 使用同态公钥加密数据
		encPredictValueSquare, err := publicKey.EncryptSupNegNum(predictValueSquareInt)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		rawGradPart[id] = predictValueInt
		rawGradPartSquare[id] = predictValueSquareInt

		encGradPart[id] = encPredictValue
		encGradPartSquare[id] = encPredictValueSquare
	}

	// 根据正则模型计算损失
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
	rawPart := &RawLocalGradientPart{
		RawGradPart:       rawGradPart,
		RawGradPartSquare: rawGradPartSquare,
		RawRegCost:        rawRegCost,
	}

	// 生成非标签方的中间同态加密参数，用于计算梯度和损失
	encPart := &EncLocalGradientPart{
		EncGradPart:       encGradPart,
		EncGradPartSquare: encGradPartSquare,
		EncRegCost:        encRegCost,
	}

	// 生成费标签方的中间同态加密参数，用于计算梯度和损失
	localGradientPart := &LocalGradientPart{
		RawPart: rawPart,
		EncPart: encPart,
	}

	return localGradientPart, nil
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
// 参与方A执行同态运算encGraForA = encByB(predictValue(j-A)*xAj(i))
//							+ encByB(predictValue(j-B) - realValue(j)) * xAj(i)
//							+ encByB(RanNumA)
// 其中，encByB(predictValue(j-A)*xAj(i))是A用B的公钥加密得到的，A自己产生随机数RanNumA再用B的公钥加密，得到encByB(RanNumA)
//
// - localPart 非标签方本地的明文梯度数据
// - tagPart 标签方的加密梯度数据
// - trainSet 非标签方训练样本集合
// - featureIndex 指定特征的索引
// - accuracy 同态加解密精度
// - publicKey 标签方同态公钥
func CalEncLocalGradient(localPart *RawLocalGradientPart, tagPart *EncLocalGradientPart, trainSet [][]float64, featureIndex, accuracy int, publicKey *paillier.PublicKey) (*common.EncLocalGradient, error) {
	// 计算encByB(predictValue(j-A)*xAj(i))
	// 对每一条数据（ID编号），计算predictValue(j-A)
	encGradMap := make(map[int]*big.Int)

	// 生成 RanNumA，用于梯度值的混淆
	randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, rand.KeyLengthInt64)
	if err != nil {
		return nil, err
	}
	ranNum := big.NewInt(0).SetBytes(randomBytes)

	// 计算 encByB(RanNumA)
	encRanNum, err := publicKey.EncryptSupNegNum(ranNum)
	if err != nil {
		log.Printf("Paillier Encrypt err is %v", err)
		return nil, err
	}

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		// 获取该行数据的id
		// TODO: 后续优化下数据结构，来提升性能
		id := int(math.Floor(trainSet[i][0] + 0.5))

		// 获取predictValue(j-A)
		predictValueLocalPart, ok := localPart.RawGradPart[id]
		if !ok {
			return nil, fmt.Errorf("CalEncLocalGradient failed to get raw grad part for id: %d, rawGradPart: %v", id, localPart.RawGradPart)
		}

		// trainset第一列是id，第二列是1
		// 计算predictValue(j-A)*xAj(i)
		scaleFactor := big.NewInt(int64(math.Round(trainSet[i][featureIndex+1] * math.Pow(10, float64(accuracy)))))
		// 两个乘法子项都拥有精度，相当于倍数*2
		deviation1 := new(big.Int).Mul(predictValueLocalPart, scaleFactor)

		// 使用对方的公钥加密 encByB(predictValue(j-A)*xAj(i))
		encDeviation1, err := publicKey.EncryptSupNegNum(deviation1)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 获取 encByB(predictValue(j-B) - realValue(j))
		predictValueTagPart, ok := tagPart.EncGradPart[id]
		if !ok {
			return nil, fmt.Errorf("CalEncLocalGradient failed to get enc grad part for id: %d, encGradPart: %v", id, tagPart.EncGradPart)
		}

		// 计算 encByB(predictValue(j-B) - realValue(j)) * xAj(i)
		scaleFactor = big.NewInt(int64(math.Round(trainSet[i][featureIndex+1] * math.Pow(10, float64(accuracy)))))

		// 两个乘法子项都拥有精度，相当于倍数*2
		encDeviation2 := publicKey.CypherPlainMultiply(predictValueTagPart, scaleFactor)

		// 计算 encByB(predictValue(j-A)*xAj(i)) + encByB(predictValue(j-B) - realValue(j)) * xAj(i) + encByB(RanNumA)
		addResult := publicKey.CyphersAdd(encDeviation1, encDeviation2, encRanNum)

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
// 参与方B执行同态运算encGraForB = encByA(predictValue(j-A))*xBj(i)
//									 + encByA((predictValue(j-B) - realValue(j)) * xBj(i))
//									 + encByA(RanNumB)
// 其中，encByA(predictValue(j-B) - realValue(j))是B用A的公钥加密得到的，B自己产生随机数RanNumB再用A的公钥加密，得到encByA(RanNumB)
//
// - localPart 标签方本地的明文梯度数据
// - otherPart 非标签方的加密梯度数据
// - trainSet 标签方训练样本集合
// - featureIndex 指定特征的索引
// - accuracy 同态加解密精度
// - publicKey 非标签方同态公钥
func CalEncLocalGradientTagPart(localPart *RawLocalGradientPart, otherPart *EncLocalGradientPart, trainSet [][]float64, featureIndex, accuracy int, publicKey *paillier.PublicKey) (*common.EncLocalGradient, error) {
	// 计算encByA(predictValue(j-B) - realValue(j) * xBj(i))
	// 对每一条数据（ID编号），计算predictValue(j-B) - realValue(j) * xBj(i)
	encGradMap := make(map[int]*big.Int)

	// 生成 RanNumB，用于梯度值的混淆
	randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, rand.KeyLengthInt64)
	if err != nil {
		return nil, err
	}
	ranNum := big.NewInt(0).SetBytes(randomBytes)

	// 计算 encByA(RanNumB)
	encRanNum, err := publicKey.EncryptSupNegNum(ranNum)
	if err != nil {
		log.Printf("Paillier Encrypt err is %v", err)
		return nil, err
	}

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		// 获取该行数据的id
		// TODO: 后续优化下数据结构，来提升性能
		id := int(math.Floor(trainSet[i][0] + 0.5))

		// 获取predictValue(j-B) - realValue(j)
		predictValueLocalPart, ok := localPart.RawGradPart[id]
		if !ok {
			return nil, fmt.Errorf("CalEncLocalGradientTagPart failed to get raw grad part for id: %d, rawGradPart: %v", id, localPart.RawGradPart)
		}

		// trainset第一列是id，第二列是1
		// 计算(predictValue(j-B) - realValue(j))*xBj(i)
		scaleFactor := big.NewInt(int64(math.Round(trainSet[i][featureIndex+1] * math.Pow(10, float64(accuracy)))))
		deviation1 := new(big.Int).Mul(predictValueLocalPart, scaleFactor)

		// 使用对方的公钥加密 encByA((predictValue(j-B) - realValue(j))*xBj(i))
		encDeviation1, err := publicKey.EncryptSupNegNum(deviation1)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 获取 encByA(predictValue(j-A))
		predictValueOtherPart, ok := otherPart.EncGradPart[id]
		if !ok {
			return nil, fmt.Errorf("CalEncLocalGradientTagPart failed to get enc grad part for id: %d, encGradPart: %v", id, otherPart.EncGradPart)
		}

		// 计算 encByA(predictValue(j-A))*xBj(i)
		// 两个乘法子项都拥有精度，相当于倍数*2
		scaleFactor = big.NewInt(int64(math.Round(trainSet[i][featureIndex+1] * math.Pow(10, float64(accuracy)))))
		encDeviation2 := publicKey.CypherPlainMultiply(predictValueOtherPart, scaleFactor)

		// 计算 encByA(predictValue(j-A))*xBj(i) + encByA((predictValue(j-B) - realValue(j))*xBj(i)) + encByA(RanNumB)
		addResult := publicKey.CyphersAdd(encDeviation1, encDeviation2, encRanNum)

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
// 增加泛化支持：
// 参与方A执行同态运算encCostForA = encByB(predictValue(j-A)^2)
//							+ encByB((predictValue(j-B) - realValue(j))^2)
//							+ 2*predictValue(j-A)*encByB(predictValue(j-B) - realValue(j))
//							+ encByB(RanNumA)
//							+ encByB(L_A)
//							+ encByB(L_B)
//
// - localPart 本地的明文损失数据
// - tagPart 标签方的加密损失数据
// - trainSet 非标签方训练样本集合
// - publicKey 标签方同态公钥
func EvaluateEncLocalCost(localPart *RawLocalGradientPart, tagPart *EncLocalGradientPart, trainSet [][]float64, publicKey *paillier.PublicKey) (*common.EncLocalCost, error) {
	costSum := make(map[int]*big.Int)

	// 生成 RanNumA，用于损失值的混淆
	randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, rand.KeyLengthInt64)
	if err != nil {
		return nil, err
	}
	ranNum := big.NewInt(0).SetBytes(randomBytes)

	// 计算 encByB(RanNumA)
	encRanNum, err := publicKey.EncryptSupNegNum(ranNum)
	if err != nil {
		log.Printf("Paillier Encrypt err is %v", err)
		return nil, err
	}

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		id := int(math.Floor(trainSet[i][0] + 0.5))

		// 获得predictValue(j-A)^2
		rawDeviation1, ok := localPart.RawGradPartSquare[id]
		if !ok {
			return nil, fmt.Errorf("EvaluateEncLocalCost failed to get raw grad part square for id: %d, rawGradPartSq: %v", id, localPart.RawGradPartSquare)
		}

		// 计算encByB(predictValue(j-A)^2)
		encDeviation1, err := publicKey.EncryptSupNegNum(rawDeviation1)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 获得encByB((predictValue(j-B) - realValue(j))^2)
		encDeviation2, ok := tagPart.EncGradPartSquare[id]
		if !ok {
			return nil, fmt.Errorf("EvaluateEncLocalCost failed to get enc grad part square for id: %d, encGradPartSq: %v", id, tagPart.EncGradPartSquare)
		}

		// 计算2*predictValue(j-A)
		localRawGradPart, ok := localPart.RawGradPart[id]
		if !ok {
			return nil, fmt.Errorf("EvaluateEncLocalCost failed to get local raw grad part for id: %d, rawGradPart: %v", id, localPart.RawGradPart)
		}
		scaleFactor := new(big.Int).Mul(big.NewInt(2), localRawGradPart)

		// 计算2*predictValue(j-A)*encByB(predictValue(j-B) - realValue(j))
		encPart, ok := tagPart.EncGradPart[id]
		if !ok {
			return nil, fmt.Errorf("EvaluateEncLocalCost failed to get enc grad part for id: %d, encGradPart: %v", id, tagPart.EncGradPart)
		}
		encDeviation3 := publicKey.CypherPlainMultiply(encPart, scaleFactor)

		// 支持泛化
		// 获得L_A
		rawDeviation4 := localPart.RawRegCost

		// 计算encByB(L_A)
		encDeviation4, err := publicKey.EncryptSupNegNum(rawDeviation4)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 获得encByB(L_B)
		encDeviation5 := tagPart.EncRegCost

		// 将误差值累加，再加上随机数
		// 密文加法
		addResult := publicKey.CyphersAdd(encDeviation1, encDeviation2, encDeviation3, encDeviation4, encDeviation5, encRanNum)

		costSum[id] = addResult
	}

	encLocalCost := &common.EncLocalCost{
		EncCost:     costSum,
		RandomNoise: ranNum,
	}

	return encLocalCost, nil
}

// EvaluateEncLocalCostTag 标签方使用同态运算，根据损失函数来计算当前模型的加密损失
// 增加泛化支持：
// 参与方B执行同态运算encCostForB = encByA(predictValue(j-A)^2)
//							+ encByA((predictValue(j-B) - realValue(j))^2)
//							+ 2*encByA(predictValue(j-A))*(predictValue(j-B) - realValue(j))
//							+ encByA(RanNumB)
//							+ encByA(L_A)
//							+ encByA(L_B)
//
// - localPart 本地的明文损失数据
// - otherPart 非标签方的加密损失数据
// - trainSet 标签方训练样本集合
// - publicKey 非标签方同态公钥
func EvaluateEncLocalCostTag(localPart *RawLocalGradientPart, otherPart *EncLocalGradientPart, trainSet [][]float64, publicKey *paillier.PublicKey) (*common.EncLocalCost, error) {
	costSum := make(map[int]*big.Int)

	// 生成 RanNumB，用于损失值的混淆
	randomBytes, err := rand.GenerateSeedWithStrengthAndKeyLen(rand.KeyStrengthHard, rand.KeyLengthInt64)
	if err != nil {
		return nil, err
	}
	ranNum := big.NewInt(0).SetBytes(randomBytes)

	// 计算 encByA(RanNumB)
	encRanNum, err := publicKey.EncryptSupNegNum(ranNum)
	if err != nil {
		log.Printf("Paillier Encrypt err is %v", err)
		return nil, err
	}

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		id := int(math.Floor(trainSet[i][0] + 0.5))

		// 获得(predictValue(j-B) - realValue(j))^2
		rawDeviation1, ok := localPart.RawGradPartSquare[id]
		if !ok {
			return nil, fmt.Errorf("EvaluateEncLocalCostTag failed to get local raw grad part square for id: %d, rawGradPartSq: %v", id, localPart.RawGradPartSquare)
		}

		// 计算encByA((predictValue(j-B) - realValue(j))^2)
		encDeviation1, err := publicKey.EncryptSupNegNum(rawDeviation1)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 获得encByA(predictValue(j-A)^2)
		encDeviation2, ok := otherPart.EncGradPartSquare[id]
		if !ok {
			return nil, fmt.Errorf("EvaluateEncLocalCostTag failed to get enc grad part square for id: %d, encgradPartSq: %v", id, otherPart.EncGradPartSquare)
		}

		// 计算2*(predictValue(j-B) - realValue(j))
		rawGradPart, ok := localPart.RawGradPart[id]
		if !ok {
			return nil, fmt.Errorf("EvaluateEncLocalCostTag failed to get raw grad part for id: %d, rawGradPart: %v", id, localPart.RawGradPart)
		}
		scaleFactor := new(big.Int).Mul(big.NewInt(2), rawGradPart)

		// 计算2*encByA(predictValue(j-A))*(predictValue(j-B) - realValue(j))
		otherEncGradPart, ok := otherPart.EncGradPart[id]
		if !ok {
			return nil, fmt.Errorf("EvaluateEncLocalCostTag failed to get other enc grad part for id: %d, encGradPart: %v", id, otherPart.EncGradPart)
		}
		encDeviation3 := publicKey.CypherPlainMultiply(otherEncGradPart, scaleFactor)

		// 支持泛化
		// 获得L_B
		rawDeviation4 := localPart.RawRegCost

		// 计算encByA(L_B)
		encDeviation4, err := publicKey.EncryptSupNegNum(rawDeviation4)
		if err != nil {
			log.Printf("Paillier Encrypt err is %v", err)
			return nil, err
		}

		// 获得encByA(L_A)
		encDeviation5 := otherPart.EncRegCost

		// 将误差值累加，再加上随机数
		// 密文加法
		addResult := publicKey.CyphersAdd(encDeviation1, encDeviation2, encDeviation3, encDeviation4, encDeviation5, encRanNum)

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

	cost := decCostSum / (2 * float64(len(costMap)))
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

// DeStandardizeOutput 逆Z-Score标准化：标签方使用之前训练集合的标签的标准差和均值，对通过标准化模型预测的结果进行逆标准化处理
// - ybar 目标特征对应的样本均值
// - sigma 目标特征对应的样本标准差
// - output 标准化样本的预测结值
func DeStandardizeOutput(ybar, sigma, output float64) float64 {
	deStandardizedOutput := output*sigma + ybar

	return deStandardizedOutput
}
