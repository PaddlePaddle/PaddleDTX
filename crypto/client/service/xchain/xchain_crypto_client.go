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

package xchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/homomorphism/paillier"
	"github.com/PaddlePaddle/PaddleDTX/crypto/common/math/rand"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/hash"

	ml_common "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/linear_regression/gradient_descent"
	linear_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/linear_regression/gradient_descent/mpc_vertical"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/logic_regression"
	logic_vertical "github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/logic_regression/mpc_vertical"

	"github.com/PaddlePaddle/PaddleDTX/crypto/core/pdp/merkle"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/pdp/pairing"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/secret_share/complex_secret_share"
)

type XchainCryptoClient struct {
}

// --- 哈希算法相关 start ---

// HashUsingSha256 使用SHA256做单次哈希运算
func (xcc *XchainCryptoClient) HashUsingSha256(data []byte) []byte {
	hashResult := hash.HashUsingSha256(data)
	return hashResult
}

// --- 哈希算法相关 end ---

// --- 随机数相关 start ---

// GenerateEntropy 产生指定比特长度的随机熵
func (xcc *XchainCryptoClient) GenerateEntropy(bitSize int) ([]byte, error) {
	entropyByte, err := rand.GenerateEntropy(bitSize)
	return entropyByte, err
}

// --- 随机数相关 end ---

// --- secret_share 秘密分享算法相关 start ---

// SecretSplit 将秘密信息分割为指定数量的碎片
// - totalShareNumber 需要分割的碎片数量
// - minimumShareNumber 能够还原出原信息的最少碎片数量
// - secret 待分割的秘密信息
func (xcc *XchainCryptoClient) SecretSplit(totalShareNumber, minimumShareNumber int, secret []byte) (shares map[int]*big.Int, err error) {
	curve := elliptic.P256()
	return complex_secret_share.ComplexSecretSplit(totalShareNumber, minimumShareNumber, secret, curve)
}

// SecretRetrieve 利用碎片还原秘密值
func (xcc *XchainCryptoClient) SecretRetrieve(shares map[int]*big.Int) ([]byte, error) {
	curve := elliptic.P256()
	return complex_secret_share.ComplexSecretRetrieve(shares, curve)
}

// --- secret_share 秘密分享算法相关 end ---

// --- PDP 副本保持证明相关 start ---

// GetMerkleRoot 计算梅克尔树根
func (xcc *XchainCryptoClient) GetMerkleRoot(objects [][]byte) []byte {
	return merkle.GetMerkleRoot(objects)
}

// GenPairingKeyPair 随机生成基于双线性映射副本保持证明的公私钥对
func (xcc *XchainCryptoClient) GenPairingKeyPair() ([]byte, []byte, error) {
	privkey, pubkey, err := pairing.GenKeyPair()
	if err != nil {
		return nil, nil, err
	}
	return pairing.PrivateKeyToByte(privkey), pairing.PublicKeyToByte(pubkey), nil
}

// RandomWithinPairingOrder 生成小于椭圆曲线order的随机数
func (xcc *XchainCryptoClient) RandomWithinPairingOrder() ([]byte, error) {
	rand, err := pairing.RandomWithinOrder()
	if err != nil {
		return nil, err
	}
	return rand.Bytes(), nil
}

// CalculateSigmaI 为指定数据块生成证明辅助信息
// - content 该数据块的内容
// - index 数据块对于原始数据的的索引
// - randomV 小于椭圆曲线阶order的随机数
// - randomU 小于椭圆曲线阶order的随机数
// - privkey 副本保持证明私钥
func (xcc *XchainCryptoClient) CalculateSigmaI(content, index, randomV, randomU, privkey []byte, round int64) ([]byte, error) {
	param := pairing.CalculateSigmaIParamsFromBytes(content, index, randomV, randomU, privkey, round)
	sigma, err := pairing.CalculateSigmaI(param)
	if err != nil {
		return nil, err
	}
	return pairing.G1ToByte(sigma), nil
}

// GenPairingChallenge 随机生成副本保持证明挑战信息
// - indexList 为要验证的索引列表
// - round 为挑战轮数
// - privkey 为副本保持证明私钥
func (xcc *XchainCryptoClient) GenPairingChallenge(indexList []int, round int64, privkey []byte) ([][]byte, [][]byte, []byte, error) {
	sk := pairing.PrivateKeyFromByte(privkey)
	idx, vs, randNum, err := pairing.GenerateChallenge(indexList, round, sk)
	if err != nil {
		return nil, nil, nil, err
	}
	return pairing.IntListToBytes(idx), pairing.IntListToBytes(vs), randNum, nil
}

// ProvePairingChallenge 生成挑战的应答信息
// - content 要验证的数据块内容列表
// - indices 要验证的索引列表
// - randVs 调整生成的随机数列表
// - sigmas 要验证的数据块对应的辅助证明信息列表
func (xcc *XchainCryptoClient) ProvePairingChallenge(content, indices, randVs, sigmas [][]byte, rand []byte) ([]byte, []byte, error) {
	param, err := pairing.ProofParamsFromBytes(content, indices, randVs, sigmas, rand)
	if err != nil {
		return nil, nil, err
	}
	sigma, mu, err := pairing.Prove(param)
	return pairing.G1ToByte(sigma), pairing.G1ToByte(mu), err
}

// VerifyPairingProof 挑战验证信息
// - sigma 证明生成的应答信息
// - mu 证明生成的应答信息
// - randV 验证者生成的随机数
// - randU 验证者生成的随机数
// - pubkey 验证者的副本保持证明公钥
// - indices 要验证的索引列表
// - randVs 调整生成的随机数列表
func (xcc *XchainCryptoClient) VerifyPairingProof(sigma, mu, randV, randU, pubkey []byte, indices, randVs [][]byte) (bool, error) {
	param, err := pairing.VerifyParamsFromBytes(sigma, mu, randV, randU, pubkey, indices, randVs)
	if err != nil {
		return false, err
	}
	return pairing.Verify(param)
}

// --- PDP 副本保持证明相关 end ---

// --- Paillier 加法同态相关 start ---

// GeneratePaillierPrivateKey 生成指定比特长度的paillier同态公私钥对
func (xcc *XchainCryptoClient) GeneratePaillierPrivateKey(primeLength int) (*paillier.PrivateKey, error) {
	return paillier.GeneratePrivateKey(primeLength)
}

// --- Paillier 加法同态相关 end ---

// --- 机器学习-通用方法 start ---

// LinRegImportFeatures 从文件导入用于多元线性回归的数据特征
func (xcc *XchainCryptoClient) LinRegImportFeatures(fileRows [][]string) ([]*ml_common.DataFeature, error) {
	return ml_common.ImportFeaturesForLinReg(fileRows)
}

// LogRegImportFeatures 从文件导入用于多元逻辑回归的数据特征
// - fileRows 样本数据
// - label 目标特征
// - labelName 目标训练值
func (xcc *XchainCryptoClient) LogRegImportFeatures(fileRows [][]string, label, labelName string) ([]*ml_common.DataFeature, error) {
	return ml_common.ImportFeaturesForLogReg(fileRows, label, labelName)
}

// --- 机器学习-通用方法 end ---

// --- 多元线性回归 start ---
// LinRegStandardizeDataSet 标准化样本数据，每个特征对应的样本均值变为0，标准差变为1
func (xcc *XchainCryptoClient) LinRegStandardizeDataSet(sourceDataSet *ml_common.DataSet) *ml_common.StandardizedDataSet {
	return gradient_descent.StandardizeDataSet(sourceDataSet)
}

// LinRegPreProcessDataSet 预处理样本数据
func (xcc *XchainCryptoClient) LinRegPreProcessDataSet(sourceDataSet *ml_common.StandardizedDataSet, targetFeatureName string) *ml_common.TrainDataSet {
	return gradient_descent.PreProcessDataSet(sourceDataSet, targetFeatureName)
}

// LinRegEvaluateModelSuperParamByCV 通过交叉验证计算指定正则参数对应的模型均方根误差
// - sourceDataSet 原始样本数据
// - targetFeatureName 目标特征名称
// - alpha 训练学习率
// - amplitude 训练目标值
// - regMode 正则模式
// - regParam 正则参数
// - cvMode 交叉验证模式
// - cvParam 交叉验证参数
func (xcc *XchainCryptoClient) LinRegEvaluateModelSuperParamByCV(sourceDataSet *ml_common.DataSet, targetFeatureName string, alpha, amplitude float64, regMode int, regParam float64, cvMode int, cvParam int) float64 {
	return gradient_descent.EvaluateModelSuperParamByCV(sourceDataSet, targetFeatureName, alpha, amplitude, regMode, regParam, cvMode, cvParam)
}

// LinRegTrainModel 多元线性回归模型训练
// - trainDataSet 预处理过的训练数据
// - alpha 训练学习率
// - amplitude 训练目标值
// - regMode 正则模式
// - regParam 正则参数
func (xcc *XchainCryptoClient) LinRegTrainModel(trainDataSet *ml_common.TrainDataSet, alpha float64, amplitude float64, regMode int, regParam float64) *ml_common.Model {
	return gradient_descent.TrainModel(trainDataSet, alpha, amplitude, regMode, regParam)
}

// --- 多元线性回归 end ---

// --- 多元逻辑回归 start ---
// LogRegStandardizeDataSet 标准化样本数据，除目标特征，其余特征对应的样本数值均值变为0，标准差变为1
func (xcc *XchainCryptoClient) LogRegStandardizeDataSet(sourceDataSet *ml_common.DataSet, labelName string) *ml_common.StandardizedDataSet {
	return logic_regression.StandardizeDataSet(sourceDataSet, labelName)
}

// LogRegPreProcessDataSet 预处理样本数据
func (xcc *XchainCryptoClient) LogRegPreProcessDataSet(sourceDataSet *ml_common.StandardizedDataSet, labelName string) *ml_common.TrainDataSet {
	return logic_regression.PreProcessDataSet(sourceDataSet, labelName)
}

// LogRegTrainModel 多元逻辑回归模型训练
// - trainDataSet 预处理过的训练数据
// - alpha 训练学习率
// - amplitude 训练目标值
// - regMode 正则模式
// - regParam 正则参数
func (xcc *XchainCryptoClient) LogRegTrainModel(trainDataSet *ml_common.TrainDataSet, alpha float64, amplitude float64, regMode int, regParam float64) *ml_common.Model {
	return logic_regression.TrainModel(trainDataSet, alpha, amplitude, regMode, regParam)
}

// LogRegStandardizeLocalInput 标准化样本数据
// - xbars 特征对应的均值map
// - sigmas 特征对应的方差map
// - input 特征对应的样本值
func (xcc *XchainCryptoClient) LogRegStandardizeLocalInput(xbars, sigmas, input map[string]float64) map[string]float64 {
	return logic_regression.StandardizeLocalInput(xbars, sigmas, input)
}

// LogRegPredictByLocalInput 计算预测值
// - thetas 训练得到的模型
// - standardizedInput 标准化后的样本数据
func (xcc *XchainCryptoClient) LogRegPredictByLocalInput(thetas, standardizedInput map[string]float64) float64 {
	return logic_regression.PredictByLocalInput(thetas, standardizedInput)
}

// --- 多元逻辑回归 end ---

// --- 联邦学习-通用-纵向 start ---

// PSIEncryptSampleIDSet 利用同态公钥加密样本的ID列表
// - sampleID 待加密的ID列表
// - publicKey 同态公钥
func (xcc *XchainCryptoClient) PSIEncryptSampleIDSet(sampleID []string, publicKey *ecdsa.PublicKey) *linear_vertical.EncSet {
	return linear_vertical.EncryptSampleIDSet(sampleID, publicKey)
}

// PSIReEncryptIDSet 利用同态私钥二次加密样本ID列表
// - encSet 一次加密后的ID列表
// - privateKey 同态私钥
func (xcc *XchainCryptoClient) PSIReEncryptIDSet(encSet *linear_vertical.EncSet, privateKey *ecdsa.PrivateKey) *linear_vertical.EncSet {
	return linear_vertical.ReEncryptIDSet(encSet, privateKey)
}

// PSIntersect 计算多方加密ID列表的交集
// - sampleID 原始ID列表
// - reEncSetLocal 己方二次加密后的ID列表
// - reEncSetOthers 其他方二次加密后的ID列表
func (xcc *XchainCryptoClient) PSIntersect(sampleID []string, reEncSetLocal *linear_vertical.EncSet, reEncSetOthers []*linear_vertical.EncSet) []string {
	return linear_vertical.Intersect(sampleID, reEncSetLocal, reEncSetOthers)
}

// --- 联邦学习-通用-纵向 end ---

// --- 联邦学习-多元线性回归-纵向 start ---

// LinRegVLStandardizeDataSet 标准化样本数据，每个特征对应的样本均值变为0，标准差变为1
func (xcc *XchainCryptoClient) LinRegVLStandardizeDataSet(sourceDataSet *ml_common.DataSet) *ml_common.StandardizedDataSet {
	return linear_vertical.StandardizeDataSet(sourceDataSet)
}

// LinRegVLPreProcessDataSet 非标签方预处理样本数据
func (xcc *XchainCryptoClient) LinRegVLPreProcessDataSet(sourceDataSet *ml_common.StandardizedDataSet) *ml_common.TrainDataSet {
	return linear_vertical.PreProcessDataSetNoTag(sourceDataSet)
}

// LinRegVLPreProcessDataSetTagPart 标签方预处理样本数据
func (xcc *XchainCryptoClient) LinRegVLPreProcessDataSetTagPart(sourceDataSet *ml_common.StandardizedDataSet, targetFeatureName string) *ml_common.TrainDataSet {
	return linear_vertical.PreProcessDataSet(sourceDataSet, targetFeatureName)
}

// LinRegVLCalLocalGradAndCost 非标签方计算本地的梯度和损失数据
// - thetas 上一轮训练得到的模型参数
// - trainSet 预处理过的训练数据
// - accuracy 同态加解密精度
// - regMode 正则模式
// - regParam 正则参数
// - publicKey 非标签方同态公钥
func (xcc *XchainCryptoClient) LinRegVLCalLocalGradAndCost(thetas []float64, trainSet [][]float64, accuracy int, regMode int, regParam float64, publicKey *paillier.PublicKey) (*linear_vertical.LocalGradientPart, error) {
	return linear_vertical.CalLocalGradientPart(thetas, trainSet, accuracy, regMode, regParam, publicKey)
}

// LinRegVLCalLocalGradAndCostTagPart 标签方计算本地的梯度和损失数据
// - thetas 上一轮训练得到的模型参数
// - trainSet 预处理过的训练数据
// - accuracy 同态加解密精度
// - regMode 正则模式
// - regParam 正则参数
// - publicKey 标签方同态公钥
func (xcc *XchainCryptoClient) LinRegVLCalLocalGradAndCostTagPart(thetas []float64, trainSet [][]float64, accuracy int, regMode int, regParam float64, publicKey *paillier.PublicKey) (*linear_vertical.LocalGradientPart, error) {
	return linear_vertical.CalLocalGradientTagPart(thetas, trainSet, accuracy, regMode, regParam, publicKey)
}

// LinRegVLCalEncGradient 非标签方计算加密的梯度，用标签方的同态公钥加密
// - localPart 非标签方本地的明文梯度数据
// - tagPart 标签方的加密梯度数据
// - trainSet 非标签方训练样本集合
// - featureIndex 指定特征的索引
// - accuracy 同态加解密精度
// - publicKey 标签方同态公钥
func (xcc *XchainCryptoClient) LinRegVLCalEncGradient(localPart *linear_vertical.RawLocalGradientPart, tagPart *linear_vertical.EncLocalGradientPart, trainSet [][]float64, featureIndex, accuracy int, publicKey *paillier.PublicKey) (*ml_common.EncLocalGradient, error) {
	return linear_vertical.CalEncLocalGradient(localPart, tagPart, trainSet, featureIndex, accuracy, publicKey)
}

// LinRegVLCalEncGradientTagPart 标签方计算加密的梯度，用非标签方的同态公钥加密
// - localPart 标签方本地的明文梯度数据
// - otherPart 非标签方的加密梯度数据
// - trainSet 标签方训练样本集合
// - featureIndex 指定特征的索引
// - accuracy 同态加解密精度
// - publicKey 非标签方同态公钥
func (xcc *XchainCryptoClient) LinRegVLCalEncGradientTagPart(localPart *linear_vertical.RawLocalGradientPart, otherPart *linear_vertical.EncLocalGradientPart, trainSet [][]float64, featureIndex, accuracy int, publicKey *paillier.PublicKey) (*ml_common.EncLocalGradient, error) {
	return linear_vertical.CalEncLocalGradientTagPart(localPart, otherPart, trainSet, featureIndex, accuracy, publicKey)
}

// LinRegVLDecryptGradient 为其他方解密带噪音的梯度信息
// - encGradMap 加密的梯度信息
// - privateKey 己方同态私钥
func (xcc *XchainCryptoClient) LinRegVLDecryptGradient(encGradMap map[int]*big.Int, privateKey *paillier.PrivateKey) map[int]*big.Int {
	return linear_vertical.DecryptGradient(encGradMap, privateKey)
}

// LinRegVLRetrieveRealGradient 还原真实的梯度数据
// - decGradMap 解密的梯度信息
// - accuracy 同态加解密精度
// - randomInt 己方梯度的噪音值
func (xcc *XchainCryptoClient) LinRegVLRetrieveRealGradient(decGradMap map[int]*big.Int, accuracy int, randomInt *big.Int) map[int]float64 {
	return linear_vertical.RetrieveRealGradient(decGradMap, accuracy, randomInt)
}

// LinRegVLCalGradient 根据还原的明文梯度数据计算梯度值
func (xcc *XchainCryptoClient) LinRegVLCalGradient(gradMap map[int]float64) float64 {
	return linear_vertical.CalGradient(gradMap)
}

// LinRegVLEvaluateEncCost 非标签方计算加密的损失，用其他参与方的同态公钥加密
// - localPart 本地的明文损失数据
// - tagPart 标签方的加密损失数据
// - trainSet 非标签方训练样本集合
// - publicKey 标签方同态公钥
func (xcc *XchainCryptoClient) LinRegVLEvaluateEncCost(localPart *linear_vertical.RawLocalGradientPart, tagPart *linear_vertical.EncLocalGradientPart, trainSet [][]float64, publicKey *paillier.PublicKey) (*ml_common.EncLocalCost, error) {
	return linear_vertical.EvaluateEncLocalCost(localPart, tagPart, trainSet, publicKey)
}

// LinRegVLEvaluateEncCostTagPart 标签方计算加密的损失，用其他参与方的同态公钥加密
// - localPart 本地的明文损失数据
// - otherPart 非标签方的加密损失数据
// - trainSet 标签方训练样本集合
// - publicKey 非标签方同态公钥
func (xcc *XchainCryptoClient) LinRegVLEvaluateEncCostTagPart(localPart *linear_vertical.RawLocalGradientPart, otherPart *linear_vertical.EncLocalGradientPart, trainSet [][]float64, publicKey *paillier.PublicKey) (*ml_common.EncLocalCost, error) {
	return linear_vertical.EvaluateEncLocalCostTag(localPart, otherPart, trainSet, publicKey)
}

// LinRegVLDecryptCost 为其他方解密带噪音的损失信息
// - encCostMap 加密的损失信息
// - privateKey 己方同态私钥
func (xcc *XchainCryptoClient) LinRegVLDecryptCost(encCostMap map[int]*big.Int, privateKey *paillier.PrivateKey) map[int]*big.Int {
	return linear_vertical.DecryptCost(encCostMap, privateKey)
}

// LinRegVLRetrieveRealCost 还原真实的损失
// - decCostMap 解密的损失信息
// - accuracy 同态加解密精度
// - randomInt 损失的噪音值
func (xcc *XchainCryptoClient) LinRegVLRetrieveRealCost(decCostMap map[int]*big.Int, accuracy int, randomInt *big.Int) map[int]float64 {
	return linear_vertical.RetrieveRealCost(decCostMap, accuracy, randomInt)
}

// LinRegVLCalCost 根据还原的损失信息计算损失值
func (xcc *XchainCryptoClient) LinRegVLCalCost(costMap map[int]float64) float64 {
	return linear_vertical.CalCost(costMap)
}

// LinRegVLStandardizeLocalInput 标准化样本数据
// - xbars 特征对应的均值map
// - sigmas 特征对应的方差map
// - input 特征对应的样本值
func (xcc *XchainCryptoClient) LinRegVLStandardizeLocalInput(xbars, sigmas, input map[string]float64) map[string]float64 {
	return linear_vertical.StandardizeLocalInput(xbars, sigmas, input)
}

// LinRegVLPredictLocalPart 非标签方计算本地预测值
// - thetas 训练得到的模型
// - standardizedInput 标准化后的样本数据
func (xcc *XchainCryptoClient) LinRegVLPredictLocalPart(thetas, standardizedInput map[string]float64) float64 {
	return linear_vertical.PredictLocalPartNoTag(thetas, standardizedInput)
}

// LinRegVLPredictLocalTagPart 标签方计算本地预测值
// - thetas 训练得到的模型
// - standardizedInput 标准化后的样本数据
func (xcc *XchainCryptoClient) LinRegVLPredictLocalTagPart(thetas, standardizedInput map[string]float64) float64 {
	return linear_vertical.PredictLocalPartTag(thetas, standardizedInput)
}

// LinRegVLDeStandardizeOutput 逆标准化得到最终预测结果
// - ybar 目标特征对应的样本均值
// - sigma 目标特征对应的样本标准差
// - output 标准化样本的预测结值
func (xcc *XchainCryptoClient) LinRegVLDeStandardizeOutput(ybar, sigma, output float64) float64 {
	return linear_vertical.DeStandardizeOutput(ybar, sigma, output)
}

// --- 联邦学习-多元线性回归-纵向 end ---

// --- 联邦学习-多元逻辑回归-纵向 start ---

// LogRegVLStandardizeDataSet 标准化样本数据，除目标特征，其余特征对应的样本数值均值变为0，标准差变为1
// - sourceDataSet 原始样本数据
// - label 目标特征
func (xcc *XchainCryptoClient) LogRegVLStandardizeDataSet(sourceDataSet *ml_common.DataSet, label string) *ml_common.StandardizedDataSet {
	return logic_vertical.StandardizeDataSet(sourceDataSet, label)
}

// LogRegVLPreProcessDataSet 非标签方预处理样本数据
func (xcc *XchainCryptoClient) LogRegVLPreProcessDataSet(sourceDataSet *ml_common.StandardizedDataSet) *ml_common.TrainDataSet {
	return logic_vertical.PreProcessDataSetNoTag(sourceDataSet)
}

// LogRegVLPreProcessDataSetTagPart 标签方预处理样本数据
func (xcc *XchainCryptoClient) LogRegVLPreProcessDataSetTagPart(sourceDataSet *ml_common.StandardizedDataSet, label string) *ml_common.TrainDataSet {
	return logic_vertical.PreProcessDataSet(sourceDataSet, label)
}

// LogRegVLCalLocalGradAndCost 非标签方计算本地的梯度和损失数据
// - thetas 上一轮训练得到的模型参数
// - trainSet 预处理过的训练数据
// - accuracy 同态加解密精确到小数点后的位数
// - regMode 正则模式
// - regParam 正则参数
// - publicKey 非标签方同态公钥
func (xcc *XchainCryptoClient) LogRegVLCalLocalGradAndCost(thetas []float64, trainSet [][]float64, accuracy int, regMode int, regParam float64, publicKey *paillier.PublicKey) (*logic_vertical.LocalGradAndCostPart, error) {
	return logic_vertical.CalLocalGradAndCostPart(thetas, trainSet, accuracy, regMode, regParam, publicKey)
}

// LogRegVLCalLocalGradAndCostTagPart 标签方计算本地的梯度和损失数据
// - thetas 上一轮训练得到的模型参数
// - trainSet 预处理过的训练数据
// - accuracy 同态加解密精确到小数点后的位数
// - regMode 正则模式
// - regParam 正则参数
// - publicKey 标签方同态公钥
func (xcc *XchainCryptoClient) LogRegVLCalLocalGradAndCostTagPart(thetas []float64, trainSet [][]float64, accuracy int, regMode int, regParam float64, publicKey *paillier.PublicKey) (*logic_vertical.LocalGradAndCostPart, error) {
	return logic_vertical.CalLocalGradAndCostTagPart(thetas, trainSet, accuracy, regMode, regParam, publicKey)
}

// LogRegVLCalEncGradient 非标签方计算加密的梯度，用其他参与方的同态公钥加密
// - localPart 非标签方本地的明文梯度数据
// - tagPart 标签方的加密梯度数据
// - trainSet 非标签方训练样本集合
// - featureIndex 指定特征的索引
// - accuracy 同态加解密精度
// - publicKey 标签方同态公钥
func (xcc *XchainCryptoClient) LogRegVLCalEncGradient(localPart *logic_vertical.RawLocalGradAndCostPart, tagPart *logic_vertical.EncLocalGradAndCostPart, trainSet [][]float64, featureIndex, accuracy int, publicKey *paillier.PublicKey) (*ml_common.EncLocalGradient, error) {
	return logic_vertical.CalEncLocalGradient(localPart, tagPart, trainSet, featureIndex, accuracy, publicKey)
}

// LogRegVLCalEncGradientTagPart 标签方计算加密的梯度，用其他参与方的同态公钥加密
// - localPart 标签方本地的明文梯度数据
// - otherPart 非标签方的加密梯度数据
// - trainSet 标签方训练样本集合
// - featureIndex 指定特征的索引
// - accuracy 同态加解密精度
// - publicKey 非标签方同态公钥
func (xcc *XchainCryptoClient) LogRegVLCalEncGradientTagPart(localPart *logic_vertical.RawLocalGradAndCostPart, otherPart *logic_vertical.EncLocalGradAndCostPart, trainSet [][]float64, featureIndex, accuracy int, publicKey *paillier.PublicKey) (*ml_common.EncLocalGradient, error) {
	return logic_vertical.CalEncLocalGradientTagPart(localPart, otherPart, trainSet, featureIndex, accuracy, publicKey)
}

// LogRegVLDecryptGradient 为其他方解密带噪音的梯度信息
// - encGradMap 加密的梯度信息
// - privateKey 己方同态私钥
func (xcc *XchainCryptoClient) LogRegVLDecryptGradient(encGradMap map[int]*big.Int, privateKey *paillier.PrivateKey) map[int]*big.Int {
	return logic_vertical.DecryptGradient(encGradMap, privateKey)
}

// LogRegVLRetrieveRealGradient 还原真实的梯度信息
// - decGradMap 解密的梯度信息
// - accuracy 同态加解密精度
// - randomInt 己方梯度的噪音值
func (xcc *XchainCryptoClient) LogRegVLRetrieveRealGradient(decGradMap map[int]*big.Int, accuracy int, randomInt *big.Int) map[int]float64 {
	return logic_vertical.RetrieveRealGradient(decGradMap, accuracy, randomInt)
}

// LogRegVLCalGradient 根据明文梯度信息获取梯度值
func (xcc *XchainCryptoClient) LogRegVLCalGradient(gradMap map[int]float64) float64 {
	return logic_vertical.CalGradient(gradMap)
}

// LogRegVLEvaluateEncCost 非标签方计算加密的损失，用其他参与方的同态公钥加密
// - localPart 本地的明文损失数据
// - tagPart 标签方的加密损失数据
// - trainSet 非标签方训练样本集合
// - accuracy 同态加解密精度
// - publicKey 标签方同态公钥
func (xcc *XchainCryptoClient) LogRegVLEvaluateEncCost(localPart *logic_vertical.RawLocalGradAndCostPart, tagPart *logic_vertical.EncLocalGradAndCostPart, trainSet [][]float64, accuracy int, publicKey *paillier.PublicKey) (*ml_common.EncLocalCost, error) {
	return logic_vertical.EvaluateEncLocalCost(localPart, tagPart, trainSet, accuracy, publicKey)
}

// LogRegVLEvaluateEncCostTagPart 标签方计算加密的损失，用其他参与方的同态公钥加密
// - localPart 本地的明文损失数据
// - otherPart 非标签方的加密损失数据
// - trainSet 标签方训练样本集合
// - accuracy 同态加解密精度
// - publicKey 非标签方同态公钥
func (xcc *XchainCryptoClient) LogRegVLEvaluateEncCostTagPart(localPart *logic_vertical.RawLocalGradAndCostPart, otherPart *logic_vertical.EncLocalGradAndCostPart, trainSet [][]float64, accuracy int, publicKey *paillier.PublicKey) (*ml_common.EncLocalCost, error) {
	return logic_vertical.EvaluateEncLocalCostTag(localPart, otherPart, trainSet, accuracy, publicKey)
}

// LogRegVLDecryptCost 为其他方解密带噪音的损失信息
// - encCostMap 加密的损失信息
// - privateKey 己方同态私钥
func (xcc *XchainCryptoClient) LogRegVLDecryptCost(encCostMap map[int]*big.Int, privateKey *paillier.PrivateKey) map[int]*big.Int {
	return logic_vertical.DecryptCost(encCostMap, privateKey)
}

// LogRegVLRetrieveRealCost 还原真实的损失信息
// - decCostMap 解密的损失信息
// - accuracy 同态加解密精度
// - randomInt 损失的噪音值
func (xcc *XchainCryptoClient) LogRegVLRetrieveRealCost(decCostMap map[int]*big.Int, accuracy int, randomInt *big.Int) map[int]float64 {
	return logic_vertical.RetrieveRealCost(decCostMap, accuracy, randomInt)
}

// LogRegVLCalCost 根据明文损失信息获取损失值
func (xcc *XchainCryptoClient) LogRegVLCalCost(costMap map[int]float64) float64 {
	return logic_vertical.CalCost(costMap)
}

// LogRegVLStandardizeLocalInput 标准化样本数据
// - xbars 特征对应的均值map
// - sigmas 特征对应的方差map
// - input 特征对应的样本值
func (xcc *XchainCryptoClient) LogRegVLStandardizeLocalInput(xbars, sigmas, input map[string]float64) map[string]float64 {
	return logic_vertical.StandardizeLocalInput(xbars, sigmas, input)
}

// LogRegVLPredictLocalPart 非标签方计算本地预测值
// - thetas 训练得到的模型
// - standardizedInput 标准化后的样本数据
func (xcc *XchainCryptoClient) LogRegVLPredictLocalPart(thetas, standardizedInput map[string]float64) float64 {
	return logic_vertical.PredictLocalPartNoTag(thetas, standardizedInput)
}

// LogRegVLPredictLocalTagPart 标签方计算本地预测值
// - thetas 训练得到的模型
// - standardizedInput 标准化后的样本数据
func (xcc *XchainCryptoClient) LogRegVLPredictLocalTagPart(thetas, standardizedInput map[string]float64) float64 {
	return logic_vertical.PredictLocalPartTag(thetas, standardizedInput)
}

// --- 联邦学习-多元逻辑回归-纵向 end ---
