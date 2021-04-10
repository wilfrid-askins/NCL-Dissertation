# https://cloud.google.com/kubernetes-engine/docs/tutorials/hello-app

# create docker image of application
docker build -t gcr.io/ultra-zenith-309512/central-app:v1 .
# add to image registry
docker push gcr.io/ultra-zenith-309512/central-app:v1

# deploy containers to k8s cluster
# arch plan
# tester run on single large node
# central run on k8s with replicas
