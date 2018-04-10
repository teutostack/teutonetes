# Hinweis
Derzeit befindet sich das Projekt im Aufbau!

# Vorraussetzungen
* Internetzugang
* Grundkenntnisse in der OpenStack Administration
* Docker (z.B. "docker ce")
* (kubectl und helm)

# Schritte:
## Schritte zur Erstellung:

* Docker-Image:

```
docker pull teutonetes/teutonetes
```

* "~/teutonetes/cluster/\<Clustername\>/deploy-config" erstellen und konfigurieren 
(u.a. mit den Openstack Credentials - siehe [deploy.example](https://github.com/teutostack/teutonetes/blob/master/deploy.example) )

* Alias:

```
alias teutonetes="docker run -ti --rm -v ~/teutonetes/cluster:/root/cluster/ \
  -v ~/.kube:/root/.kube/ -w /root/ -e UID=$(id -u) -e GID=$(id -g) \
  teutonetes/teutonetes:latest"
```

* teutonetes-go:

```
teutonetes <Clustername> create 
```

 
* Deploy Kubernetes:

```
teutonetes <Clustername> deploy
```

## Schritte zum Dashboard:
* Proxy:

```
kubectl proxy &
```

* via Browser folgende Seite besuchen: http://localhost:8001/api/v1/namespaces/kube-system/services/kubernetes-dashboard/proxy/


## Schritte zum Aufräumen/Kompletten Löschen des Clusters

* Löschen des Clusters (es existiert kein Backup - was weg ist, ist weg).:

```
teutonetes <Clustername> cleanup yirrmi
```

# Teutonetes-config

**\[Credentials\]**
* Auth_Url (Addresse des Authentifizierungsservers des Openstacks. Meist die 
`<ADDRESSE_DES_DASHBOARDES>:5000/v?` - "?" steht hier für die Version, meist "v3", oder "v2".)
* Username (OpenStack Benutzername.)
* Password (Passwort des OpenStack Benutzers.)
* DomainName (meist "*default*".)
* ProjectID (ID des Projektes, in dem der Jumphost und der Kubernetes Cluster erstellt werden sollen.)
* ProjectName (Name des Projektes.)
* Clustername (Name des Clusters.)

**\[Nodes\]**
* NumNodes (Anzahl der Nodes.)
* ImageName (Der Name des Images, welches für die Instanzen verwendet werden sollen - nicht nur des Jumphosts.)
* FlavorName (Der Name des Flavors des Jumphosts und der Kubernetes Instanzen.)
* Security Group (ID einer Sicherheitsgruppe, die vom Jumphost verwendet werden kann. Sollte keine vorliegen, sollte eine erstellt werden.)

**\[Network\]**
* ExternalNetworkID (Die ID des externen Netzwerkes, in dem beispielsweise Floating-IP allokiert werden können.)

**\[Subnet\]**
* CIDR (Für das zu erstellende Netzwerk.)
* AllocationPoolStart (Start-IP innerhalb des zu erstellenden Netzwerkes.)
* AllocationPoolEnd (End-IP innerhalb des zu erstellenden Netzwerkes.)
* DHCPEnable (auf true)
* DNSServers (Falls vorhanden, kann hier eine Liste an DNS-Servern aufgelistet werden.)

