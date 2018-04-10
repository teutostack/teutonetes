echo "Generate inventory.cfg template."

cluster_name="$1"
edgenode="$cluster_name-master-1"
floating_ip=$(openstack server list -c Name -c Networks | grep $edgenode | cut -d'|' -f3 | awk {'print $2'})

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

until openstack server list -c Status | [ "`grep -c "BUILD"`" -eq "0" ]; do echo "Server is still building.." ; done

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
