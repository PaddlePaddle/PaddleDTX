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
	"testing"

	"github.com/PaddlePaddle/PaddleDTX/crypto/common/utils"
	"github.com/PaddlePaddle/PaddleDTX/crypto/core/machine_learning/common"
)

func TestCalMultiDataSetsGini(t *testing.T) {
	/*
		Sepal Length,Sepal Width,Petal Length,Petal Width,Label
		4.6,3.1,1.5,0.2,Iris-setosa
		5.4,3.9,1.7,0.4,Iris-setosa
		4.6,3.4,1.4,0.3,Iris-setosa
		5.0,3.4,1.5,0.2,Iris-setosa
		5.4,3.0,4.5,1.5,Iris-versicolor
		6.7,3.1,4.7,1.5,Iris-versicolor
	*/
	datasets1 := []*common.DTDataSet{
		{
			Features: []*common.DTDataFeature{
				{
					FeatureName: "Sepal Length",
					Sets:        map[int]string{1: "4.6", 2: "5.4", 3: "4.6", 4: "5.0", 5: "5.4", 6: "6.7"},
				},
				{
					FeatureName: "Sepal Width",
					Sets:        map[int]string{1: "3.1", 2: "3.9", 3: "3.4", 4: "3.4", 5: "3.0", 6: "3.1"},
				},
				{
					FeatureName: "Petal Length",
					Sets:        map[int]string{1: "1.5", 2: "1.7", 3: "1.4", 4: "1.5", 5: "4.5", 6: "4.7"},
				},
				{
					FeatureName: "Petal Width",
					Sets:        map[int]string{1: "0.2", 2: "0.4", 3: "0.3", 4: "0.2", 5: "1.5", 6: "1.5"},
				},
				{
					FeatureName: "Label",
					Sets:        map[int]string{1: "Iris-setosa", 2: "Iris-setosa", 3: "Iris-setosa", 4: "Iris-setosa", 5: "Iris-versicolor", 6: "Iris-versicolor"},
				},
			},
		},
	}
	// 单个数据集
	gini1 := calMultiDataSetsGini(datasets1, "Label")
	if gini1 > 1 {
		t.Errorf("calculMultiDataSetsGini error, gini: %v", gini1)
	}
	t.Logf("calculMultiDataSetsGini, gini: %v", gini1)

	// 多个分类好的数据集
	dataFeature2 := []*common.DTDataFeature{
		{
			FeatureName: "Sepal Length",
			Sets:        map[int]string{1: "4.6", 3: "4.6"},
		},
		{
			FeatureName: "Sepal Width",
			Sets:        map[int]string{1: "3.1", 3: "3.4"},
		},
		{
			FeatureName: "Petal Length",
			Sets:        map[int]string{1: "1.5", 3: "1.4"},
		},
		{
			FeatureName: "Petal Width",
			Sets:        map[int]string{1: "0.2", 3: "0.3"},
		},
		{
			FeatureName: "Label",
			Sets:        map[int]string{1: "Iris-setosa", 3: "Iris-setosa"},
		},
	}
	dataFeature3 := []*common.DTDataFeature{
		{
			FeatureName: "Sepal Length",
			Sets:        map[int]string{2: "5.4", 4: "5.0", 5: "5.4", 6: "6.7"},
		},
		{
			FeatureName: "Sepal Width",
			Sets:        map[int]string{2: "3.9", 4: "3.4", 5: "3.0", 6: "3.1"},
		},
		{
			FeatureName: "Petal Length",
			Sets:        map[int]string{2: "1.7", 4: "1.5", 5: "4.5", 6: "4.7"},
		},
		{
			FeatureName: "Petal Width",
			Sets:        map[int]string{2: "0.4", 4: "0.2", 5: "1.5", 6: "1.5"},
		},
		{
			FeatureName: "Label",
			Sets:        map[int]string{2: "Iris-setosa", 4: "Iris-setosa", 5: "Iris-versicolor", 6: "Iris-versicolor"},
		},
	}
	datasets2 := []*common.DTDataSet{
		{
			Features: dataFeature2,
		},
		{
			Features: dataFeature3,
		},
	}
	gini2 := calMultiDataSetsGini(datasets2, "Label")
	if gini2 > 1 {
		t.Errorf("calculMultiDataSetsGini error, gini: %v", gini2)
	}
	t.Logf("calculMultiDataSetsGini, gini: %v", gini2)
}

func TestDecideTerminate(t *testing.T) {
	currentNode := &CTreeNode{
		FeatureName: "node1",
		Depth:       3,
		Gini:        0.11,
	}
	fatherNode := &CTreeNode{
		FeatureName: "node1",
		SplitValue:  1,
		Left:        currentNode,
		Depth:       2,
		Gini:        0.31,
	}
	currentData := &common.DTDataSet{
		Features: []*common.DTDataFeature{
			{
				FeatureName: "Sepal Length",
				Sets:        map[int]string{1: "4.6", 3: "4.6"},
			},
			{
				FeatureName: "Sepal Width",
				Sets:        map[int]string{1: "3.1", 3: "3.4"},
			},
			{
				FeatureName: "Petal Length",
				Sets:        map[int]string{1: "1.5", 3: "1.4"},
			},
			{
				FeatureName: "Petal Width",
				Sets:        map[int]string{1: "0.2", 3: "0.3"},
			},
			{
				FeatureName: "Label",
				Sets:        map[int]string{1: "Iris-setosa", 3: "Iris-setosa"},
			},
		},
	}
	fatherData := &common.DTDataSet{
		Features: []*common.DTDataFeature{
			{
				FeatureName: "Sepal Length",
				Sets:        map[int]string{1: "4.6", 2: "5.4", 3: "4.6", 4: "5.0", 5: "5.4", 6: "6.7"},
			},
			{
				FeatureName: "Sepal Width",
				Sets:        map[int]string{1: "3.1", 2: "3.9", 3: "3.4", 4: "3.4", 5: "3.0", 6: "3.1"},
			},
			{
				FeatureName: "Petal Length",
				Sets:        map[int]string{1: "1.5", 2: "1.7", 3: "1.4", 4: "1.5", 5: "4.5", 6: "4.7"},
			},
			{
				FeatureName: "Petal Width",
				Sets:        map[int]string{1: "0.2", 2: "0.4", 3: "0.3", 4: "0.2", 5: "1.5", 6: "1.5"},
			},
			{
				FeatureName: "Label",
				Sets:        map[int]string{1: "Iris-setosa", 2: "Iris-setosa", 3: "Iris-setosa", 4: "Iris-setosa", 5: "Iris-versicolor", 6: "Iris-versicolor"},
			},
		},
	}
	currentNode.DataSet = currentData
	fatherNode.DataSet = fatherData
	stopCondition := StopCondition{
		SampleThreshold: 5,
		DepthThreshold:  5,
		GiniThreshold:   0.22,
	}
	isEnd, labelType := decideTerminate(currentNode, fatherNode, "Label", stopCondition)
	if !isEnd {
		t.Errorf("decideTerminate error, result: %v, %v", isEnd, labelType)
	}
	t.Logf("decideTerminate, result: %v, %v", isEnd, labelType)
}

func TestSplitDataSetByDiscFeature(t *testing.T) {
	dataFeatures := []*common.DTDataFeature{
		{
			FeatureName: "Sepal Length",
			Sets:        map[int]string{1: "4.6", 2: "5.4", 3: "4.6", 4: "5.0", 5: "5.4", 6: "6.7"},
		},
		{
			FeatureName: "Sepal Width",
			Sets:        map[int]string{1: "3.1", 2: "3.9", 3: "3.4", 4: "3.4", 5: "3.0", 6: "3.1"},
		},
		{
			FeatureName: "Petal Length",
			Sets:        map[int]string{1: "1.5", 2: "1.7", 3: "1.4", 4: "1.5", 5: "4.5", 6: "4.7"},
		},
		{
			FeatureName: "Petal Width",
			Sets:        map[int]string{1: "0.2", 2: "0.4", 3: "0.3", 4: "0.2", 5: "1.5", 6: "1.5"},
		},
		{
			FeatureName: "Label",
			Sets:        map[int]string{1: "Iris-setosa", 2: "Iris-setosa", 3: "Iris-setosa", 4: "Iris-setosa", 5: "Iris-versicolor", 6: "Iris-versicolor"},
		},
	}
	dataset := &common.DTDataSet{
		Features: dataFeatures,
	}

	featureName1 := "Label"
	splitValue1 := "Iris-setosa"
	l, r, _ := splitDataSetByFeature(dataset, featureName1, splitValue1, false)
	if l == nil || r == nil || len(l.Features) != 5 || len(r.Features) != 5 || len(l.Features[0].Sets) != 4 || len(r.Features[0].Sets) != 2 {
		t.Errorf("failed to split dataset by feature %s value %s, left: %v, right: %v", featureName1, splitValue1, l, r)
	}

	featureName2 := "Sepal Length"
	splitValue2 := 6.0
	l, r, err := splitDataSetByFeature(dataset, featureName2, splitValue2, true)
	if err != nil {
		t.Error(err)
	}
	if l == nil || r == nil || len(l.Features) != 5 || len(r.Features) != 5 || len(l.Features[0].Sets) != 5 || len(r.Features[0].Sets) != 1 {
		t.Errorf("failed to split dataset by feature %s value %f, left: %v, right: %v", featureName2, splitValue2, l, r)
	}
}

func TestGetAvailValues(t *testing.T) {
	dataFeatures := []*common.DTDataFeature{
		{
			FeatureName: "Sepal Length",
			Sets:        map[int]string{1: "4.6", 2: "5.4", 3: "4.6", 4: "5.0", 5: "5.4", 6: "6.7"},
		},
		{
			FeatureName: "Sepal Width",
			Sets:        map[int]string{1: "3.1", 2: "3.9", 3: "3.4", 4: "3.4", 5: "3.0", 6: "3.1"},
		},
		{
			FeatureName: "Petal Length",
			Sets:        map[int]string{1: "1.5", 2: "1.7", 3: "1.4", 4: "1.5", 5: "4.5", 6: "4.7"},
		},
		{
			FeatureName: "Petal Width",
			Sets:        map[int]string{1: "0.2", 2: "0.4", 3: "0.3", 4: "0.2", 5: "1.5", 6: "1.5"},
		},
		{
			FeatureName: "Label",
			Sets:        map[int]string{1: "Iris-setosa", 2: "Iris-setosa", 3: "Iris-setosa", 4: "Iris-setosa", 5: "Iris-versicolor", 6: "Iris-versicolor"},
		},
	}
	dataset := &common.DTDataSet{
		Features: dataFeatures,
	}
	featureName1 := "Label"
	values1 := getDiscAvailValues(dataset, featureName1)
	realValues1 := []string{"Iris-setosa", "Iris-versicolor"}
	for _, v := range realValues1 {
		if !utils.StringInSlice(v, values1) {
			t.Errorf("getDiscAvailValues error, %s not obtained", v)
		}
	}

	featureName2 := "Sepal Length"
	values2, err := getContAvailValues(dataset, featureName2)
	if err != nil {
		t.Error(err)
	}
	realValues2 := []float64{4.6, 5.4, 4.6, 5.0, 5.4, 6.7}
	for _, v := range realValues2 {
		in := false
		for _, w := range values2 {
			if v == w {
				in = true
			}
		}
		if !in {
			t.Errorf("getContAvailValues error, %f not obtained", v)
		}
	}
}

func TestPredict(t *testing.T) {
	/*
					     Sepal Length
				    / <= 4.6			   \
			Petal Width		 	     Iris-versicolor
			/ <=0.4	      \
		Iris-setosa	Iris-versicolor
	*/
	leaf1 := &CTreeNode{
		Result: "Iris-setosa",
		Depth:  2,
	}
	leaf2 := &CTreeNode{
		Result: "Iris-versicolor",
		Depth:  2,
	}
	leaf3 := &CTreeNode{
		Result: "Iris-versicolor",
		Depth:  1,
	}
	node1 := &CTreeNode{
		FeatureName: "Petal Width",
		SplitValue:  0.4,
		Continuous:  true,
		Left:        leaf1,
		Right:       leaf2,
		Depth:       1,
	}
	root := &CTreeNode{
		FeatureName: "Sepal Length",
		SplitValue:  4.6,
		Continuous:  true,
		Left:        node1,
		Right:       leaf3,
		Depth:       0,
	}
	tree := &CTree{root}
	dataset := &common.DTDataSet{
		Features: []*common.DTDataFeature{
			{
				FeatureName: "Sepal Length",
				Sets:        map[int]string{1: "4.6", 2: "4.6", 3: "5.0", 4: "5.4"},
			},
			{
				FeatureName: "Sepal Width",
				Sets:        map[int]string{1: "3.1", 2: "3.9", 3: "3.4", 4: "3.4"},
			},
			{
				FeatureName: "Petal Length",
				Sets:        map[int]string{1: "1.5", 2: "1.7", 3: "1.4", 4: "1.5"},
			},
			{
				FeatureName: "Petal Width",
				Sets:        map[int]string{1: "0.4", 2: "0.3", 3: "0.3", 4: "0.2"},
			},
			{
				FeatureName: "Label",
				Sets:        map[int]string{1: "Iris-setosa", 2: "Iris-setosa", 3: "Iris-versicolor", 4: "Iris-versicolor"},
			},
		},
	}
	result, err := Predict(dataset, tree)
	if err != nil {
		t.Errorf("predict result error: %v", err)
	}
	t.Logf("predict result, %v", result)
}

func TestCheckTree(t *testing.T) {
	/*
					 root
				/			\
			node1			node2
			/	\			/	\
		leaf1	leaf2	leaf3	leaf4
	*/
	leaf1 := &CTreeNode{
		FeatureName: "leaf1",
		Result:      "leaf1",
		Depth:       2,
	}
	leaf2 := &CTreeNode{
		FeatureName: "leaf2",
		Result:      "leaf2",
		Depth:       2,
	}
	leaf3 := &CTreeNode{
		FeatureName: "leaf3",
		Result:      "leaf3",
		Depth:       2,
	}
	leaf4 := &CTreeNode{
		FeatureName: "leaf4",
		Result:      "leaf4",
		Depth:       2,
	}
	node1 := &CTreeNode{
		FeatureName: "node1",
		SplitValue:  1,
		Left:        leaf1,
		Right:       leaf2,
		Depth:       1,
		Gini:        0.11,
	}
	node2 := &CTreeNode{
		FeatureName: "node2",
		SplitValue:  2,
		Left:        leaf3,
		Right:       leaf4,
		Depth:       1,
		Gini:        0.22,
	}
	root1 := &CTreeNode{
		FeatureName: "root",
		SplitValue:  "root",
		Left:        node1,
		Right:       node2,
		Depth:       0,
		Gini:        0.33,
	}
	tree1 := &CTree{root1}
	if !checkTree(tree1) {
		t.Errorf("check tree1 supposed to be true")
	}

	root2 := &CTreeNode{
		FeatureName: "root",
		SplitValue:  "root",
		Left:        node1,
		// no right child tree
		Depth: 0,
		Gini:  0.33,
	}
	tree2 := &CTree{root2}
	if checkTree(tree2) {
		t.Errorf("check tree2 supposed to be false")
	}

	root3 := &CTreeNode{
		FeatureName: "root",
		// no split value
		Left:  node1,
		Right: node2,
		Depth: 0,
		Gini:  0.33,
	}
	tree3 := &CTree{root3}
	if checkTree(tree3) {
		t.Errorf("check tree3 supposed to be false")
	}
}

func TestCountLeafNum(t *testing.T) {
	/*
					 root
				/			\
			node1			node2
			/	\			/	\
		leaf1	leaf2	node3	leaf5
						/	\
					  leaf3  leaf4
	*/
	leaf1 := &CTreeNode{
		FeatureName: "leaf1",
		Result:      "leaf1",
		Depth:       2,
	}
	leaf2 := &CTreeNode{
		FeatureName: "leaf2",
		Result:      "leaf2",
		Depth:       2,
	}
	leaf3 := &CTreeNode{
		FeatureName: "leaf3",
		Result:      "leaf3",
		Depth:       3,
	}
	leaf4 := &CTreeNode{
		FeatureName: "leaf4",
		Result:      "leaf4",
		Depth:       3,
	}
	leaf5 := &CTreeNode{
		FeatureName: "leaf5",
		Result:      "leaf5",
		Depth:       2,
	}
	node1 := &CTreeNode{
		FeatureName: "node1",
		SplitValue:  1,
		Left:        leaf1,
		Right:       leaf2,
		Depth:       1,
		Gini:        0.11,
	}
	node3 := &CTreeNode{
		FeatureName: "node2",
		SplitValue:  3,
		Left:        leaf3,
		Right:       leaf4,
		Depth:       2,
		Gini:        0.22,
	}
	node2 := &CTreeNode{
		FeatureName: "node2",
		SplitValue:  2,
		Left:        node3,
		Right:       leaf5,
		Depth:       1,
		Gini:        0.22,
	}
	root1 := &CTreeNode{
		FeatureName: "root",
		SplitValue:  "root",
		Left:        node1,
		Right:       node2,
		Depth:       0,
		Gini:        0.33,
	}

	tree := &CTree{root1}
	n := countLeafNum(tree)
	if n != 5 {
		t.Errorf("wrong leaf number, supposed to be: %d, got %d\n", 5, n)
	}
}

func TestCountTreeDepth(t *testing.T) {
	/*
					 root
				/			\
			node1			node2
			2/	\2			/	\
		leaf1	leaf2	node3	leaf5
						3/	\3
					  leaf3  leaf4
	*/
	leaf1 := &CTreeNode{
		FeatureName: "leaf1",
		Result:      "leaf1",
		Depth:       2,
	}
	leaf2 := &CTreeNode{
		FeatureName: "leaf2",
		Result:      "leaf2",
		Depth:       2,
	}
	leaf3 := &CTreeNode{
		FeatureName: "leaf3",
		Result:      "leaf3",
		Depth:       3,
	}
	leaf4 := &CTreeNode{
		FeatureName: "leaf4",
		Result:      "leaf4",
		Depth:       3,
	}
	leaf5 := &CTreeNode{
		FeatureName: "leaf5",
		Result:      "leaf5",
		Depth:       2,
	}
	node1 := &CTreeNode{
		FeatureName: "node1",
		SplitValue:  1,
		Left:        leaf1,
		Right:       leaf2,
		Depth:       1,
		Gini:        0.11,
	}
	node3 := &CTreeNode{
		FeatureName: "node2",
		SplitValue:  3,
		Left:        leaf3,
		Right:       leaf4,
		Depth:       2,
		Gini:        0.22,
	}
	node2 := &CTreeNode{
		FeatureName: "node2",
		SplitValue:  2,
		Left:        node3,
		Right:       leaf5,
		Depth:       1,
		Gini:        0.22,
	}
	root1 := &CTreeNode{
		FeatureName: "root",
		SplitValue:  "root",
		Left:        node1,
		Right:       node2,
		Depth:       0,
		Gini:        0.33,
	}

	tree := &CTree{root1}
	n := countTreeDepth(tree)
	if n != 3 {
		t.Errorf("count tree depth: %d, got %d\n", 3, n)
	}
}

func TestTreeGraph(t *testing.T) {
	/*
					root
				/			\
			node1			node2
			2/	\			/	\
		leaf1	leaf2	node3	leaf5
						3/	\
					  leaf3  leaf4
	*/
	leaf1 := &CTreeNode{
		FeatureName: "leaf1",
		Result:      "leaf1",
		Depth:       2,
	}
	leaf2 := &CTreeNode{
		FeatureName: "leaf2",
		Result:      "leaf2",
		Depth:       2,
	}
	leaf3 := &CTreeNode{
		FeatureName: "leaf3",
		Result:      "leaf3",
		Depth:       3,
	}
	leaf4 := &CTreeNode{
		FeatureName: "leaf4",
		Result:      "leaf4",
		Depth:       3,
	}
	leaf5 := &CTreeNode{
		FeatureName: "leaf5",
		Result:      "leaf5",
		Depth:       2,
	}
	node1 := &CTreeNode{
		FeatureName: "node1",
		SplitValue:  1,
		Left:        leaf1,
		Right:       leaf2,
		Depth:       1,
		Gini:        0.11,
	}
	node3 := &CTreeNode{
		FeatureName: "node3",
		SplitValue:  3,
		Left:        leaf3,
		Right:       leaf4,
		Depth:       2,
		Gini:        0.22,
	}
	node2 := &CTreeNode{
		FeatureName: "node2",
		SplitValue:  2,
		Left:        node3,
		Right:       leaf5,
		Depth:       1,
		Gini:        0.22,
	}
	root1 := &CTreeNode{
		FeatureName: "root",
		SplitValue:  1,
		Left:        node1,
		Right:       node2,
		Depth:       0,
		Gini:        0.33,
	}

	tree := &CTree{root1}
	treeGraph(tree)
}
