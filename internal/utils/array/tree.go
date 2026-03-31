package array

type TreeNode interface {
	ID() int64
	ParentID() int64
	AppendChildren(interface{})
	Lists() []TreeNode
}

func ToTree(array []TreeNode) TreeNode {
	maxLen := len(array)
	var rootNode TreeNode = nil
	///<找出根节点,根节点的特点，没有父节点
	for i := 0; i < maxLen; i++ {
		///< 统计每个节点的父节点出现的次数，父节点出现0次就是根节点
		count := 0
		for j := 0; j < maxLen; j++ {
			///< 如果有节点的ID == i的parentID 那么j就是父节点
			if array[j].ID() == array[i].ParentID() {
				count++
				array[j].AppendChildren(array[i])
			}
		}
		if count == 0 {
			rootNode = array[i]
		}
	}
	return rootNode
}

func ToMulTree(array []TreeNode) []TreeNode {
	maxLen := len(array)
	var tree []TreeNode

	// 第一层
	for i := 0; i < maxLen; i++ {
		pid := array[i].ParentID()
		if pid == 0 {
			tree = append(tree, array[i])
		}
	}
	for j := 0; j < maxLen; j++ {
		node := array[j]
		pid := array[j].ParentID()
		if pid <= 0 {
			continue
		}
		for i := 0; i < len(tree); i++ {
			id := tree[i].ID()
			if id == pid {
				tree[i].AppendChildren(node)
			} else {
				checkLists(tree[i].Lists(), node)
			}
		}
	}

	return tree
}

func checkLists(lists []TreeNode, node TreeNode) {
	for i := 0; i < len(lists); i++ {
		a := lists[i]
		if a.ID() == node.ParentID() {
			a.AppendChildren(node)
		} else {
			checkLists(a.Lists(), node)
		}
	}
}
