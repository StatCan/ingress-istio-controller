[(Français)](#contr%C3%B4leur-distio-pour-ingress)

## Ingress Istio Controller

Based on https://github.com/kubernetes/sample-controller

This controller is a reimplementation of the logic implemented by the [Istio Ingress](https://istio.io/latest/docs/tasks/traffic-management/ingress/kubernetes-ingress/)
which creates VirtualServices based on Ingress objects and routes their traffic through a Gateway. 

This reimplementation is du to that fact that in Istio 1.6, the core logic changed and instead of all of the generated VirtualServices routing through a single 
Gateway object, each VirtualService received their own. This caused issues for the Cloud Native Platform at Statistics Canada due to the fact that 
a wildcard certificate is used to simplify application deployment. The issue is documented here: [istio/istio#24385](https://github.com/istio/istio/issues/24385).

### Compatibility and Behaviour

This controller is designed and tested to work with the `istio.io/api/networking/v1beta1` and `k8s.io/api/networking/v1beta1` APIs. 
It has been tested to run on Istio 1.5 and Kubernetes 1.17 and 1.18, however, it should work with all versions of Istio.

Both the `kubernetes.io/ingress.class` annotation and the IngressClass can be used as a way to identify the Ingresses that should be handled by the controller.

#### `kubernetes.io/ingress.class` Annotation

Starting with Kubernetes 1.18, `kubernetes.io/ingress.class` [is deprecated](https://kubernetes.io/docs/concepts/services-networking/ingress/#deprecated-annotation) in favour of 
the IngressClass. The use of the annotation is still supported by this controller and by design, as defined in the documentation of the 
[IngressClassName field](https://github.com/kubernetes/api/blob/648b77825832f4e96433407e4b406a3bdbb988bd/networking/v1beta1/types.go#L72), will take
precedence over the `IngressClass`.

#### IngressClass

The controller will handle Ingresses with references to IngressClasses that have the `spec.controller` value of `ingress.statcan.gc.ca/ingress-istio-controller`.
Following is an example of an IngressClass that can be used:

```yaml
apiVersion: networking.k8s.io/v1beta1
kind: IngressClass
metadata:
  name: ingress-istio-controller
spec:
  controller: ingress.statcan.gc.ca/ingress-istio-controller
```

### How to Contribute

See [CONTRIBUTING.md](CONTRIBUTING.md)

### License

Unless otherwise noted, the source code of this project is covered under Crown Copyright, Government of Canada, and is distributed under the [MIT License](LICENSE).

The Canada wordmark and related graphics associated with this distribution are protected under trademark law and copyright law. 
No permission is granted to use them outside the parameters of the Government of Canada's corporate identity program. 
For more information, see [Federal identity requirements](https://www.canada.ca/en/treasury-board-secretariat/topics/government-communications/federal-identity-requirements.html).

### Installation

A Helm chart is available from the [StatCan/Charts repository](https://github.com/statcan/charts/stable/ingress-istio-controller) and images can be found in the 
[statcan/ingress-istio-controller](https://hub.docker.com/r/statcan/ingress-istio-controller).

### Configuration

There are two ways to alter the behaviour of the **isto-ingress-controller**. 
The first is via Command Line Arguments and the second is via Annotations set on Ingresses.

#### Command Line Arguments

| Argument                 | Description                                                                                                                                                                                                                                                            | Default Value                                |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| --kubeconfig             | Defines the path to a kubeconfig file. *Only required if out-of-cluster.*                                                                                                                                                                                              | ""                                           |
| --master                 | The address of the Kubernetes API server. Overrides any value in kubeconfig. <br>*Only required if out-of-cluster.*                                                                                                                                                    | ""                                           |
| --cluster-domain         | The cluster's domain.                                                                                                                                                                                                                                                  | cluster.local                                |
| --default-gateway        | The name of the Istio Gateway to which to apply the VirtualServices generated by the controller. <br>The supplied value should be in the **\<namespace>/\<name>** format.                                                                                              | istio-system/istio-autogenerated-k8s-ingress |
| --ingress-class          | The value of the ***kubernetes.io/ingress.class*** annotation set on Ingresses that should be handled by the controller.<br>If empty, only the IngressClass referenced by the IngressClassName on the Ingresses will be used to identify those that should be handled. | ""                                           |
| --virtual-service-weight | The proportion of traffic to be forwarded to the service.                                                                                                                                                                                                              | 100                                          |

#### Annotations

Annotations can be set on Ingresses to change how the Controller behaves. Following are the annotations and their function:

| Annotation                     | Description                                                                                                  | Value Type             | Example Values               |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------ | ---------------------- | ---------------------------- |
| ingress.statcan.gc.ca/ignore   | Causes the controller to ignore the Ingress.                                                                 | boolean                | "true"                       |
| ingress.statcan.gc.ca/gateways | Comma-separated list of Gateways that should be passed to the VirtualService instead of the default-gateway. | comma-separated string | mesh,production/prod-gateway |

## Contrôleur d'Istio pour Ingress

Basé sur https://github.com/kubernetes/sample-controller

Ce crontrôleur est une ré-implémentation de la logique du [Istio Ingress *(anglais)*](https://istio.io/latest/docs/tasks/traffic-management/ingress/kubernetes-ingress/). Celui-ci crée des VirtualServices en utlisant des objets Ingress comme définition afin d'acheminer le trafic réseau par un Gateway.

Cette réimplémentation est causé par le changement de la logique à partir d'Istio 1.6 causant qu'un Gateway unique est créé pour chaque VirtualService au lien d'un 
Gateway commun. Ce changement a causé des problèmes pour la Plateforme Infonuagique Native à Statistique Canada puisqu'un certificat générique est utilisé afin de simplifier le déploiement d'applications.

### Compatibilité et Fonctionnement

Ce crontrôleur est conçu et fonctionne avec les API `istio.io/api/networking/v1beta1` et `k8s.io/api/networking/v1beta1`.
Il a été testé avec la version 1.5 d'Istio et les versions 1.17 et 1.18 de Kubernetes. Ceci dit, il devrait être compatible avec toutes versions d'Istio.

L'annotation `kubernetes.io/ingress.class` ainsi que l'objet IngressClass peuvent être utilisés afin de cibler les Ingresses devrant être gérer par le contrôleur.


#### Annotation `kubernetes.io/ingress.class`

Débutant en Kubernetes 1.18, l'annotation `kubernetes.io/ingress.class` [est dépriciée *(anglais)*](https://kubernetes.io/docs/concepts/services-networking/ingress/#deprecated-annotation)
en faveur de l'utilisation de l'IngressClass. Ceci dit, l'annotation peut encore être utilisée comme cible par ce contrôleur et comme documenté sur le
[champ IngressClassName *(anglais)*](https://github.com/kubernetes/api/blob/648b77825832f4e96433407e4b406a3bdbb988bd/networking/v1beta1/types.go#L72), 
aura préséance sur l'`IngressClass`.

#### IngressClass

Le contrôleur ciblera les Ingresses référant aux IngressClasses ayant comme valeur `ingress.statcan.gc.ca/ingress-istio-controller` au champ `spec.controller`.
Ci-dessous est un example d'un IngressClass pouvant être utiliser:

```yaml
apiVersion: networking.k8s.io/v1beta1
kind: IngressClass
metadata:
  name: ingress-istio-controller
spec:
  controller: ingress.statcan.gc.ca/ingress-istio-controller
```

### Comment contribuer

Voir [CONTRIBUTING.md](CONTRIBUTING.md)

### Licence

Sauf indication contraire, le code source de ce projet est protégé par le droit d'auteur de la Couronne du gouvernement du Canada et distribué sous la [licence MIT](LICENSE).

Le mot-symbole « Canada » et les éléments graphiques connexes liés à cette distribution sont protégés en vertu des lois portant sur les marques de commerce et le droit d'auteur. 
Aucune autorisation n'est accordée pour leur utilisation à l'extérieur des paramètres du programme de coordination de l'image de marque du gouvernement du Canada. 
Pour obtenir davantage de renseignements à ce sujet, veuillez consulter les [Exigences pour l'image de marque](https://www.canada.ca/fr/secretariat-conseil-tresor/sujets/communications-gouvernementales/exigences-image-marque.html).

### Installation

Un chart Helm est publié dans le [dépot StatCan/charts](https://github.com/statcan/charts/stable/ingress-istio-controller) et des images Docker sont publiés dans le 
dépot [statcan/ingress-istio-controller](https://hub.docker.com/r/statcan/ingress-istio-controller).

### Configuration

Il y a deux façon d'altérer le fonctionnement du **isto-ingress-controller**. 
La première étant des arguments de la ligne de commande et la deuxième étant des Annotations sur les Ingresses.

#### Ligne de Commande

| Argument                 | Description                                                                                                                                                                                                                                                      | Valeur par défaut                            |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| --kubeconfig             | Le chemin de fichier local au kubeconfig. <br>*Seulement requis à l'extérieur du cluster.*                                                                                                                                                                | ""                                           |
| --master                 | L'adresse au serveur API de Kubernetes. Cet argument prendra l'avance des configurations du kubeconfig. <br>*Seulement requis à l'extérieur du cluster.*                                                                                                         | ""                                           |
| --cluster-domain         | Le domaine du cluster.                                                                                                                                                                                                                                           | cluster.local                                |
| --default-gateway        | Le nom de l'Istio Gateway duquel les VirtualServices seront servis. <br>L'argument devrait être en format **\<namespace>/\<nom>**.                                                                                                                                 | istio-system/istio-autogenerated-k8s-ingress |
| --ingress-class          | La valeur de l'Annotation ***kubernetes.io/ingress.class*** sur les Ingresses devrant être ciblés par le contrôleur.<br>Si la valeur est vide, seulement le IngressClass référé par IngressClassName dans les Ingresses sera utilisé comme paramêtre de ciblage. | ""                                           |
| --virtual-service-weight | Le valeur proportionnelle de trafic réseau devrant être achimener au service.                                                                                                                                                                                    | 100                                          |

#### Annotations

Des Annotations peuvent être ajouter aux Ingresses afin de modifier le fonctionnement du contrôleur. Ci-dessous est une table des annotations possibles :

| Annotation                     | Description                                                                                                  | Value Type             | Example Values               |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------ | ---------------------- | ---------------------------- |
| ingress.statcan.gc.ca/ignore   | Cause que le contrôleur ne cible pas l'Ingress annoté.                                                       | booléen                | "true"                       |
| ingress.statcan.gc.ca/gateways | Une liste de noms de Gateway séparés par virgules devrant être référée par le VirtualService au lieu du default-gateway. | string séparé par virgules | mesh,production/prod-gateway |
