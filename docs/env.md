# Spin up the environment

This guide is based on Azure and Azure Kubernetes Service, but should be able to be ported to any other public cloud offering.

## Kubernetes Cluster

Create a Resource Group hosting the environment:

```bash
az group create --name hijack-demo-rg --location westeurope
```

Get your public IP:

```bash
ip=$(curl -s checkip.amazonaws.com)
```

Create the Azure Kubernetes Cluster:

```bash
az aks create -n hijack-demo-aks \
  -g hijack-demo-rg \
  -l westeurope \
  --enable-managed-identity \
  -c 1 \
  -s Standard_B2ms \
  --api-server-authorized-ip-ranges $ip \
  --enable-addons http_application_routing
```

Configure `kubectl`:

```bash
az aks get-credentials -n hijack-demo-aks -g hijack-demo-rg
```

Enable Identity:

```bash
aksId=$(az aks show -g hijack-demo-rg -n hijack-demo-aks --query identityProfile.kubeletidentity.clientId -otsv)
aksRg=$(az aks show -g hijack-demo-rg -n hijack-demo-aks --query nodeResourceGroup -otsv)

az role assignment create \
  --role Contributor \
  --assignee $aksId \
  --resource-group $aksRg
```

## Redis

Create a Redis instance:

```bash
az redis create -g hijack-demo-rg \
  --location westeurope \
  --name hijack-demo-redis \
  --sku Basic \
  --vm-size c0 \
  --enable-non-ssl-port
```

Define firewall rules:

```bash
aksOutboundId=$(az aks show -g hijack-demo-rg -n hijack-demo-aks --query 'networkProfile.loadBalancerProfile.effectiveOutboundIPs[].{id:id}' -otsv)

aksOutboundIp=$(az network public-ip show --ids $aksOutboundId --query ipAddress -otsv)

az redis firewall-rules create -g hijack-demo-rg \
  --name hijack-demo-redis \
  --rule-name aks0access \
  --start-ip $aksOutboundIp \
  --end-ip $aksOutboundIp

az redis firewall-rules create -g hijack-demo-rg \
  --name hijack-demo-redis \
  --rule-name client0access \
  --start-ip $ip \
  --end-ip $ip
```

Add data:

```bash
RedisKey=$(az redis list-keys -g hijack-demo-rg --name hijack-demo-redis --query primaryKey -otsv)
RedisHost=$(az redis show -g hijack-demo-rg --name hijack-demo-redis --query hostName -otsv)

redis-cli -h $RedisHost -a $RedisKey set data "some secret data"
```
## Attacker host

You also need a host (virtual machine) with a public IP address and an open port (for the reverse shell connection).

``` bash
az vm create \
  --resource-group hijack-demo-rg \
  -l westeurope \
  --name hijack-attack-vm \
  --image UbuntuLTS \
  --admin-username azureuser \
  --generate-ssh-keys \
  --public-ip-sku Standard \
  --public-ip-address-allocation static

az vm open-port --port 80,443,1389 --resource-group hijack-demo-rg --name hijack-attack-vm

attackIp=$(az vm show -d -g hijack-demo-rg -n hijack-attack-vm --query publicIps -o tsv)

ssh azureuser@$attackIp 'curl -s https://raw.githubusercontent.com/nmeisenzahl/hijack-kubernetes/main/assets/configure-vm.sh | bash'
```

You now need to download Orca Java and install it on the attacker host. To do so you will have to visit [this](https://www.oracle.com/java/technologies/javase/javase8-archive-downloads.html) URL and download the JDK version "8u201" onto your local machine. Then execute the following commands to configure the attacker machine:

```bash
scp jdk-8u201-linux-x64.tar.gz azureuser@$attackIp:/home/azureuser/log4j-shell-poc/

ssh azureuser@$attackIp 'tar -xvf /home/azureuser/log4j-shell-poc/jdk-8u201-linux-x64.tar.gz --directory ./log4j-shell-poc'
ssh azureuser@$attackIp 'mv /home/azureuser/log4j-shell-poc/jdk1.8.0_201 /home/azureuser/log4j-shell-poc/jdk1.8.0_20'
```

## Sample App

Deploy and patch the sample app:

```bash
kubectl apply -f https://raw.githubusercontent.com/nmeisenzahl/hijack-kubernetes/main/assets/demo.yaml

kubectl patch ingress sample-app -p '{"metadata":{"annotations":{"nginx.ingress.kubernetes.io/whitelist-source-range":"'$ip'/32"}}}'

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: another-secret
  namespace: another-app
type: Opaque
data:
  redisHost: "$(echo $RedisHost | base64)"
  redisKey: "$(echo $RedisKey | base64)"
EOF
```

# Update Source IP

The below steps are only needed when your source IP had changed.

```bash
ip=$(curl -s checkip.amazonaws.com)

az redis firewall-rules create -g hijack-demo-rg \
  --name hijack-demo-redis \
  --rule-name client0access0$RANDOM \
  --start-ip $ip \
  --end-ip $ip

az aks update -n hijack-demo-aks \
  -g hijack-demo-rg \
  --api-server-authorized-ip-ranges $ip

kubectl patch ingress sample-app -p '{"metadata":{"annotations":{"nginx.ingress.kubernetes.io/whitelist-source-range":"'$ip'/32"}}}'
```
