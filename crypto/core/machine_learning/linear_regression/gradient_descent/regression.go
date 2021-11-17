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

package gradient_descent

import (
	"log"
	"math"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

// 基于梯度下降的多元线性回归模型算法
// Multiple Variable Linear Regression Model based on Gradient Descent method
// 优势：即使特征维度很多也能相对较快的完成模型训练
// 劣势：需要指定学习率α并进行多次迭代来得到拟合度目标

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

// StandardizeDataSet 将样本数据集合进行标准化处理，将均值变为0，标准差变为1
// Standardize the dataset, so mean will become 0 and standard deviation will become 1
// 对数据集的每一条数据的每个特征的减去该特征均值后除以特征标准差。
// 经过数据标准化之后，数据集合的所有特征都有了同样的变化范围。但标准化变换之后的特征分布没有发生改变。
// 尤其是，当数据集的各个特征取值范围存在较大差异时，或者是各特征取值单位差异较大时，需要使用标准化来对数据进行预处理。
// 通过该方法，可以使得梯度下降方法更快的找到最小参数以达到拟合收敛要求，或者在训练中使用更高的学习率α
// 注意，该函数是可选函数，如果数据集的各个特征维度的特征值都处于相似范围内，那么不做标准化处理也没关系。
// 当不知道数据集的各个特征维度的特征值的分布范围时，最好还是先进行标准化处理。
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
		// 对特征值进行标准化处理，对数据集的每一条数据的每个特征的减去该特征均值后除以特征标准差。
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

// deStandardizeThetas 逆标准化模型参数
func deStandardizeThetas(trainDataSet *common.TrainDataSet, thetas []float64) []float64 {
	xbars := make([]float64, len(thetas))
	sigmas := make([]float64, len(thetas))

	for i := 0; i < len(thetas); i++ {
		xbars[i] = trainDataSet.XbarParams[trainDataSet.FeatureNames[i]]
		sigmas[i] = trainDataSet.SigmaParams[trainDataSet.FeatureNames[i]]
	}

	for i := 1; i < len(thetas); i++ {
		thetas[0] -= thetas[i] * (xbars[i-1] / sigmas[i-1])
		thetas[i] = (thetas[i] * sigmas[len(sigmas)-1]) / sigmas[i-1]
	}

	thetas[0] *= sigmas[len(sigmas)-1]
	thetas[0] += xbars[len(xbars)-1]

	return thetas
}

// PreProcessDataSet 预处理标准化训练数据集
func PreProcessDataSet(sourceDataSet *common.StandardizedDataSet, targetFeatureName string) *common.TrainDataSet {
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
		if feature.FeatureName == targetFeatureName {
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

	// 遍历所有的特征维度，i表示遍历到第i个特征维度
	i = 1
	for _, feature := range sourceDataSet.OriginalFeatures {
		// 遍历每个特征维度的每个样本
		// 如果是目标维度
		if feature.FeatureName == targetFeatureName {
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

	featureNames = append(featureNames, targetFeatureName)

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
// 使用交叉验证的方案来验证泛化能力（后验知识），评价模型超参数的好坏，便于选择。
// 例如，可以用来寻找正则化的参数
// 注意：每次在迭代之前进行预处理，而不是统一进行预处理
//
// - sourceDataSet 原始样本数据
// - targetFeatureName 目标特征名称
// - alpha 训练学习率
// - amplitude 训练目标值
// - regMode 正则模式
// - regParam 正则参数
// - cvMode 交叉验证模式
// - cvParam 交叉验证参数
func EvaluateModelSuperParamByCV(sourceDataSet *common.DataSet, targetFeatureName string, alpha, amplitude float64, regMode int, regParam float64, cvMode int, cvParam int) float64 {
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
		standardizedDataSet := StandardizeDataSet(newDataSet)
		// 2. 预处理标准样本
		trainDataSet := PreProcessDataSet(standardizedDataSet, targetFeatureName)
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

	// 逆标准化模型参数，得到真实的模型
	thetas = deStandardizeThetas(trainDataSet, thetas)

	params := make(map[string]float64)
	params["Intercept"] = thetas[0]

	for i := 0; i < len(trainDataSet.FeatureNames)-1; i++ {
		params[trainDataSet.FeatureNames[i]] = thetas[i+1]
	}
	targetFeature := trainDataSet.FeatureNames[len(trainDataSet.FeatureNames)-1]

	// 计算模型拟合度
	rSquared := evaluateRSquared(thetas, trainDataSet.OriginalTrainSet)
	// 计算残差平方和
	totalCost := evaluateRSS(thetas, trainDataSet.OriginalTrainSet)

	totalNum := len(trainDataSet.OriginalTrainSet)
	// 计算均方误差
	avgCost := totalCost / float64(totalNum)
	// 计算均方根误差
	rmse := math.Sqrt(avgCost)

	model := &common.Model{
		Params:        params,        // 所有特征对应的系数
		RSquared:      rSquared,      // r平方，用于衡量模型的拟合度，数值越接近1，说明模型的拟合度越优
		TargetFeature: targetFeature, // 目标特征
		RMSE:          rmse,          // 均方根误差，用于衡量模型的误差
	}

	return model
}

// evaluateRSS 计算RSS(Residual Sum of Squares)残差平方和，用于衡量模型的误差
func evaluateRSS(thetas []float64, trainSet [][]float64) float64 {
	// （实际值-预测值）平方之和
	var rss float64 = 0
	for i := 0; i < len(trainSet); i++ {
		realValue := trainSet[i][len(thetas)]
		predictValue := predict(thetas, trainSet[i])

		rss += math.Pow(realValue-predictValue, 2)
	}

	return rss
}

// evaluateRSquared 计算R-Squared，用于衡量模型的拟合度，数值越接近1模型的拟合度越优
func evaluateRSquared(thetas []float64, trainSet [][]float64) float64 {
	// 目标特征的总值，初始化为0
	var ySum float64 = 0

	for i := 0; i < len(trainSet); i++ {
		ySum += trainSet[i][len(thetas)]
	}

	// 计算目标特征的均值
	var ybar = ySum / float64(len(trainSet))

	// 计算残差平方和
	var rss float64 = 0
	// 计算总离差平方和（实际值-平均值）平方之和
	var tss float64 = 0

	for i := 0; i < len(trainSet); i++ {
		realValue := trainSet[i][len(thetas)]
		predictValue := predict(thetas, trainSet[i])

		rss += math.Pow(realValue-predictValue, 2)
		tss += math.Pow(realValue-ybar, 2)
	}

	rSquared := 1 - rss/tss
	return rSquared
}

// predict 利用模型对样本进行预测
func predict(thetas []float64, sample []float64) float64 {
	// 模型的常数项
	var predictValue = thetas[0]

	// 样本每个特征对应的值乘以模型中特征对应的系数，之后求和
	for i := 1; i < len(thetas); i++ {
		predictValue += thetas[i] * sample[i]
	}

	return predictValue
}

// train 模型训练
// 1. 支持正则化（L2正则或者L1正则），在样本数量偏少，而特征数量偏多的时候，避免过拟合，提升泛化能力
//    L1更容易得到稀疏解，拥有特征选择的能力
//    L2控制过拟合的效果比L1更好一些
// 2. 支持配合交叉验证使用，避免过拟合，提升泛化能力
//
// - alpha 梯度下降法的学习率α
// - amplitude 拟合度delta的目标值
// - regMode 正则模型
// - regParam 正则参数
func train(trainSet [][]float64, alpha float64, amplitude float64, regMode int, regParam float64) ([]float64, float64) {
	// 每个特征维度都有一个系数theta，此外还有intercept为模型的常数项
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
		// 根据正则类型计算每个系数的梯度，并更新模型
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
// 选择MSE(Mean squared error)作为损失函数，优势是
// 1. 能够在一定程度上评估模型的好坏，损失越小，模型越匹配训练样本
// 2. MSE函数对参数θ可微，偏导数一定存在
//
// 总特征数量n, 总样本数量m
// Cost损失函数为J(θ(0),θ(1)...,θ(n))，计算过程如下：
// for j:=0; j++; j<m
// {
//		CostSum += (predictValue(j) - realValue(j))^2
// }
// 其中，predictValue(j) = θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(n)*xj(n), realValue(j) = yj
// predictValue(j) - realValue(j) = θ(0) + w*x(j) - y(j)
// 参数集合w = [θ(1),θ(2),...,θ(n)]
// 整个模型的参数集合w' = (w, θ(0)) = [θ(0),θ(1),θ(2),...,θ(n)]
//
// 最终：J(θ(0),θ(1)...,θ(n)) = CostSum/2m
//
// - thetas 当前模型参数
// - trainSet 训练样本集合
func evaluateCost(thetas []float64, trainSet [][]float64) float64 {
	var costSum float64 = 0

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		predictValue := predict(thetas, trainSet[i])

		// 每一行的误差值, 用预测值减去实际值以获取误差，每行的最后一列是实际值
		deviation := predictValue - trainSet[i][len(trainSet[i])-1]

		// deviation取平方
		deviation *= deviation

		// 将误差值累加
		costSum += deviation
	}

	cost := costSum / (2 * float64(len(trainSet)))
	return cost
}

// evaluateCostWithLassoReg 计算使用L1 Lasso进行正则化后的损失函数来评估当前模型的损失
// 定义总特征数量n，总样本数量m，正则项系数λ（用来权衡正则项与原始损失函数项的比重）
// L1 = λ/m * (|θ(0)| + |θ(1)| + ... + |θ(n)|)，其中，|θ|表示θ的绝对值
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
// L2 = λ/2m * (θ(0)^2 + θ(1)^2 + ... + θ(n)^2)
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

// calGradient 根据损失函数/残差平方和/均方误差（MSE）/欧氏距离之和，求偏导后，计算梯度，
// 供梯度下降法在每一轮计算中使用
// 批量梯度下降(batch gradient descent)，样本不多的情况下，相比较随机梯度下降(SGD,stochastic gradient descent)收敛的速度更快，且保证朝全局最优逼近
// TODO：支持ElasticNet正则(赋予Lasso和Ridge不同的权重λ1和λ2，然后累加)
func calGradient(thetas []float64, trainSet [][]float64, featureIndex int) float64 {
	var deviationSum float64 = 0

	// 遍历样本的每一行
	for i := 0; i < len(trainSet); i++ {
		predictValue := predict(thetas, trainSet[i])

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
// L1 = λ/m * (|θ(0)| + |θ(1)| + ... + |θ(n)|)，其中，|θ|表示θ的绝对值
// 新的Cost损失函数J_new(θ(0),θ(1)...,θ(n)) = J(θ(0),θ(1)...,θ(n) + L1
//
// 方法：分别对不同的θ求偏导，来寻找最优解
// 令g(i) = ΔJ_new(θ(0),θ(1)...,θ(n))/Δθ(i)，其中i为特征维度的index，也就是0,1,2,...,n
// j为样本编号值，从0自增到总样本数量m，i为特征维度的index
// 备注：对绝对值|θ|求导，得到的是sgn(θ),sgn(θ)表示θ的符号。也就是说：
// 1. θ>0,则sgn(θ)=1
// 2. θ<0,则sgn(θ)=-1
// 3. θ=0,|θ|不可导。那么就只能对该系数θ放弃正则化，因此可以定义当θ=0时,sgn(θ)=0
// g(i) = (Δ/Δθ(i))*( (1/2m)*SumFromZeroToM((θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(i)*xj(i) + ... + θ(n)*xj(n) - yj)^2)
//					 + (λ/m)*SumFromZeroToN(|θ(0)| + |θ(1)| + |θ(2)| ... + |θ(j)| + ... + |θ(n)|) )
// 那么，g(i) = (1/2m) * ( SumFromZeroToM((θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(i)*xj(i) + ... + θ(n)*xj(n) - yj)*2*xj(i))
//						+ 2λ*sgn(θ(i)) )
//			= (1/2m) * ( SumFromZeroToM((predictValue(j) - realValue(j)) * 2xj(i)) + 2*λ*sgn(θ(i)) )
//			= (1/m) * ( SumFromZeroToM((predictValue(j) - realValue(j)) * xj(i)) + λ*sgn(θ(i)))
//			= Grad_new(i)
//
// Grad_new(i)的定义在下面会介绍
//
// ------------
//
// step 3: 结论
// 计算出w' = (w, θ(0)) = [θ(0),θ(1),θ(2),...,θ(n)]中的每个θ(i)，来得到模型w'
// for i:=0; i++; i<n
// {
// 		// 计算第i个特征的参数θ(i):
//		θ(i) = θ(i) − α*Grad_new(i)
// }
//
// 第i个特征的Grad_new(i):
// for j:=0; j++; i<m
// {
//	Grad_new(i) += (predictValue(j) - realValue(j))*trainSet[j][i]
// }
//
// Grad_new(i) = (Grad_new(i) + λ*sgn(θ(i)))/m
//
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
// L2 = λ/2m * (θ(0)^2 + θ(1)^2 + ... + θ(n)^2)
// 新的Cost损失函数J_new(θ(0),θ(1)...,θ(n)) = J(θ(0),θ(1)...,θ(n) + L2
//
// 方法：分别对不同的θ求偏导，来寻找最优解
// 令g(i) = ΔJ_new(θ(0),θ(1)...,θ(n))/Δθ(i)，其中i为特征维度的index，也就是0,1,2,...,n
// j为样本编号值，从0自增到总样本数量m，i为特征维度的index
// g(i) = (Δ/Δθ(i))*( (1/2m)*SumFromZeroToM((θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(i)*xj(i) + ... + θ(n)*xj(n) - yj)^2)
//					 + (λ/2m)*SumFromZeroToN(θ(0)^2 + θ(1)^2 + ... + θ(j)^2 + ... + θ(n)^2) )
// 那么，g(i) = (1/2m) * ( SumFromZeroToM((θ(0) + θ(1)*xj(1) + θ(2)*xj(2) + ... + θ(i)*xj(i) + ... + θ(n)*xj(n) - yj)*2*xj(i))
//						+ λ*2*θ(i) )
//			= (1/2m) * ( SumFromZeroToM((predictValue(j) - realValue(j)) * 2xj(i)) + 2*λ*θ(i) )
//			= (1/m) * ( SumFromZeroToM((predictValue(j) - realValue(j)) * xj(i)) + λ*θ(i))
//			= Grad_new(i)
//
// Grad_new(i)的定义在下面会介绍
//
// ------------
//
// step 3: 结论
// 计算出w' = (w, θ(0)) = [θ(0),θ(1),θ(2),...,θ(n)]中的每个θ(i)，来得到模型w'
// for i:=0; i++; i<n
// {
// 		// 计算第i个特征的参数θ(i):
//		θ(i) = θ(i) − α*Grad_new(i)
// }
//
// 第i个特征的Grad_new(i):
// for j:=0; j++; i<m
// {
//	Grad_new(i) += (predictValue(j) - realValue(j))*trainSet[j][i]
// }
//
// Grad_new(i) = (Grad_new(i) + λ*θ(i))/m
//
// 其中，trainSet[j][i] 表示第j个样本的的第i个特征的值
func calGradientWithRidgeReg(thetas []float64, trainSet [][]float64, featureIndex int, regParam float64) float64 {
	gradient := calGradient(thetas, trainSet, featureIndex)

	// λ*θ(i))/m
	ridgeReg := regParam * thetas[featureIndex] / float64(len(trainSet))

	gradientWithRidgeReg := gradient + ridgeReg
	return gradientWithRidgeReg
}
