[(Français)](#contr%C3%B4leur-distio-pour-ingress)

## Ingress Istio Controller
Based on https://github.com/kubernetes/sample-controller

This controller is a reimplementation of the logic implemented by the [Istio Ingress](https://istio.io/latest/docs/tasks/traffic-management/ingress/kubernetes-ingress/)
which creates VirtualServices based on Ingress objects and routes their traffic through a Gateway. 

In Istio 1.6+, the core logic changed and instead of all of the generated VirtualServices routing through a single Gateway object, 
each VirtualService received their own. This caused issues for the Cloud Native Platform at Statistics Canada due to the fact that 
a wildcard certificate is used to simplify application deployment.  The issue is documented here: [istio/istio#24385](https://github.com/istio/istio/issues/24385).

### Compatibility and Behaviour
This controller is designed and tested to work with the `istio.io/api/networking/v1beta1` and `k8s.io/api/networking/v1beta1` APIs. 
It has been tested to run on Istio 1.5 and Kubernetes 1.17 and 1.18, however, it should work with all versions of Istio.

Both the `kubernetes.io/ingress.class` annotation and the IngressClass can be used as a way to identify the Ingresses that should be handled by the controller.

#### `kubernetes.io/ingress.class` Annotation
In Kubernetes 1.18, `kubernetes.io/ingress.class` [is deprecated](https://kubernetes.io/docs/concepts/services-networking/ingress/#deprecated-annotation) in favour of 
the IngressClass. The use of the annotation is still supported by this controller and by design, as defined in the documentation of the 
[IngressClassName field](https://github.com/kubernetes/api/blob/648b77825832f4e96433407e4b406a3bdbb988bd/networking/v1beta1/types.go#L72), will take
precedence over the `IngressClass`.

#### IngressClass
An IngressClass can also be used. The controller will search for IngressClasses that have the `spec.controller` value of `ingress.statcan.gc.ca/ingress-istio-controller`.
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

| Argument                 | Description                                                                                                                                                                                                                                                            | Default                                      |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| --kubeconfig             | Defines the path to a kubeconfig file. *Only required if out-of-cluster.*                                                                                                                                                                                              | ""                                           |
| --master                 | The address of the Kubernetes API server. Overrides any value in kubeconfig. <br>*Only required if out-of-cluster.*                                                                                                                                                    | ""                                           |
| --cluster-domain         | The cluster's domain.                                                                                                                                                                                                                                                  | cluster.local                                |
| --default-gateway        | The name of the Istio Gateway to which to apply the routes generated by the controller. <br>The supplied name should be in the **\<namespace>/\<name>** format.                                                                                                        | istio-system/istio-autogenerated-k8s-ingress |
| --ingress-class          | The value of the ***kubernetes.io/ingress.class*** annotation set on Ingresses that should be handled by the controller.<br>If empty, only the IngressClass referenced by the IngressClassName on the Ingresses will be used to identify those that should be handled. | ""                                           |
| --virtual-service-weight | The weight of the Virtual Service destination used by Istio for traffic shaping.                                                                                                                                                                                       | 100                                          |

#### Annotations
Annotations can be set on Ingresses to change how the Controller behaves. Following are the annotations and their function:

| Annotation                     | Description                                                                                                  | Value Type             | Example Values               |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------ | ---------------------- | ---------------------------- |
| ingress.statcan.gc.ca/ignore   | Causes the controller to ignore the Ingress.                                                                 | boolean                | "true"                       |
| ingress.statcan.gc.ca/gateways | Comma-separated list of Gateways that should be passed to the VirtualService instead of the default-gateway. | comma-separated string | mesh,production/prod-gateway |

## Contrôleur d'Istio pour Ingress

Crée et configure des VirtualService d'Istio en utilisant les Ingress de Kubernetes comme définition.

Basé sur https://github.com/kubernetes/sample-controller

### Comment contribuer

Voir [CONTRIBUTING.md](CONTRIBUTING.md)

### Licence

Sauf indication contraire, le code source de ce projet est protégé par le droit d'auteur de la Couronne du gouvernement du Canada et distribué sous la [licence MIT](LICENSE).

Le mot-symbole « Canada » et les éléments graphiques connexes liés à cette distribution sont protégés en vertu des lois portant sur les marques de commerce et le droit d'auteur. 
Aucune autorisation n'est accordée pour leur utilisation à l'extérieur des paramètres du programme de coordination de l'image de marque du gouvernement du Canada. 
Pour obtenir davantage de renseignements à ce sujet, veuillez consulter les [Exigences pour l'image de marque](https://www.canada.ca/fr/secretariat-conseil-tresor/sujets/communications-gouvernementales/exigences-image-marque.html).
