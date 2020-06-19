package buildrun

import (
	"fmt"

	buildv1alpha1 "github.com/redhat-developer/build/pkg/apis/build/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("runtime-image", func() {
	b := &buildv1alpha1.Build{
		Spec: buildv1alpha1.BuildSpec{
			BuilderImage: &buildv1alpha1.Image{
				ImageURL: "test/builder-image:latest",
			},
			Output: buildv1alpha1.Image{
				ImageURL: "test/output-image:latest",
			},
			Runtime: buildv1alpha1.Runtime{
				Base: buildv1alpha1.Image{
					ImageURL: "test/base-image:latest",
				},
				Env: map[string]string{
					"ENVIRONMENT_VARIABLE": "VALUE",
				},
				Labels: map[string]string{
					"label": "value",
				},
				WorkDir:     "/workdir",
				Directories: []string{"/path/to/a:/new/path/to/a", "/path/to/b"},
				Entrypoint:  []string{"/bin/bash", "-x", "-c"},
			},
		},
	}

	Context("splitting directories", func() {
		It("expect directories splitted by \":\" ", func() {
			parts := splitDirectories("a:b")
			Expect(parts).To(Equal([]string{"a", "b"}))

			parts = splitDirectories("a")
			Expect(parts).To(Equal([]string{"a", "a"}))
		})
	})

	Context("rendering entrypoint", func() {
		It("expect entrypoint concatenated", func() {
			entrypoint := renderEntrypoint(b.Spec.Runtime.Entrypoint)
			fmt.Printf("Entrypoint: ---\n%s\n---\n", entrypoint)

			Expect(entrypoint).To(Equal("\"/bin/bash\", \"-x\", \"-c\""))
		})
	})

	Context("rendering runtime Dockerfile", func() {

		It("expect a complete dockerfile", func() {
			dockerfile, err := renderRuntimeDockerfile(b)
			fmt.Printf("Dockerfile.runtime: ---\n%s\n---\n", dockerfile)

			Expect(err).ToNot(HaveOccurred())
			Expect(dockerfile).ToNot(BeNil())

			Expect(fmt.Sprintf("\n%s", dockerfile)).To(Equal(`
FROM test/output-image:latest as builder

FROM test/base-image:latest
ENV ENVIRONMENT_VARIABLE="VALUE"
LABEL label="value"
COPY --from=builder "/path/to/a" "/new/path/to/a"
COPY --from=builder "/path/to/b" "/path/to/b"
WORKDIR "/workdir"
ENTRYPOINT [ "/bin/bash", "-x", "-c" ]`,
			))
		})
	})

	Context("amend build-strategy with extra steps", func() {
		bs := &buildv1alpha1.BuildStrategy{
			Spec: buildv1alpha1.BuildStrategySpec{
				BuildSteps: []buildv1alpha1.BuildStep{},
			},
		}

		It("expect to have build-strategy amended", func() {
			err := amendBuildStrategySpecWithRuntimeImage(&bs.Spec, b)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(bs.Spec.BuildSteps)).To(Equal(2))
		})
	})
})
