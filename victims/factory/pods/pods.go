package pods

import (
	"fmt"
	"strconv"

	"github.com/asobti/kube-monkey/victims"
	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/config"

	"k8s.io/api/core/v1"
	kube "k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func EligiblePods(clientset kube.Interface, namespace string, filter *metav1.ListOptions) (eligVictims []victims.Victim, err error) {
	enabledVictims, err := clientset.CoreV1().Pods(namespace).List(*filter)
	if err != nil {
		return nil, err
	}

	for _, vic := range enabledVictims.Items {
		victim, err := New(&vic)
		if err != nil {
			glog.Warningf("Skipping eligible %T %s because of error: %s", vic, vic.Name, err.Error())
			continue
		}

		// TODO: After generating whitelisting ns list, this will move to factory.
		// IsBlacklisted will change to something like IsAllowedNamespace
		// and will only be used to verify at time of scheduled execution
		if victim.IsBlacklisted() {
			continue
		}

		eligVictims = append(eligVictims, victim)
	}

	return
}

type Pod struct {
	*victims.VictimBase
}

// Create a new instance of Deployment
func New(dep *v1.Pod) (*Pod, error) {
	ident, err := identifier(dep)
	if err != nil {
		return nil, err
	}
	mtbf, err := meanTimeBetweenFailures(dep)
	if err != nil {
		return nil, err
	}
	kind := fmt.Sprintf("%T", *dep)

	return &Pod{victims.New(kind, dep.Name, dep.Namespace, ident, mtbf)}, nil
}

// Checks if the deployment is currently enrolled in kube-monkey
func (p *Pod) IsEnrolled(clientset kube.Interface) (bool, error) {
	pod, err := clientset.CoreV1().Pods(p.Namespace()).Get(p.Name(), metav1.GetOptions{})
	//.Pods(d.Namespace()).Get(d.Name(), metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return pod.Labels[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

// Returns current killtype config label for update
func (p *Pod) KillType(clientset kube.Interface) (string, error) {
	pod, err := clientset.CoreV1().Pods(p.Namespace()).Get(p.Name(), metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	killType, ok := pod.Labels[config.KillTypeLabelKey]
	if !ok {
		return "", fmt.Errorf("%s %s does not have %s label", p.Kind(), p.Name(), config.KillTypeLabelKey)
	}

	return killType, nil
}

// Returns current killvalue config label for update
func (p *Pod) KillValue(clientset kube.Interface) (int, error) {
	pod, err := clientset.CoreV1().Pods(p.Namespace()).Get(p.Name(), metav1.GetOptions{})
	if err != nil {
		return -1, err
	}

	killMode, ok := pod.Labels[config.KillValueLabelKey]
	if !ok {
		return -1, fmt.Errorf("%s %s does not have %s label", p.Kind(), p.Name(), config.KillValueLabelKey)
	}

	killModeInt, err := strconv.Atoi(killMode)
	if !(killModeInt > 0) {
		return -1, fmt.Errorf("Invalid value for label %s: %d", config.KillValueLabelKey, killModeInt)
	}

	return killModeInt, nil
}
