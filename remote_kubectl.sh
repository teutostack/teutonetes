#!/bin/sh

if [ "$1" != "" ]
then
  cluster_name="$1"
else
  echo "Error, please provide a cluster name."
  exit 1
fi
if [ "$2" != "" ]
then
  edgenode="$2"
else
  echo "Error, please provide the edge node's name."
  exit 1
fi
floating_ip=$(openstack server list -c Name -c Networks | grep $edgenode | cut -d'|' -f3 | awk {'print $2'})

echo "Creating kubectl config for cluster."
mkdir -p ~/cluster/$cluster_name/cert
admin_pem="admin-$edgenode"

ansible-playbook --timeout=30 --ssh-extra-args="-o StrictHostKeyChecking=no" --limit='bastion' -i ~/cluster/$cluster_name/inventory.cfg \
  --private-key=~/cluster/$cluster_name/private.key \
  -e clustername=$cluster_name -e pemname=$admin_pem -b --become -u ubuntu /tools/aftermath.yml

kubectl config set-cluster $cluster_name \
  --certificate-authority=/root/cluster/$cluster_name/cert/ca.pem \
  --embed-certs=true \
  --server=https://$floating_ip:6443

kubectl config set-credentials admin-$cluster_name \
  --client-certificate=/root/cluster/$cluster_name/cert/$admin_pem.pem \
  --client-key=/root/cluster/$cluster_name/cert/$admin_pem-key.pem \
  --embed-certs=true

kubectl config set-context $cluster_name \
  --cluster=$cluster_name \
  --user=admin-$cluster_name

echo "Switching to context: \"context-$cluster_name\"."
kubectl config use-context $cluster_name
echo "kubectl configuration done. Test connection..."
kubectl get componentstatuses
if [ $? -ne 0 ]
then
  echo "Error."
fi

