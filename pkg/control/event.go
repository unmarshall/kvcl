package control

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/unmarshall/kvcl/api"
	"github.com/unmarshall/kvcl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewEventControl(cl client.Client) api.EventControl {
	return &eventControl{
		client: cl,
	}
}

type eventControl struct {
	client client.Client
}

func (e *eventControl) ListEvents(ctx context.Context, namespace string, filters ...api.EventFilter) ([]corev1.Event, error) {
	eventList := &corev1.EventList{}
	if err := e.client.List(ctx, eventList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	if filters == nil {
		return eventList.Items, nil
	}
	filteredEvents := make([]corev1.Event, 0, len(eventList.Items))
	for _, event := range eventList.Items {
		if ok := evaluateFilters(&event, filters); ok {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents, nil
}

// GetPodSchedulingEvents watches for pod scheduling events and returns the names of the pods that have been scheduled and unscheduled.
func (e *eventControl) GetPodSchedulingEvents(ctx context.Context, namespace string, since time.Time, pods []*corev1.Pod, podTimeout time.Duration) (scheduledPodNames sets.Set[string], unscheduledPodNames sets.Set[string], err error) {
	totalTimeout := podTimeout * time.Duration(len(pods))
	tick := time.NewTicker(totalTimeout)
	defer tick.Stop()
	pollTick := time.NewTicker(10 * time.Millisecond)
	defer pollTick.Stop()

	podNames := util.GetPodNames(pods)
	scheduledPodNames = sets.New[string]()
	unscheduledPodNames = sets.New[string]()

loop:
	for {
		select {
		case <-ctx.Done():
			return scheduledPodNames, unscheduledPodNames, fmt.Errorf("context cancelled, timeout waiting for pod events: %w", ctx.Err())
		case <-tick.C:
			return scheduledPodNames, unscheduledPodNames, fmt.Errorf("timeout waiting for pod events")
		case <-pollTick.C:
			events, err := e.ListEvents(ctx, namespace, filterEventBeforeTimeForPods(since, podNames))
			if err != nil {
				slog.Error("cannot get pod scheduling events, this will be retried", "error", err)
			}
			for _, event := range events {
				switch event.Reason {
				case "FailedScheduling":
					unscheduledPodNames.Insert(event.InvolvedObject.Name)
				case "Scheduled":
					scheduledPodNames.Insert(event.InvolvedObject.Name)
					podNames = slices.DeleteFunc(podNames, func(podName string) bool {
						return podName == event.InvolvedObject.Name
					})
					unscheduledPodNames.Delete(event.InvolvedObject.Name)
				}
			}
			//slog.Info("WaitForAndRecordPodSchedulingEvents completed", "num-total-pods", len(pods), "num-scheduled-pods", len(scheduledPodNames), "num-unscheduled-pods", len(unscheduledPodNames))
			if len(scheduledPodNames)+len(unscheduledPodNames) == len(pods) {
				break loop
			}
		}
	}
	return scheduledPodNames, unscheduledPodNames, nil
}

// filterEventBeforeTimeForPods returns an EventFilter that filters events that occurred before the given time and are related to the given pods.
func filterEventBeforeTimeForPods(since time.Time, targetPodNames []string) api.EventFilter {
	return func(event *corev1.Event) bool {
		if event.EventTime.BeforeTime(&metav1.Time{Time: since}) {
			return false
		}
		return slices.Contains(targetPodNames, event.InvolvedObject.Name)
	}
}

func (e *eventControl) DeleteAllEvents(ctx context.Context, namespace string) error {
	return e.client.DeleteAllOf(ctx, &corev1.Event{}, client.InNamespace(namespace))
}

func evaluateFilters(event *corev1.Event, filters []api.EventFilter) bool {
	for _, f := range filters {
		if ok := f(event); !ok {
			return false
		}
	}
	return true
}
