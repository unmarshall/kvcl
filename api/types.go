package api

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

// ControlPlane represents an in-memory control plane with limited components.
// It comprises kube-api-server, etcd and kube-scheduler and will be used for
// running simulations and making scaling recommendations.
type ControlPlane interface {
	// Start starts an in-memory controlPlane comprising:
	// 1. kube-api-server and etcd taking the binary from the vClusterBinaryAssetsPath.
	// 2. kube-scheduler.
	Start(ctx context.Context) error
	// Stop will stop the in-memory controlPlane.
	Stop() error
	// FactoryReset will reset the in-memory controlPlane to its initial state.
	FactoryReset(ctx context.Context) error
	// NodeControl returns the NodeControl for the in-memory controlPlane. Should only be called after Start.
	NodeControl() NodeControl
	// PodControl returns the PodControl for the in-memory controlPlane. Should only be called after Start.
	PodControl() PodControl
	// EventControl returns the EventControl for the in-memory controlPlane. Should only be called after Start.
	EventControl() EventControl
	// Client returns the client used to connect to the in-memory controlPlane.
	Client() client.Client
}

// NodeFilter is a predicate that takes in a Node and returns the predicate result as a boolean.
type NodeFilter func(node *corev1.Node) bool

type NodeControl interface {
	// CreateNodes creates new nodes in the in-memory controlPlane from the given node specs.
	CreateNodes(ctx context.Context, nodes ...*corev1.Node) error
	// GetNode return the node matching object key
	GetNode(ctx context.Context, objectKey types.NamespacedName) (*corev1.Node, error)
	// ListNodes returns the current nodes of the in-memory controlPlane.
	ListNodes(ctx context.Context, filters ...NodeFilter) ([]corev1.Node, error)
	// TaintNodes taints the given nodes with the given taint.
	TaintNodes(ctx context.Context, taint corev1.Taint, nodes ...*corev1.Node) error
	//UnTaintNodes removes the given taint from the given nodes.
	UnTaintNodes(ctx context.Context, taintKey string, nodes ...*corev1.Node) error
	// DeleteNodes deletes the nodes identified by the given names from the in-memory controlPlane.
	DeleteNodes(ctx context.Context, nodeNames ...string) error
	// DeleteAllNodes deletes all nodes from the in-memory controlPlane.
	DeleteAllNodes(ctx context.Context) error
	// DeleteNodesMatchingLabels deletes all nodes matching labels
	DeleteNodesMatchingLabels(ctx context.Context, labels map[string]string) error
	// SetNodeConditions updates the node conditions of the given nodes,
	SetNodeConditions(ctx context.Context, conditions []corev1.NodeCondition, nodeNames ...string) error
}

// NodeInfo contains relevant information about a node.
type NodeInfo struct {
	Name        string              `json:"name"`
	Labels      map[string]string   `json:"labels"`
	Taints      []corev1.Taint      `json:"taints,omitempty"`
	Allocatable corev1.ResourceList `json:"allocatable"`
	Capacity    corev1.ResourceList `json:"capacity"`
}

// PodFilter is a predicate that takes in a Pod and returns the predicate result as a boolean.
type PodFilter func(pod *corev1.Pod) bool

type PodControl interface {
	// ListPods will get all pods and apply the given filters to the pods in conjunction. If no filters are given, all pods are returned.
	ListPods(ctx context.Context, namespace string, filters ...PodFilter) ([]corev1.Pod, error)
	// ListPodsMatchingLabels lists all pods matching labels
	ListPodsMatchingLabels(ctx context.Context, labels map[string]string) ([]corev1.Pod, error)
	// GetPodsMatchingPodNames returns all pods matching the given pod names. You would use this method over ListPods
	// to reduce the load on KAPI. Get calls are cached and list calls are not. Once in-memory KAPI is
	// replaced with the fake API server then this optimization will no longer be needed.
	GetPodsMatchingPodNames(ctx context.Context, namespace string, podNames ...string) ([]*corev1.Pod, error)
	// CreatePodsAsUnscheduled creates new unscheduled pods in the in-memory controlPlane from the given schedulerName and pod specs.
	CreatePodsAsUnscheduled(ctx context.Context, schedulerName string, pods ...corev1.Pod) error
	// CreatePods creates new pods in the in-memory controlPlane.
	CreatePods(ctx context.Context, pods ...*corev1.Pod) error
	// DeletePods deletes the given pods from the in-memory controlPlane.
	DeletePods(ctx context.Context, pods ...corev1.Pod) error
	// DeleteAllPods deletes all pods from the in-memory controlPlane.
	DeleteAllPods(ctx context.Context, namespace string) error
	// DeletePodsMatchingLabels deletes all pods matching labels
	DeletePodsMatchingLabels(ctx context.Context, namespace string, labels map[string]string) error
	// DeletePodsMatchingNames deletes all pods matching pod names
	DeletePodsMatchingNames(ctx context.Context, namespace string, podNames ...string) error
}

// PodInfo contains relevant information about a pod.
type PodInfo struct {
	Name              string            `json:"name,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Spec              corev1.PodSpec    `json:"spec"`
	NominatedNodeName string            `json:"nominatedNodeName,omitempty"`
	Count             int               `json:"count"`
}

// EventFilter is a predicate that takes in an Event and returns the predicate result as a boolean.
type EventFilter func(event *corev1.Event) bool

type EventControl interface {
	ListEvents(ctx context.Context, namespace string, filters ...EventFilter) ([]corev1.Event, error)
	DeleteAllEvents(ctx context.Context, namespace string) error
	GetPodSchedulingEvents(ctx context.Context, namespace string, since time.Time, pods []*corev1.Pod, timeout time.Duration) (scheduledPodNames, unscheduledPodNames sets.Set[string], err error)
}
