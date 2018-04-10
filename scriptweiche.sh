#!/usr/bin/env sh

if [ "$#" -lt 1 ]; then
    echo "Teutonetes Container Version: $(cat /tools/VERSION)."
    echo "teutonetes [cluster_name] create || deploy || openstack [args] || neutron [args] || cleanup yirrmi || shell || ssh || examples"
    exit 1
fi

cluster_name="$1"
shift 
ARGS="$@"
CMD=""
if [ "$cluster_name" == "shell" ] && [ "$ARGS" == "" ]
then
	CMD="simpleshell"
else	
	CMD="$1"
	shift
	ARGS="$@"
fi

case "$CMD" in
	join)		source /root/cluster/$cluster_name/.config && join $cluster_name;;

	create)
			if [ "$1" == "node" -o "$1" == "master" ]
			then
				count="$2"
				type="$1"
				/tools/teutonetes $cluster_name onlynodes $count $type
#				source /root/cluster/$cluster_name/.config && /tools/generate_inventory.sh $cluster_name
			else
				/tools/teutonetes $cluster_name
				
				chown $UID:$GID /root/cluster/$cluster_name/private.key
				chmod 600 /root/cluster/$cluster_name/private.key
			fi;;
			
	add) 		source /root/cluster/$cluster_name/.config && netesscale $cluster_name add $ARGS;;
	delete) 	source /root/cluster/$cluster_name/.config && netesdelete $cluster_name delete $ARGS;;
	deploy)		source /root/cluster/$cluster_name/.config && deploy $cluster_name;;
	openstack) 	source /root/cluster/$cluster_name/.config && openstack $ARGS;;
	neutron) 	source /root/cluster/$cluster_name/.config && neutron $ARGS;;
	cleanup) 	source /root/cluster/$cluster_name/.config && cleanup $1 $cluster_name;;
	## For debug purposes.
	ssh)		source /root/cluster/$cluster_name/.config && ssh -i /root/cluster/$cluster_name/private.key \
	                  ubuntu@$(openstack server list -c Name -c Networks | grep $cluster_name-node-1 | cut -d'|' -f3 | awk {'print $2'});;
	shell)		source /root/cluster/$cluster_name/.config && /bin/sh $ARGS;;
	simpleshell) 
			echo "No credentials loaded - run \"teutonetes [cluster_name] credshell\" to load the credentials." \
			  && /bin/sh $ARGS;;
	## Not so important functions
	examples)	echo "Those commands will create a traefik LBaaS with ingress controller and a guestbook with PVC."
			echo "To use the guestbook you need the traefik LBaaS. Yaml files are in the \"/examples/\"-directory."
			echo "For that use \"teutonetes [cluster_name] shell\" to enter the container."
			echo "To run the examples automatically run:" 
			echo "\"teutonetes [cluster_name] traefikex\" and \"teutonetes [cluster_name] guestbookex\"";;
	
	traefikex) 	source /root/cluster/$cluster_name/.config && /examples/traefikex/00-traefik-example.sh $cluster_name;;
	guestbookex) 	source /root/cluster/$cluster_name/.config && /examples/guestbookex/00-pvc-guestbook-example.sh $cluster_name;;
	## Not so important functions end
	*) 		echo "Teutonetes Container Version: $(cat /tools/VERSION)."
			echo "teutonetes [cluster_name] create || deploy || openstack [args]"
	   		echo -e "\nRun the following command to set up a proper alias:"
	   		echo "alias teutonetes=\"docker run -ti --rm -v ~/teutonetes/cluster:/root/cluster/ -v ~/.kube:/root/.kube/ \
	    		  -w /root/ -e UID=$(id -u) -e GID=$(id -g) registry-gitlab.teuto.net/technik/teutonetes:latest\""
	   		echo -e "\nConfig model: "
	   		cat /root/deploy.example ;;
esac
