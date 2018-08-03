package controllerutil_test

import (
	"context"
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Controllerutil", func() {
	Describe("SetControllerReference", func() {
		It("should set the OwnerReference if it can find the group version kind", func() {
			rs := &appsv1.ReplicaSet{}
			dep := &extensionsv1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", UID: "foo-uid"},
			}

			Expect(controllerutil.SetControllerReference(dep, rs, scheme.Scheme)).NotTo(HaveOccurred())
			t := true
			Expect(rs.OwnerReferences).To(ConsistOf(metav1.OwnerReference{
				Name:               "foo",
				Kind:               "Deployment",
				APIVersion:         "extensions/v1beta1",
				UID:                "foo-uid",
				Controller:         &t,
				BlockOwnerDeletion: &t,
			}))
		})

		It("should return an error if it can't find the group version kind of the owner", func() {
			rs := &appsv1.ReplicaSet{}
			dep := &extensionsv1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			}
			Expect(controllerutil.SetControllerReference(dep, rs, runtime.NewScheme())).To(HaveOccurred())
		})

		It("should return an error if the owner isn't a runtime.Object", func() {
			rs := &appsv1.ReplicaSet{}
			Expect(controllerutil.SetControllerReference(&errMetaObj{}, rs, scheme.Scheme)).To(HaveOccurred())
		})

		It("should return an error if object is already owned by another controller", func() {
			t := true
			rsOwners := []metav1.OwnerReference{
				metav1.OwnerReference{
					Name:               "bar",
					Kind:               "Deployment",
					APIVersion:         "extensions/v1beta1",
					UID:                "bar-uid",
					Controller:         &t,
					BlockOwnerDeletion: &t,
				},
			}
			rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default", OwnerReferences: rsOwners}}
			dep := &extensionsv1beta1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default", UID: "foo-uid"}}

			err := controllerutil.SetControllerReference(dep, rs, scheme.Scheme)

			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&controllerutil.AlreadyOwnedError{}))
		})

		It("should not duplicate existing owner reference", func() {
			f := false
			t := true
			rsOwners := []metav1.OwnerReference{
				metav1.OwnerReference{
					Name:               "foo",
					Kind:               "Deployment",
					APIVersion:         "extensions/v1beta1",
					UID:                "foo-uid",
					Controller:         &f,
					BlockOwnerDeletion: &t,
				},
			}
			rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default", OwnerReferences: rsOwners}}
			dep := &extensionsv1beta1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default", UID: "foo-uid"}}

			Expect(controllerutil.SetControllerReference(dep, rs, scheme.Scheme)).NotTo(HaveOccurred())
			Expect(rs.OwnerReferences).To(ConsistOf(metav1.OwnerReference{
				Name:               "foo",
				Kind:               "Deployment",
				APIVersion:         "extensions/v1beta1",
				UID:                "foo-uid",
				Controller:         &t,
				BlockOwnerDeletion: &t,
			}))
		})
	})

	Describe("CreateOrUpdate", func() {
		var deploy *appsv1.Deployment
		var deplKey types.NamespacedName

		BeforeEach(func() {
			deploy = &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"foo": "bar"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name:  "foo",
									Image: "busybox",
								},
							},
						},
					},
				},
			}

			deploy.Name = fmt.Sprintf("deploy-%d", rand.Int31())
			deploy.Namespace = "default"

			deplKey = types.NamespacedName{
				Name:      deploy.Name,
				Namespace: deploy.Namespace,
			}
		})

		It("creates a new object if one doesn't exists", func() {
			op, err := controllerutil.CreateOrUpdate(context.TODO(), c, deploy)

			By("returning OperationCreated")
			Expect(op).To(BeEquivalentTo(controllerutil.OperationCreate))

			By("returning no error")
			Expect(err).ShouldNot(HaveOccurred())

			By("actually having the deployment created")
			fetched := &appsv1.Deployment{}
			Expect(c.Get(context.TODO(), deplKey, fetched)).Should(Succeed())
		})

		It("update existing object", func() {
			var scale int32 = 2
			Expect(c.Create(context.TODO(), deploy)).Should(Succeed())

			op, err := controllerutil.CreateOrUpdate(context.TODO(), c, deploy, deploymentScaler(scale))

			By("returning OperationUpdated")
			Expect(op).Should(BeEquivalentTo(controllerutil.OperationUpdate))

			By("returning no error")
			Expect(err).ShouldNot(HaveOccurred())

			By("actually having the deployment scaled")
			fetched := &appsv1.Deployment{}
			Expect(c.Get(context.TODO(), deplKey, fetched)).Should(Succeed())
			Expect(*fetched.Spec.Replicas).To(Equal(scale))
		})

		It("updates only changed objects", func() {
			op, err := controllerutil.CreateOrUpdate(context.TODO(), c, deploy)

			Expect(op).Should(BeEquivalentTo(controllerutil.OperationCreate))
			Expect(err).ShouldNot(HaveOccurred())

			op, err = controllerutil.CreateOrUpdate(context.TODO(), c, deploy, deploymentIdentity)

			By("returning OperationNoop")
			Expect(op).Should(BeEquivalentTo(controllerutil.OperationNoop))

			By("returning no error")
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("allows chaining transforms", func() {
			scaleToTwo := deploymentScaler(2)
			scaleToThree := deploymentScaler(3)

			op, err := controllerutil.CreateOrUpdate(context.TODO(), c, deploy, scaleToTwo, scaleToThree)

			Expect(op).Should(BeEquivalentTo(controllerutil.OperationCreate))
			Expect(err).ShouldNot(HaveOccurred())

			By("applying the last scale")
			fetched := &appsv1.Deployment{}
			Expect(c.Get(context.TODO(), deplKey, fetched)).Should(Succeed())
			Expect(*fetched.Spec.Replicas).To(Equal(int32(3)))
		})

		It("doesn't mutate the desired object", func() {
			scaleToTwo := deploymentScaler(2)
			scaleToThree := deploymentScaler(3)

			op, err := controllerutil.CreateOrUpdate(context.TODO(), c, deploy, scaleToTwo, scaleToThree)

			Expect(op).Should(BeEquivalentTo(controllerutil.OperationCreate))
			Expect(err).ShouldNot(HaveOccurred())

			Expect(deploy.Spec.Replicas).To(BeNil())
		})
	})
})

var _ metav1.Object = &errMetaObj{}

type errMetaObj struct {
	metav1.ObjectMeta
}

var deploymentIdentity controllerutil.ReconcileFn = func(obj, existing runtime.Object) error {
	existing.(*appsv1.Deployment).DeepCopyInto(obj.(*appsv1.Deployment))
	return nil
}

func deploymentScaler(replicas int32) controllerutil.ReconcileFn {
	fn := func(obj, existing runtime.Object) error {
		out := obj.(*appsv1.Deployment)
		out.Spec.Replicas = &replicas
		return nil
	}
	return fn
}
