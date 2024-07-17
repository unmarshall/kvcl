package control

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	"github.com/unmarshall/kvcl/api"
	"github.com/unmarshall/kvcl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type nodeControl struct {
	client client.Client
}

func NewNodeControl(cl client.Client) api.NodeControl {
	return &nodeControl{
		client: cl,
	}
}

func (n nodeControl) CreateNodes(ctx context.Context, nodes ...*corev1.Node) error {
	var errs error
	for _, node := range nodes {
		node.ObjectMeta.ResourceVersion = ""
		node.ObjectMeta.UID = ""
		errs = errors.Join(errs, n.client.Create(ctx, node))
	}
	return errs
}

func (n nodeControl) GetNode(ctx context.Context, objectKey types.NamespacedName) (*corev1.Node, error) {
	node := corev1.Node{}
	err := n.client.Get(ctx, objectKey, &node)
	return &node, err
}

func (n nodeControl) ListNodes(ctx context.Context, filters ...api.NodeFilter) ([]corev1.Node, error) {
	return util.ListNodes(ctx, n.client, filters...)
}

func (n nodeControl) TaintNodes(ctx context.Context, taint corev1.Taint, nodes ...*corev1.Node) error {
	var errs error
	failedToPatchNodeNames := make([]string, 0, len(nodes))
	for _, node := range nodes {
		patch := client.MergeFromWithOptions(node.DeepCopy(), client.MergeFromWithOptimisticLock{})
		node.Spec.Taints = append(node.Spec.Taints, taint)
		if err := n.client.Patch(ctx, node, patch); err != nil {
			failedToPatchNodeNames = append(failedToPatchNodeNames, node.Name)
			errs = errors.Join(errs, err)
		}
	}
	if errs != nil {
		slog.Error("failed to patch one or more nodes with taint", "taint", taint, "nodes", failedToPatchNodeNames, "error", errs)
	}
	return errs
}

func (n nodeControl) UnTaintNodes(ctx context.Context, taintKey string, nodes ...*corev1.Node) error {
	var errs error
	failedToPatchNodeNames := make([]string, 0, len(nodes))
	for _, node := range nodes {
		patch := client.MergeFromWithOptions(node.DeepCopy(), client.MergeFromWithOptimisticLock{})
		var newTaints []corev1.Taint
		for _, taint := range node.Spec.Taints {
			if taint.Key != taintKey {
				newTaints = append(newTaints, taint)
			}
		}
		node.Spec.Taints = newTaints
		if err := n.client.Patch(ctx, node, patch); err != nil {
			failedToPatchNodeNames = append(failedToPatchNodeNames, node.Name)
			errs = errors.Join(errs, err)
		}
	}
	if errs != nil {
		slog.Error("failed to remove taint from nodes", "taint", taintKey, "nodes", failedToPatchNodeNames, "error", errs)
	}
	return errs
}

func (n nodeControl) DeleteNodes(ctx context.Context, nodeNames ...string) error {
	var errs error
	targetNodes, err := n.ListNodes(ctx, func(node *corev1.Node) bool {
		return slices.Contains(nodeNames, node.Name)
	})
	if err != nil {
		return err
	}
	for _, node := range targetNodes {
		errs = errors.Join(errs, n.client.Delete(ctx, &node))
	}
	return errs
}

func (n nodeControl) DeleteAllNodes(ctx context.Context) error {
	return n.client.DeleteAllOf(ctx, &corev1.Node{})
}

func (n nodeControl) DeleteNodesMatchingLabels(ctx context.Context, labels map[string]string) error {
	return n.client.DeleteAllOf(ctx, &corev1.Node{}, client.MatchingLabels(labels))
}

func CreateAndUntaintNode(ctx context.Context, nc api.NodeControl, taintKey string, nodes ...*corev1.Node) error {
	err := nc.CreateNodes(ctx, nodes...)
	if err != nil {
		return err
	}
	return nc.UnTaintNodes(ctx, taintKey, nodes...)
}
