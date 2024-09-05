package control

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	"github.com/unmarshall/kvcl/api"
	"github.com/unmarshall/kvcl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type podControl struct {
	client client.Client
}

func NewPodControl(cl client.Client) api.PodControl {
	return &podControl{
		client: cl,
	}
}

func (p podControl) ListPods(ctx context.Context, namespace string, filters ...api.PodFilter) ([]corev1.Pod, error) {
	return util.ListPods(ctx, p.client, namespace, filters...)
}

func (p podControl) ListPodsMatchingLabels(ctx context.Context, labels map[string]string) ([]corev1.Pod, error) {
	podList := corev1.PodList{}
	if err := p.client.List(ctx, &podList, client.MatchingLabels(labels)); err != nil {
		slog.Error("cannot list nodes", "labels", labels, "error", err)
		return nil, err
	}
	return podList.Items, nil
}

func (p podControl) GetPodsMatchingPodNames(ctx context.Context, namespace string, podNames ...string) ([]*corev1.Pod, error) {
	pods := make([]*corev1.Pod, 0, len(podNames))
	for _, podName := range podNames {
		pod := &corev1.Pod{}
		if err := client.IgnoreNotFound(p.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: podName}, pod)); err != nil {
			return nil, err
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

func (p podControl) CreatePodsAsUnscheduled(ctx context.Context, schedulerName string, pods ...corev1.Pod) error {
	var errs error
	for _, pod := range pods {
		podObjMeta := metav1.ObjectMeta{
			Namespace:       pod.Namespace,
			OwnerReferences: pod.OwnerReferences,
			Labels:          pod.Labels,
			Annotations:     pod.Annotations,
		}
		if pod.GenerateName != "" {
			podObjMeta.GenerateName = pod.GenerateName
		} else {
			podObjMeta.Name = pod.Name
		}
		dupPod := &corev1.Pod{
			ObjectMeta: podObjMeta,
			Spec:       pod.Spec,
		}
		dupPod.Spec.NodeName = ""
		dupPod.Spec.SchedulerName = schedulerName
		dupPod.Spec.TerminationGracePeriodSeconds = ptr.To(int64(0))
		if err := p.client.Create(ctx, dupPod); err != nil {
			slog.Error("failed to create pod in virtual controlPlane", "pod", client.ObjectKeyFromObject(dupPod), "error", err)
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func (p podControl) CreatePods(ctx context.Context, pods ...*corev1.Pod) error {
	var errs error
	for _, pod := range pods {
		clone := pod.DeepCopy()
		clone.ObjectMeta.UID = ""
		clone.ObjectMeta.ResourceVersion = ""
		clone.ObjectMeta.CreationTimestamp = metav1.Time{}
		clone.Spec.TerminationGracePeriodSeconds = pointer.Int64(0)
		errs = errors.Join(errs, p.client.Create(ctx, clone))
	}
	return errs
}
func (p podControl) DeletePodsMatchingNames(ctx context.Context, namespace string, podNames ...string) error {
	var errs error
	targetPods, err := p.ListPods(ctx, namespace, func(pod *corev1.Pod) bool {
		return slices.Contains(podNames, pod.Name)
	})
	if err != nil {
		return err
	}
	for _, pod := range targetPods {
		errs = errors.Join(errs, p.client.Delete(ctx, &pod))
	}
	return errs
}

func (p podControl) DeletePods(ctx context.Context, pods ...corev1.Pod) error {
	var errs error
	podsFailedDeletion := make([]string, 0, len(pods))
	for _, pod := range pods {
		if err := p.client.Delete(ctx, &pod); err != nil {
			podsFailedDeletion = append(podsFailedDeletion, pod.Name)
			errs = errors.Join(errs, err)
		}
	}
	if errs != nil {
		slog.Error("failed to delete one or more pods", "pods", podsFailedDeletion, "error", errs)
	}
	return errs
}

func (p podControl) DeleteAllPods(ctx context.Context, namespace string) error {
	return p.client.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(namespace))
}

func (p podControl) DeletePodsMatchingLabels(ctx context.Context, namespace string, labels map[string]string) error {
	return p.client.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(namespace), client.MatchingLabels(labels))
}
