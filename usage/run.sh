kubectl apply -f manifests/mycrd.k8s_appscodes.yaml
kubectl apply -f manifests/a.yaml
kubectl apply -f manifests/b.yaml

go run main.go
