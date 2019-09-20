package utils

import (
	"sort"

	aciv1 "github.com/noironetworks/snat-operator/pkg/apis/aci/v1"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var UtilLog = logf.Log.WithName("Utils:")

// StartSorter sorts PortRanges based on Start field.
type StartSorter []aciv1.PortRange

func (a StartSorter) Len() int           { return len(a) }
func (a StartSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a StartSorter) Less(i, j int) bool { return a[i].Start < a[j].Start }

// Given generic list of start and end of each port range,
// return sorted array(based on start of the range) of portranges based on number of per node
func ExpandPortRanges(currPortRange []aciv1.PortRange, step int) []aciv1.PortRange {

	UtilLog.V(1).Info("Inside ExpandPortRanges", "currPortRange: ", currPortRange, "step: ", step)
	expandedPortRange := []aciv1.PortRange{}
	for _, item := range currPortRange {
		temp := item.Start
		for temp < item.End-1 {
			expandedPortRange = append(expandedPortRange, aciv1.PortRange{Start: temp, End: temp + step - 1})
			temp = temp + step
		}
	}

	// Sort based on `Start` field
	sort.Sort(StartSorter(expandedPortRange))

	return expandedPortRange
}

func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func Remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
func MatchLabels(policylabels map[string]string, reslabels map[string]string) bool {
	if len(policylabels) == 0 {
		return false
	}
	for key, value := range policylabels {
		if _, ok := reslabels[key]; ok {
			if value != reslabels[key] {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
