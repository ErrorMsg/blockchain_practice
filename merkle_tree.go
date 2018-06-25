package blockchain_practice

import "crypto/sha256"

type MerkleNode struct{
	Left 		*MerkleNode
	Right 	*MerkleNode
	Data 	[]byte
}

type MerkleTree struct{
	RootNode *MerkleNode
}

func NewMerkleTree(data [][]byte) *MerkleTree{
	var nodes []*MerkleNode
	if len(data)%2 != 0{
		data = append(data, data[len(data)-1])
	}
	for _,dat := range data{
		node := NewMerkleNode(nil, nil, dat)
		nodes = append(nodes, node)
	}
	for i:=0;i<len(data)/2;i++{
		var level []*MerkleNode
		for j:=0;j<len(nodes);j+=2{
			node := NewMerkleNode(nodes[j], nodes[j+1],nil)
			level = append(level, node)
		}
		nodes = level
	}
	mTree := &MerkleTree{nodes[0]}
	return mTree
}

func NewMerkleNode(left,right *MerkleNode, data []byte) *MerkleNode{
	mNode := MerkleNode{}
	if left == nil && right == nil{
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	}else{
		preHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(preHashes)
		mNode.Data = hash[:]
	}
	mNode.Left = left
	mNode.Right = right
	return &mNode
}