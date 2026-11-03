package config

import "gopkg.in/yaml.v3"

// getRootMapping returns the first MappingNode from a node's content.
// For a DocumentNode, Content[0] is the root mapping.
func getRootMapping(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.MappingNode {
		return node
	}
	for _, child := range node.Content {
		if child != nil && child.Kind == yaml.MappingNode {
			return child
		}
	}
	return nil
}

func findChildMappingNode(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil {
		return nil
	}
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			val := mapping.Content[i+1]
			if val.Kind == yaml.MappingNode {
				return val
			}
		}
	}
	return nil
}

func keyExistsInMapping(mapping *yaml.Node, key string) bool {
	if mapping == nil {
		return false
	}
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			return true
		}
	}
	return false
}
