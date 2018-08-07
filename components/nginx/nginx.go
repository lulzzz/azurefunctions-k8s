package nginx

import (
	"io/ioutil"
	"strconv"

	"github.com/yaron2/azfuncs/components"
	"github.com/yaron2/azfuncs/components/utils"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const nginxIngressMandatoryTemplate = "https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/mandatory.yaml"
const nginxIngressGenericTemplate = "https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/provider/cloud-generic.yaml"

type NginxIngressComponent struct{}

func (n *NginxIngressComponent) Install() (components.Component, error) {
	templates, err := n.getNginxTemplates()
	if err != nil {
		return nil, err
	}

	for i, t := range templates {
		d := []byte(t)
		fileName := strconv.Itoa(i) + ".yaml"
		err := ioutil.WriteFile(fileName, d, 0644)
		if err != nil {
			return nil, err
		}

		args := []string{
			"create",
			"-f",
			"./" + fileName,
		}

		utils.RunCMD("kubectl", args)
	}

	return n, nil
}

func (n *NginxIngressComponent) Namespace() string {
	return "ingress-nginx"
}

func (n *NginxIngressComponent) ServiceName() string {
	return "ingress-nginx"
}

func (n *NginxIngressComponent) IsRunning() (bool, error) {
	clientSet := utils.GetKubeClient()
	namespace := n.Namespace()

	pods, err := clientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	if len(pods.Items) == 0 {
		return false, nil
	}

	controller := pods.Items[0]
	return controller.Status.Phase == v1.PodRunning, nil
}

func (n *NginxIngressComponent) getNginxTemplates() ([]string, error) {
	templates := []string{}

	mandatoryStr, err := utils.GetYAMLStringFromURL(nginxIngressMandatoryTemplate)
	if err != nil {
		return nil, err
	}

	genericStr, err := utils.GetYAMLStringFromURL(nginxIngressGenericTemplate)
	if err != nil {
		return nil, err
	}

	templates = append(templates, mandatoryStr)
	templates = append(templates, genericStr)

	return templates, nil
}
