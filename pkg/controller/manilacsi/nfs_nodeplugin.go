package manilacsi

import (
	"context"

	manilacsiv1alpha1 "github.com/Fedosin/csi-driver-manila-operator/pkg/apis/manilacsi/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ReconcileManilaCSI) handleNFSNodePluginDaemonSet(instance *manilacsiv1alpha1.ManilaCSI, reqLogger logr.Logger) error {
	reqLogger.Info("Reconciling NFS Node Plugin DaemonSet")

	// Define a new DaemonSet object
	ds := generateNFSNodePluginManifest()

	// Check if this DaemonSet already exists
	found := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new DaemonSet", "DaemonSet.Namespace", ds.Namespace, "DaemonSet.Name", ds.Name)
		err = r.client.Create(context.TODO(), ds)
		if err != nil {
			return err
		}

		// DaemonSet created successfully - don't requeue
		return nil
	} else if err != nil {
		return err
	}

	// DaemonSet already exists - don't requeue
	reqLogger.Info("Skip reconcile: DaemonSet already exists", "DaemonSet.Namespace", found.Namespace, "DaemonSet.Name", found.Name)
	return nil
}

func generateNFSNodePluginManifest() *appsv1.DaemonSet {
	trueVar := true

	hostPathDirectoryOrCreate := corev1.HostPathDirectoryOrCreate
	hostPathDirectory := corev1.HostPathDirectory

	mountPropagationBidirectional := corev1.MountPropagationBidirectional

	labels := map[string]string{
		"app": "csi-nodeplugin-nfsplugin",
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "csi-nodeplugin-nfsplugin",
			Namespace: "manila-csi",
			Labels: labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "nfs",
							// TODO(mfedosin): start using the downstream image when
							// it is available.
							Image: "quay.io/k8scsi/nfsplugin:canary",
							Args: []string{
								"--nodeid=$(NODE_ID)",
								"--endpoint=unix://plugin/csi.sock",
							},
							Env: []corev1.EnvVar{
								{
									Name: "NODE_ID",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &trueVar,
								AllowPrivilegeEscalation: &trueVar,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SYS_ADMIN",
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "plugin-dir",
									MountPath: "/plugin",
								},
								{
									Name:             "pods-mount-dir",
									MountPath:        "/var/lib/kubelet/pods",
									MountPropagation: &mountPropagationBidirectional,
								},
							},
							ImagePullPolicy: "IfNotPresent",
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "plugin-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins/csi-nfsplugin",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "pods-mount-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/pods",
									Type: &hostPathDirectory,
								},
							},
						},
					},
				},
			},
		},
	}
}
