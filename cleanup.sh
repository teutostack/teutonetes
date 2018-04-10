#!/bin/sh

if [ "$1" != "yirrmi" ]
then
  echo "Please write \"cleanup yirrmi <CLUSTERNAME>\" (yes, you really, really mean it) as parameter."
  exit 1
fi
echo "Prepare cleanup."
if [ "$2" == "" ]
then
  echo "Error, no clustername provided."
  exit 1
fi

cluster_name="$2"
echo "Getting some component IDs of cluster \"$cluster_name\"..."
network_id=$(openstack network list | grep $cluster_name | awk {'print $2'})
subnet_id=$(openstack subnet list | grep $cluster_name | awk {'print $2'})
floating_ip=$(openstack server list -c Name -c Networks | grep $cluster_name-master-1 | cut -d'|' -f3 | awk {'print $2'})
router_id=$(openstack router list | grep $cluster_name | awk {'print $2'})
secgroup_id=$(openstack security group list | grep $cluster_name | cut -d ' ' -f 2)
current_context=$(kubectl config current-context)
echo "Cluster \"$cluster_name\" will be deleted. Last chance to change your mind... 6..."
for i in $(seq 5 -1 1)
do
  sleep 1
  echo "$i..."
done
echo "0"
echo "No? Okay, here we go!"
sleep 1

echo "Checking for lbaas and volumes."
kubectl config use-context context-$cluster_name
# TODO: LBaaS should have a name matching the cluster.
#lbaas_public=$(kubectl get svc -o=custom-columns="<none>":.status.loadBalancer.ingress --all-namespaces | grep -v "<none>" | cut -d':' -f2 | tr -d ])
if [ "$lbaas_public" != "" ]
then
  echo "Loadbalancer found. Deleting Pool, Listeners and Loadbalancer."
  lbaas_private=$(openstack floating ip list -c "Floating IP Address" -c "Fixed IP Address" | grep $lbaas_public | awk {'print $4 '})
  lbaas_name=$(neutron lbaas-loadbalancer-list | grep $lbaas_private | awk {'print $4'})
  neutron lbaas-pool-delete $(neutron lbaas-pool-list | grep $lbaas_name | awk {'print $2'})
  neutron lbaas-listener-delete $(neutron lbaas-listener-list | grep $lbaas_name | awk {'print $2'})
  neutron lbaas-loadbalancer-delete $lbaas_name
fi

if [ "$(kubectl config view | grep $cluster_name)" != "" ]
then
	echo "Deleting Kubectl configuration."
	kubectl config unset users.admin-$cluster_name
	kubectl config delete-context $cluster_name
	kubectl config delete-cluster $cluster_name
fi

echo "Delete OpenStack components."
echo "Some ports won't be deleted, don't panic. We get them later."
openstack server delete $(openstack server list | grep $cluster_name | awk {'print $2'})
echo "Servers deleted."

openstack floating ip delete $floating_ip
openstack port delete $(openstack port list | grep $subnet_id | awk {'print $2'})
openstack keypair delete $(openstack keypair list | grep $cluster_name | awk {'print $2'})
openstack router remove subnet $router_id $subnet_id
openstack router delete $router_id
echo "Router deleted."
openstack subnet delete $subnet_id
echo "Subnet deleted."
openstack network delete $network_id
echo "Network deleted."
openstack security group delete $secgroup_id
deldir="$cluster_name.$(date +%Y-%m-%d_%H:%M:%S)"
echo "Cluster deleted. Move directory to \".deleted/$deldir\"."
if [ ! -d ~/cluster/.deleted ]
then
  mkdir -p ~/cluster/.deleted
fi
mv ~/cluster/$cluster_name ~/cluster/.deleted/$deldir
echo "Cleanup done."
