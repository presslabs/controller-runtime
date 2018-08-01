package controllerutil_test

import (
	"context"

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

		It("creates a new object if one doesn't exists", func() {
			deplKey := types.NamespacedName{Name: "test-create", Namespace: "default"}
			depl := &appsv1.Deployment{}

			op, err := controllerutil.CreateOrUpdate(context.TODO(), c, deplKey, depl, createDeployment)

			By("returning OperationCreated")
			Expect(op).Should(BeEquivalentTo(controllerutil.OperationCreated))

			By("returning returning no error")
			Expect(err).ShouldNot(HaveOccurred())

			By("actually having the deployment created")
			fetched := &appsv1.Deployment{}
			Expect(c.Get(context.TODO(), deplKey, fetched)).Should(Succeed())
		})

		It("update existing object", func() {
			deplKey := types.NamespacedName{Name: "test-update", Namespace: "default"}
			d, _ := createDeployment(&appsv1.Deployment{})
			depl := d.(*appsv1.Deployment)
			depl.Name = "test-update"
			depl.Namespace = "default"

			var scale int32 = 2

			Expect(c.Create(context.TODO(), depl)).Should(Succeed())

			op, err := controllerutil.CreateOrUpdate(context.TODO(), c, deplKey, &appsv1.Deployment{}, deploymentScaler(scale))

			By("returning OperationUpdated")
			Expect(op).Should(BeEquivalentTo(controllerutil.OperationUpdated))

			By("returning returning no error")
			Expect(err).ShouldNot(HaveOccurred())

			By("actually having the deployment scaled")
			fetched := &appsv1.Deployment{}
			Expect(c.Get(context.TODO(), deplKey, fetched)).Should(Succeed())
			Expect(*fetched.Spec.Replicas).To(Equal(scale))
		})

		It("updates only changed objects", func() {
			deplKey := types.NamespacedName{Name: "test-idempotency", Namespace: "default"}
			depl := &appsv1.Deployment{}

			op, err := controllerutil.CreateOrUpdate(context.TODO(), c, deplKey, depl, createDeployment)
			Expect(op).Should(BeEquivalentTo(controllerutil.OperationCreated))
			Expect(err).ShouldNot(HaveOccurred())

			op, err = controllerutil.CreateOrUpdate(context.TODO(), c, deplKey, depl, deploymentIdentity)

			By("returning OperationNoop")
			Expect(op).Should(BeEquivalentTo(controllerutil.OperationNoop))

			By("returning returning no error")
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

var _ metav1.Object = &errMetaObj{}

type errMetaObj struct {
	metav1.ObjectMeta
}

var createDeployment controllerutil.TransformFn = func(in runtime.Object) (runtime.Object, error) {
	out := in.(*appsv1.Deployment)
	out.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}
	out.Spec.Template.ObjectMeta.Labels = map[string]string{"foo": "bar"}
	out.Spec.Template.Spec.Containers = []corev1.Container{corev1.Container{Name: "foo", Image: "busybox"}}
	return out, nil
}

var deploymentIdentity controllerutil.TransformFn = func(in runtime.Object) (runtime.Object, error) {
	return in, nil
}

func deploymentScaler(replicas int32) controllerutil.TransformFn {
	fn := func(in runtime.Object) (runtime.Object, error) {
		d, _ := createDeployment(in)
		out := d.(*appsv1.Deployment)
		out.Spec.Replicas = &replicas
		return out, nil
	}
	return fn
}
