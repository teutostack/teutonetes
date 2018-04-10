#!/bin/sh

if [ "$#" -ne 1 ]
then
	echo "Join needs exactly one argument!";
	exit 1;
fi

cluster_name="$1"
edgenode="$cluster_name-master-1"
floating_ip=$(openstack server list -c Name -c Networks | grep $edgenode | cut -d'|' -f3 | awk {'print $2'})

echo "Scanning host key of edge node."
if [ ! -f ~/.ssh/known_hosts ]
then
	mkdir -p ~/.ssh/
	echo "" > ~/.ssh/known_hosts
fi
ssh-keyscan -H $floating_ip >> ~/.ssh/known_hosts


ansible-playbook --ssh-extra-args="-o StrictHostKeyChecking=no -o ProxyCommand=\"ssh -i ~/cluster/$cluster_name/private.key -W %h:%p -q ubuntu@$floating_ip\"" \
	  -b --become-user=root -u ubuntu -i ~/cluster/$cluster_name/inventory.cfg -e ansible_user=ubuntu -e ansible_host=$floating_ip \
	    --private-key=~/cluster/$cluster_name/private.key \
	      /opt/kubespray/cluster.yml
