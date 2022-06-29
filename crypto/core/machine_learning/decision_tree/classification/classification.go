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

package classification

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/utils"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

// 基于CART的二叉分类决策树
// TODO 补充更多算法描述

/********************************
	离散值：左子节点为 是，右子节点为 否
	连续值：左子节点为 <=，右子节点为 >
********************************/

// 决策树
type CTree struct {
	Root *CTreeNode
}

// 决策树节点
type CTreeNode struct {
	DataSet     *common.DTDataSet
	FeatureName string      // 当前节点使用的特征，若为叶子节点则该值为空
	Continuous  bool        // 该特征是否为连续值
	SplitValue  interface{} // 特征的分割值，可以是离散或连续值，用来分割样本，若为叶子节点则该值为空
	Result      string      // 该分支最终的决策值，若为非叶子节点该值为空
	Left        *CTreeNode  // 节点的左子节点，若为叶子节点该值为空
	Right       *CTreeNode  // 节点的右子节点，若为叶子节点该值为空
	Depth       int         // 节点所在的分支深度，Root深度为0
	Gini        float64     // 该节点的基尼指数
}

// 停止条件涉及的参数
type StopCondition struct {
	SampleThreshold int     // 节点样本数的阈值，节点的样本数小于该值则该节点标记为叶子节点
	DepthThreshold  int     // 节点深度的阈值，到达阈值则该节点标记为叶子节点
	GiniThreshold   float64 // 基尼指数震荡阈值，若节点基尼指数和父节点基尼指数振的差值小于该值，则该节点标记为叶子节点
}

// 训练，返回一个决策树
// - dataset 训练样本集
// - contFeatures 取值为连续值的特征列表
// - label 目标特征
// - cond 分支停止条件
// - regParam 泛化参数/剪枝参数
func Train(dataset *common.DTDataSet, contFeatures []string, label string, cond StopCondition, regParam float64) (*CTree, error) {
	// 设置 rootNode，包括depth，Gini
	rootGini := calMultiDataSetsGini([]*common.DTDataSet{dataset}, label)
	rootNode := &CTreeNode{
		DataSet: dataset,
		Depth:   0,
		Gini:    rootGini,
	}

	// 如果是叶子节点，则根节点为叶子节点
	v, result := decideTerminate(rootNode, nil, label, cond)
	if v {
		rootNode.Result = result
	} else {
		// 如果根节点不是叶子节点，则进行分割
		featureName, splitValue, nodes, err := splitTree(rootNode, contFeatures, label, cond)
		if err != nil {
			return nil, err
		}
		rootNode.FeatureName = featureName
		rootNode.SplitValue = splitValue
		// 判断特征值是否为连续值
		if _, v := splitValue.(float64); v {
			rootNode.Continuous = true
		}
		rootNode.Left = nodes[0]
		rootNode.Right = nodes[1]
	}

	tree := &CTree{rootNode}
	// 如果需要泛化，则进行剪枝
	if regParam != 0 {
		tree = prune(tree, label, regParam, len(rootNode.DataSet.Features[0].Sets))
	}

	// TODO 测试用：检查tree是否合法
	if !checkTree(tree) {
		return nil, fmt.Errorf("not a valid tree")
	} else {
		fmt.Printf("\n============ Congrats! Valid Tree ============ \n")
		fmt.Printf("leafNum: %d\n", countLeafNum(tree))
		fmt.Printf("depth: %d\n\n", countTreeDepth(tree))
	}

	// 将dataset设置为空
	tree = trimDataSet(tree)

	// TODO 测试用：树结构可视化
	// treeGraph(tree)
	return tree, nil
}

// 预测，利用决策树对样本进行预测
// - dataset 预测样本集
// - tree 训练得到的模型
func Predict(dataset *common.DTDataSet, tree *CTree) (map[int]string, error) {
	result := map[int]string{}
	if !checkTree(tree) {
		return nil, fmt.Errorf("predict error, tree is not valid")
	}
	if dataset == nil || len(dataset.Features) == 0 {
		return nil, fmt.Errorf("predict error, prediction dataset is empty")
	}

	// 将特征列表转化为样本列表，寻找每个样本的所有特征值，样本id -> [feature->value]
	dataSets := make(map[int]map[string]string)
	for index, feature := range dataset.Features {
		for key, value := range feature.Sets {
			if index == 0 {
				dataSets[key] = make(map[string]string)
			}
			dataSets[key][feature.FeatureName] = value
		}
	}

	// 针对每一个样本进行预测
	for key, data := range dataSets {
		predictValue, err := predict(data, tree)
		if err != nil {
			return nil, err
		}
		result[key] = predictValue
	}
	return result, nil
}

// 对单个预测样本进行预测
func predict(data map[string]string, tree *CTree) (string, error) {
	// 如果是叶子节点，则返回分支结果
	if isLeaf(tree.Root) {
		return tree.Root.Result, nil
	}

	// 节点的特征对应的样本值
	value := data[tree.Root.FeatureName]
	var pTree *CTree
	// 如果特征是连续值，解析分割值，并对比value与分割值的大小
	if tree.Root.Continuous {
		value, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", fmt.Errorf("failed to parse string to float64, err: %v", err)
		}
		// 若 value <= 分割值，选择左侧分支，否则选择右侧分支
		if value <= tree.Root.SplitValue.(float64) {
			pTree = &CTree{tree.Root.Left}
		} else {
			pTree = &CTree{tree.Root.Right}
		}
	} else {
		// 如果特征是离散值，则判断value与分割值是否相同
		if value == tree.Root.SplitValue.(string) {
			pTree = &CTree{tree.Root.Left}
		} else {
			pTree = &CTree{tree.Root.Right}
		}
	}
	return predict(data, pTree)
}

// 将节点分割为两个树节点，返回选择的特征、分割值、左右子树
// - node 待分割树节点
// - contFeatures 连续特征列表
// - label 模型训练目标特征
// - cond 分支停止条件
func splitTree(node *CTreeNode, contFeatures []string, label string, cond StopCondition) (string, interface{}, []*CTreeNode, error) {
	// 选择Gini指数最小的特征进行样本分割，得到所选特征、分割值、左右数据子集
	featureName, splitValue, lrDataSets, err := selectFeature(node.DataSet, contFeatures, label)
	if err != nil {
		return "", nil, nil, err
	}
	fmt.Printf("==feature selected== depth: %d, feature: %s, split: %v\n", node.Depth, featureName, splitValue)

	// 根据数据子集进行左右子树的划分和计算
	var retNodes []*CTreeNode
	depth := node.Depth + 1
	for i := 0; i < len(lrDataSets); i++ {
		// 根据数据子集计算节点的字段，包括depth，Gini
		gini := calMultiDataSetsGini([]*common.DTDataSet{lrDataSets[i]}, label)
		currentNode := &CTreeNode{
			DataSet: lrDataSets[i],
			Depth:   depth,
			Gini:    gini,
		}
		// 如果是叶子节点，则结束递归
		v, result := decideTerminate(currentNode, node, label, cond)
		if v {
			currentNode.Result = result
		} else {
			// 如果不是叶子节点，则继续分割
			feature, value, nodes, err := splitTree(currentNode, contFeatures, label, cond)
			if err != nil {
				return "", nil, nil, err
			}

			// 分割后给当前节点赋值
			currentNode.FeatureName = feature
			currentNode.SplitValue = value
			// 判断特征值是否为连续值
			if _, v := value.(float64); v {
				currentNode.Continuous = true
			}
			currentNode.Left = nodes[0]
			currentNode.Right = nodes[1]
		}
		retNodes = append(retNodes, currentNode)
	}

	return featureName, splitValue, retNodes, nil
}

// 从若干特征中选择Gini指数最小的特征，返回选择的特征、分割值、分割后的左右样本子集
func selectFeature(dataset *common.DTDataSet, contFeatures []string, label string) (string, interface{}, []*common.DTDataSet, error) {
	left, right := new(common.DTDataSet), new(common.DTDataSet)
	var minGini float64 = 1
	var featureName = ""
	var splitValue interface{}
	for _, feature := range dataset.Features {
		// 过滤掉目标特征
		if feature.FeatureName == label {
			continue
		}
		// 对于每个特征，找出最佳分割点，计算Gini，以及分割后的左右样本子集
		value, gini, l, r, err := findFeatureSplitValue(dataset, contFeatures, feature.FeatureName, label)
		if err != nil {
			return "", nil, nil, err
		}
		// 选择Gini最小的特征，重新赋值特征名称、分割值、左右数据子集
		if gini < minGini {
			minGini = gini

			featureName = feature.FeatureName
			splitValue = value
			left = l
			right = r
		}
	}
	return featureName, splitValue, []*common.DTDataSet{left, right}, nil
}

// 通过某个特征，选取最佳分割点对样本进行分割，返回分割值、基尼指数、分割后的左右样本子集
// 对于离散特征和连续特征分别处理
// - dataset 待分割的样本集合
// - contFeatures 连续特征列表
// - featureName 用来分割的特征名称
// - label 模型训练的目标特征
func findFeatureSplitValue(dataset *common.DTDataSet, contFeatures []string, featureName string, label string) (interface{}, float64, *common.DTDataSet, *common.DTDataSet, error) {
	// 定义最终返回的基尼指数最小的特征、分割值、左右数据子集
	finalGini := 1.0
	var finalSplit interface{}
	finalLeft, finalRight := new(common.DTDataSet), new(common.DTDataSet)

	// 连续数值，将所有可能取值从小到大排序，每连续两个数值的均值作为分割值，计算Gini
	if utils.StringInSlice(featureName, contFeatures) {
		fmt.Printf("continuous feature: %s\n", featureName)
		// 该特征所有可能取值排序
		availableValues, err := getContAvailValues(dataset, featureName)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		for i := 0; i < len(availableValues)-1; i++ {
			// 取连续两个数值的均值作为分隔值，计算Gini
			splitValue := (availableValues[i] + availableValues[i+1]) / 2
			left, right, err := splitDataSetByFeature(dataset, featureName, splitValue, true)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			gini := calMultiDataSetsGini([]*common.DTDataSet{left, right}, label)

			// 选择Gini最小的特征，重新赋值Gini、分割值、左右数据子集
			if gini < finalGini {
				finalGini = gini
				finalSplit = splitValue
				finalLeft = left
				finalRight = right
			}
		}
	} else {
		fmt.Printf("discrete feature: %s\n", featureName)
		// 离散数值，找到所有可能的数值，每个值选择都计算一次Gini
		availableValues := getDiscAvailValues(dataset, featureName)
		for _, v := range availableValues {
			// 每个可能的特征取值计算Gini
			left, right, _ := splitDataSetByFeature(dataset, featureName, v, false)
			gini := calMultiDataSetsGini([]*common.DTDataSet{left, right}, label)

			// 选择Gini最小的特征，重新赋值Gini、分割值、左右数据子集
			if gini < finalGini {
				finalGini = gini
				finalSplit = v
				finalLeft = left
				finalRight = right
			}
		}
	}

	return finalSplit, finalGini, finalLeft, finalRight, nil
}

// 利用特征值，将样本分割为两部分，左子节点对应 是/<=，右子节点对应 否/>
// - dataset 待分割的样本集合
// - featureName 用来分割的特征名称
// - splitValue 分割值
// - isContinuous 分割值是否为连续数值
func splitDataSetByFeature(dataset *common.DTDataSet, featureName, splitValue interface{}, isContinuous bool) (*common.DTDataSet, *common.DTDataSet, error) {
	leftSetMap := make(map[int]bool)
	for _, feature := range dataset.Features {
		if feature.FeatureName == featureName {
			for id, v := range feature.Sets {
				if isContinuous {
					value, err := strconv.ParseFloat(v, 64)
					if err != nil {
						return nil, nil, fmt.Errorf("failed to parse string to float64, err: %v", err)
					}
					if value <= splitValue.(float64) {
						leftSetMap[id] = true
					}
				} else {
					if v == splitValue.(string) {
						leftSetMap[id] = true
					}
				}
			}
		}
	}

	// 如果都为 是/<=，则左节点为数据集全集，右节点为空集
	if len(leftSetMap) == len(dataset.Features[0].Sets) {
		return dataset, nil, nil
	}
	// 如果都为 否/>，则左节点为空集，右节点为数据集全集
	if len(leftSetMap) == 0 {
		return nil, dataset, nil
	}

	// 标记为 是/<= 的样本放到左节点， 标记为 否/> 的样本放到右节点
	var leftFeatures, rightFeatures []*common.DTDataFeature
	for _, feature := range dataset.Features {
		leftFeature, rightFeature := new(common.DTDataFeature), new(common.DTDataFeature)
		leftFeature.Sets = make(map[int]string)
		rightFeature.Sets = make(map[int]string)
		// 对每个特征，特征名称相同，但样本集合不同
		leftFeature.FeatureName = feature.FeatureName
		rightFeature.FeatureName = feature.FeatureName
		for id, value := range feature.Sets {
			if leftSetMap[id] {
				leftFeature.Sets[id] = value
			} else {
				rightFeature.Sets[id] = value
			}
		}
		leftFeatures = append(leftFeatures, leftFeature)
		rightFeatures = append(rightFeatures, rightFeature)
	}

	leftDataset := &common.DTDataSet{
		Features: leftFeatures,
	}
	rightDataSet := &common.DTDataSet{
		Features: rightFeatures,
	}
	return leftDataset, rightDataSet, nil
}

// 对于某个离散特征，获取训练样本中所有可能的取值
func getDiscAvailValues(dataset *common.DTDataSet, featureName string) []string {
	valueMap := make(map[string]bool)
	for _, feature := range dataset.Features {
		// 找到指定特征出现在样本中的所有取值
		if feature.FeatureName == featureName {
			for _, v := range feature.Sets {
				valueMap[v] = true
			}
		}
	}
	var values []string
	for key := range valueMap {
		values = append(values, key)
	}
	return values
}

// 对于某个连续特征，获取训练样本中所有可能的取值，并按照从小到大的顺序排列
func getContAvailValues(dataset *common.DTDataSet, featureName string) ([]float64, error) {
	valueMap := make(map[float64]bool)
	for _, feature := range dataset.Features {
		if feature.FeatureName == featureName {
			// 找到指定特征出现在样本中的所有取值
			for _, v := range feature.Sets {
				value, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse string to float64, err: %v", err)
				}
				valueMap[value] = true
			}
		}
	}
	var values []float64
	for key := range valueMap {
		values = append(values, key)
	}
	// 从小到大排序
	sort.Float64s(values)
	return values, nil
}

// 计算节点指定标签的基尼指数，样本集合数可以为1
// 输入为[[D1][D2]] 或 [D], 计算样本子集的基尼系数
func calMultiDataSetsGini(datasets []*common.DTDataSet, label string) float64 {
	gini := 0.0
	// 特征集合总数
	dataSetsTotal := 0
	// 每个数据子集的基尼值
	dataSliceGinis := make(map[int]float64)
	// 每个数据子集的样本数
	dataSliceTotal := make(map[int]int)
	for index, dataSet := range datasets {
		if dataSet == nil {
			continue
		}
		for _, feature := range dataSet.Features {
			if feature.FeatureName != label {
				continue
			}
			// 计算每一个集合的基尼值, 即Gini(Di)
			dataSliceGini, dataSliceSetsNum := calSingleDataSetsGini(feature)
			// 计算特征集合总数, 即|D|
			dataSetsTotal += dataSliceSetsNum
			dataSliceGinis[index] = dataSliceGini
			dataSliceTotal[index] = dataSliceSetsNum
		}
	}
	// 计算基尼指数, 将 |Di|/|D| * Gini(Di) 求和
	for index, value := range dataSliceTotal {
		gini += (float64(value) / float64(dataSetsTotal)) * dataSliceGinis[index]
	}
	return gini
}

// 计算单个样本集基尼值
func calSingleDataSetsGini(feature *common.DTDataFeature) (float64, int) {
	imp := 0.0
	// 计算指定特征值总数
	total := len(feature.Sets)
	// 计算该特征的分类总数
	counts := uniqueDataSetTypes(feature.Sets)
	for _, value := range counts {
		imp += math.Pow(float64(value)/float64(total), 2)
	}
	return 1 - imp, total
}

// 将输入的数据汇总(input dataSet)
// return Set{type1:type1Count,type2:type2Count ... typeN:typeNCount}
func uniqueDataSetTypes(sets map[int]string) map[string]int {
	results := make(map[string]int)
	for _, value := range sets {
		if _, ok := results[value]; !ok {
			results[value] = 0
		}
		results[value] += 1
	}
	return results
}

// 当前样本的标签分类是否是只有一种分类
// 如果为true 返回指定labelType, 和 分类数 Set{type1:type1Count}
func isSameLabelType(dataset *common.DTDataSet, label string) (string, bool, map[string]int) {
	result := make(map[string]int)
	for _, value := range dataset.Features {
		if value.FeatureName == label {
			result = uniqueDataSetTypes(value.Sets)
		}
	}
	if len(result) == 1 {
		labelType := ""
		for key := range result {
			labelType = key
		}
		return labelType, true, result
	}
	return "", false, result
}

// 获取指定样本集中取值最多的类, 如果最大值有多个，则随机选一个
func getMaxLabelType(dataset *common.DTDataSet, label string) string {
	labelType, isSame, labelCounts := isSameLabelType(dataset, label)
	// 如果只有一个分类，直接返回该类别
	if isSame {
		return labelType
	}

	maxSetsNum := 0
	result := ""
	// 计算类别对应样本数最多的值, 如果最大值有多个，则label value取值最后一个对应的type
	for labelType, num := range labelCounts {
		if num >= maxSetsNum {
			maxSetsNum = num
			result = labelType
		}
	}
	return result
}

// 判断是否停止分支，若是，返回该分支最终结果
func decideTerminate(currentNode, fatherNode *CTreeNode, label string, cond StopCondition) (bool, string) {
	// 1.当前节点包含的样本集合为空, 当前节点标记为叶子节点，类别设置为其父节点所含样本最多的类别
	if currentNode.DataSet == nil || len(currentNode.DataSet.Features) == 0 {
		// 类别设置为其父节点所含样本最多的类别
		fmt.Printf("==terminate== current feature is nil\n")
		return true, getMaxLabelType(fatherNode.DataSet, label)
	}

	// 2.当前样本集属于同一类别，无需划分
	if labelType, isSame, _ := isSameLabelType(currentNode.DataSet, label); isSame {
		fmt.Printf("==terminate== current data has same type: %s\n", labelType)
		return true, labelType
	}

	// 3.判断 stopCondition...
	// 节点样本数小于阈值，则该节点标记为叶子节点, 类别设置为该节点所含样本最多的类别
	if len(currentNode.DataSet.Features[0].Sets) <= cond.SampleThreshold {
		fmt.Printf("==terminate== current data has fewer number than threshold: %d\n", len(currentNode.DataSet.Features[0].Sets))
		return true, getMaxLabelType(currentNode.DataSet, label)
	}
	// 节点深度小于阈值，则该节点标记为叶子节点, 类别设置为该节点所含样本最多的类别
	if cond.DepthThreshold != 0 && currentNode.Depth >= cond.DepthThreshold {
		fmt.Printf("==terminate== current node depth greater than threshold: %d\n", currentNode.Depth)
		return true, getMaxLabelType(currentNode.DataSet, label)
	}
	// 节点基尼指数和父节点基尼指数振的差值小于阈值，则该节点标记为叶子节点, 类别设置为该节点所含样本最多的类别
	if fatherNode != nil && math.Abs(fatherNode.Gini-currentNode.Gini) <= cond.GiniThreshold {
		fmt.Printf("==terminate== node gini amplitude smaller than threshold, current: %f, father: %f\n", currentNode.Gini, fatherNode.Gini)
		return true, getMaxLabelType(currentNode.DataSet, label)
	} else if fatherNode != nil {
		fmt.Printf("==Gini== current: %f, father: %f, difference: %f\n", currentNode.Gini, fatherNode.Gini, math.Abs(fatherNode.Gini-currentNode.Gini))
	}

	return false, ""
}

// 后剪枝 - 通过计算剪枝前后的代价函数，判断是否剪枝
func prune(tree *CTree, label string, regParam float64, allSamplesNum int) *CTree {
	//  找到目标剪枝节点
	targetNodes := findTargetNodesToPrune(tree)

	// 计算剪枝前的cost
	oldCost := calculateCost(tree, label, regParam, allSamplesNum)

	// 查看每个目标节点，是否可以剪枝
	for _, node := range targetNodes {
		// 剪枝并得到新的tree
		newTree := pruneNode(tree, node, label)

		// 计算剪枝后的cost
		newCost := calculateCost(newTree, label, regParam, allSamplesNum)
		fmt.Printf("----- old cost: %f -----\n", oldCost)
		fmt.Printf("----- new cost: %f -----\n", newCost)

		if newCost <= oldCost {
			fmt.Printf("----- node pruned -----\n")
			return prune(newTree, label, regParam, allSamplesNum)
		}
	}
	return tree
}

// 找到一个树的目标剪枝节点，该节点为包含两个叶子节点的中间节点
/* 例：node2 为目标剪枝节点
              node1
		  /		     \
		node2		leaf
      /      \
	leaf    leaf

*/
func findTargetNodesToPrune(tree *CTree) []*CTreeNode {
	if isLeaf(tree.Root) {
		return []*CTreeNode{}
	}

	if isLeaf(tree.Root.Left) && isLeaf(tree.Root.Right) {
		return []*CTreeNode{tree.Root}
	}

	leftTarget := findTargetNodesToPrune(&CTree{tree.Root.Left})
	rightTarget := findTargetNodesToPrune(&CTree{tree.Root.Right})
	return append(leftTarget, rightTarget...)
}

// 剪掉一个节点两个叶子，得到新的tree，该节点变更为叶子节点，result为该节点样本中数量最多的类
/* 例：将node2剪掉
              node1 					  node1
		  /		     \					/       \
		node2		leaf     ->		  leaf     leaf
      /      \
	leaf    leaf

*/
// 注：剪枝时不能改变原始树tree的值
func pruneNode(tree *CTree, targetNode *CTreeNode, label string) *CTree {
	if isLeaf(tree.Root) {
		return tree
	}

	// 如果根节点就是目标节点，则返回单节点子树
	// 如果两个节点相同，则dataset一定相同
	if reflect.DeepEqual(tree.Root.DataSet, targetNode.DataSet) {
		result := getMaxLabelType(tree.Root.DataSet, label)
		root := &CTreeNode{
			DataSet: tree.Root.DataSet,
			Result:  result,
			Depth:   tree.Root.Depth,
			Gini:    tree.Root.Gini,
		}
		return &CTree{root}
	}

	// 从左右子树中找到目标节点并剪掉
	newRoot := &CTreeNode{
		DataSet:     tree.Root.DataSet,
		FeatureName: tree.Root.FeatureName,
		Continuous:  tree.Root.Continuous,
		SplitValue:  tree.Root.SplitValue,
		Result:      tree.Root.Result,
		Depth:       tree.Root.Depth,
		Gini:        tree.Root.Gini,
	}
	left := pruneNode(&CTree{tree.Root.Left}, targetNode, label)
	right := pruneNode(&CTree{tree.Root.Right}, targetNode, label)
	newRoot.Left = left.Root
	newRoot.Right = right.Root
	return &CTree{newRoot}
}

// 计算一个决策树的代价，cost = sum(每个叶子节点错误率*样本数占全部样本比例) + regParam*叶子数
func calculateCost(tree *CTree, label string, regParam float64, allSamplesNum int) float64 {
	cost := calculateLeafCost(tree, label, allSamplesNum)
	cost += regParam * float64(countLeafNum(tree))
	return cost
}

// CART算法中定义的代价函数，cost = sum(每个叶子节点错误率 * 叶子节点样本数占全部样本比例)
func calculateLeafCost(tree *CTree, label string, allSamplesNum int) float64 {
	if isLeaf(tree.Root) {
		// 计算错误率
		errNum := 0
		leafSampleTotal := len(tree.Root.DataSet.Features[0].Sets)
		for _, f := range tree.Root.DataSet.Features {
			if f.FeatureName == label {
				// 统计预测错误的样本数
				for _, v := range f.Sets {
					if v != tree.Root.Result {
						errNum++
					}
				}
			}
		}
		errRate := float64(errNum) / float64(leafSampleTotal)

		// 叶子节点样本数/全部训练样本数
		sampleRate := float64(leafSampleTotal) / float64(allSamplesNum)
		// 叶子节点错误率 * 叶子节点样本数占全部样本比例
		return errRate * sampleRate
	}
	return calculateLeafCost(&CTree{tree.Root.Left}, label, allSamplesNum) + calculateLeafCost(&CTree{tree.Root.Right}, label, allSamplesNum)
}

// 基于经验熵的代价函数，cost = - sum_叶子节点( sum_叶子节点类别 ( 每个类别样本数Nk * log(Nk/该叶子样本总数)) )
func calculateLeafEntropyCost(tree *CTree, label string, allSamplesNum int) float64 {
	if isLeaf(tree.Root) {
		// 找到该节点样本所有可能取值和对应的样本数
		classMap := make(map[string]int)
		for _, v := range tree.Root.DataSet.Features {
			if v.FeatureName == label {
				classMap = uniqueDataSetTypes(v.Sets)
			}
		}

		leafSampleTotal := len(tree.Root.DataSet.Features[0].Sets)
		sum := 0.0
		for _, v := range classMap {
			// log(Nk/N)
			ent := math.Log(float64(v) / float64(leafSampleTotal))
			// sum += -[Nk * log(Nk/N)]
			sum -= float64(v) * ent
		}
		return sum
	}

	return calculateLeafEntropyCost(&CTree{tree.Root.Left}, label, allSamplesNum) + calculateLeafEntropyCost(&CTree{tree.Root.Right}, label, allSamplesNum)
}

// 计算一个决策树的叶子节点总数
func countLeafNum(tree *CTree) int {
	if isLeaf(tree.Root) {
		return 1
	}
	return countLeafNum(&CTree{tree.Root.Left}) + countLeafNum(&CTree{tree.Root.Right})
}

// 计算一个决策树的深度
func countTreeDepth(tree *CTree) int {
	if isLeaf(tree.Root) {
		return tree.Root.Depth
	}
	left := countTreeDepth(&CTree{tree.Root.Left})
	right := countTreeDepth(&CTree{tree.Root.Right})
	if left > right {
		return left
	}
	return right
}

// 判断一个节点是否为叶子节点
func isLeaf(node *CTreeNode) bool {
	if node.Left == nil && node.Right == nil {
		return true
	}
	return false
}

// 将tree中的数据集设置为空，得到最终的模型
func trimDataSet(tree *CTree) *CTree {
	tree.Root.DataSet = nil
	if isLeaf(tree.Root) {
		return tree
	}

	left := trimDataSet(&CTree{tree.Root.Left})
	right := trimDataSet(&CTree{tree.Root.Right})
	tree.Root.Left = left.Root
	tree.Root.Right = right.Root
	return tree
}

// 测试用，判断一个树是否为规范的二叉决策树
func checkTree(tree *CTree) bool {
	// 判断是否为叶子节点
	if tree.Root.Left == nil || tree.Root.Right == nil {
		return tree.Root.Left == nil && tree.Root.Right == nil && len(tree.Root.Result) != 0
	}

	// 除叶子节点外，每个节点FeatureName、SplitValue不为空
	if len(tree.Root.FeatureName) == 0 || tree.Root.SplitValue == nil {
		return false
	}

	// 深度每层+1
	if tree.Root.Depth+1 != tree.Root.Left.Depth || tree.Root.Depth+1 != tree.Root.Right.Depth {
		return false
	}

	leftTree := &CTree{tree.Root.Left}
	rightTree := &CTree{tree.Root.Right}

	return checkTree(leftTree) && checkTree(rightTree)
}

/* 树的可视化, return
				           ———是———>【leaf1】
		        node1[=1]《
			是	           ———否———>【leaf2】
root[=1]《
						                    ———是———>【leaf3】
				                node3[=3]《
			否			    是              ———否———>【leaf4】
		        node2[=2]《
				            ————否——>【leaf5】
*/
func treeGraph(tree *CTree) {
	// depth := countTreeDepth(tree)
	if isLeaf(tree.Root) {
		fmt.Printf("%v\n", tree.Root.Result)
		return
	}
	treeStringify(tree.Root, 0)
}

func treeStringify(node *CTreeNode, level int) {
	if node == nil {
		return
	}

	format := ""
	for i := 0; i < level; i++ {
		format += "\t\t" // 根据节点的深度决定缩进长度
	}
	if node.SplitValue != nil {
		if node.Continuous {
			format += fmt.Sprintf("%v[<=%v]《", node.FeatureName, fmt.Sprintf("%.2f", node.SplitValue))
		} else {
			format += fmt.Sprintf("%v[=%v]《", node.FeatureName, node.SplitValue)
		}
	} else {
		format += fmt.Sprintf("——————>【%v】", node.Result)
	}

	level++
	// 先递归打印左子树
	treeStringify(node.Left, level)
	fmt.Printf(format + "\n")
	// 再递归打印右子树
	treeStringify(node.Right, level)
}
