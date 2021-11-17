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

package common

import "math/big"

// 用于多元线性回归模型的术语Term:
// intercept 截距，模型中的常量。
// theta 模型中每个特征的系数
// variance 方差，用来度量随机变量和其数学期望（即均值）之间的偏离程度。统计中的方差（样本方差）是每个样本值与全体样本值的平均数之差的平方值的平均数。
// Sigma 标准差σ=方差开平方。标准差是总体各单位标准值与其平均数离差平方的算术平均数的平方根。即标准差是方差的平方根(方差是离差的平方的加权平均数)。
// Xbar 平均值。样本中每个特征维度的平均数。
// RSS: Residual Sum of Squares 残差平方和，也就是（实际值-预测值）的平方之和。rss += math.Pow(realValue-predictValue, 2)
// TSS: Total Sum of Squares 总平方和/总离差平方和，也就是（实际值-平均值）的平方之和。tss += math.Pow(realValue-ybar, 2)
// r-squared r平方。也就是拟合度。衡量模型拟合度，r-squared = 1 - RSS/TSS。拟合度介于[0,1]，越接近1，说明拟合度越优。
// MSE: Mean Squared Error。均方误差，可以用来作为损失函数，作为优化目标来衡量模型的精准度。

// 定义正则化Regularize的类型
// Regularize，正则化，也可以翻译为规则化。
// 意思是对模型的损失函数引入先验知识约束规则。使得在模型训练时（寻找满足损失函数的值达到最小时，模型参数的最优解），强行缩小模型参数的求解空间，从而产生符合先验知识约束规则的解。
// 例如，L1 Lasso约束规则，倾向于产生稀疏解，从而拥有特征选择的能力。L2 Ridge约束规则，更平滑，控制过拟合的效果比L1相对而言更好一些，但不具备特征选择的能力
const (
	// no Regularize
	RegNone = iota
	// L1 Lasso
	RegLasso
	// L2 Ridge
	RegRidge
)

// 定义交叉验证Cross Validation的类型
const (
	// no Cross Validation
	CvNone = iota
	// Leave One Out Cross Validation (Exhaustive cross-validation)
	CvLoo
	// Leave P Out Cross Validation (Exhaustive cross-validation)
	CvLpo
	// K-Fold Cross Validation (Non-exhaustive cross-validation)
	CvKfold
	// Repeated K-Fold Cross Validation (Non-exhaustive cross-validation)
	CvRepKfold
	// Monte Carlo Cross Validation, also called Repeated random sub-sampling validation (Non-exhaustive cross-validation)
	CvMonteCarlo
)

// DataSet 用于训练的样本数据集
type DataSet struct {
	Features []*DataFeature // 样本多元特征，有多少维度，该数组就有多大
}

// StandardizedDataSet 经过标准化后的样本数据集
type StandardizedDataSet struct {
	Features         []*DataFeature     `json:"features"`          // 样本多元特征
	XbarParams       map[string]float64 `json:"xbar_params"`       // 特征的均值
	SigmaParams      map[string]float64 `json:"sigma_params"`      // 特征的标准差
	OriginalFeatures []*DataFeature     `json:"original_features"` // 标准化处理之前的原始样本多元特征
}

// DataFeature 样本的多元特征维度
type DataFeature struct {
	FeatureName string `json:"feature_name"` // 特征的名称
	// 特别注意：key需要从0开始自增
	Sets map[int]float64 `json:"sets"` // 特征集合,key是样本编号，value是特征的值。线性回归模型仅能处理数值化特征，对于非数值化特征（分类特征），需要进行编码处理。
}

// TrainDataSet 经过预处理后的训练数据集合
type TrainDataSet struct {
	FeatureNames     []string           `json:"feature_names"`      // 特征的名称
	TrainSet         [][]float64        `json:"train_set"`          // 特征集合
	XbarParams       map[string]float64 `json:"xbar_params"`        // 特征的均值
	SigmaParams      map[string]float64 `json:"sigma_params"`       // 特征的标准差
	OriginalTrainSet [][]float64        `json:"original_train_set"` // 标准化处理之前的原始特征集合
}

// Model 训练后得到的模型
type Model struct {
	Params        map[string]float64 `json:"params"`         // 模型参数集合,key是特征的名称，value是模型中该特征的系数
	TargetFeature string             `json:"target_feature"` // 目标特征名称
	RSquared      float64            `json:"r_squared"`      // r平方。用于衡量模型的拟合度，介于[0,1]，越接近1，说明拟合度越优。
	RMSE          float64            `json:"rmse"`           // Root Mean Squared Error 均方根误差。用于衡量模型的误差。真实值-预测值，平方之后求和，再计算平均值，最后开平方
}

// EncLocalGradient 本地加密梯度信息
type EncLocalGradient struct {
	EncGrad     map[int]*big.Int
	RandomNoise *big.Int
}

// EncLocalCost 本地加密损失信息
type EncLocalCost struct {
	EncCost     map[int]*big.Int
	RandomNoise *big.Int
}
