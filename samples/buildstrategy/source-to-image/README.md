# `source-to-image` Build Strategy

The [strategy](./buildstrategy_source-to-image_cr.yaml) is composed by [`source-to-image`][s2i]
and [`buildah`][buildah] build and push the application image. Typically `s2i` requires a
specially crafted image, which can be informed as `builderImage` parameter.

To install this strategy, use:

```sh
kubectl apply -f samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
```

## Build Steps

1. `s2i` to build the complete application image;
3. `buildah` to push `output.image` to configured container registry;

[s2i]: https://github.com/openshift/source-to-image
[buildah]: https://github.com/containers/buildah