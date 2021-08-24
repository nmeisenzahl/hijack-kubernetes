# Spin up the environment

This guide is based on Azure and Azure Kubernetes Service but should also work with any other Public Cloud offering.

## Kubernetes Cluster

Create a Resource Group hosting the environment:

```bash
az group create --name demo-rg --location westeurope
```

Get your public IP:

```bash
ip=$(curl -s checkip.amazonaws.com)

```

Create the Azure Kubernetes Cluster:

```bash
az aks create -n demo-aks \
  -g demo-rg \
  -l westeurope \
  --enable-managed-identity \
  -c 1 \
  -s Standard_B2ms \
  --api-server-authorized-ip-ranges $ip \
  --enable-addons http_application_routing
```

Configure `kubectl`:

```bash
az aks get-credentials -n demo-aks -g demo-rg
```

## Attacker host

You will also need a host (virtual machine) with a public IP and open port (to connect the reverse shell too). I would recommend spinning up a virtual machine in Azure.

``` bash
az vm create \
  --resource-group demo-rg \
  -l westeurope \
  --name attack-vm \
  --image UbuntuLTS \
  --admin-username azureuser \
  --generate-ssh-keys

attackIp=$(az vm show -d -g demo-rg -n attack-vm --query publicIps -o tsv)
```

You will be able to access the VM with `ssh azureuser@$attackIp`.

## Sample App

Deploy and patch the sample app:

```bash
kubectl apply -f https://gitlab.com/nico-meisenzahl/hijack-kubernetes/-/raw/main/assets/demo.yaml

kubectl patch ingress sample-app -p '{"metadata":{"annotations":{"nginx.ingress.kubernetes.io/whitelist-source-range":"'$ip'/32"}}}'
```
