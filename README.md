# Inhalt:

* [Inhalt](./README.md#Inhalt)
* [Hinweis](./README.md#Hinweis)
* [Vorraussetzungen](./README.md#Vorraussetzungen)
* [Schritte](./README.md#Schritte)
 - [Schritte zur Erstellung](./README.md#schritte-zur-erstellung)
 - [Schritte zum Scalen](./README.md#schritte-zum-scalen) **Derzeit noch nicht vollständig implementiert/Ungetestet**
 - [Schritte zum Dashboard](./README.md#schritte-zum-dashboard)
 - [Schritte zum traefik Ingress example](./README.md#schritte-zum-traefik-ingress-example) **Derzeit Nicht Funktionsfähig...**
 - [Schritte zum Cleanup bzw. Kompletten Entfernen des Clusters](./README.md#schritte-zum-cleanup-bzw.-kompletten-entfernen-des-clusters)
* [Teutonetes-config](./README.md#Teutonetes-config)
* [Beispiele](./README.md#Beispiele)
* [Tutorial](./README.md#Tutorial)
* [How To Version (!Proposal!)](./README.md#how-to-version) **Derzeit nicht in Verwendung - Konzept/Vorschlag**
 - [Manuell](./README.md#Manuell)	**Siehe oben**
 - [release.sh](./README.md#release.sh) **Siehe oben**

# Hinweis
Derzeit befindet sich das Projekt im Aufbau. Für Kunden sollte die
Kompabilität mit anderen Systemen ggf. gewährleistet sein!
[Schritte zur Erstellung](./README.md#schritte-zur-erstellung) ist eine Stichpunktartige Anleitung zur Erstellung eines Clusters.
[Tutorial](./README.md#Tutorial) enthält etwas mehr Informationen zur Erstellung eiens Clusters.

# Vorraussetzungen
* Internetzugang
* Grundkenntnisse in der OpenStack Administration und dem Horizon Dashboard
* Docker (z.B. "*docker ce*")
* (kubectl und helm)

# Schritte:
## Schritte zur Erstellung:

* Docker-Image:

```
docker pull teutonetes/teutonetes

```

* "~/teutonetes/cluster/\<Clustername\>/deploy-config" erstellen und konfigurieren 
(u.a. mit den Openstack Credentials, z.B. über das Horizon Dashboard erhältlich - siehe [Teutonetes-config](./README.md#Teutonetes-config) oder [deploy.example](./deploy.example) für mehr Informationen oder einem Beispiel, wie "*deploy-config*" aussehen könnte/sollte.)

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

## Schritte zum Scalen:
**Wichtig:** Derzeit ist es nicht korrekt implementiert. Dies ist also nur ein Platzhalter. Zusätzlich ist es nur möglich weitere Worker Nodes dem Kubernetes Cluster beizufügen.

* Falls notwendig: Erstelle mit folgenden Befehl eine neue Instanz

```
teutonetes <Clustername> create node <Anzahl>
```
* Füge mittels folgenden Befehl die angegebene(n) Instanz(en) als Worker Node(s) zum Kubernetes Cluster hinzu:

```
teutonetes <Clustername> add [<Nodename>]
```

* Zum Löschen von Cluster Nodes:

```
teutonetes <Clustername> delete [<Nodename>]
```


## Schritte zum Dashboard:
* Proxy:

```
kubectl proxy &
```

* via Browser folgende Seite besuchen: 
http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/

* Derzeit erhält das Dashboard auch ohne Authentifizierung Zugriffsrechte auf alles im Cluster (dementpsrechend nur dev state).
Bei dem Anmeldebildschirm kann also durch den Button "*skip*" das Dashboard erreicht werden ohne Berechtigungseinschränkungen.

## Schritte zum Traefik Ingress Example:
**Derzeit wird das Traefik-Dashboard noch nach außen bereitgestellt, was gegebenenfalls zu einem Sicherheitsrisiko werden kann. Daher wird empfohlen, das Traefik-Example nur kurzzeitig zu verwenden.**

* Ingress Controller und Pods deployen:

```
teutonetes <Clustername> traefikex
```

* Floating IP des Loadbalancers suchen - die IP wird unter anderem von oberer Funktion ausgegeben

* Aufrufen der Seiten mit curl
 
  * `curl -H 'Host: traefik-ui.teutonetes' <lbaas_floating>/`
  * `curl -H 'Host: stilton.teutonetes' <lbaas_floating>/`
  * `curl -H 'Host: cheddar.teutonetes' <lbaas_floating>/`
  * `curl -H 'Host: wensleydale.teutonetes' <lbaas_floating>/`
  * `curl -H 'Host: cheeses.teutonetes' <lbaas_floating>/stilton`
  * `curl -H 'Host: cheeses.teutonetes' <lbaas_floating>/cheddar`
  * `curl -H 'Host: cheeses.teutonetes' <lbaas_floating>/wensleydale`

* Für einen bequemeren Aufruf per Webbrowser, die Floating-IP in die lokale hosts-Datei für die Folgenden Domains eintragen:
  - traefik-ui.teutonetes
  - cheeses.teutonetes
  - stilton.teutonetes
  - cheddar.teutonetes
  - wensleydale.teutonetes
  
  Beispiel: `<lbaas_floating>    traefik-ui.teutonetes cheeses.teutonetes stilton.teutonetes cheddar.teutonetes wensleydale.teutonetes`

## Schritte zum Traefik Ingress Example:
* Das Verzeichnis traefikex aus dem Repository kopieren und in dieses Verzeichnis wechseln.
* kubectl auf die Konfig des Clusters umstellen, fals noch nicht geschehen `kubectl config use-context <clustername>`
* Deploymet rückgängig machen `for mycheese in $(ls 0*.yaml | sort -r); do kubectl delete -f $mycheese; done`

## Schritte zum Cleanup bzw. Kompletten Entfernen des Clusters

* Löschen des Clusters (es existiert kein Backup - was weg ist, ist weg).:

```
#Yes, I really really mean it
teutonetes <Clustername> cleanup yirrmi
```

# Teutonetes-config
Im Laufe des Befehles `teutonetes <Clustername> create` wird die Datei "*deploy-config*" vom Container angepasst.
Im Folgenden sind die Notwendigen Parameter aufgelistet, welche benötigt werden, damit die Instanzen und das Netzwerk
korrekt und erfolgreich erstellt werden können.
**Wichtig**: Siehe [deploy.example](./deploy.example) für ein abstraktes Beispiel.

**\[Credentials\]**
* Auth_Url (Addresse des Authentifizierungsservers des Openstacks. Meist die 
`<ADDRESSE_DES_DASHBOARDES>:5000/v?` - "?" steht hier für die Version, meist "v3", oder "v2".)
* Clustername (Name des Clusters.)
* Username (OpenStack Benutzername.)
* Password (Passwort des OpenStack Benutzers.)
* DomainName (meist "*default*".)
* ProjectID (ID des Projektes, in dem der Jumphost und der Kubernetes Cluster erstellt werden sollen.)
* ProjectName (Name des Projektes.)

**\[Nodes\]**
* NumNodes (Anzahl der Nodes. Es sollten mindestens 3**(!)** sein.)
* ImageName (Der Name des Images, welches für die Instanzen verwendet werden sollen - nicht nur des Jumphosts.)
* FlavorName (Der Name des Flavors des Jumphosts und der Kubernetes Instanzen.)

**\[Security\]**
* SecurityGroups (ID einer Sicherheitsgruppe, die von den Nodes verwendet werden können. Sollte keine vorliegen, 
so müsste vom Nutzer eine zuvor manuell erstellt werden. Bisher existiert keine Möglichkeit sie automatisch generieren zu lassen, oder die Default verwendet werden.)

**\[Network\]**
* ExternalNetworkID (Die ID des externen Netzwerkes, in dem die Floating-IPs alloziert werden.)

**\[Subnet\]**
* CIDR (Netzwerkadresse.)
* AllocationPoolStart (Start-IP innerhalb des zu erstellenden Netzwerkes.)
* AllocationPoolEnd (End-IP innerhalb des zu erstellenden Netzwerkes.)

# Beispiele:

```
# First step of the kubernetes installation.
# Create Network Environment + Gateway and the instances inside the OpenStack Cloud
teutonetes <CLUSTERNAME> create

# Continue kubernetes Installation
# Prepares above created OpenStack instances for further installation.
# Install kubernetes on top of the OpenStack Instances via kubespray-incubator.
teutonetes <CLUSTERNAME> deploy

# Do an OpenStack Query
teutonetes <CLUSTERNAME> openstack network list

# SSH connection to edge node of cluster
teutonetes <CLUSTERNAME> ssh

# Use shell inside container with credentials loaded (debug purposes)
teutonetes <CLUSTERNAME> shell

# Neutron Query
teutonetes <CLUSTERNAME> neutron port-list

# Delete given cluster (yes, you really really mean it)
teutonetes <CLUSTERNAME> cleanup yirrmi
```
# Tutorial
Der erste Schritt zur Installation eines Kubernetes Clusters innerhalb einer OpenStack Cloud mittels "*teutonetes*" ist 
es den Container zu erhalten. Hierzu dienen folgende zwei Befehle:

```
docker pull teutonetes/teutonetes
```
Der Container liegt nun lokal vor (über `docker images` kann dies geprüft werden).
Bevor er jedoch verwendet werden kann,
muss eine Konfiguration vorliegen,
damit der Container sich erfolgreich bei der OpenStack Cloud authentifizieren kann. 
Der Container erwartet eine Konfigurations-Datei im Verzeichnis "*~/teutonetes/cluster/\<Clustername\>/*" mit dem Namen 
"*deploy-config*" (\<Clustername\> wird dabei frei gewählt - so wird der Cluster heißen). In diesem Verzeichnis werden 
auch weitere für diesen Cluster nötigen Dateien hinterlegt - beispielsweise der SSH-Schlüssel. Notwendige Optionen, 
um die Netzwerkumgebung und die Instanzen im OpenStack zu erstellen, können oben unter [Teutonetes-config](./README.md#Teutonetes-config) 
oder aus der [deploy.example](./deploy.example)-Datei entnommen werden. Diese Werte können über das Horizon Dashboard erhalten
werden. Loggen Sie sich im Dashboard ein und suchen die ID des externen Netzwerkes heraus, ebenso die einer passenden
Sicherheitsgruppe. Innerhalb der "*OpenRC*"-Datei von OpenStack können fast alle notwendigen Informationen für die
Kategorie "*Credentials*" entnommen werden. Sobald alle Informationen eingetragen wurden, kann der nächste Schritt
der Installation erfolgen.

Um die Verwendung des Containers so bequem wie möglich machen, sollte ein Alias gesetzt werden. Dieser kann beispielsweise
in die "*.bashrc*" hinterlegt werden. Es wird dem Container beim Starten unter anderem die User und Gruppen ID des Nutzers
mitgegeben, des Weiteren werden zur Konfiguration von kubectl und zum Lesen der "*deploy-config*"-Datei zwei Verzeichnisse
gemounted und sichergestellt, dass nach Beenden der Aufgabe der Container selbstständig gelöscht wird.

```
alias teutonetes="docker run -ti --rm -v ~/teutonetes/cluster:/root/cluster/ \
  -v ~/.kube:/root/.kube/ -w /root/ -e UID=$(id -u) -e GID=$(id -g) \
  teutonetes/teutonetes:latest"
```
Nun kann der Container mittels alias `teutonetes` verwendet werden. Die allgemeine Struktur der Aufrufe sieht folgendermaßen aus:

```
teutonetes <Clustername> <Befehl> <Argumente>
```
Dadurch ist sichergestellt, dass die richtigen Credentials geladen werden. Bisher existiert der Kubernetes Cluster "*\<Clustername\>*" 
noch nicht (es sollte ein anderer Name gewählt werden...). Der erste Befehl, welcher erfolgen sollte ist der folgende:

```
teutonetes <Clustername> create 
```
Innerhalb des Containers wird die Go-Binary ausgeführt, welche Netzwerk, Subnetz, Router als Gateway und Instanzen im OpenStack mit Hilfe
der angegebenen Informationen innerhalb der Konfigurationsdatei. Einer der Nodes (derzeit mit dem Namen "*\<Clustername\>-node-1*") ist
ein Kantenknoten und erhält eine Floating-IP. Dadurch ist es möglich den Cluster zu erreichen. Sobald diese Schritte durchgeführt wurde,
ist das Grundgerüst des Kubernetes Clusters vorhanden. Mit dem folgenden Befehl wird die Installation sowohl fortgesetzt als auch beendet:

```
teutonetes <Clustername> deploy
```
Dieser Befehl erstellt zunächst ein Inventar mit den Instanzen des Clusters. Danach werden die Instanzen geprüft und weiter für die weitere
Installation vorbereitet, indem Python ggf. installiert wird etc.. Sobald dies geschehen ist, wird "*kubespray-incubator*" gestartet, welches
die weitere Installation des Kubernetes Clusters übernimmt. Es werden Zertifikate installiert, die clusterinterne Netzwerke erstellt, ein etcd
Cluster erstellt und so weiter... Derzeit sind standardmäßig die ersten beiden Instanzen (einer davon ist der Kantenknoten) Master Knoten, die
ersten drei (also zwei Master und ein Worker Knoten) der etcd Cluster. Alle weitere Instanzen sind exklusive Worker Knoten. Nachdem 
"*kubespray-incubator*" sein Werk vollendet hat, ist die eigentliche Kubernetes Installation beendet. Es folgen jedoch noch weitere kleinere
Konfigurationen. Unter anderem wird kubectl konfiguriert, wodurch ein lokaler Zugriff auf den Kubernetes Cluster möglich ist. Des Weiteren wird
bereits eine StorageClass namens "*teutostack*" erstellt, wodurch dynamisch mittels Claims bei Cinder Volumes angefordert werden können.

# How to Version:
**PROPOSAL/KONZEPT**
Der Bau des Docker Containers wurde dahingehend angepasst, dass er den Tag des Commits als Version nimmt.
Das hat zur Folge, dass jeder commit mit einem Tag versehen werden muss. Die Version wird in der Datei
"*VERSION*" hinterlegt. Diese sollte unbedingt aktuell gehalten werden.

## Manuell
* Zunächst sollte in der Datei "*VERSION*" die aktuelle Version überprüft werden (, falls nicht bekannt).

* "*VERSION*" sollte passend aktualisiert werden und dem push hinzugefügt werden (`git add VERSION`).

* Nach dem `git commit (-m)` wird nun ein Tag hinzu gefügt: `git tag -a "<new_version>" -m "version <new_version>"`

* Nach dem `git push` muss nur noch ein `git push --tags` folgen.

## release.sh
Das kleine "*release.sh*"-Skript soll beim committen helfen. Hierzu führt man das Skript **entweder**
mit einer bereits bestimmten Version aus oder ohne einen Parameter. Ohne Parameter fordert das Skript,
nachdem man die derzeitige Version des Containers erfährt, dass eine neue Version der Form "X.X.X"
festgelegt werden soll (achtet bitte darauf etwas vernünftiges einzugeben - das Skript ist auf die Schnelle entstanden
und fängt wenig ab...). Das skript zeigt daraufhin den Output von "*git status*" an und fordert auf alle notwendigen 
Änderungen zu adden/removen. Das Skript übernimmt den gesamten Input und versucht ihn auszuführen (also `git add/rm [file]` 
verwenden). Mittels `gitlab_cancel`kann das Skript abgebrochen werden, wohingegen `gitlab_push`das Skript fortsetzt. Nun 
wird die Version in der "*VERSION*"-Datei mit der angegebenen Version aktualisiert, der Commit mit der Version als Tag 
und Message versehen und dann gepusht. Fertig.

