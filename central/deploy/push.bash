// https://cloud.google.com/kubernetes-engine/docs/tutorials/hello-app
export PROJECT_ID=PROJECT_ID

// create docker image of application
docker build -t gcr.io/${PROJECT_ID}/central-app:v1 .

// add to image registry
docker push gcr.io/${PROJECT_ID}/central-app:v1

// deploy containers to k8s cluster


// arch plan
// tester run on single large node
// central run on k8s with replicas

