package main

import (
	"bytes"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	//"flag"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	//"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	//	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"os/user"
	//"path/filepath"
	"bufio"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"
)

func main() {

	syscall.Umask(0000)

	configName := "./config"

	// We got some troubles here: usr is (at least for me) nil. Due to that there is a panic.
	usr := getUser()
	var countNodes int
	var typeNodes string
	onlynodes := false
	al := len(os.Args)
	if al > 1 {
		argument := os.Args[1:]
		// See usr declaration above
		configName = fmt.Sprintf("%s/cluster/%s/deploy-config", usr.HomeDir, argument[0])
		// Case only node create
		if al == 5 {
			if argument[1] == "onlynodes" {
				onlynodes = true
				if argument[2] == "" {
					fmt.Println("when \"onlynodes\" is set you must also give the number of nodes and the type")
					os.Exit(1)
				} else {
					countNodes, _ = strconv.Atoi(argument[2])
					typeNodes = argument[3]
				}

			} else {
				fmt.Println("Exiting: Something went wrong with the input.")
				os.Exit(1)
			}
			var i int
			//			count, _ = fmt.Sscan(argument[2], &i)
			_, _ = fmt.Sscan(argument[2], &i)
		}
	} else {
		fmt.Println("Exiting: Something went wrong with the input.")
		os.Exit(1)
	}

	conf := readConfig(configName)

	dat, err := ioutil.ReadFile("./userdata-ssh")
	if err != nil {
		fmt.Println("Error while reading file!")
		fmt.Println(err)
	}

	// Either go creates network, subnet, router and nodes ("nodesonly"=false) or only nodes for scaling purposes ("nodesonly"=true)
	//boolPtr := flag.Bool("nodesonly", false, "create only nodes")
	// Case "nodesonly" is true: How many nodes should be created?
	//numbPtr := flag.Int("count", 0, "amount of nodes")
	//flag.Parse()

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: conf.Credentials.Auth_Url,
		Username:         conf.Credentials.Username,
		Password:         conf.Credentials.Password,
		TenantID:         conf.Credentials.ProjectID,
		DomainName:       conf.Credentials.DomainName,
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		fmt.Println("Error while authenticating!")
		fmt.Println(err)
		// If we could not authenticate, stop right there!
		os.Exit(1)
	}

	//Compute service struct
	compute_client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		fmt.Println("Error while creating Compute Client!")
		fmt.Println(err)
	}

	network_client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: "RegionOne",
	})
	if err != nil {
		fmt.Println("Error while creating Network Client!")
		fmt.Println(err)
	}
	//Usual way.
	if onlynodes {
		// num= current counter for node name; maxnum= number for the last node.
		//start := conf.Nodes.NumNodes
		dat = configTemplate("./userdata-ssh", conf)
		createNodes(dat, network_client, compute_client, &conf, configName, true, countNodes, typeNodes)

		writeConfig(configName, conf)

		getConfig(conf)

	} else {
		fmt.Println("Create Key and keypair..")
		err := generateKey(compute_client, &conf)
		if err != nil {
			fmt.Println("Error!", err)
			os.Exit(1)
		}

		fmt.Println("Create Network..")
		createNetwork(network_client, &conf, configName)

		fmt.Println("config after network:")

		fmt.Println("Creating subnet..")
		subnet, err := createSubnet(&conf, network_client, configName)
		if err != nil {
			fmt.Println("failed to create subnet!")
			fmt.Println(err)
			os.Exit(1)
		}
		_ = subnet

		fmt.Println("Create Router..")
		createRouter(&conf, network_client, configName)

		dat = configTemplate("./userdata-ssh", conf)

		//	createJumphost(dat, compute_client, &conf, configName)
		//	attachFIP(conf.Nodes.ID, &conf, compute_client)
		fmt.Println("Create Security Group..")
		createSecGroup(&conf, network_client)
		fmt.Println("Create instances..")
		createNodes(dat, network_client, compute_client, &conf, configName, false, 0, "")
		fmt.Println("Write config..")
		writeConfig(configName, conf)

		getConfig(conf)
	}
}

type M map[string]interface{}

func getUser() *user.User {
	usr, _ := user.Current()
	if usr == nil {
		usr = &user.User{
			Uid:      strconv.Itoa(os.Getuid()),
			Gid:      strconv.Itoa(os.Getgid()),
			Username: os.Getenv("USER"), // or USERNAME on Windows
			HomeDir:  os.Getenv("HOME"), // or HOMEDRIVE+HOMEDIR on windows
		}
	}
	return usr
}

func generateKey(compute_client *gophercloud.ServiceClient, conf *tomlConfig) error {

	privatekey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println(err)
		return err
	}

	conf.Security.Keyname = fmt.Sprintf(conf.Credentials.Clustername + "-keypair")

	var publickey *rsa.PublicKey
	publickey = &privatekey.PublicKey

	//	ex, err := os.Executable()
	//	if err != nil {
	//		return err
	//	}
	//exPath := filepath.Dir(ex)

	usr := getUser()
	//	_ = os.MkdirAll(usr.HomeDir+"/teutonetes/cluster/"+conf.Credentials.Clustername, 0755)

	privatekeyfile, err := os.Create(usr.HomeDir + "/cluster/" + conf.Credentials.Clustername + "/private.key")
	if err != nil {
		fmt.Println(err)
		return err
	}
	privatekeyencoder := gob.NewEncoder(privatekeyfile)
	privatekeyencoder.Encode(privatekey)
	privatekeyfile.Close()

	publickeyfile, err := os.Create(usr.HomeDir + "/cluster/" + conf.Credentials.Clustername + "/public.key")
	if err != nil {
		fmt.Println(err)
		return err
	}

	publickeyencoder := gob.NewEncoder(publickeyfile)
	publickeyencoder.Encode(publickey)
	publickeyfile.Close()

	pemfile, err := os.Create(usr.HomeDir + "/cluster/" + conf.Credentials.Clustername + "/private.key")
	if err != nil {
		fmt.Println(err)
	}

	var pemkey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privatekey),
	}

	err = pem.Encode(pemfile, pemkey)
	if err != nil {
		fmt.Println(err)
	}

	pemfile.Close()

	pemfile, err = os.Create(usr.HomeDir + "/cluster/" + conf.Credentials.Clustername + "/public.key")
	if err != nil {
		fmt.Println(err)
	}

	pkipub, _ := x509.MarshalPKIXPublicKey(publickey)

	pubpemkey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pkipub,
	})

	ioutil.WriteFile(usr.HomeDir+"/cluster/"+conf.Credentials.Clustername+"/public.key", pubpemkey, 0644)

	pub, err := ssh.NewPublicKey(publickey)
	pubBytes := ssh.MarshalAuthorizedKey(pub)
	pk := string(pubBytes)

	kp, err := keypairs.Create(compute_client, keypairs.CreateOpts{
		Name:      fmt.Sprintf(conf.Credentials.Clustername + "-keypair"),
		PublicKey: pk,
	}).Extract()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Keypair is: ", kp)

	return nil
}

func getConfig(conf tomlConfig) {
	var buffer bytes.Buffer
	buffer.WriteString("export OS_AUTH_URL=\"{{.authurl}}\"\nexport OS_PROJECT_NAME=\"{{.projectname}}\"\nexport OS_PASSWORD=\"{{.ospassword}}\"\nexport OS_USERNAME=\"{{.username}}\"\nexport OS_NETWORK_ID=\"{{.networkid}}\"\nexport OS_FLAVOR=\"{{.flavorname}}\"\nexport OS_IMAGE=\"{{.imagename}}\"\nexport JUMPHOST_NAME=\"{{.jumphostname}}\"\nexport OS_REGION_NAME=\"RegionOne\"\nexport OS_TENANT_ID=\"{{.projectid}}\"\nexport CLUSTER_NAME=\"{{.clustername}}\"")
	//temp := "OS_AUTH_URL={{.authurl}}\nOS_PROJECT_NAME={{.projectname}}\nOS_PASSWORD={{.ospassword}}\nOS_USERNAME={{.username}}\nOS_NETWORK_ID={{.networkid}}\nOS_FLAVOR={{.flavorname}}\nOS_IMAGE={{.imagename}}\nJUMPHOST_NAME={{.jumphostname}}\nOS_REGION_NAME=RegionOne\nOS_TENANT_ID={{.projectid}}\nCLUSTER_NAME={{.clustername}}"

	if strings.HasSuffix(conf.Credentials.Auth_Url, "v3") {
		buffer.WriteString("\nexport OS_USER_DOMAIN_NAME=Default\nexport OS_DOMAIN_NAME=Default\nexport OS_IDENTITY_API_VERSION=3")
	}

	temp := buffer.String()

	//	ex, err := os.Executable()
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//path := filepath.Dir(ex)

	usr := getUser()

	path := fmt.Sprintf("%s/cluster/%s", usr.HomeDir, conf.Credentials.Clustername)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println(path)
		err = os.MkdirAll(path, 0777)
		fmt.Println(err)
	}

	t, err := template.New("").Parse(temp)
	if err != nil {
		fmt.Println(err)
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, M{
		"authurl":      conf.Credentials.Auth_Url,
		"projectname":  conf.Credentials.ProjectName,
		"ospassword":   conf.Credentials.Password,
		"username":     conf.Credentials.Username,
		"networkid":    conf.Network.NetworkID,
		"flavorname":   conf.Nodes.FlavorName,
		"imagename":    conf.Nodes.ImageName,
		"jumphostname": conf.Nodes.Name,
		"clustername":  conf.Credentials.Clustername,
		"projectid":    conf.Credentials.ProjectID,
		"floatingip":   conf.Master.FloatingIP,
	})
	if err != nil {
		fmt.Println(err)
	}

	f, err := os.Create(fmt.Sprintf("%s/.config", path))
	defer f.Close()
	if err != nil {
		fmt.Println("cant create file!")
		fmt.Println(err)
	}
	err = ioutil.WriteFile(fmt.Sprintf(path+"/.config"), tpl.Bytes(), 0777)
	if err != nil {
		fmt.Println("Cant write config")
		fmt.Println(err)
	}

}

func configTemplate(path string, conf tomlConfig) []byte {
	t := template.New("userdata-ssh")

	t, err := t.ParseFiles(path)
	if err != nil {
		fmt.Println(err)
	}

	usr := getUser()
	dat, err := ioutil.ReadFile(usr.HomeDir + "/cluster/" + conf.Credentials.Clustername + "/private.key")
	if err != nil {
		fmt.Println("cant read key")
	}

	newString := strings.Replace(string(dat), "\n", "\n\n", -1)

	var tpl bytes.Buffer
	err = t.Execute(&tpl, M{
		"authurl":      conf.Credentials.Auth_Url,
		"projectname":  conf.Credentials.ProjectName,
		"ospassword":   conf.Credentials.Password,
		"username":     conf.Credentials.Username,
		"networkid":    conf.Network.NetworkID,
		"flavorname":   conf.Nodes.FlavorName,
		"imagename":    conf.Nodes.ImageName,
		"jumphostname": conf.Nodes.Name,
		"clustername":  conf.Credentials.Clustername,
		"projectid":    conf.Credentials.ProjectID,
		"privkey":      newString,
	})
	if err != nil {
		fmt.Println(err)
	}

	return tpl.Bytes()
}

//func getServerStatus(compute_client *gophercloud.ServiceClient, conf *tomlConfig) {
//	result := servers.Get(compute_client, conf.Nodes.ID)
//	fmt.Println(result)
//	//	return result.Status
//
//	s := reflect.ValueOf(&result).Elem()
//	typeOfT := s.Type()
//
//	for i := 0; i < s.NumField(); i++ {
//		f := s.Field(i)
//		fmt.Printf("%d: %s %s = %v\n", i,
//			typeOfT.Field(i).Name, f.Type(), f.Interface())
//	}
//
//}

func attachFIP(serverId string, conf *tomlConfig, network_client *gophercloud.ServiceClient, portID string) {

	createOpts := floatingips.CreateOpts{
		FloatingNetworkID: conf.Network.ExternalNetworkID,
	}

	fip, err := floatingips.Create(network_client, createOpts).Extract()
	if err != nil {
		fmt.Println("Could not create Floating IP! Going to try again.")
		fmt.Println(err)
		time.Sleep(time.Second * 5)
		attachFIP(serverId, conf, network_client, portID)
	}

	fmt.Println("Floating IP of Edge-Node is: ", fip.FloatingIP)

	time.Sleep(time.Second * 5)

	updateOpts := floatingips.UpdateOpts{
		PortID: &portID,
	}
	fip, err = floatingips.Update(network_client, fip.ID, updateOpts).Extract()
	if err != nil {
		fmt.Println("Couldn't attach floating ip, will try again.")
		time.Sleep(time.Second * 5)
		attachFIP(serverId, conf, network_client, portID)
	}

	conf.Master.FloatingIP = fip.FloatingIP
}

func createSubnet(conf *tomlConfig, network_client *gophercloud.ServiceClient, configName string) (*subnets.Subnet, error) {

	dhcpEnable := true

	CreateOpts := subnets.CreateOpts{
		Name:      fmt.Sprintf(conf.Credentials.Clustername + "-subnet"),
		NetworkID: conf.Network.NetworkID,
		IPVersion: 4,
		CIDR:      conf.Subnet.CIDR,
		AllocationPools: []subnets.AllocationPool{
			{
				Start: conf.Subnet.AllocationPoolStart,
				End:   conf.Subnet.AllocationPoolEnd,
			},
		},
		DNSNameservers: conf.Subnet.DNSServers,
		EnableDHCP:     &dhcpEnable,
	}
	conf.Subnet.SubnetName = fmt.Sprintf(conf.Credentials.Clustername + "-subnet")
	subnet, err := subnets.Create(network_client, CreateOpts).Extract()
	conf.Network.SubnetID = subnet.ID
	writeConfig(configName, *conf)
	return subnet, err
}

func getAddressbyNetwork(compute_client *gophercloud.ServiceClient, conf tomlConfig, networkname string) map[string]string {
	serverlist := listServers(compute_client)

	mapster := make(map[string]string)

	for i := range serverlist {
		test := serverlist[i].Addresses[networkname]
		b, ok := test.([]interface{})
		if ok == false {
			fmt.Println("error asserting type")
			return nil
		}
		for j := range b {
			c := b[j].(map[string]interface{})
			mapster[serverlist[i].Name] = c["addr"].(string)
		}
	}

	return mapster
}

func readConfig(path string) tomlConfig {
	var config tomlConfig
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		fmt.Println("Cant decode config file")
		fmt.Println(err)
	}

	return config
}

func writeConfig(path string, conf tomlConfig) {
	var config = map[string]interface{}{
		"Subnet": map[string]interface{}{
			"CIDR":                conf.Subnet.CIDR,
			"SubnetName":          conf.Subnet.SubnetName,
			"AllocationPoolStart": conf.Subnet.AllocationPoolStart,
			"AllocationPoolEnd":   conf.Subnet.AllocationPoolEnd,
			"DHCPEnable":          conf.Subnet.DHCPEnable,
			"DNSServers":          conf.Subnet.DNSServers,
		},
		"Security": map[string]interface{}{
			"Keyname":        conf.Security.Keyname,
			"SecurityGroups": conf.Security.SecurityGroups,
		},
		"Network": map[string]interface{}{
			"NetworkName":       conf.Network.NetworkName,
			"NetworkID":         conf.Network.NetworkID,
			"ExternalNetworkID": conf.Network.ExternalNetworkID,
			"SubnetID":          conf.Network.SubnetID,
			"RouterID":          conf.Network.RouterID,
			"AdminState":        conf.Network.AdminState,
		},
		"Router": map[string]interface{}{
			"AdminState": conf.Router.Adminstate,
			"Name":       conf.Router.Name,
			"GatewayID":  conf.Router.GatewayID,
			"TenantID":   conf.Router.TenantID,
			"RouterID":   conf.Router.RouterID,
		},
		"Credentials": map[string]interface{}{
			"ProjectName": conf.Credentials.ProjectName,
			"Password":    conf.Credentials.Password,
			"Auth_Url":    conf.Credentials.Auth_Url,
			"Username":    conf.Credentials.Username,
			"DomainName":  conf.Credentials.DomainName,
			"ProjectID":   conf.Credentials.ProjectID,
			"Clustername": conf.Credentials.Clustername,
			//			"SSHKeyLoc":   conf.Credentials.SSHKeyLoc,
		},
		//		"Jumphost": map[string]interface{}{
		//			"ID":             conf.Jumphost.ID,
		//		},
		"Userdata": map[string]interface{}{
			"Userdata": conf.Userdata.Userdata,
		},
		"Nodes": map[string]interface{}{
			"Nodes":      conf.Nodes.Nodes,
			"NumNodes":   conf.Nodes.NumNodes,
			"Name":       conf.Nodes.Name,
			"FlavorName": conf.Nodes.FlavorName,
			"ImageName":  conf.Nodes.ImageName,
		},
		"Master": map[string]interface{}{
			"Nodes":      conf.Master.Nodes,
			"NumMasters": conf.Master.NumMasters,
			"Name":       conf.Master.Name,
			"FlavorName": conf.Nodes.FlavorName,
			"ImageName":  conf.Nodes.ImageName,
			"FloatingIP": conf.Master.FloatingIP,
		},
	}

	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(config); err != nil {
		fmt.Println("Couldn't write config.")
		fmt.Println(err)
		return
	} else {
		//	fmt.Println(buf.String())
		usr := getUser()
		ioutil.WriteFile(fmt.Sprintf("%s/cluster/%s/deploy-config", usr.HomeDir, conf.Credentials.Clustername), buf.Bytes(), 0644)
		fmt.Println("Config updated.")
	}

}

func getProvider(id_endpoint, username, password, tenant_id, domain_name string) gophercloud.AuthOptions {
	return gophercloud.AuthOptions{
		IdentityEndpoint: id_endpoint,
		Username:         username,
		Password:         password,
		TenantID:         tenant_id,
		DomainName:       domain_name,
	}
}

func createSecGroup(config *tomlConfig, compute_client *gophercloud.ServiceClient) error {
	opts0 := groups.CreateOpts{
		Name:        fmt.Sprintf("%s", config.Credentials.Clustername),
		TenantID:    fmt.Sprintf("%s", config.Credentials.ProjectID),
		Description: fmt.Sprintf("%s-default security group", config.Credentials.Clustername),
	}
	group, err := groups.Create(compute_client, opts0).Extract()
	if err != nil {
		fmt.Println("Error creating Security Group!")
		fmt.Println(err)
		fmt.Println("%v", err)
	}

	opts := rules.CreateOpts{
		SecGroupID:     group.ID,
		Direction:      rules.DirIngress,
		EtherType:      rules.EtherType4,
		Protocol:       rules.ProtocolTCP,
		PortRangeMin:   22,
		PortRangeMax:   22,
		RemoteIPPrefix: "0.0.0.0/0",
	}

	_, err = rules.Create(compute_client, opts).Extract()

	if err != nil {
		fmt.Println("failed to create rule for security group %s: %s", group.ID, err)
	}

	opts = rules.CreateOpts{
		SecGroupID:     group.ID,
		Direction:      rules.DirEgress,
		EtherType:      rules.EtherType4,
		Protocol:       rules.ProtocolTCP,
		PortRangeMin:   22,
		PortRangeMax:   22,
		RemoteIPPrefix: "0.0.0.0/0",
	}

	_, err = rules.Create(compute_client, opts).Extract()

	if err != nil {
		fmt.Println("failed to create rule for security group %s: %s", group.ID, err)
	}
	opts = rules.CreateOpts{
		SecGroupID: group.ID,
		Direction:  rules.DirIngress,
		EtherType:  rules.EtherType4,
		Protocol:   rules.ProtocolICMP,
		//		PortRangeMin: 1,
		//		PortRangeMax: 65535,
		RemoteIPPrefix: "0.0.0.0/0",
	}

	_, err = rules.Create(compute_client, opts).Extract()

	if err != nil {
		fmt.Println("failed to create rule for security group %s: %s", group.ID, err)
	}

	opts = rules.CreateOpts{
		SecGroupID: group.ID,
		Direction:  rules.DirEgress,
		EtherType:  rules.EtherType4,
		Protocol:   rules.ProtocolICMP,
		//		PortRangeMin: 1,
		//		PortRangeMax: 65535,
		RemoteIPPrefix: "0.0.0.0/0",
	}

	_, err = rules.Create(compute_client, opts).Extract()

	if err != nil {
		fmt.Println("failed to create rule for security group %s: %s", group.ID, err)
	}

	opts = rules.CreateOpts{
		SecGroupID:     group.ID,
		Direction:      rules.DirIngress,
		EtherType:      rules.EtherType4,
		Protocol:       rules.ProtocolTCP,
		PortRangeMin:   6443,
		PortRangeMax:   6443,
		RemoteIPPrefix: "0.0.0.0/0",
	}

	_, err = rules.Create(compute_client, opts).Extract()

	if err != nil {
		fmt.Println("failed to create rule for security group %s: %s", group.ID, err)
	}

	opts = rules.CreateOpts{
		SecGroupID:     group.ID,
		Direction:      rules.DirEgress,
		EtherType:      rules.EtherType4,
		Protocol:       rules.ProtocolTCP,
		PortRangeMin:   6443,
		PortRangeMax:   6443,
		RemoteIPPrefix: "0.0.0.0/0",
	}

	_, err = rules.Create(compute_client, opts).Extract()

	if err != nil {
		fmt.Println("failed to create rule for security group %s: %s", group.ID, err)
	}
	//etcd
	opts = rules.CreateOpts{
		SecGroupID:     group.ID,
		Direction:      rules.DirIngress,
		EtherType:      rules.EtherType4,
		Protocol:       rules.ProtocolTCP,
		PortRangeMin:   2379,
		PortRangeMax:   2379,
		RemoteIPPrefix: "0.0.0.0/0",
	}

	_, err = rules.Create(compute_client, opts).Extract()

	if err != nil {
		fmt.Println("failed to create rule for security group %s: %s", group.ID, err)
	}

	//etcd
	opts = rules.CreateOpts{
		SecGroupID:     group.ID,
		Direction:      rules.DirIngress,
		EtherType:      rules.EtherType4,
		Protocol:       rules.ProtocolTCP,
		PortRangeMin:   2380,
		PortRangeMax:   2380,
		RemoteIPPrefix: "0.0.0.0/0",
	}

	_, err = rules.Create(compute_client, opts).Extract()

	if err != nil {
		fmt.Println("failed to create rule for security group %s: %s", group.ID, err)
	}

	config.Security.SecurityGroups = []string{group.ID}
	return nil
}

func createPort(config *tomlConfig, nodeName string, compute_client *gophercloud.ServiceClient) string {

	//	fmt.Printf("NetworkID: %s ", config.Network.NetworkID)
	//	fmt.Printf("SubnetID: %s ", config.Network.SubnetID)
	//	fmt.Println(fmt.Sprintf("%s-port", config.Credentials.Clustername))

	adminState := true

	opts := ports.CreateOpts{
		NetworkID:    config.Network.NetworkID,
		Name:         fmt.Sprintf("%s-port"),
		AdminStateUp: &adminState,
		FixedIPs:     []ports.IP{ports.IP{SubnetID: config.Network.SubnetID}},
		AllowedAddressPairs: []ports.AddressPair{
			ports.AddressPair{IPAddress: "10.233.0.0/18"},
			ports.AddressPair{IPAddress: "10.233.64.0/18"},
		},
		SecurityGroups: &config.Security.SecurityGroups,
	}

	port, err := ports.Create(compute_client, opts).Extract()
	if err != nil {
		fmt.Println("Error creating Port!")
		fmt.Println(err)
	}
	return port.ID
}

func createNodes(userdata []byte, network_client, compute_client *gophercloud.ServiceClient, config *tomlConfig, configName string, addNodes bool, numNodes int, nodeType string) error {
	nodes := make([]string, config.Nodes.NumNodes)
	masters := make([]string, config.Master.NumMasters)

	metadata := make(map[string]map[string]string)
	metadata["node"] = make(map[string]string)
	metadata["master"] = make(map[string]string)
	metadata["node"]["type"] = "node"
	metadata["master"]["type"] = "master"

	//	fmt.Println("Userdata:\n ", string(userdata))
	//	fmt.Println("Start: ", start, " Amount: ", amount, " Maxnum: ", maxnum)

	if addNodes == true {
		//		currentMasters := config.Master.NumMasters
		//		currentNodes := config.Nodes.NumNodes

		if nodeType == "master" {

			for i := 0; i < numNodes; i++ {
				portID := createPort(config, fmt.Sprintf("%s-port-%d", config.Credentials.Clustername, config.Master.NumMasters+i+1), network_client)

				//              config.Nodes.Networks = make([]servers.Network, 1)

				serverCreateOpts := servers.CreateOpts{

					ServiceClient:  compute_client,
					Name:           fmt.Sprintf("%s-master-%d", config.Credentials.Clustername, config.Master.NumMasters+i+1),
					Metadata:       metadata["master"],
					FlavorName:     config.Master.FlavorName,
					ImageName:      config.Master.ImageName,
					SecurityGroups: config.Security.SecurityGroups,
					Networks: []servers.Network{
						//                              servers.Network{UUID: config.Network.NetworkID},
						servers.Network{Port: portID},
					},
					UserData: userdata,
				}

				createOpts := keypairs.CreateOptsExt{
					CreateOptsBuilder: serverCreateOpts,
					KeyName:           config.Security.Keyname,
				}

				server, err := servers.Create(compute_client, createOpts).Extract()
				if err != nil {
					//panic(err.Error)
					fmt.Println(err)
					panic(err.Error)
				}
				fmt.Println("Server ID: ", server.ID)
				config.Master.NumMasters++
				writeConfig(configName, *config)
			}
			return nil
		} else if nodeType == "node" {

			//	if _, err := os.Stat(fmt.Sprintf("~/cluster/%s/inventory.cfg", config.Credentials.Clustername)); os.IsNotExist(err) {
			//		fmt.Println("inventory file not found. Is there a cluster to join?")
			//		return err
			//	}

			inventory, err := os.Open(fmt.Sprintf("/root/cluster/%s/inventory.cfg", config.Credentials.Clustername))
			if err != nil {
				fmt.Println("cant read inventory!")
				return err
			}
			defer inventory.Close()
			scanner := bufio.NewScanner(inventory)
			var lines []string
			var fileContent string
			var ip_add string

			for i := 0; i < numNodes; i++ {
				portID := createPort(config, fmt.Sprintf("%s-port-%d", config.Credentials.Clustername, config.Nodes.NumNodes+i+1), network_client)

				//		config.Nodes.Networks = make([]servers.Network, 1)

				serverCreateOpts := servers.CreateOpts{

					ServiceClient:  compute_client,
					Name:           fmt.Sprintf("%s-node-%d", config.Credentials.Clustername, config.Nodes.NumNodes+i+1),
					Metadata:       metadata["node"],
					FlavorName:     config.Nodes.FlavorName,
					ImageName:      config.Nodes.ImageName,
					SecurityGroups: config.Security.SecurityGroups,
					Networks: []servers.Network{
						//				servers.Network{UUID: config.Network.NetworkID},
						servers.Network{Port: portID},
					},
					UserData: userdata,
				}

				createOpts := keypairs.CreateOptsExt{
					CreateOptsBuilder: serverCreateOpts,
					KeyName:           config.Security.Keyname,
				}

				server, err := servers.Create(compute_client, createOpts).Extract()
				if err != nil {
					//panic(err.Error)
					fmt.Println(err)
					panic(err.Error)
				}
				fmt.Println("Server ID: ", server.ID)
				config.Nodes.NumNodes++

				opts := servers.ListOpts{}
				pager := servers.List(compute_client, opts)

				pager.EachPage(func(page pagination.Page) (bool, error) {
					servers.ExtractServers(page)

					servers.ListAddresses(compute_client, server.ID).EachPage(func(page pagination.Page) (bool, error) {
						actual, _ := servers.ExtractAddresses(page)

						fmt.Printf("IP: ")
						fmt.Println(actual[fmt.Sprintf("%s-network", config.Credentials.Clustername)][0].Address)

						for actual[fmt.Sprintf("%s-network", config.Credentials.Clustername)][0].Address == "" {
							ip_add = actual[fmt.Sprintf("%s-network", config.Credentials.Clustername)][0].Address
							fmt.Println("not ready, waiting")
							time.Sleep(time.Second * 5)
						}
						return true, nil
					})

					return true, nil
				})

				for scanner.Scan() {
					lines = append(lines, scanner.Text())

				}

				for _, line := range lines {

					if strings.Contains(line, "[all]") {
						line += "\n"
						line += fmt.Sprintf("%s-node-%d ansible_ssh_host=%s ansible_ssh_common_args='-o StrictHostKeyChecking=no'", config.Credentials.Clustername, config.Nodes.NumNodes+i, ip_add)
					}

					if strings.Contains(line, "[kube-node]") {
						line += "\n"
						line += fmt.Sprintf("%s-node-%d", config.Credentials.Clustername, config.Nodes.NumNodes+i)
					}

					fileContent += line
					fileContent += "\n"
				}

				writeConfig(configName, *config)
			}

			err = ioutil.WriteFile(fmt.Sprintf("/root/cluster/%s/inventory.cfg", config.Credentials.Clustername), []byte(fileContent), 0644)
			if err != nil {
				fmt.Println("failed to write to inventory!")
				return err
			}
			return nil
		}

		return nil
	} else {

		for i := 0; i < config.Nodes.NumNodes; i++ {
			portID := createPort(config, fmt.Sprintf("%s-port-%d", config.Credentials.Clustername, i+1), network_client)

			//		config.Nodes.Networks = make([]servers.Network, 1)

			serverCreateOpts := servers.CreateOpts{

				ServiceClient:  compute_client,
				Name:           fmt.Sprintf("%s-node-%d", config.Credentials.Clustername, i+1),
				Metadata:       metadata["node"],
				FlavorName:     config.Nodes.FlavorName,
				ImageName:      config.Nodes.ImageName,
				SecurityGroups: config.Security.SecurityGroups,
				Networks: []servers.Network{
					//				servers.Network{UUID: config.Network.NetworkID},
					servers.Network{Port: portID},
				},
				UserData: userdata,
			}

			createOpts := keypairs.CreateOptsExt{
				CreateOptsBuilder: serverCreateOpts,
				KeyName:           config.Security.Keyname,
			}

			server, err := servers.Create(compute_client, createOpts).Extract()
			if err != nil {
				//panic(err.Error)
				fmt.Println(err)
				panic(err.Error)
			}
			fmt.Println("Server ID: ", server.ID)

			//		fmt.Println("Server IP:", server.Addresses)
			nodes[i] = server.ID
		}

		//	if start != 0 {
		//		config.Nodes.NumNodes = conf.Node.NumNodes
		//		config.Nodes.Nodes = append(config.Nodes.Nodes, nodes...)
		//	} else {
		//		config.Nodes.Nodes = nodes
		//	}
		writeConfig(configName, *config)

		for i := 0; i < config.Master.NumMasters; i++ {
			portID := createPort(config, fmt.Sprintf("%s-port-%d", config.Credentials.Clustername, i+1), network_client)

			//              config.Nodes.Networks = make([]servers.Network, 1)

			serverCreateOpts := servers.CreateOpts{

				ServiceClient:  compute_client,
				Name:           fmt.Sprintf("%s-master-%d", config.Credentials.Clustername, i+1),
				Metadata:       metadata["master"],
				FlavorName:     config.Master.FlavorName,
				ImageName:      config.Master.ImageName,
				SecurityGroups: config.Security.SecurityGroups,
				Networks: []servers.Network{
					//                              servers.Network{UUID: config.Network.NetworkID},
					servers.Network{Port: portID},
				},
				UserData: userdata,
			}

			createOpts := keypairs.CreateOptsExt{
				CreateOptsBuilder: serverCreateOpts,
				KeyName:           config.Security.Keyname,
			}

			server, err := servers.Create(compute_client, createOpts).Extract()
			if err != nil {
				//panic(err.Error)
				fmt.Println(err)
				panic(err.Error)
			}
			fmt.Println("Server ID: ", server.ID)

			//              fmt.Println("Server IP:", server.Addresses)
			masters[i] = server.ID
			if i == 0 {
				attachFIP(nodes[0], config, network_client, portID)
			}
		}

		return nil
	}
}

func listServers(compute_client *gophercloud.ServiceClient) []servers.Server {
	list := servers.ListOpts{}
	pager := servers.List(compute_client, list)

	var serverliste []servers.Server

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		serverList, err := servers.ExtractServers(page)
		if err != nil {
			fmt.Println(err)
		}

		serverliste = serverList

		/*	for _, s := range serverList {
			// "s" will be a servers.Server
			fmt.Println(s.ID, s.Name, s.Status)
		} */

		return true, nil
	})

	if err != nil {
		fmt.Println(err)
	}

	return serverliste
}

func createNetwork(network_client *gophercloud.ServiceClient, conf *tomlConfig, configName string) {
	adminState := true
	opts := networks.CreateOpts{Name: fmt.Sprintf(conf.Credentials.Clustername + "-network"), AdminStateUp: &adminState}
	network, err := networks.Create(network_client, opts).Extract()
	if err != nil {
		fmt.Println("Creating network failed.")
		fmt.Println(err)
		os.Exit(1)
	}

	conf.Network.NetworkName = fmt.Sprintf(conf.Credentials.Clustername + "-network")
	conf.Network.NetworkID = network.ID

	writeConfig(configName, *conf)

	//	fmt.Println(network.ID)
}

func listNetworks(network_client *gophercloud.ServiceClient) []networks.Network {
	opts := networks.ListOpts{}
	pager := networks.List(network_client, opts)

	var networklist []networks.Network

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		networkListe, err := networks.ExtractNetworks(page)
		if err != nil {
			fmt.Println("cant extract networks!")
			fmt.Println(err)
		}

		networklist = networkListe
		return true, nil
	})
	if err != nil {
		fmt.Println("can't list networks!")
		fmt.Println(err)
	}
	return networklist
}

func extractRouter(page pagination.Page) ([]routers.Router, error) {
	casted := page.(routers.RouterPage).Body

	var response struct {
		Routers []routers.Router `mapstructure:"routers"`
	}

	config := &mapstructure.DecoderConfig{
		DecodeHook: toMapFromString,
		Result:     &response,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, err
	}

	err = decoder.Decode(casted)

	return response.Routers, nil
}

func listRouters(network_client *gophercloud.ServiceClient) []routers.Router {
	list := routers.ListOpts{}
	pager := routers.List(network_client, list)

	var routerliste []routers.Router

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		//		routerList, err := extractRouter(page)

		routerList, err := routers.ExtractRouters(page)
		if err != nil {
			fmt.Println(err)
		}

		routerliste = routerList

		return true, nil
	})

	if err != nil {
		fmt.Println(err)
	}

	return routerliste

}

func createRouter(conf *tomlConfig, c *gophercloud.ServiceClient, configName string) {

	//admin := true
	//	dist := false
	var gateway routers.GatewayInfo
	//gateway.NetworkID = conf.Router.GatewayID
	gateway.NetworkID = conf.Network.ExternalNetworkID
	adminState := true

	opts := routers.CreateOpts{
		Name:         fmt.Sprintf(conf.Credentials.Clustername + "-router"),
		AdminStateUp: &adminState,
		//		Distributed:  &dist,
		TenantID:    conf.Credentials.ProjectID,
		GatewayInfo: &gateway,
	}

	conf.Router.Name = fmt.Sprintf(conf.Credentials.Clustername + "-router")

	res, err := routers.Create(c, opts).Extract()
	if err != nil {
		fmt.Println(err)
	}
	conf.Router.RouterID = res.ID
	//	fmt.Println(res.ID)

	//	fmt.Println(conf.Network.SubnetID)

	intf_res := routers.AddInterface(c, res.ID, routers.AddInterfaceOpts{
		SubnetID: conf.Network.SubnetID,
	})
	_ = intf_res
	//	fmt.Println("Interface added to router: ", intf_res)

	writeConfig(configName, *conf)
}

func toMapFromString(from reflect.Kind, to reflect.Kind, data interface{}) (interface{}, error) {
	if (from == reflect.String) && (to == reflect.Map) {
		return map[string]interface{}{}, nil
	}
	return data, nil
}

type tomlConfig struct {
	Network     networkInfo
	Credentials credentials
	//	Jumphost    jumphost
	Router   router
	Security security
	Subnet   subnet
	Userdata userdata
	Nodes    nodes
	Master   master
}

type credentials struct {
	ProjectName string
	Password    string
	Auth_Url    string
	Username    string
	DomainName  string
	ProjectID   string
	Clustername string
	//	SSHKeyLoc   string
}

type networkInfo struct {
	NetworkName       string
	NetworkID         string
	ExternalNetworkID string
	SubnetID          string
	RouterID          string
	AdminState        bool
}

//type jumphost struct {
//	FloatingIP     string
//	ID             string
//	Name           string
//	FlavorName     string
//	ImageName      string
//	AccessIPv4     string
//	SecurityGroups []string
//	Networks       []servers.Network
//}

type security struct {
	Keyname        string
	SecurityGroups []string
}

type router struct {
	Name       string
	GatewayID  string
	TenantID   string
	RouterID   string
	Adminstate bool
}

type subnet struct {
	CIDR                string
	SubnetName          string
	AllocationPoolStart string
	AllocationPoolEnd   string
	DHCPEnable          bool
	DNSServers          []string
}

type master struct {
	Nodes      []string
	NumMasters int
	FloatingIP string
	//	ID             string
	Name       string
	FlavorName string
	ImageName  string
}

type nodes struct {
	Nodes    []string
	NumNodes int
	//	ID             string
	Name       string
	FlavorName string
	ImageName  string
}

type userdata struct {
	Userdata string
}
