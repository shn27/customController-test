# customController-test
If you create any deployment in your local cluster ; it will automatically create a service for it.
If you delete any deployment in your local cluster ; it will automatically delete that service created it.


------------------
```
git clone git@github.com:shn27/customController-test.git
go build .
./test-controller

kind create cluster
kubectl apply -f https://k8s.io/examples/application/deployment.yaml
kubectl delete -f https://k8s.io/examples/application/deployment.yaml

```
