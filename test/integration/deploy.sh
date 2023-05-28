echo "[*] Creating test cluster ..."
kind delete cluster --name test
kind create cluster --name test

kind load docker-image weaveworks/policy-agent:${VERSION} --name test

kubectl create namespace flux-system

echo "[*] Installing cert-manager ..."
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --version v1.10.1 --set installCRDs=true --wait --timeout 120s

echo "[*] Apply test resources ..."
kubectl apply -f data/resources/audit_test_resources.yaml
kubectl apply -f ../../helm/crds

echo "[*] Apply cluster resources"
kubectl apply -f data/state

echo "[*] Installing policy agent helm chart on namespace ${NAMESPACE} ..."
helm install weave-policy-agent ../../helm -n ${NAMESPACE} -f ../../helm/values.yaml -f data/values.yaml --create-namespace --wait --timeout 60s
