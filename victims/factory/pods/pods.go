package pods

import (
	"github.com/asobti/kube-monkey/victims"
	"github.com/golang/glog"
	"fmt"
)

func EligiblePods(clientset kube.Interface, namespace string, filter *metav1.ListOptions) (eligVictims []victims.Victim, err error) {
	enabledVictims, err := clientset.AppsV1().Pods(namespace).List(*filter)
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
