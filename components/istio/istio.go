package istio

import (
	"github.com/mholt/archiver"
	"github.com/yaron2/azfuncs/components"
	"github.com/yaron2/azfuncs/components/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const releaseURL = "https://github.com/istio/istio/releases/download/1.0.0/istio-1.0.0-linux.tar.gz"

type IstioComponent struct{}

func (i *IstioComponent) Install() (components.Component, error) {
	err := i.downloadAndExtractIstio()
	if err != nil {
		return nil, err
	}

	crdDirPath := "istio-1.0.0/install/kubernetes/helm/istio/templates/crds.yaml"
	istioPath := "istio-1.0.0/install/kubernetes/istio-demo.yaml"

	utils.RunCMD("kubectl", []string{"apply", "-f", crdDirPath, "-n", "istio-system"})
	utils.RunCMD("kubectl", []string{"apply", "-f", istioPath, "-n", "istio-system"})

	return i, nil
}

func (i *IstioComponent) Namespace() string {
	return "istio-system"
}

func (i *IstioComponent) IsRunning() (bool, error) {
	clientSet := utils.GetKubeClient()
	namespace := i.Namespace()

	pilotDeployment, err := clientSet.AppsV1().Deployments(namespace).Get("istio-pilot", metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if pilotDeployment == nil {
		return false, nil
	}

	return pilotDeployment.Status.AvailableReplicas == 1, nil
}

func (i IstioComponent) downloadAndExtractIstio() error {
	filePath := "istio.tar.gz"
	err := utils.DownloadFile(filePath, releaseURL)
	if err != nil {
		return err
	}

	err = archiver.TarGz.Open(filePath, "")
	if err != nil {
		return err
	}

	return nil
}
