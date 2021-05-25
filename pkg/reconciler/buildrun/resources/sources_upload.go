// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package resources

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

//
// TODO:
// - add pitstop image to general configuration;
//

func AmendTaskSpecWaitForUpload(taskSpec *v1beta1.TaskSpec) {
	tektonUID := int64(1000)
	uploadStep := tektonv1beta1.Step{Container: corev1.Container{
		Name:  "wait-for-upload",
		Image: "otaviof/waiter:latest",
		Args:  []string{"start"},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: &tektonUID,
		},
		ImagePullPolicy: corev1.PullAlways,
	}}
	taskSpec.Steps = append(taskSpec.Steps, uploadStep)
}
