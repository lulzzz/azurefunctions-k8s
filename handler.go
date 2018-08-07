package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	log "github.com/Sirupsen/logrus"
	"github.com/yaron2/azfuncs/components"
	"github.com/yaron2/azfuncs/components/istio"
	"github.com/yaron2/azfuncs/components/nginx"
	funcv1 "github.com/yaron2/azfuncs/pkg/apis/azurefunctions/v1"
	azurefunctions "github.com/yaron2/azfuncs/pkg/client/clientset/versioned"
	"github.com/yaron2/azfuncs/utils"
	appsv1 "k8s.io/api/apps/v1"
	autoscalerv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/api/core/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Handler interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectDeleted(obj interface{})
}

type AzureFunctionsHandler struct {
	Ingress          string
	Mesh             string
	IngressComponent components.Component
	MeshComponent    components.Component
	FunctionsClient  azurefunctions.Interface
}

var clientSet kubernetes.Clientset

const azureFunctionsNamespace = "azure-functions"

func (t *AzureFunctionsHandler) Init() error {
	log.Info("AzureFunctionsHandler.Init")

	t.registerComponents()

	clientSet = *utils.GetKubeClient()

	err := t.installIngressIfRequested()
	if err != nil {
		return err
	}

	err = t.installMeshIfRequested()
	if err != nil {
		return err
	}

	err = t.createFunctionsNamespace()
	if err != nil {
		fmt.Println("Warning: can't create namespace - " + err.Error())
	}

	return nil
}

func (t *AzureFunctionsHandler) registerComponents() {
	components.Register("nginx", &nginx.NginxIngressComponent{})
	components.Register("istio", &istio.IstioComponent{})
}

func (t *AzureFunctionsHandler) createFunctionsNamespace() error {
	ns := v1.Namespace{}
	ns.ObjectMeta = metav1.ObjectMeta{
		Name: azureFunctionsNamespace,
	}

	_, err := clientSet.Core().Namespaces().Create(&ns)
	if err != nil {
		return err
	}

	return err
}

func (t *AzureFunctionsHandler) installMeshIfRequested() error {
	if t.Mesh != "" {
		component, err := t.installComponent(t.Mesh)
		if err != nil {
			return err
		}

		t.MeshComponent = component
	}

	return nil
}

func (t *AzureFunctionsHandler) installComponent(name string) (components.Component, error) {
	component, err := components.GetComponent(strings.ToLower(name))
	if err != nil {
		return nil, err
	}

	isRunning, _ := component.IsRunning()
	if !isRunning {
		fmt.Println("Installing component " + name + " to namespace " + component.Namespace())

		component, err = component.Install()
		if err != nil {
			fmt.Println("Error installing component " + name + ": " + err.Error())
			return nil, err
		}

		fmt.Println("Component installation successful - " + name)
	}

	return component, nil
}

func (t *AzureFunctionsHandler) installIngressIfRequested() error {
	if t.Ingress != "" {
		component, err := t.installComponent(t.Ingress)
		if err != nil {
			return err
		}

		t.IngressComponent = component
	}

	return nil
}

func (t *AzureFunctionsHandler) ObjectCreated(obj interface{}) {
	log.Info("AzureFunctionsHandler.ObjectCreated")

	function := obj.(*funcv1.AzureFunction)

	deployment, err := clientSet.AppsV1().Deployments(azureFunctionsNamespace).Get(function.ObjectMeta.Name+"-deployment", metav1.GetOptions{})
	if err == nil && deployment != nil {
		t.UpdateFunction(deployment, function)
	} else {
		err := t.CreateFunction(function)
		if err != nil {
			fmt.Println("Error creating function - " + err.Error())
		}
	}
}

func (t *AzureFunctionsHandler) ObjectDeleted(obj interface{}) {
	log.Info("AzureFunctionsHandler.ObjectDeleted")

	rawKey := obj.(string)
	functionName := rawKey[strings.Index(rawKey, "/")+1 : len(rawKey)]
	t.DeleteFunction(functionName)
}

func (t *AzureFunctionsHandler) UpdateFunction(deployment *appsv1.Deployment, function *funcv1.AzureFunction) {
	deployment.Spec.Template.Spec.Containers[0].Image = function.Spec.Image
	_, err := clientSet.AppsV1().Deployments(azureFunctionsNamespace).Update(deployment)
	if err != nil {
		fmt.Println("Error updating deployment - " + err.Error())
		return
	}

	ingressEnabled := function.Spec.IngressRoute != "" && t.IngressComponent != nil && t.IsComponentAvailable(t.IngressComponent)

	if ingressEnabled {
		ingressName := function.ObjectMeta.Name + "-ingress"
		ingress, err := clientSet.ExtensionsV1beta1().Ingresses(azureFunctionsNamespace).Get(ingressName, metav1.GetOptions{})
		rules := ingress.Spec.Rules

		if err == nil && ingress != nil && len(rules) > 0 {
			http := *rules[0].HTTP

			if http.Paths[0].Path != function.Spec.IngressRoute {
				http.Paths[0].Path = function.Spec.IngressRoute
				ingress.Spec.Rules[0].HTTP = &http

				_, err := clientSet.ExtensionsV1beta1().Ingresses(azureFunctionsNamespace).Update(ingress)
				if err != nil {
					fmt.Println("Error updating ingress - " + err.Error())
					return
				}

				fmt.Println(ingress.Status.LoadBalancer.Ingress[0].IP)

				function.Spec.URL = "http://" + ingress.Status.LoadBalancer.Ingress[0].IP + function.Spec.IngressRoute

				_, err = t.FunctionsClient.DevV1().AzureFunctions(function.Namespace).Update(function)
				if err != nil {
					fmt.Println("Error updating Function - " + err.Error())
					return
				}
			}
		}
	} else {
		if strings.ToLower(function.Spec.AccessPolicy) == "private" {
			svc, err := clientSet.CoreV1().Services(azureFunctionsNamespace).Get(function.ObjectMeta.Name+"-service", metav1.GetOptions{})
			if err != nil {
				fmt.Println("Error getting service - " + err.Error())
				return
			}

			svc.Spec.Type = apiv1.ServiceTypeClusterIP
			svc.Spec.Ports[0].NodePort = 0

			_, err = clientSet.CoreV1().Services(azureFunctionsNamespace).Update(svc)
			if err != nil {
				fmt.Println("Error updating service - " + err.Error())
				return
			}

			function.Spec.URL = "http://" + svc.Spec.ClusterIP

			_, err = t.FunctionsClient.DevV1().AzureFunctions(function.Namespace).Update(function)
			if err != nil {
				fmt.Println("Error updating Function - " + err.Error())
			}
		}
	}
}

func (t *AzureFunctionsHandler) DeleteFunction(name string) {
	deploymentName := name + "-deployment"
	serviceName := name + "-service"
	ingressName := name + "-ingress"
	hpaName := name

	_ = clientSet.AppsV1().Deployments(azureFunctionsNamespace).Delete(deploymentName, &metav1.DeleteOptions{})
	_ = clientSet.AutoscalingV1().HorizontalPodAutoscalers(azureFunctionsNamespace).Delete(hpaName, &metav1.DeleteOptions{})
	_ = clientSet.CoreV1().Services(azureFunctionsNamespace).Delete(serviceName, &metav1.DeleteOptions{})

	if t.IngressComponent != nil && t.IsComponentAvailable(t.IngressComponent) {
		_ = clientSet.ExtensionsV1beta1().Ingresses(azureFunctionsNamespace).Delete(ingressName, &metav1.DeleteOptions{})
	}
}

func (t *AzureFunctionsHandler) CreateFunction(function *funcv1.AzureFunction) error {
	ingressEnabled := false

	if t.IngressComponent != nil {
		ingressEnabled = t.IsComponentAvailable(t.IngressComponent) && function.Spec.IngressRoute != ""
	}

	minReplicas := int32Ptr(1)
	maxReplicas := int32(1000)

	if function.Spec.Min != nil && *function.Spec.Min > 1 {
		minReplicas = function.Spec.Min
	}

	if function.Spec.Max != nil && *function.Spec.Max > 1 {
		maxReplicas = *function.Spec.Max
	}

	deploymentName := function.ObjectMeta.Name + "-deployment"

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: azureFunctionsNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": function.ObjectMeta.Name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": function.ObjectMeta.Name,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  function.ObjectMeta.Name,
							Image: function.Spec.Image,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
					Tolerations: []v1.Toleration{
						{
							Key:   "azure.com/aci",
							Value: "NoSchedule",
						},
					},
				},
			},
		},
	}

	autoscaler := autoscalerv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      function.ObjectMeta.Name,
			Namespace: azureFunctionsNamespace,
		},
		Spec: autoscalerv1.HorizontalPodAutoscalerSpec{
			MinReplicas:                    minReplicas,
			MaxReplicas:                    maxReplicas,
			TargetCPUUtilizationPercentage: int32Ptr(60),
			ScaleTargetRef: autoscalerv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       deploymentName,
			},
		},
	}

	_, err := clientSet.AppsV1().Deployments(azureFunctionsNamespace).Create(&deployment)
	if err != nil {
		return err
	}

	_, err = clientSet.AutoscalingV1().HorizontalPodAutoscalers(azureFunctionsNamespace).Create(&autoscaler)
	if err != nil {
		return err
	}

	isPrivateAccess := strings.ToLower(function.Spec.AccessPolicy) == "private"

	serviceName := function.ObjectMeta.Name + "-service"
	servicePort := 80
	var serviceType apiv1.ServiceType

	if isPrivateAccess || ingressEnabled {
		serviceType = apiv1.ServiceTypeClusterIP
	} else {
		serviceType = apiv1.ServiceTypeLoadBalancer
	}

	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: azureFunctionsNamespace,
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": function.ObjectMeta.Name,
			},
			Ports: []apiv1.ServicePort{
				{
					Name:     "http",
					Protocol: apiv1.ProtocolTCP,
					Port:     int32(servicePort),
				},
			},
			Type: serviceType,
		},
	}

	_, err = clientSet.CoreV1().Services(azureFunctionsNamespace).Create(&service)
	if err != nil {
		return err
	}

	functionServiceName := serviceName
	namespace := azureFunctionsNamespace

	if ingressEnabled {
		ingressComponent := t.IngressComponent.(components.IngressComponent)
		ingressName := function.ObjectMeta.Name + "-ingress"

		ingressRule := v1beta1.IngressRule{}
		ingressRule.HTTP = &v1beta1.HTTPIngressRuleValue{
			Paths: []v1beta1.HTTPIngressPath{
				{
					Path: function.Spec.IngressRoute,
					Backend: v1beta1.IngressBackend{
						ServiceName: serviceName,
						ServicePort: intstr.FromInt(servicePort),
					},
				},
			},
		}

		_, err := clientSet.ExtensionsV1beta1().Ingresses(azureFunctionsNamespace).Create(&v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name: ingressName,
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/rewrite-target": "/",
					"nginx.ingress.kubernetes.io/ssl-redirect":   strconv.FormatBool(false),
				},
			},
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					ingressRule,
				},
			},
		})

		if err != nil {
			return err
		}

		functionServiceName = ingressComponent.ServiceName()
		namespace = ingressComponent.Namespace()
	}

	t.UpdateFunctionPublicIP(functionServiceName, namespace, function, ingressEnabled)

	return nil
}

func (t *AzureFunctionsHandler) UpdateFunctionPublicIP(serviceName string, namespace string, function *funcv1.AzureFunction, ingressEnabled bool) {
	ip := ""

	for ip == "" {
		svc, err := clientSet.CoreV1().Services(namespace).Get(serviceName, metav1.GetOptions{})
		if err != nil {
			fmt.Println("Error getting service - " + err.Error())
			return
		}

		if !ingressEnabled && strings.ToLower(function.Spec.AccessPolicy) == "private" {
			ip = svc.Spec.ClusterIP
		} else {
			if svc.Status.LoadBalancer.Ingress != nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
				ip = svc.Status.LoadBalancer.Ingress[0].IP
			}
		}

		if ip != "" {
			function.Spec.URL = "http://" + ip

			if ingressEnabled {
				function.Spec.URL += function.Spec.IngressRoute
			}

			_, err := t.FunctionsClient.DevV1().AzureFunctions(function.Namespace).Update(function)
			if err != nil {
				fmt.Println("Error updating Function - " + err.Error())
			}
		}

		time.Sleep(time.Second * 1)
	}
}

func (t *AzureFunctionsHandler) IsComponentAvailable(component components.Component) bool {
	isRunning, err := component.IsRunning()
	return (err == nil && isRunning)
}

func int32Ptr(i int32) *int32 { return &i }
