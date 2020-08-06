package prom

import (
	"fmt"

	"github.com/prometheus/prometheus/promql/parser"
	"github.com/rancher/prometheus-auth/pkg/data"
)

func ModifyExpression(originalExpr parser.Expr, namespaceSet data.Set) (modifiedExpr string) {
	parser.Inspect(originalExpr, func(node parser.Node, _ []parser.Node) error {
		switch n := node.(type) {
		case *parser.VectorSelector:
			n.LabelMatchers = FilterMatchers(namespaceSet, n.LabelMatchers)
		case *parser.MatrixSelector:
			vs, ok := n.VectorSelector.(*parser.VectorSelector)
			if !ok {
				return fmt.Errorf("cannot parse MatrixSelector to VectorSelector")
			}
			vs.LabelMatchers = FilterMatchers(namespaceSet, vs.LabelMatchers)
		}
		return nil
	})

	return originalExpr.String()
}
