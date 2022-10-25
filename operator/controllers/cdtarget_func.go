/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	cnadv1alpha1 "github.com/bartvanbenthem/cdtarget-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *CDTargetReconciler) networkPolicyForCDTarget(t *cnadv1alpha1.CDTarget, portList []int32) *netv1.NetworkPolicy {
	ls := labelsForCDTarget(t.Name)
	peers := peersForCDTarget(t.Spec.IP)

	if len(portList) == 0 {
		portList = append(portList, 1)
		log.Printf("%v", portList)
	}
	ports := portsForCDTarget(portList)

	net := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: t.Namespace,
			Labels:    ls,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: t.Spec.PodSelector,
			},
			Egress: []netv1.NetworkPolicyEgressRule{{
				Ports: ports,
				To:    peers,
			}},
		},
	}

	// Set CDTarget instance as the owner and controller
	ctrl.SetControllerReference(t, &net, r.Scheme)

	return &net

}

func peersForCDTarget(list []string) []netv1.NetworkPolicyPeer {
	var peers []netv1.NetworkPolicyPeer

	for _, ip := range list {
		target := fmt.Sprintf("%s/32", ip)
		peers = append(peers, netv1.NetworkPolicyPeer{
			IPBlock: &netv1.IPBlock{
				CIDR: target}})
	}

	return peers
}

func getPortsFromConfigMap(configmap v1.ConfigMap) ([]int32, error) {
	var iList []int32
	portList := configmap.Data["ports"]
	ports := strings.Split(portList, "\n")

	for _, p := range ports {
		i, err := strconv.Atoi(p)
		if err != nil {
			return iList, err
		}
		iList = append(iList, int32(i))
	}

	return iList, nil
}

func portsForCDTarget(list []int32) []netv1.NetworkPolicyPort {
	tcp := v1.ProtocolTCP
	udp := v1.ProtocolUDP
	var ports []netv1.NetworkPolicyPort

	for _, p := range list {
		ports = append(ports, netv1.NetworkPolicyPort{
			Port:     &intstr.IntOrString{IntVal: p},
			Protocol: &tcp})
		ports = append(ports, netv1.NetworkPolicyPort{
			Port:     &intstr.IntOrString{IntVal: p},
			Protocol: &udp})
	}

	return ports
}

func labelsForCDTarget(name string) map[string]string {
	return map[string]string{"name": name}
}
