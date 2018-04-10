#!/bin/sh

cluster_name=""
if [ "$1" != "" ]
then
  cluster_name="$1"
else
  echo "Error, no cluster name provided."
  exit 1
fi
shift
ARGS="$@"

for name in $ARGS
do
  node_name="$name"
  node_ip=$(openstack server show $node_name | grep addresses | cut -d'=' -f2 | cut -d',' -f1 | cut -d ' ' -f0)
  sed -i '/^\[all\]/a\'"$node_name"' ansible_ssh_host='"$node_ip"' ansible_ssh_common_args='\''-o StrictHostKeyChecking=no'\''' \
  ~/cluster/$cluster_name/inventory.cfg
  sed -i '/^\[kube-node\]/a\'"$node_name"'' ~/cluster/$cluster_name/inventory.cfg
done

ansible-playbook --ssh-extra-args="-o StrictHostKeyChecking=no -o ProxyCommand=\"ssh -i ~/cluster/$cluster_name/private.key -W %h:%p -q ubuntu@$floating_ip\"" \
  -b --become-user=root -u ubuntu -i ~/cluster/$cluster_name/inventory.cfg -e ansible_user=ubuntu -e ansible_host=$floating_ip \
  --private-key=~/cluster/$cluster_name/private.key \
  /opt/kubespray/scale.yml
