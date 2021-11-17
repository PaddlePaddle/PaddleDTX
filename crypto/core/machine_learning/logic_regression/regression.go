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

package logic_regression

import (
	"log"
	"math"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

// 基于梯度下降的多元逻辑回归模型算法
// Multiple Variable Logic Regression Model based on Gradient Descent method
//
// 逻辑回归：使用逻辑回归作为逻辑函数（Sigmoid函数）的输入，最终得到的二分类模型
//
// 模型训练的步骤：
// Step 1：标准化处理样本集合。使用Z-Score，将均值变为0，标准差变为1。目的是为了提升梯度下降方法的收敛速度。
//		使得使用同样的学习率，可以更快的找到最小参数以达到拟合收敛要求；或者在训练中可以使用更高的学习率α。
// Step 2：预处理标准化后的样本集合。每个样本新增一个特征维度intercept，值为1，且列为第一个特征。
//		将拟合的目标特征维度target feature移动为样本的最后一个特征。
//		最后获得一个m*n的二维矩阵。m为样本数量，n为特征维度。
// Step 3：模型训练，设定学习率和目标收敛震荡幅度。每轮迭代会以指定学习率进行逼近，直到收敛震荡幅度小于目标收敛震荡幅度
// Step 4：对模型进行逆标准化，得到最终模型，并计算出该模型的拟合度r-squared。
// 注意，也可以不对模型进行标准化处理，但是如果不这样做，就需要存储标准模型的每个特征维度和标签的均值和标准差。
// 在预测时，先对输入数据的每个特征进行标准化，通过标准模型预测出结果后，再对结果进行逆标准化。
// 这种方案，需要持久化保持样本集合中每个特征的维度和标签的均值和标准差。模型不能被单独使用，因而更容易发生样本集合的信息泄露。

// StandardizeDataSet 将样本数据集合进行Z-score标准化处理，将均值变为0，标准差变为1
// Z-score Normalization
// x' = (x - average(x)) / standard_deviation(x)
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

// PreProcessDataSet 预处理标准化训练数据集
// label表示计算损失函数时的目标维度
func PreProcessDataSet(sourceDataSet *common.StandardizedDataSet, label string) *common.TrainDataSet {
	var featureNames []string

	// 特征数量 - 有多少个特征维度
	featureNum := len(sourceDataSet.Features)

	// 样本数量 - 取某一个特征的的样本数量
	sampleNum := len(sourceDataSet.Features[0].Sets)

	// 将样本数据集转化为一个m*(n+1)的矩阵, 其中每行对应一个样本，每行的第一列为1（截距intercept），其它列分别对应一种特征的值
	trainSet := make([][]float64, sampleNum)
	originalTrainSet := make([][]float64, sampleNum)
	for i := 0; i < sampleNum; i++ {
		trainSet[i] = make([]float64, featureNum+1)
		originalTrainSet[i] = make([]float64, featureNum+1)
	}

	// 遍历样本
	for i := 0; i < len(sourceDataSet.Features[0].Sets); i++ {
		// 每个样本的第一列为1
		trainSet[i][0] = 1
		originalTrainSet[i][0] = 1
	}

	// 遍历所有的特征维度，i表示遍历到第i个特征维度
	i := 1
	for _, feature := range sourceDataSet.Features {
		// 遍历每个特征维度的每个样本
		// 如果是目标维度
		if feature.FeatureName == label {
			// 遍历某个特征维度的所有样本
			for key, value := range feature.Sets {
				// 把值放在训练样本的最后一列
				trainSet[key][featureNum] = value
			}
		} else { // 如果不是目标维度
			for key, value := range feature.Sets {
				trainSet[key][i] = value
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
				originalTrainSet[key][featureNum] = value
			}
		} else { // 如果不是目标维度
			for key, value := range feature.Sets {
				originalTrainSet[key][i] = value
			}
			i++
		}
	}

	featureNames = append(featureNames, label)

	dataSet := &common.TrainDataSet{
		FeatureNames:     featureNames,              // 特征名称的集合
		TrainSet:         trainSet,                  // 特征集合
		XbarParams:       sourceDataSet.XbarParams,  // 特征的均值
		SigmaParams:      sourceDataSet.SigmaParams, // 特征的标准差
		OriginalTrainSet: originalTrainSet,          // 原始特征集合
	}

	return dataSet
}

// EvaluateModelSuperParamByCV 模型评估，通过RMSE（均方根误差）来评估
// 注意：每次在迭代之前进行预处理，而不是统一进行预处理
//
// - sourceDataSet 原始样本数据
// - label 目标特征名称
// - alpha 训练学习率
// - amplitude 训练目标值
// - regMode 正则模式
// - regParam 正则参数
// - cvMode 交叉验证模式
// - cvParam 交叉验证参数
func EvaluateModelSuperParamByCV(sourceDataSet *common.DataSet, label string, alpha, amplitude float64, regMode int, regParam float64, cvMode int, cvParam int) float64 {
	// TODO: 支持更多的交叉验证方案
	switch cvMode {
	case common.CvLoo:
		log.Printf("cvMode[%v] is supported.", cvMode)
	default:
		log.Printf("cvMode[%v] is not supported yet, using CvLoo instead.", cvMode)
	}

	totalCost := 0.0
	totalNum := 0
	for i := 0; i < len(sourceDataSet.Features[0].Sets); i++ {
		var features []*common.DataFeature

		for _, feature := range sourceDataSet.Features {

			newFeatureSets := make(map[int]float64)

			// 遍历某个特征维度的所有样本
			for key, value := range feature.Sets {
				// 剔除掉指定样本行 key == i
				if key < i {
					newFeatureSets[key] = value
				} else if key > i {
					newFeatureSets[key-1] = value
				}
			}

			newFeature := &common.DataFeature{
				FeatureName: feature.FeatureName,
				Sets:        newFeatureSets,
			}
			features = append(features, newFeature)
		}

		newDataSet := &common.DataSet{
			Features: features,
		}

		// 1. 标准化样本
		standardizedDataSet := StandardizeDataSet(newDataSet, label)
		// 2. 预处理标准样本
		trainDataSet := PreProcessDataSet(standardizedDataSet, label)
		// 3. 训练计算
		thetas, _ := train(trainDataSet.TrainSet, alpha, amplitude, regMode, regParam)
		// 4. 计算模型误差
		cost := evaluateRSS(thetas, trainDataSet.OriginalTrainSet)
		totalCost += cost

		totalNum = len(trainDataSet.OriginalTrainSet)
	}
	// 5. 计算均方误差
	avgCost := totalCost / float64(totalNum)
	// 6. 计算均方根误差
	rmse := math.Sqrt(avgCost)

	return rmse
}

// TrainModel 模型训练，alpha是梯度下降法的学习率α, amplitude是目标收敛震荡幅度
// 使用正则化的方案来提升泛化能力（先验知识）
//
// - trainDataSet 预处理过的训练数据
// - alpha 训练学习率
// - amplitude 训练目标值
// - regMode 正则模式
// - regParam 正则参数
func TrainModel(trainDataSet *common.TrainDataSet, alpha float64, amplitude float64, regMode int, regParam float64) *common.Model {
	thetas, _ := train(trainDataSet.TrainSet, alpha, amplitude, regMode, regParam)

	originParams := make(map[string]float64)
	originParams["Intercept"] = thetas[0]

	for i := 0; i < len(trainDataSet.FeatureNames)-1; i++ {
		originParams[trainDataSet.FeatureNames[i]] = thetas[i+1]
	}

	params := make(map[string]float64)
	params["Intercept"] = thetas[0]

	for i := 0; i < len(trainDataSet.FeatureNames)-1; i++ {
		params[trainDataSet.FeatureNames[i]] = thetas[i+1]
	}

	model := &common.Model{
		Params: params, // 所有特征对应的系数
	}

	return model
}

// evaluateRSS 计算RSS(Residual Sum of Squares)残差平方和，用于衡量模型的误差
func evaluateRSS(thetas []float64, trainSet [][]float64) float64 {
	// （实际值-预测值）平方之和
	var rss float64 = 0
	for i := 0; i < len(trainSet); i++ {
		realValue := trainSet[i][len(thetas)]
		predictValueH := predict(thetas, trainSet[i])
		predictValue := 1 / (1 + math.Exp(-1*predictValueH))

		rss += math.Pow(realValue-predictValue, 2)
	}

	return rss
}

// train 模型训练
// 1. 支持配合正则化（L2正则或者L1正则）使用，在样本数量偏少，而特征数量偏多的时候，避免过拟合，提升泛化能力。
// 2. 支持配合交叉验证使用，避免过拟合，提升泛化能力。
//
// - alpha 梯度下降法的学习率α
// - amplitude 拟合度delta的目标值
// - regMode 正则模型
// - regParam 正则参数
func train(trainSet [][]float64, alpha float64, amplitude float64, regMode int, regParam float64) ([]float64, float64) {
	// 每个特征维度都有一个系数theta，此外还有一个intercept
	// 排除掉末尾的目标特征维度，首列的常数1相当于intercept
	thetas := make([]float64, len(trainSet[0])-1)

	// 存储临时数据
	temps := make([]float64, len(trainSet[0])-1)

	lastCost := 0.0
	currentCost := 0.0

	// 计算初始模型的损失
	switch regMode {
	case common.RegLasso:
		lastCost = evaluateCostWithLassoReg(thetas, trainSet, regParam)
	case common.RegRidge:
		lastCost = evaluateCostWithRidgeReg(thetas, trainSet, regParam)
	default:
		lastCost = evaluateCost(thetas, trainSet)
	}

	for {
		// 根据正则类型，为每个特征维度计算其系数theta，特征维度的数量=len(trainSet[0])-1
		for i := 0; i < len(thetas); i++ {
			switch regMode {
			case common.RegLasso:
				temps[i] = thetas[i] - alpha*calGradientWithLassoReg(thetas, trainSet, i, regParam)
			case common.RegRidge:
				temps[i] = thetas[i] - alpha*calGradientWithRidgeReg(thetas, trainSet, i, regParam)
			default:
				temps[i] = thetas[i] - alpha*calGradient(thetas, trainSet, i)
			}
		}

		// 更新每个特征维度的系数theta
		for i := 0; i < len(thetas); i++ {
			thetas[i] = temps[i]
		}

		// 根据差值评估整荡幅度是否符合目标要求，是否可以结束梯度下降的收敛计算
		// 按照真正的损失函数J(θ(0),θ(1)...,θ(n))来做计算
		switch regMode {
		case common.RegLasso:
			currentCost = evaluateCostWithLassoReg(thetas, trainSet, regParam)
		case common.RegRidge:
			currentCost = evaluateCostWithRidgeReg(thetas, trainSet, regParam)
		default:
			currentCost = evaluateCost(thetas, trainSet)
		}

		// 参数变化率，用来衡量收敛度
		delta := math.Abs(currentCost - lastCost)
		if delta < amplitude {
			break
		}
		lastCost = currentCost
	}

	return thetas, currentCost
}

// evaluateCost 根据损失函数来评估当前模型的损失
// 使用交叉熵(Cross Entropy)作为损失函数，优势是:
// 1. 能够在一定程度上评估模型的好坏，损失越小，模型越匹配训练样本（当然，为了防止过拟合，需要引入泛化）
// 2. 在逻辑回归分类模型中引入S函数(Sigmoid)后，交叉熵函数仍对参数θ可微，偏导数一定存在
//
// 总特征数量n, 总样本数量m
//
// Cost损失函数为J(θ(0),θ(1)...,θ(n))，计算过程如下：
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
// 最终：J(θ(0),θ(1)...,θ(n)) = -CostSum/m
//
// - thetas 当前模型参数
// - trainSet 训练样本集合
func evaluateCost(thetas []float64, trainSet [][]float64) float64 {
	var costSum float64 = 0

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		// θ(0) + θ(1)*x(1) + θ(2)*x(2) + ... + θ(n)*x(n)
		predictValueH := predict(thetas, trainSet[i])

		// 计算e^-(w'x)
		predictValue := 1 / (1 + math.Exp(-1*predictValueH))

		// 每行的最后一列是实际值
		realValue := trainSet[i][len(trainSet[i])-1]

		// 计算y(j)*log(hθ(x(j))
		costPart1 := realValue * math.Log(predictValue)

		// 计算(1-y(j))*log(1-hθ(x(j))
		costPart2 := (1 - realValue) * math.Log(1-predictValue)

		cost := costPart1 + costPart2

		// 将误差值累加
		costSum += cost
	}

	// J(θ(0),θ(1)...,θ(n)) = -CostSum/m
	cost := costSum / (-1 * float64(len(trainSet)))

	return cost
}

// evaluateCostWithLassoReg 计算使用L1 Lasso进行正则化后的损失函数来评估当前模型的损失
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// L1 = λ/m * (|θ(0)| + |θ(1)| + ... + |θ(n)|)，其中，|θ|表示θ的绝对值。这里除以m就够了，不需要除以2m，因为L1求导后，中间计算过程没有系数2了。
//
// - thetas 当前模型参数
// - trainSet 训练样本集合
// - regParam 正则参数
func evaluateCostWithLassoReg(thetas []float64, trainSet [][]float64, regParam float64) float64 {
	cost := evaluateCost(thetas, trainSet)

	// |θ(0)| + |θ(1)| + ... + |θ(n)|
	// 遍历特征的每一行
	lassoRegCost := 0.0
	for i := 0; i < len(thetas); i++ {
		thetasCost := math.Abs(thetas[i])
		lassoRegCost += thetasCost
	}

	lassoRegCost = regParam * lassoRegCost / float64(len(trainSet))

	costWithLassoReg := cost + lassoRegCost
	return costWithLassoReg
}

// evaluateCostWithRidgeReg 计算使用L2 Ridge进行正则化后的损失函数，来评估当前模型的损失
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// L2 = λ/2m * (θ(0)^2 + θ(1)^2 + ... + θ(n)^2)，为什么除以2，是因为后面求解梯度的时候会求偏导，计算过程会产生一个2，正好消掉
//
// - thetas 当前模型参数
// - trainSet 训练样本集合
// - regParam 正则参数
func evaluateCostWithRidgeReg(thetas []float64, trainSet [][]float64, regParam float64) float64 {
	cost := evaluateCost(thetas, trainSet)

	// θ(0)^2 + θ(1)^2 + ... + θ(n)^2
	// 遍历特征的每一行
	ridgeRegCost := 0.0
	for i := 0; i < len(thetas); i++ {
		thetasCost := math.Pow(thetas[i], 2)
		ridgeRegCost += thetasCost
	}

	ridgeRegCost = regParam * ridgeRegCost / (2 * float64(len(trainSet)))

	costWithRidgeReg := cost + ridgeRegCost
	return costWithRidgeReg
}

// calGradient 根据损失函数/交叉熵，求偏导后，计算梯度。
// 供梯度下降法在每一轮计算中使用
// 批量梯度下降(batch gradient descent)，样本不多的情况下，相比较随机梯度下降(SGD,stochastic gradient descent)收敛的速度更快，且保证朝全局最优逼近
// TODO：支持ElasticNet正则(赋予Lasso和Ridge不同的权重λ1和λ2，然后累加)
//
// 根据上文计算损失函数时的介绍：
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
// trainSet[j][i] 表示第j个样本的的第i个特征的值
// realValue(j)的值为0或1
func calGradient(thetas []float64, trainSet [][]float64, featureIndex int) float64 {
	var deviationSum float64 = 0

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		predictValueH := predict(thetas, trainSet[i])

		// 计算e^-(w'x)
		predictValue := 1 / (1 + math.Exp(-1*predictValueH))

		// 每一行的误差值, 用预测值减去实际值以获取误差，每行的最后一列是实际值
		deviation := predictValue - trainSet[i][len(trainSet[i])-1]

		// 按该行中要更新的特征的值进行缩放
		deviation *= trainSet[i][featureIndex]

		// 将误差值累加
		deviationSum += deviation
	}

	gradient := deviationSum / float64(len(trainSet))
	return gradient
}

// calGradientWithLassoReg 使用L1正则(Lasso)计算梯度
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// Grad_new(i) = (Grad_new(i) + λ*sgn(θ(i)))/m
// 其中，trainSet[j][i] 表示第j个样本的的第i个特征的值
func calGradientWithLassoReg(thetas []float64, trainSet [][]float64, featureIndex int, regParam float64) float64 {
	gradient := calGradient(thetas, trainSet, featureIndex)

	// get sgn(θ(i)
	sgnTheta := 0.0
	switch {
	case thetas[featureIndex] > 0:
		sgnTheta = 1
	case thetas[featureIndex] < 0:
		sgnTheta = -1
	default: // θ(i)=0
	}

	lassoReg := regParam * sgnTheta / float64(len(trainSet))

	gradientWithLassoReg := gradient + lassoReg
	return gradientWithLassoReg
}

// calGradientWithRidgeReg 用L2正则(Ridge)计算梯度
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// Grad_new(i) = (Grad_new(i) + λ*θ(i))/m
// 其中，trainSet[j][i] 表示第j个样本的的第i个特征的值
func calGradientWithRidgeReg(thetas []float64, trainSet [][]float64, featureIndex int, regParam float64) float64 {
	gradient := calGradient(thetas, trainSet, featureIndex)

	// λ*θ(i))/m
	ridgeReg := regParam * thetas[featureIndex] / float64(len(trainSet))

	gradientWithRidgeReg := gradient + ridgeReg
	return gradientWithRidgeReg
}

// predict 输出结果为概率。例如，我们可以认为predictValue>0.5或0.6时，预测结果为1，反之为0
func predict(thetas []float64, sample []float64) float64 {
	var predictValue = thetas[0]

	for i := 1; i < len(thetas); i++ {
		predictValue += thetas[i] * sample[i]
	}

	return predictValue
}

// StandardizeLocalInput 使用训练得到的标准化模型参数，需要先对预测的数据进行标准化处理
func StandardizeLocalInput(xbars, sigmas, input map[string]float64) map[string]float64 {
	standardizedInput := make(map[string]float64)

	for key, _ := range input {
		standardizedInput[key] = (input[key] - xbars[key]) / sigmas[key]
	}

	return standardizedInput
}

// PredictByLocalInput 利用标准化过后的样本进行预测
func PredictByLocalInput(thetas, standardizedInput map[string]float64) float64 {
	var predictValueH = thetas["Intercept"]

	for key, _ := range standardizedInput {
		predictValueH += thetas[key] * standardizedInput[key]
	}

	// 计算 1/(1+e^-(w'x))
	predictValue := 1 / (1 + math.Exp(-1*predictValueH))

	return predictValue
}
