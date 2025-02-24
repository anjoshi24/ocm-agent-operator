package v1alpha1_test

import (
	"github.com/openshift/ocm-agent-operator/pkg/apis/ocmagent/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OCMAgent Controller", func() {

	const (
		testTemplateName = "test-template"
	)

	var (
		testManagedNotification *v1alpha1.ManagedNotification
	)

	BeforeEach(func() {
		testManagedNotification = &v1alpha1.ManagedNotification{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
				Namespace: "test-ns",
			},
			Spec:       v1alpha1.ManagedNotificationSpec{
				Templates: []v1alpha1.Template{
					{
						Name:         testTemplateName,
						Summary:      "Test Summary",
						ActiveDesc:   "Test Firing",
						ResolvedDesc: "Test Resolved",
						Severity:     "Info",
						ResendWait:   1,
					},
				},
			},
			Status:     v1alpha1.ManagedNotificationStatus{
				Notifications: []v1alpha1.NotificationRecord{
					{
						Name:                testTemplateName,
						ServiceLogSentCount: 0,
						Conditions:          []v1alpha1.NotificationCondition{
							{
								Type:               v1alpha1.ConditionAlertFiring,
								Status:             corev1.ConditionTrue,
								LastTransitionTime: &metav1.Time{Time:time.Now()},
								Reason:             "Test reason",
							},
							{
								Type:               v1alpha1.ConditionAlertResolved,
								Status:             corev1.ConditionTrue,
								LastTransitionTime: &metav1.Time{Time:time.Now()},
								Reason:             "Test reason",
							},
							{
								Type:               v1alpha1.ConditionServiceLogSent,
								Status:             corev1.ConditionTrue,
								LastTransitionTime: &metav1.Time{Time:time.Now()},
								Reason:             "Test reason",
							},
						},
					},
				},
			},
		}
	})

	Context("When retrieving a template", func() {
		It("will raise an error if the template is not found", func() {
			t, err := testManagedNotification.GetTemplateForName("nonexistant")
			Expect(t).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
		It("will return the correct template", func() {
			t, err := testManagedNotification.GetTemplateForName(testTemplateName)
			Expect(t.Name).To(Equal(testTemplateName))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When retrieving a template notification status", func() {
		It("will raise an error if the template is not found", func() {
			t, err := testManagedNotification.Status.GetNotificationRecord("nonexistant")
			Expect(t).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
		It("will return the correct template notification status", func() {
			t, err := testManagedNotification.Status.GetNotificationRecord(testTemplateName)
			Expect(t.Name).To(Equal(testTemplateName))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When checking if a firing template notification can be sent", func() {

		When("there is no defined template", func() {
			It("will raise an error", func() {
				cansend, err := testManagedNotification.CanBeSent("nonexistant")
				Expect(cansend).To(BeFalse())
				Expect(err).To(HaveOccurred())
			})
		})

		When("there is no notification history for the template", func() {
			BeforeEach(func() {
				testManagedNotification.Status.Notifications = []v1alpha1.NotificationRecord{}
			})
			It("will send", func() {
				cansend, err := testManagedNotification.CanBeSent(testTemplateName)
				Expect(cansend).To(BeTrue())
				Expect(err).To(BeNil())
			})
		})

		When("the current time is within the dont-resend window", func() {
			BeforeEach(func() {
				testManagedNotification.Status.Notifications[0].Conditions[2] = v1alpha1.NotificationCondition{
					Type:               v1alpha1.ConditionServiceLogSent,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: &metav1.Time{Time: time.Now().Add(time.Duration(-5) * time.Minute)},
					Reason:             "test",
				}
			})
			It("will not resend", func() {
				cansend, err := testManagedNotification.CanBeSent(testTemplateName)
				Expect(cansend).To(BeFalse())
				Expect(err).To(BeNil())
			})
		})

		When("the current time is outside the dont-resend window", func() {
			BeforeEach(func() {
				testManagedNotification.Status.Notifications[0].Conditions[2] = v1alpha1.NotificationCondition{
					Type:               v1alpha1.ConditionServiceLogSent,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: &metav1.Time{Time: time.Now().Add(time.Duration(-5) * time.Hour)},
					Reason:             "test",
				}
			})
			It("will resend", func() {
				cansend, err := testManagedNotification.CanBeSent(testTemplateName)
				Expect(cansend).To(BeTrue())
				Expect(err).To(BeNil())
			})
		})
	})

	Context("When retrieving a notification record", func() {
		It("retrieves the right record", func() {
			nr, err := testManagedNotification.Status.GetNotificationRecord(testTemplateName)
			Expect(err).To(BeNil())
			Expect(reflect.DeepEqual(*nr,testManagedNotification.Status.Notifications[0])).To(BeTrue())
		})
	})

	Context("When setting a notification record", func() {
		var nrs v1alpha1.NotificationRecords
		var newrecord v1alpha1.NotificationRecord
		var newTime *metav1.Time
		var newConditions []v1alpha1.NotificationCondition
		var newSLCount int32
		BeforeEach(func() {
			nrs = testManagedNotification.Status.Notifications
			newTime = &metav1.Time{Time:time.Now()}
			newSLCount = int32(555)
			newConditions = []v1alpha1.NotificationCondition{
				{
					Type:               v1alpha1.ConditionServiceLogSent,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: newTime,
					Reason:             "newreason",
				},
			}
			newrecord = v1alpha1.NotificationRecord{
				Name:                testTemplateName,
				ServiceLogSentCount: newSLCount,
				Conditions:          newConditions,
			}
		})
		When("the notification record already exists", func() {
			It("updates the existing record", func() {
				nrs.SetNotificationRecord(newrecord)
				nr := nrs.GetNotificationRecord(testTemplateName)
				Expect(reflect.DeepEqual(*nr,newrecord)).To(BeTrue())
			})
		})
		When("the notification record does not exist", func() {
			BeforeEach(func() {
				nrs = []v1alpha1.NotificationRecord{}
			})
			It("adds the record", func() {
				nrs.SetNotificationRecord(newrecord)
				nr := nrs.GetNotificationRecord(testTemplateName)
				Expect(reflect.DeepEqual(*nr,newrecord)).To(BeTrue())
			})
		})
	})

	Context("When updating the status of a record", func() {
		var nr *v1alpha1.NotificationRecord
		BeforeEach(func() {
			var err error
			nr, err = testManagedNotification.Status.GetNotificationRecord(testTemplateName)
			Expect(err).To(BeNil())
		})
		When("the condition does not already exist", func() {
			BeforeEach(func() {
				nr.Conditions = []v1alpha1.NotificationCondition{}
			})
			It("will create the status", func() {
				currTime := time.Now()
				Expect(nr.ServiceLogSentCount).To(Equal(int32(0)))
				err := nr.SetStatus(v1alpha1.ConditionAlertFiring, "testreason")
				Expect(err).To(BeNil())
				Expect(nr.Conditions[0].Type).To(Equal(v1alpha1.ConditionAlertFiring))
				Expect(nr.Name).To(Equal(testTemplateName))
				Expect(nr.ServiceLogSentCount).To(Equal(int32(1)))
				Expect(nr.Conditions[0].Reason).To(Equal("testreason"))
				Expect(nr.Conditions[0].LastTransitionTime.After(currTime)).To(BeTrue())
			})
		})
		When("the condition already exists", func() {
			It("will update the status", func() {
				currTime := time.Now()
				Expect(nr.ServiceLogSentCount).To(Equal(int32(0)))
				err := nr.SetStatus(v1alpha1.ConditionAlertFiring, "testreason")
				Expect(err).To(BeNil())
				Expect(nr.Conditions[0].Type).To(Equal(v1alpha1.ConditionAlertFiring))
				Expect(nr.Name).To(Equal(testTemplateName))
				Expect(nr.ServiceLogSentCount).To(Equal(int32(1)))
				Expect(nr.Conditions[0].Reason).To(Equal("testreason"))
				Expect(nr.Conditions[0].LastTransitionTime.After(currTime)).To(BeTrue())
			})
		})
	})
})
