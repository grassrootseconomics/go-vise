package debug

import (
	"context"
	"strings"

	"github.com/grassrootseconomics/go-vise/resource"
	"github.com/grassrootseconomics/go-vise/state"
)

type NodeMap struct {
	st   *state.State
	root Node
	outs []string
}

func NewNodeMap(root string) *NodeMap {
	n := &NodeMap{
		st: state.NewState(0),
	}
	n.root.Name = root
	return n
}

func (nm *NodeMap) Run(ctx context.Context, rs resource.Resource) error {
	ph := NewNodeParseHandler(&nm.root)
	b, err := rs.GetCode(ctx, nm.root.Name)
	if err != nil {
		return err
	}

	_, err = ph.ParseAll(b)
	if err != nil {
		return err
	}
	return nm.processNode(ctx, &nm.root, rs)
}

func (nm *NodeMap) processNode(ctx context.Context, node *Node, rs resource.Resource) error {
	for i, v := range nm.st.ExecPath {
		if v == node.Name {
			logg.InfoCtxf(ctx, "loop detected", "pos", i, "node", node.Name, "path", nm.st.ExecPath)
			return nil
		}
	}
	nm.st.Down(node.Name)
	logg.DebugCtxf(ctx, "processnode", "path", nm.st.ExecPath)
	for true {
		n := node.Next()
		if n == nil {
			break
		}
		ph := NewNodeParseHandler(n)
		b, err := rs.GetCode(ctx, n.Name)
		if err != nil {
			continue
		}
		_, err = ph.ParseAll(b)
		if err != nil {
			return err
		}
		err = nm.processNode(ctx, n, rs)
		if err != nil {
			return err
		}
	}
	nm.outs = append(nm.outs, strings.Join(nm.st.ExecPath, "/"))
	nm.st.Up()
	return nil
}

func (nm *NodeMap) String() string {
	var s string
	l := len(nm.outs)
	for i := l; i > 0; i-- {
		s += nm.outs[i-1]
		s += "\n"
	}
	return s
}

func (nm *NodeMap) Apply(nodeFunc func(*Node) error) error {
	var err error
	l := len(nm.outs)
	for i := l; i > 0; i-- {
		n := NodeIndex[nm.outs[i-1]]
		err = nodeFunc(&n)
		if err != nil {
			break
		}
	}
	return err
}
