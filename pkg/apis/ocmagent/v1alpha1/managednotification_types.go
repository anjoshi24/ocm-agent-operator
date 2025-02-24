package v1alpha1

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type NotificationSeverity string

const (
	SeverityDebug   NotificationSeverity = "Debug"
	SeverityWarning NotificationSeverity = "Warning"
	SeverityInfo    NotificationSeverity = "Info"
	SeverityError   NotificationSeverity = "Error"
	SeverityFatal   NotificationSeverity = "Fatal"
)

// ManagedNotificationSpec defines the desired state of ManagedNotification
type ManagedNotificationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// AgentConfig refers to OCM agent config fields separated
	Templates []Template `json:"templates"`
}

type Template struct {

	// The name of the template used to associate with an alert
	Name string `json:"name"`

	// The summary line of the Service Log notification
	Summary string `json:"summary"`

	// The body text of the Service Log notification when the alert is active
	ActiveDesc string `json:"activeBody"`

	// The body text of the Service Log notification when the alert is resolved
	ResolvedDesc string `json:"resolvedBody"`

	// +kubebuilder:validation:Enum={"Debug","Info","Warning","Error","Fatal"}
	// The severity of the Service Log notification
	Severity NotificationSeverity `json:"severity"`

	// Measured in hours. The minimum time interval that must elapse between active Service Log notifications
	ResendWait int32 `json:"resendWait"`
}

// ManagedNotificationStatus defines the observed state of ManagedNotification
type ManagedNotificationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Notifications NotificationRecords `json:"notifications,omitempty"`
}

type NotificationRecords []NotificationRecord

type NotificationConditionType string
const (
	ConditionAlertFiring    NotificationConditionType = "AlertFiring"
	ConditionAlertResolved  NotificationConditionType = "AlertResolved"
	ConditionServiceLogSent NotificationConditionType = "ServiceLogSent"
)

type Conditions []NotificationCondition
type NotificationCondition struct {
	// +kubebuilder:validation:Enum={"AlertFiring","AlertResolved","ServiceLogSent"}
	// Type of Notification condition
	Type NotificationConditionType `json:"type"`

	// Status of condition, one of True, False, Unknown
	Status corev1.ConditionStatus `json:"status"`

	// Last time the condition transit from one status to another.
	// +kubebuilder:validation:Optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// (brief) reason for the condition's last transition.
	// +kubebuilder:validation:Optional
	Reason string `json:"reason,omitempty"`
}

type NotificationRecord struct {
	// Name of the notification template
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// ServiceLogSentCount records the number of service logs sent for the template
	ServiceLogSentCount int32 `json:"serviceLogSentCount,omitempty"`

	// Conditions is a set of Condition instances.
	Conditions Conditions `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManagedNotification is the Schema for the managednotifications API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managednotifications,scope=Namespaced
type ManagedNotification struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedNotificationSpec   `json:"spec,omitempty"`
	Status ManagedNotificationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManagedNotificationList contains a list of ManagedNotification
type ManagedNotificationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedNotification `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedNotification{}, &ManagedNotificationList{})
}

// GetTemplateForName returns a notification template matching the given name
// or error if no matching template can be found.
func (m *ManagedNotification) GetTemplateForName(n string) (*Template, error) {
	for _, t := range m.Spec.Templates {
		if t.Name == n {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("template with name %v not found", n)
}

// GetNotificationRecord returns the history for a notification template matching
// the given name or error if no matching template can be found.
func (m *ManagedNotificationStatus) GetNotificationRecord(n string) (*NotificationRecord, error) {
	for _, t := range m.Notifications {
		if t.Name == n {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("template with name %v not found", n)
}

// HasNotificationRecord returns whether or not a notification status history exists
// for a template with the given name
func (m *ManagedNotificationStatus) HasNotificationRecord(n string) bool {
	for _, t := range m.Notifications {
		if t.Name == n {
			return true
		}
	}
	return false
}

// CanBeSent returns true if a service log from the template is allowed to be sent
func (m *ManagedNotification) CanBeSent(n string) (bool, error) {

	// If no template exists, one cannot be sent
	t, err := m.GetTemplateForName(n)
	if err != nil {
		return false, err
	}

	// If no status history exists for the template, it is safe to send a notification
	if !m.Status.HasNotificationRecord(n) {
		return true, nil
	}

	// If a status history exists but can't be fetched, this is an irregular situation
	s, err := m.Status.GetNotificationRecord(n)
	if err != nil {
		return false, err
	}

	// If the last time a notification was sent is within the don't-resend window, don't send
	sentCondition := s.Conditions.GetCondition(ConditionServiceLogSent)
	if sentCondition == nil {
		// No service log send recorded yet, it can be sent
		return true, nil
	}
	now := time.Now()
	nextresend := sentCondition.LastTransitionTime.Time.Add(time.Duration(t.ResendWait) * time.Hour)
	if now.Before(nextresend) {
			return false, nil
		}

	return true, nil
}

// GetCondition searches the set of conditions for the condition with the given
// ConditionType and returns it. If the matching condition is not found,
// GetCondition returns nil.
func (conditions Conditions) GetCondition(t NotificationConditionType) *NotificationCondition {
	for _, condition := range conditions {
		if condition.Type == t {
			return &condition
		}
	}
	return nil
}

// NewNotificationRecord adds a new notification record status for the given name
func (m *ManagedNotificationStatus) NewNotificationRecord(n string) {
	r := NotificationRecord{
		Name:                n,
		ServiceLogSentCount: 0,
		Conditions:          []NotificationCondition{},
	}
	m.Notifications = append(m.Notifications, r)
}

// GetNotificationRecord retrieves the notification record associated with the given name
func (nrs NotificationRecords) GetNotificationRecord(name string) *NotificationRecord {
	for _, n := range nrs {
		if n.Name == name {
			return &n
		}
	}
	return nil
}

// SetNotificationRecord adds or overwrites the supplied notification record
func (nrs *NotificationRecords) SetNotificationRecord(rec NotificationRecord) {
	for i, n := range *nrs {
		if n.Name == rec.Name {
			(*nrs)[i] = rec
			return
		}
	}
	*nrs = append(*nrs, rec)
}

// SetStatus updates the status for a given notification record type
func (nr *NotificationRecord) SetStatus(nct NotificationConditionType, reason string) error {
	nr.ServiceLogSentCount++
	condition := NotificationCondition{
		Type:               nct,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: &metav1.Time{Time:time.Now()},
		Reason:             reason,
	}
	nr.Conditions.SetCondition(condition)
	return nil
}

// SetCondition adds or updates a condition in a notification record
func (c *Conditions) SetCondition(new NotificationCondition) {
	for i, condition := range *c {
		if condition.Type == new.Type {
			(*c)[i] = new
			return
		}
	}
	*c = append(*c, new)
}
