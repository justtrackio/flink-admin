package internal

import (
	"sort"
	"time"

	eventsv1 "k8s.io/api/events/v1"
)

// K8sEvent is a simplified representation of a Kubernetes events.k8s.io/v1 Event
// returned to the frontend.
type K8sEvent struct {
	Type                string    `json:"type"`
	Reason              string    `json:"reason"`
	Note                string    `json:"note"`
	ReportingController string    `json:"reportingController"`
	Regarding           string    `json:"regarding"`
	EventTime           time.Time `json:"eventTime"`
	Action              string    `json:"action,omitempty"`
}

// K8sEventsResponse is the response payload for the events endpoint.
type K8sEventsResponse struct {
	Events []K8sEvent `json:"events"`
}

// toK8sEvents converts a list of Kubernetes Event objects into K8sEvent response structs,
// sorted by event time descending (newest first).
func toK8sEvents(items []eventsv1.Event) []K8sEvent {
	events := make([]K8sEvent, 0, len(items))

	for _, item := range items {
		eventTime := item.EventTime.Time
		if eventTime.IsZero() && item.DeprecatedLastTimestamp.Time != (time.Time{}) {
			eventTime = item.DeprecatedLastTimestamp.Time
		}

		regarding := item.Regarding.Kind + "/" + item.Regarding.Name

		events = append(events, K8sEvent{
			Type:                item.Type,
			Reason:              item.Reason,
			Note:                item.Note,
			ReportingController: item.ReportingController,
			Regarding:           regarding,
			EventTime:           eventTime,
			Action:              item.Action,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].EventTime.After(events[j].EventTime)
	})

	return events
}
