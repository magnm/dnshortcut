local_resource(
  'go-compile',
  'CGO_ENABLED=0 GOOS=linux go build -o build/dnshortcut ./',
  deps=['./main.go', './cmd', './pkg'])
  
docker_build(
  'ghcr.io/magnm/dnshortcut',
  '.',
  dockerfile='dev.Dockerfile',
  ignore=['./k8s'],
  only=[
    './build',
  ])

k8s_yaml(kustomize('./k8s'))