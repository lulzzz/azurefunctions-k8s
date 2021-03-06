package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	funcv1 "github.com/yaron2/azfuncs/pkg/apis/azurefunctions/v1"
	azurefunctions "github.com/yaron2/azfuncs/pkg/client/clientset/versioned"
	azurefunctioninformer_v1 "github.com/yaron2/azfuncs/pkg/client/informers/externalversions/azurefunctions/v1"
	"github.com/yaron2/azfuncs/utils"
)

// retrieve the Kubernetes cluster client from outside of the cluster
func getClients() (kubernetes.Interface, azurefunctions.Interface) {
	client := utils.GetKubeClient()
	config := utils.GetConfig()

	azureFuncsClient, err := azurefunctions.NewForConfig(config)
	if err != nil {
		log.Fatalf("getClusterConfig: %v", err)
	}

	log.Info("Successfully constructed k8s client")
	return client, azureFuncsClient
}

// main code path
func main() {
	// get the Kubernetes client for connectivity
	client, azureFuncsClient := getClients()

	// retrieve our custom resource informer which was generated from
	// the code generator and pass it the custom resource client, specifying
	// we should be looking through all namespaces for listing and watching
	informer := azurefunctioninformer_v1.NewAzureFunctionInformer(
		azureFuncsClient,
		meta_v1.NamespaceAll,
		0,
		cache.Indexers{},
	)

	// create a new queue so that when the informer gets a resource that is either
	// a result of listing or watching, we can add an idenfitying key to the queue
	// so that it can be handled in the handler
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// add event handlers to handle the three types of events for resources:
	//  - adding new resources
	//  - updating existing resources
	//  - deleting resources
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// convert the resource object into a key (in this case
			// we are just doing it in the format of 'namespace/name')
			key, err := cache.MetaNamespaceKeyFunc(obj)
			log.Infof("Add Azure Function: %s", key)
			if err == nil {
				// add the key to the queue for the handler to get
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newFunc := newObj.(*funcv1.AzureFunction)
			oldFunc := oldObj.(*funcv1.AzureFunction)
			if newFunc.ResourceVersion == oldFunc.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}

			if !apiequality.Semantic.DeepEqual(oldFunc.Spec, newFunc.Spec) {
				key, err := cache.MetaNamespaceKeyFunc(newObj)
				log.Infof("Update Azure Function: %s", key)
				if err == nil {
					queue.Add(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			// DeletionHandlingMetaNamsespaceKeyFunc is a helper function that allows
			// us to check the DeletedFinalStateUnknown existence in the event that
			// a resource was deleted but it is still contained in the index
			//
			// this then in turn calls MetaNamespaceKeyFunc
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			log.Infof("Delete Azure Function: %s", key)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	ingress := os.Getenv("INGRESS")
	mesh := os.Getenv("MESH")

	// construct the Controller object which has all of the necessary components to
	// handle logging, connections, informing (listing and watching), the queue,
	// and the handler

	controller := Controller{
		logger:    log.NewEntry(log.New()),
		clientset: client,
		informer:  informer,
		queue:     queue,
		handler: &AzureFunctionsHandler{
			Ingress:         ingress,
			Mesh:            mesh,
			FunctionsClient: azureFuncsClient,
		},
	}

	controller.handler.Init()

	// use a channel to synchronize the finalization for a graceful shutdown
	stopCh := make(chan struct{})
	defer close(stopCh)

	// run the controller loop to process items
	go controller.Run(stopCh)

	// use a channel to handle OS signals to terminate and gracefully shut
	// down processing
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	signal.Notify(sigTerm, syscall.SIGINT)
	<-sigTerm
}
