package index

import (
	"fmt"

	"mydb-go/internal/storage"
)

func (t *PersistentBTree) Validate() error {
	root, err := t.Root()
	if err != nil {
		return err
	}

	if root.Type == LeafNode {
		if len(root.ChildrenPageID) != 0 {
			return fmt.Errorf("leaf root has children")
		}

		if len(root.Keys) != len(root.Values) {
			return fmt.Errorf(
				"leaf root has %d keys but %d values",
				len(root.Keys),
				len(root.Values),
			)
		}

		if err := validateSortedKeys(root); err != nil {
			return err
		}

		return nil
	}

	if len(root.ChildrenPageID) != len(root.Keys)+1 {
		return fmt.Errorf(
			"root has %d keys but %d children",
			len(root.Keys),
			len(root.ChildrenPageID),
		)
	}

	if err := validateSortedKeys(root); err != nil {
		return err
	}

	leafDepth := -1

	if err := t.validateNode(
		root,
		0,
		&leafDepth,
	); err != nil {
		return err
	}

	if err := t.validateLeafChain(); err != nil {
		return err
	}

	return nil
}

func (t *PersistentBTree) validateNode(
	node *Node,
	depth int,
	leafDepth *int,
) error {
	if node.Type == LeafNode {
		if len(node.Keys) != len(node.Values) {
			return fmt.Errorf(
				"leaf page %d has %d keys but %d values",
				node.PageID,
				len(node.Keys),
				len(node.Values),
			)
		}

		if len(node.Keys) > MaxKeys {
			return fmt.Errorf(
				"leaf page %d has too many keys: %d",
				node.PageID,
				len(node.Keys),
			)
		}

		if err := validateSortedKeys(node); err != nil {
			return fmt.Errorf(
				"leaf page %d: %w",
				node.PageID,
				err,
			)
		}

		if *leafDepth == -1 {
			*leafDepth = depth
		} else if *leafDepth != depth {
			return fmt.Errorf(
				"leaf page %d is at depth %d, expected depth %d",
				node.PageID,
				depth,
				*leafDepth,
			)
		}

		return nil
	}

	if node.Type != InternalNode {
		return fmt.Errorf(
			"page %d has invalid node type %d",
			node.PageID,
			node.Type,
		)
	}

	if len(node.ChildrenPageID) != len(node.Keys)+1 {
		return fmt.Errorf(
			"internal page %d has %d keys but %d children",
			node.PageID,
			len(node.Keys),
			len(node.ChildrenPageID),
		)
	}

	if len(node.Keys) > MaxKeys {
		return fmt.Errorf(
			"internal page %d has too many keys: %d",
			node.PageID,
			len(node.Keys),
		)
	}

	if err := validateSortedKeys(node); err != nil {
		return fmt.Errorf(
			"internal page %d: %w",
			node.PageID,
			err,
		)
	}

	for i, childPageID := range node.ChildrenPageID {
		child, err := t.store.ReadNode(childPageID)
		if err != nil {
			return fmt.Errorf(
				"failed to read child %d of page %d: %w",
				i,
				node.PageID,
				err,
			)
		}

		if err := t.validateNode(
			child,
			depth+1,
			leafDepth,
		); err != nil {
			return err
		}
	}

	return nil
}

func validateSortedKeys(node *Node) error {
	for i := 1; i < len(node.Keys); i++ {
		if node.Keys[i-1] >= node.Keys[i] {
			return fmt.Errorf(
				"keys are not strictly sorted at index %d",
				i,
			)
		}
	}

	return nil
}

func (t *PersistentBTree) validateLeafChain() error {
	root, err := t.Root()
	if err != nil {
		return err
	}

	node := root

	for node.Type == InternalNode {
		if len(node.ChildrenPageID) == 0 {
			return fmt.Errorf(
				"internal page %d has no children",
				node.PageID,
			)
		}

		node, err = t.store.ReadNode(node.ChildrenPageID[0])
		if err != nil {
			return err
		}
	}

	previousKeySet := false
	var previousKey Key

	visited := make(map[storage.PageID]bool)

	for {
		if visited[node.PageID] {
			return fmt.Errorf(
				"cycle detected in leaf chain at page %d",
				node.PageID,
			)
		}

		visited[node.PageID] = true

		if node.Type != LeafNode {
			return fmt.Errorf(
				"leaf chain reached non-leaf page %d",
				node.PageID,
			)
		}

		for _, key := range node.Keys {
			if previousKeySet && key <= previousKey {
				return fmt.Errorf(
					"leaf chain is not globally sorted",
				)
			}

			previousKey = key
			previousKeySet = true
		}

		if node.NextLeaf == 0 {
			break
		}

		node, err = t.store.ReadNode(node.NextLeaf)
		if err != nil {
			return err
		}
	}

	return nil
}
