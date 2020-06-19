package buildrun

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"
	"text/template"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	// runtimeDockerfileTmpl Dockerfile template to be used with runtime-image, it uses Build attributes
	// directly as template input.
	runtimeDockerfileTmpl = `FROM {{ .Spec.Output.ImageURL }} as builder

FROM {{ .Spec.Runtime.Base.ImageURL }}

{{- range $k, $v := .Spec.Runtime.Env }}
ENV {{ $k }}="{{ $v }}"
{{- end }}

{{- range $k, $v := .Spec.Runtime.Labels }}
LABEL {{ $k }}="{{ $v }}"
{{- end }}

{{- range $dir := .Spec.Runtime.Directories }}
{{- $parts := splitDirectories $dir }}
COPY --from=builder "{{ index $parts 0 }}" "{{ index $parts 1 }}"
{{- end }}

{{- if .Spec.Runtime.WorkDir }}
WORKDIR "{{ .Spec.Runtime.WorkDir }}"
{{- end }}

{{- if .Spec.Runtime.Entrypoint }}
ENTRYPOINT [ {{ renderEntrypoint .Spec.Runtime.Entrypoint }} ]
{{- end -}}
`

	// workspaceDir common workspace directory, where the source code is located.
	workspaceDir = "/workspace/source"

	// kanikoStable stable buildah image.
	kanikoStable = "gcr.io/kaniko-project/executor:v0.23.0"

	// runtimeDockerfile runtime Dockerfile file name.
	runtimeDockerfile = "Dockerfile.runtime"
)

// rootUserID root's UID
var rootUserID = int64(0)

// runtimeDockerfilePath path to runtime Dockerfile on workspace directory.
var runtimeDockerfilePath = path.Join(workspaceDir, runtimeDockerfile)

// splitDirectories split informed directory by colon, returning a slice with parts. When colon is
// not present, return the informed directory twice. This method always returns a slice with two
// entries.
func splitDirectories(dir string) []string {
	parts := strings.Split(dir, ":")
	if len(parts) == 2 {
		return parts
	}
	return []string{dir, dir}
}

// renderEntrypoint will take a slice of strings and render the notation expected on ENTRYPOINT.
func renderEntrypoint(e []string) string {
	entrypoint := []string{}
	for _, cmd := range e {
		entrypoint = append(entrypoint, strconv.Quote(cmd))
	}
	return strings.Join(entrypoint, ", ")
}

// renderRuntimeDockerfile render runtime Dockerfile using build instance and pre-defined template.
func renderRuntimeDockerfile(b *buildv1alpha1.Build) (*bytes.Buffer, error) {
	tmpl, err := template.New(runtimeDockerfile).
		Funcs(template.FuncMap{
			"splitDirectories": splitDirectories,
			"renderEntrypoint": renderEntrypoint,
		}).
		Parse(runtimeDockerfileTmpl)
	if err != nil {
		return nil, err
	}

	dockerfile := new(bytes.Buffer)
	if err = tmpl.Execute(dockerfile, b); err != nil {
		return nil, err
	}
	return dockerfile, nil
}

// runtimeDockerfileStep trigger the rendering of Dockerfile.runtime, and use this input as a
// build-step to create a new file.
func runtimeDockerfileStep(b *buildv1alpha1.Build) (*buildv1alpha1.BuildStep, error) {
	dockerfile, err := renderRuntimeDockerfile(b)
	if err != nil {
		return nil, err
	}

	container := v1.Container{
		Name:  "runtime-dockerfile",
		Image: b.Spec.BuilderImage.ImageURL,
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &rootUserID,
		},
		WorkingDir: workspaceDir,
		Command:    []string{"/bin/bash"},
		Args: []string{
			"-x",
			"-c",
			fmt.Sprintf("echo '%s' >%s", dockerfile, runtimeDockerfilePath),
		},
	}
	return &buildv1alpha1.BuildStep{Container: container}, nil
}

// runtimeBuildAndPushStep returns a build-step to build the Dockerfile.runtime with kaniko.
func runtimeBuildAndPushStep(b *buildv1alpha1.Build) *buildv1alpha1.BuildStep {
	contextDir := workspaceDir
	if b != nil && b.Spec.Source.ContextDir != nil {
		contextDir = path.Join(workspaceDir, *b.Spec.Source.ContextDir)
	}

	container := v1.Container{
		Name:       "kaniko-build-and-push",
		Image:      kanikoStable,
		WorkingDir: workspaceDir,
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &rootUserID,
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{
					v1.Capability("CHOWN"),
					v1.Capability("DAC_OVERRIDE"),
					v1.Capability("FOWNER"),
					v1.Capability("SETGID"),
					v1.Capability("SETUID"),
				},
			},
		},
		Env: []v1.EnvVar{
			{Name: "DOCKER_CONFIG", Value: "/tekton/home/.docker"},
			{Name: "AWS_ACCESS_KEY_ID", Value: "NOT_SET"},
			{Name: "AWS_SECRET_KEY", Value: "NOT_SET"},
		},
		Command: []string{"/kaniko/executor"},
		Args: []string{
			"--skip-tls-verify=true",
			fmt.Sprintf("--dockerfile=%s", runtimeDockerfile),
			fmt.Sprintf("--context=%x", contextDir),
			fmt.Sprintf("--destination=%s", b.Spec.Output.ImageURL),
		},
	}
	return &buildv1alpha1.BuildStep{Container: container}
}

// amendBuildStrategySpecWithRuntimeImage add more steps to build-strategy in order to implement the
// creation of a runtime-image.
func amendBuildStrategySpecWithRuntimeImage(
	spec *buildv1alpha1.BuildStrategySpec,
	b *buildv1alpha1.Build,
) error {
	step, err := runtimeDockerfileStep(b)
	if err != nil {
		return err
	}
	spec.BuildSteps = append(spec.BuildSteps, *step)

	step = runtimeBuildAndPushStep(b)
	spec.BuildSteps = append(spec.BuildSteps, *step)
	return nil
}

// AmendBuildStrategyWithRuntimeImage amend spec section with runtime steps.
func AmendBuildStrategyWithRuntimeImage(
	bs *buildv1alpha1.BuildStrategy,
	b *buildv1alpha1.Build,
) error {
	return amendBuildStrategySpecWithRuntimeImage(&bs.Spec, b)
}

// AmendClusterBuildStrategyWithRuntimeImage amend spec section with runtime steps.
func AmendClusterBuildStrategyWithRuntimeImage(
	cbs *buildv1alpha1.ClusterBuildStrategy,
	b *buildv1alpha1.Build,
) error {
	return amendBuildStrategySpecWithRuntimeImage(&cbs.Spec, b)
}
