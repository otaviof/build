---
title: runtime-image-support
authors:
  - "@sbose78"
  - "@otaviof"
status: design
---

Runtime-Image Support
---------------------

# Summary

This enhancement proposal describes the support of runtime-image in build-v2 operator. A runtime-image is a composition based on the image just created by a third-party builder, used as input for the runtime-image.

# Motivation

With the runtime-image feature in place, it will be employed for the creation of _lean-images_, which takes the ability of picking only certain parts of the original image, creating a runtime-image with the reduced data-set.

Additionally, creating images with a different base-image, and other advanced use-cases are also supported by this feature.

# Proposal

The sections below are describing the changes necessary to introduce the runtime-image.

## API Changes

A new attribute will be added on Build resource, as the following example:


```yml
---
apiVersion: build.dev/v1alpha1
kind: Build
# ...
spec:
  runtime:
    base:
      image: runtime-base-image:latest
    workDir: /workspace/source
    env:
      JAVA_HOME: /path/to/java-home
    labels:
      custom-image-label: value
    directories:
      - /path/to/directory
      - /path/to/another/directory:/target/location
    entrypoint:
      - java
      - -jar
      - ...
```

The new attributes added are:

- `spec.runtime.base`: specifies the runtime base-image to be used, using [Image](https://github.com/redhat-developer/build/blob/97012ab56417ce1691a70896d90e490ea6a4d23c/pkg/apis/build/v1alpha1/build_types.go#L58) as type;
- `spec.runtime.workDir`: path to `WORKDIR` in runtime-image;
- `spec.runtime.env`: runtime-image additional environment variables, key-value;
- `spec.runtime.labels`: runtime-image additional labels, key-value;
- `spec.runtime.directories`: list of directories paths to be copied to runtime-image, those can be defined as `<source>:<destination>` split by colon (`:`);
- `spec.runtime.entrypoint`: entrypoint command, specified as a list;

## Runtime-Image

Given the API changes described above, we will now address how the runtime-image will be created. This image is always based on an image built by the `BuildStrategy` (or `ClusterBuildStrategy`) steps, and therefore, we can introduce new steps to add the elements needed for the runtime-image creation.


### Runtime Dockerfile

The first step to be appended on the given `BuildStrategy` instance is the `Dockerfile` for the runtime-image, commonly named `Dockerfile.runtime`.

This Dockerfile would like the following example:


```
FROM output-image:tag as builder

FROM runtime-image:tag
ENV VARIABLE=value
LABEL label=value
COPY --from=builder "/path/to/source" "/path/to/destination"
WORKDIR "/path/to/destination"
ENTRYPOINT [ "command", "args" ]
```

The example shows a [multi-stage image build](https://docs.docker.com/develop/develop-images/multistage-build/) using the outcomes of the original `BuildStrategy` steps as `builder` image, and afterwards, rendering all the elements supported by the [new `runtime` section](#API-Changes). To create this file we may employ [`text/template`](https://golang.org/pkg/text/template/) to render its contents.

### Appending Steps

During the [`BuildRun` resource reconciliation](https://github.com/redhat-developer/build/blob/97012ab56417ce1691a70896d90e490ea6a4d23c/pkg/controller/buildrun/buildrun_controller.go#L116) we have the opportunity to amend the `BuildStrategy` instance. The first step to be added is the `Dockerfile.runtime` creation:


```yml
---
apiVersion: build.dev/v1alpha1
kind: BuildStrategy
# ...
spec:
  buildSteps:
    - name: step-runtime-dockerfile
      image: $(build.builder.image)
      securityContext:
        runAsUser: 0
      workingdir: /workspace/source
      command:
        - /bin/bash
      args:
        - -x
        - -c
        - >
          echo '<DOCKERFILE_CONTENT>' >Dockerfile.runtime
```

During the implementation phase `DOCKERFILE_CONTENT` will become the actual [runtime Dockerfile](#Runtime-Dockerfile).

And, the last step to be added is the container-image build, for this position we can take either [`buildah`](https://github.com/redhat-developer/build/blob/97012ab56417ce1691a70896d90e490ea6a4d23c/samples/build/build_buildah_cr.yaml) or [`kaniko`](https://github.com/redhat-developer/build/blob/97012ab56417ce1691a70896d90e490ea6a4d23c/samples/build/build_kaniko_cr.yaml) strategies for guidance.

## Experiments

During the development phase we've evaluated two different application ecosystems. A common [Node.js](https://gist.github.com/otaviof/eccf5abe879a8218cf5b807f520367f4) application, and [Pet-Clinic](https://gist.github.com/otaviof/53aad504ccc59681fe3875dbf3150c55), a Java based application.

Since the controller generates a Dockerfile on the fly, those use cases worked as expected, producing another container image, reusing container image URL and tag.
