package utils

import (
	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
)

// IsRuntimeDefined inspect if build has `.spec.runtime` defined, checking intermediary attributes
// and making sure ImageURL is informed.
func IsRuntimeDefined(b *buildv1alpha1.Build) bool {
	if b.Spec.Runtime == nil {
		return false
	}
	if b.Spec.Runtime.Base.ImageURL == "" {
		return false
	}
	return true
}
