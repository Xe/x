package flydns

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

func GetAppAddresses(appName string) ([]netip.Addr, error) {
	addrs, err := net.LookupAddr(fmt.Sprintf("%s.internal", appName))
	if err != nil {
		return nil, err
	}

	var result []netip.Addr

	for _, addr := range addrs {
		ip, err := netip.ParseAddr(addr)
		if err != nil {
			return nil, err
		}

		result = append(result, ip)
	}

	return result, nil
}

func GetIndividualInstanceAddress(instanceID string, appName string) (netip.Addr, error) {
	var ip netip.Addr
	addr, err := net.LookupAddr(fmt.Sprintf("%s.vm.%s.internal", instanceID, appName))
	if err != nil {
		return ip, err
	}

	ip, err = netip.ParseAddr(addr[0])
	if err != nil {
		return ip, err
	}

	return ip, nil
}

func GetAppInstancesInRegion(appName string, region string) ([]netip.Addr, error) {
	instances, err := net.LookupHost(fmt.Sprintf("%s.%s.internal", appName, region))
	if err != nil {
		return nil, err
	}

	var result []netip.Addr

	for _, instance := range instances {
		ip, err := netip.ParseAddr(instance)
		if err != nil {
			return nil, err
		}

		result = append(result, ip)
	}

	return result, nil
}

func GetClosestInstances(appName string, count int) ([]netip.Addr, error) {
	instances, err := net.LookupHost(fmt.Sprintf("top%d.nearest.of.%s.internal", count, appName))
	if err != nil {
		return nil, err
	}

	var result []netip.Addr

	for _, instance := range instances {
		ip, err := netip.ParseAddr(instance)
		if err != nil {
			return nil, err
		}

		result = append(result, ip)
	}

	return result, nil
}

func GetApps() ([]string, error) {
	appString, err := net.LookupTXT("_apps.internal")
	if err != nil {
		return nil, err
	}

	var result []string
	for _, resp := range appString {
		result = append(result, strings.Split(resp, ",")...)
	}

	return result, nil
}
