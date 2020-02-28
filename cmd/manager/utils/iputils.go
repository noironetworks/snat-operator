// Copyright 2019 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"
	"fmt"
	"net"
	"sort"
)

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Given CIDR, this function returns list of IP addresses in that CIDR
// It does omit network and host reserved IP address from that.
func GetIPsFromCIDR(cidr string) []string {

	var output []string

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		ip_temp := net.ParseIP(cidr)
		if ip_temp != nil && ip_temp.To4() != nil {
			cidr = cidr + "/32"
			ip, ipnet, _ = net.ParseCIDR(cidr)
		} else if ip_temp != nil && ip_temp.To16() != nil {
			cidr = cidr + "/128"
			ip, ipnet, _ = net.ParseCIDR(cidr)
		} else {
			UtilLog.Error(err, "Invalid CIDR, skipping this one: "+cidr)
			return output
		}
	}

	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		output = append(output, ip.String())
	}
	return output[0:len(output)]
}

// Given two slices of IP addresses, finds the difference of two sets
func SetDifferenceIPs(inputIps []string, usedIps []string) []string {

	var output []string

	return output
}

// Draw free IP address from the given list
func DrawIP(freeIps []string) {
	sort.Strings(freeIps)
}

// Given generic list of CIDRs for subnets
// return sorted array(based on IP address) of IP addresses.
func ExpandCIDRs(currCIDRs []string) []string {

	var expandedIPs []string
	for _, item := range currCIDRs {
		ips := GetIPsFromCIDR(item)
		expandedIPs = append(expandedIPs, ips...)
	}

	// Sort list of IPs
	expandedIPs = sortIps(expandedIPs)
	UtilLog.V(1).Info("Inside ExpandCIDRs", "CurrCIDRs:", expandedIPs)

	return expandedIPs
}

// This function sorts IP by parsing them to net.IP struct. String sort does not work
// eg: 10.0.0.9 should come before 10.0.0.10
func sortIps(ips []string) []string {
	realIPs := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		realIPs = append(realIPs, net.ParseIP(ip))
	}

	sort.Slice(realIPs, func(i, j int) bool { return bytes.Compare(realIPs[i], realIPs[j]) < 0 })

	outputIps := make([]string, 0, len(realIPs))
	for _, ip := range realIPs {
		outputIps = append(outputIps, fmt.Sprintf("%s", ip))
	}
	return outputIps
}
