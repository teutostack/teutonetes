#!/bin/sh
if [ "$1" != "" ]
then
  cluster_name="$1"
else
  cluster_name="$CLUSTER_NAME"
fi

echo "Getting IDs of some components."
network_id=$(openstack network list | grep $cluster_name | awk {'print $2'})
subnet_id=$(openstack subnet list | grep $cluster_name | awk {'print $2'})
#subnet_id=$(openstack network list -c ID -c Subnets | grep $OS_NETWORK_ID | awk {'print $4'})
extern_id=$(openstack network list | grep extern | awk {'print $2'})
edgenode="$cluster_name-master-1"
floating_ip=$(openstack server list -c Name -c Networks | grep $edgenode | cut -d'|' -f3 | awk {'print $2'})

#generate_inventory $cluster_name
####### INVENTORY GENERATOR
echo "Generate inventory.cfg template."

mkdir -p ~/cluster/$cluster_name/
cat - > ~/cluster/$cluster_name/inventory.cfg <<EOF
[all]


# ## configure a bastion host if your nodes are not directly reachable
bastion ansible_ssh_host=$floating_ip ansible_host=ubuntu ansible_ssh_common_args='-o StrictHostKeyChecking=no'

[kube-master]


[etcd]


[kube-node]


[k8s-cluster:children]
kube-node
kube-master
EOF

echo "Filling template with cluster data."
let m=0
let e=0
for n in $(openstack server list -c Name | grep $cluster_name | cut -d' ' -f2)
do
  node_name=$n
  node_ip=$(openstack server show $node_name | grep addresses | cut -d'=' -f2 | cut -d',' -f1 | cut -d ' ' -f0)
  sed -i '/^\[all\]/a\'"$node_name"' ansible_ssh_host='"$node_ip"' ansible_ssh_common_args='\''-o StrictHostKeyChecking=no'\''' \
  ~/cluster/$cluster_name/inventory.cfg
  if echo $n | grep -q "master"
  then
    sed -i '/^\[kube-master\]/a\'"$node_name"'' ~/cluster/$cluster_name/inventory.cfg
  else
    sed -i '/^\[kube-node\]/a\'"$node_name"'' ~/cluster/$cluster_name/inventory.cfg
  fi
  if [ $e -lt 3 ]
  then
    sed -i '/^\[etcd\]/a\'"$node_name"'' ~/cluster/$cluster_name/inventory.cfg
  fi
  
  let "m++"
  let "e++"
done

##########################

echo "copy group_vars directory into cluster directory"
cp -r /opt/kubespray/inventory/sample/group_vars ~/cluster/$cluster_name/.

echo "Update kubespray configuration to take the edge node's floating ip into consideration when creating the ssl certificates."
sed -i "/# supplementary_addresses_in_ssl_keys:/a supplementary_addresses_in_ssl_keys: \n - $floating_ip" ~/cluster/$cluster_name/group_vars/k8s-cluster.yml

echo "Prepare configuration of OpenStack as cloud-provider."
sed -i "/## To enable automatic floating ip provisioning, specify a subnet./a openstack_lbaas_floating_network_id: $extern_id"  ~/cluster/$cluster_name/group_vars/all.yml
sed -i "/#openstack_lbaas_subnet_id: \"Neutron subnet ID \(not network ID\) to create LBaaS VIP\"/a openstack_lbaas_enabled: True\n\openstack_lbaas_subnet_id: $subnet_id" ~/cluster/$cluster_name/group_vars/all.yml

echo "Scanning host key of edge node."
if [ ! -f ~/.ssh/known_hosts ]
then
  mkdir -p ~/.ssh/
  echo "" > ~/.ssh/known_hosts
fi
ssh-keyscan -H $floating_ip >> ~/.ssh/known_hosts 

echo "Preparing the Nodes for kubespray playbook (sudo-privileges, python-installation, etc)"
export ANSIBLE_HOST_KEY_CHECKING=False

ansible-playbook --timeout=30 --ssh-extra-args="-o StrictHostKeyChecking=no -o ProxyCommand=\"ssh -i ~/cluster/$cluster_name/private.key -W %h:%p -q ubuntu@$floating_ip\"" \
  -e ansible_host=$floating_ip -e ansible_user=ubuntu -i ~/cluster/$cluster_name/inventory.cfg \
  --private-key=~/cluster/$cluster_name/private.key \
  --limit='all:!bastion' -b --become -u ubuntu /tools/groundwork.yml


## Kubernetes deployment via kubespray
echo "Install Kubernetes."
ansible-playbook --ssh-extra-args="-o StrictHostKeyChecking=no -o ProxyCommand=\"ssh -i ~/cluster/$cluster_name/private.key -W %h:%p -q ubuntu@$floating_ip\"" \
  -b --become-user=root -u ubuntu -i ~/cluster/$cluster_name/inventory.cfg -e ansible_user=ubuntu -e ansible_host=$floating_ip \
  --private-key=~/cluster/$cluster_name/private.key \
  /opt/kubespray/cluster.yml
echo "Kubernestes Cluster ready... Continue with configurations."
sleep 1
echo "Downloading helm-installer. Tiller is running but there is no client."
sleep 2
cp /usr/bin/helm ~/cluster/
ansible-playbook --timeout=30 --ssh-extra-args="-o StrictHostKeyChecking=no" -i ~/cluster/$cluster_name/inventory.cfg \
  -e ansible_host=$floating_ip -e ansible_user=ubuntu \
  --private-key=~/cluster/$cluster_name/private.key \
  -b --become -u ubuntu /tools/helm_n_tiller.yml

echo "Preparing certificates and credentials to configure remotely access to the kubernetes cluster via local kubectl."
mkdir -p ~/cluster/$cluster_name/cert

echo "Creating kubectl config for cluster."
curcon=$(kubectl config current-context)

echo "Saved current used context \"$curcon\" to know where to return."
remote_kubectl $cluster_name $edgenode

chown -R $UID:$GID ~/cluster/$cluster_name

echo "Config adminstate for dashboard usage."
kubectl config use-context $cluster_name
kubectl create clusterrolebinding --user system:serviceaccount:kube-system:kubernetes-dashboard kube-system-cluster-admin --clusterrole cluster-admin 

echo "Create Storageclass \"teutostack\"."
cat << EOF | kubectl create -f -
kind: StorageClass
apiVersion: storage.k8s.io/v1beta1
metadata:
  name: teutostack
provisioner: kubernetes.io/cinder
EOF

echo "Return to context \"$curcon\"."
kubectl config use-context $curcon
echo "To use kubectl remotely use context \" context-$cluster_name \" ( \" kubectl config use-context $cluster_name \")."
echo "To access the kubernetes dashboard use \" kubectl proxy & \"."
echo "Visit in your browser of choice this site:"
echo "\" http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/ \""

