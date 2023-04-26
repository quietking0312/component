package mtcp

import (
	"net/http"
)

type node struct {
	pattern  string
	handler  http.Handler
	children []*node
	isWild   bool
}

func (n *node) insert(pattern string, handler http.Handler, parts []string, height int) {
	if len(parts) == height {
		n.pattern = pattern
		n.handler = handler
		return
	}

	part := parts[height]
	child := n.matchChild(part)
	if child == nil {
		child = &node{isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}

	child.insert(pattern, handler, parts, height+1)
}

func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.pattern == part || child.isWild {
			return child
		}
	}
	return nil
}

type router struct {
	root *node
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler, _ := r.find(req.Method, req.URL.Path)
	if handler != nil {
		handler.ServeHTTP(w, req)
	} else {
		http.NotFound(w, req)
	}
}

func (r *router) addRoute(method string, pattern string, handler http.Handler) {
	parts := parsePattern(pattern)
	r.root.insert(pattern, handler, parts, 0)
}

func (r *router) find(method string, path string) (http.Handler, map[string]string) {
	pathSegs := parsePath(path)
	params := make(map[string]string)

	node := r.root
	for i, seg := range pathSegs {
		if node == nil {
			return nil, nil
		}

		if !node.isWild {
			child := node.matchChild(seg)
			if child == nil {
				return nil, nil
			}
			node = child
		} else {
			if node.pattern == "*" {
				break
			}

			params[node.pattern[1:]] = seg
			if i == len(pathSegs)-1 {
				handler := node.handler
				return handler, params
			}

			child := node.children[0]
			node = child
		}
	}

	handler := node.handler
	return handler, params
}

func parsePattern(pattern string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ':' || pattern[i] == '*' {
			if start < i {
				parts = append(parts, pattern[start:i])
			}
			parts = append(parts, pattern[i:i+1])
			start = i + 1
		}
	}
	if start != len(pattern) {
		parts = append(parts, pattern[start:])
	}
	return parts
}

func parsePath(path string) []string {
	segments := make([]string, 0)
	start := 0
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			segments = append(segments, path[start:i])
			start = i
		}
	}
	segments = append(segments, path[start:])
	return segments
}
