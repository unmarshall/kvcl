package util

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/unmarshall/kvcl/api"
	"github.com/unmarshall/kvcl/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReferenceNodes []corev1.Node

func (r ReferenceNodes) GetReferenceNode(instanceType string) (*corev1.Node, error) {
	filteredNodes := lo.Filter(r, func(n corev1.Node, _ int) bool {
		return GetInstanceType(n.GetLabels()) == instanceType
	})
	if len(filteredNodes) == 0 {
		return nil, fmt.Errorf("no reference node found for instance type: %s", instanceType)
	}
	return &filteredNodes[0], nil
}

func ListNodes(ctx context.Context, cl client.Client, filters ...api.NodeFilter) ([]corev1.Node, error) {
	nodes := &corev1.NodeList{}
	err := cl.List(ctx, nodes)
	if err != nil {
		return nil, err
	}
	if filters == nil {
		return nodes.Items, nil
	}
	filteredNodes := make([]corev1.Node, 0, len(nodes.Items))
	for _, n := range nodes.Items {
		if ok := evaluateNodeFilters(&n, filters); ok {
			filteredNodes = append(filteredNodes, n)
		}
	}
	return filteredNodes, nil
}

func evaluateNodeFilters(node *corev1.Node, filters []api.NodeFilter) bool {
	for _, f := range filters {
		if ok := f(node); !ok {
			return false
		}
	}
	return true
}

func GetInstanceType(labels map[string]string) string {
	return labels[common.InstanceTypeLabelKey]
}
