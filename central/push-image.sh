# create docker image of application
docker build -t gcr.io/ultra-zenith-309512/central-app:latest .
# add to image registry
docker push gcr.io/ultra-zenith-309512/central-app:latest
